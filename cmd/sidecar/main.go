// Command sidecar runs a localhost HTTP proxy that exposes an OpenAI-compatible REST API.
// It translates incoming Python/Node.js JSON requests into ECDSA-signed mTLS Zero-Trust Swarm messages.
// It also acts as an Ingress reverse-proxy to allow Python agents to serve capabilities.
package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/hamza-imran/zero-trust-swarm/pkg/idp"
	"github.com/hamza-imran/zero-trust-swarm/pkg/policy"
	"github.com/hamza-imran/zero-trust-swarm/pkg/protocol"
	"github.com/hamza-imran/zero-trust-swarm/pkg/queue"
	"github.com/hamza-imran/zero-trust-swarm/pkg/registry"
	"github.com/hamza-imran/zero-trust-swarm/pkg/transport"
)

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model     string          `json:"model"` // We map this to the Target Swarm Capability!
	Messages  []OpenAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
}

type Sidecar struct {
	AgentID      string
	SwarmTLS     *transport.SwarmTLS
	PrivKey      *ecdsa.PrivateKey
	RegistryAddr string
	TaskQueue    *queue.TaskQueue
	UpstreamURL  string
	PolicyEngine *policy.Engine
}

func main() {
	id := flag.String("id", "agent-sidecar", "Sidecar identity (attested by IdP)")
	listen := flag.String("listen", "127.0.0.1:8080", "HTTP port to expose OpenAI-compatible API")
	registryAddr := flag.String("registry", "127.0.0.1:9000", "Discovery registry address")
	listenMtls := flag.String("listen-mtls", "", "Port to listen for incoming secure Swarm connections (e.g. 127.0.0.1:8081)")
	upstream := flag.String("upstream", "http://127.0.0.1:5000/webhook", "Local HTTP URL of the Python agent for incoming tasks")
	capsFlag := flag.String("caps", "", "Comma-separated capabilities to register with the Swarm")
	flag.Parse()

	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║ Zero-Trust Swarm Protocol — Local Sidecar Proxy║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Printf("  Sidecar ID:    %s\n", *id)
	fmt.Printf("  OpenAI API:    http://%s/v1/chat/completions\n\n", *listen)

	// 1. Fetch secure in-memory identity from IdP daemon
	fmt.Println("[1/2] Connecting to Workload Identity Provider (IdP)...")
	identity, err := idp.FetchIdentity(*id)
	if err != nil {
		log.Fatalf("❌ Failed to fetch dynamic credentials: %v", err)
	}

	cert, err := tls.X509KeyPair([]byte(identity.CertPEM), []byte(identity.KeyPEM))
	if err != nil {
		log.Fatalf("❌ Failed to parse X509 keypair: %v", err)
	}

	privKeyAny := cert.PrivateKey
	ecdsaPrivKey, ok := privKeyAny.(*ecdsa.PrivateKey)
	if !ok {
		log.Fatalf("❌ Private key is not ECDSA")
	}

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM([]byte(identity.CAPEM))

	swarmTLS := &transport.SwarmTLS{
		AgentID: *id,
		TLSConfig: &tls.Config{
			Certificates:       []tls.Certificate{cert},
			ClientCAs:          caPool,
			RootCAs:            caPool,
			ClientAuth:         tls.RequireAndVerifyClientCert,
			MinVersion:         tls.VersionTLS13,
			InsecureSkipVerify: true,
			VerifyConnection: func(cs tls.ConnectionState) error {
				if len(cs.PeerCertificates) == 0 {
					return fmt.Errorf("no peer certificates presented")
				}
				peerCert := cs.PeerCertificates[0]
				opts := x509.VerifyOptions{
					Roots:         caPool,
					Intermediates: x509.NewCertPool(),
					KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
				}
				if _, err := peerCert.Verify(opts); err != nil {
					return fmt.Errorf("certificate verification failed: %w", err)
				}
				return nil
			},
		},
		CACertPool: caPool,
	}

	// Initialize Persistent Task Queue
	q, err := queue.NewTaskQueue("sidecar_queue.db")
	if err != nil {
		log.Fatalf("❌ Failed to initialize SQLite task queue: %v", err)
	}
	defer q.Close()

	engine := policy.NewEngine([]policy.Rule{
		{
			Effect:     policy.Allow,
			Principals: []string{"spiffe://swarm.local/agent/*"}, // Allow registered agents to connect
			Actions:    []protocol.MessageType{protocol.TypeTask, protocol.TypePing, protocol.TypeResult},
		},
	})

	s := &Sidecar{
		AgentID:      *id,
		SwarmTLS:     swarmTLS,
		PrivKey:      ecdsaPrivKey,
		RegistryAddr: *registryAddr,
		TaskQueue:    q,
		UpstreamURL:  *upstream,
		PolicyEngine: engine,
	}

	go s.retryQueueLoop()

	if *listenMtls != "" {
		fmt.Printf("[2/2]👂 Sidecar Ingress mTLS listening on %s -> forwarding to %s\n", *listenMtls, *upstream)
		go s.runIngressListener(*listenMtls)

		if *capsFlag != "" {
			caps := strings.Split(*capsFlag, ",")
			for i := range caps {
				caps[i] = strings.TrimSpace(caps[i])
			}
			go s.runHeartbeatLoop(*listenMtls, caps)
		}
	}

	http.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	http.HandleFunc("/v1/tasks/async", s.handleAsyncTask)

	fmt.Printf("[2/2]👂 Sidecar Egress Proxy running on http://%s\n", *listen)
	log.Fatal(http.ListenAndServe(*listen, nil))
}

func (s *Sidecar) runIngressListener(addr string) {
	ln, err := s.SwarmTLS.Listen(addr)
	if err != nil {
		log.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go s.handleIncomingSwarmConnection(conn)
	}
}

func (s *Sidecar) handleIncomingSwarmConnection(conn net.Conn) {
	defer conn.Close()
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return
	}

	if err := tlsConn.Handshake(); err != nil {
		log.Printf("❌ TLS handshake failed: %v", err)
		return
	}

	peerID := transport.ExtractPeerID(tlsConn)

	msg, err := protocol.ReadMessage(conn)
	if err != nil {
		log.Printf("Read error: %v", err)
		return
	}
	
	if len(tlsConn.ConnectionState().PeerCertificates) > 0 {
		peerPubKey, ok := tlsConn.ConnectionState().PeerCertificates[0].PublicKey.(*ecdsa.PublicKey)
		if ok {
			if err := msg.Verify(peerPubKey); err != nil {
				log.Printf("❌ [INTEGRITY FAILURE] Payload signature invalid from %s: %v", peerID, err)
				return
			}
		}
	}

	if !s.PolicyEngine.Evaluate(peerID, msg.Type) {
		log.Printf("⛔ [POLICY DENY] Action %s denied for principal %s", msg.Type, peerID)
		reply := protocol.NewMessage(protocol.TypeResult, s.AgentID, msg.From, "⛔ POLICY DENY: Unauthorized")
		reply.Sign(s.PrivKey)
		protocol.WriteMessage(conn, reply)
		return
	}

	var reply *protocol.Message
	switch msg.Type {
	case protocol.TypeTask:
		// Forward task payload to Python Upstream
		reqBody, _ := json.Marshal(map[string]string{
			"sender":  msg.From,
			"payload": msg.Payload,
		})
		
		httpResp, err := http.Post(s.UpstreamURL, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			log.Printf("⚠️ Failed to forward task to upstream %s: %v", s.UpstreamURL, err)
			reply = protocol.NewMessage(protocol.TypeResult, s.AgentID, msg.From, "ERROR: Python agent unreachable")
		} else {
			defer httpResp.Body.Close()
			bodyBytes, _ := io.ReadAll(httpResp.Body)
			reply = protocol.NewMessage(protocol.TypeResult, s.AgentID, msg.From, string(bodyBytes))
		}
	case protocol.TypePing:
		reply = protocol.NewMessage(protocol.TypePong, s.AgentID, msg.From, "PONG")
	default:
		reply = protocol.NewMessage(protocol.TypeResult, s.AgentID, msg.From, "Unsupported message type")
	}

	reply.Sign(s.PrivKey)
	protocol.WriteMessage(conn, reply)
}

func (s *Sidecar) runHeartbeatLoop(listenAddr string, caps []string) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	register := func() {
		conn, err := s.SwarmTLS.Dial(s.RegistryAddr)
		if err != nil {
			return
		}
		defer conn.Close()

		rec := registry.AgentRecord{
			AgentID:      s.AgentID,
			Address:      listenAddr,
			Capabilities: caps,
		}
		recBytes, _ := json.Marshal(rec)

		msg := protocol.NewMessage(protocol.TypeRegister, s.AgentID, "swarm-registry", string(recBytes))
		msg.Sign(s.PrivKey)
		protocol.WriteMessage(conn, msg)
	}

	register()
	for range ticker.C {
		register()
	}
}

func (s *Sidecar) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OpenAIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Messages) == 0 {
		http.Error(w, "Messages cannot be empty", http.StatusBadRequest)
		return
	}

	targetCapability := req.Model
	if targetCapability == "" {
		http.Error(w, "Model (capability) must be specified", http.StatusBadRequest)
		return
	}

	taskPayload := req.Messages[len(req.Messages)-1].Content

	conn, err := s.SwarmTLS.Dial(s.RegistryAddr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect to registry: %v", err), http.StatusInternalServerError)
		return
	}
	
	queryMsg := protocol.NewMessage(protocol.TypeQuery, s.AgentID, "swarm-registry", targetCapability)
	queryMsg.Sign(s.PrivKey)
	protocol.WriteMessage(conn, queryMsg)

	resp, err := protocol.ReadMessage(conn)
	conn.Close()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read registry response: %v", err), http.StatusInternalServerError)
		return
	}

	var qResp registry.QueryResponse
	json.Unmarshal([]byte(resp.Payload), &qResp)

	if len(qResp.Agents) == 0 {
		http.Error(w, fmt.Sprintf("No active swarm agents found with capability: %s", targetCapability), http.StatusNotFound)
		return
	}

	target := qResp.Agents[0]

	peerConn, err := s.SwarmTLS.Dial(target.Address)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect to peer %s: %v", target.AgentID, err), http.StatusInternalServerError)
		return
	}
	defer peerConn.Close()

	taskMsg := protocol.NewMessage(protocol.TypeTask, s.AgentID, target.AgentID, taskPayload)
	taskMsg.MaxTokens = req.MaxTokens
	taskMsg.Sign(s.PrivKey)
	protocol.WriteMessage(peerConn, taskMsg)

	resultMsg, err := protocol.ReadMessage(peerConn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read result from peer: %v", err), http.StatusInternalServerError)
		return
	}

	aiResp := OpenAIResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   targetCapability,
		Choices: []OpenAIChoice{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: resultMsg.Payload,
				},
				FinishReason: "stop",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(aiResp)
}

type AsyncRequest struct {
	TargetID string `json:"target_id"`
	Payload  string `json:"payload"`
}

func (s *Sidecar) handleAsyncTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AsyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msg := protocol.NewMessage(protocol.TypeTask, s.AgentID, req.TargetID, req.Payload)
	msg.Sign(s.PrivKey)

	if err := s.TaskQueue.Enqueue(req.TargetID, *msg); err != nil {
		http.Error(w, "Failed to enqueue task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "queued", "target": req.TargetID})
}

func (s *Sidecar) retryQueueLoop() {
	ticker := time.NewTicker(15 * time.Second)
	for range ticker.C {
		msgs, err := s.TaskQueue.FetchAll()
		if err != nil || len(msgs) == 0 {
			continue
		}

		for _, qm := range msgs {
			conn, err := s.SwarmTLS.Dial(s.RegistryAddr)
			if err != nil {
				continue
			}

			queryMsg := protocol.NewMessage(protocol.TypeQuery, s.AgentID, "swarm-registry", "")
			queryMsg.Sign(s.PrivKey)
			protocol.WriteMessage(conn, queryMsg)

			resp, err := protocol.ReadMessage(conn)
			conn.Close()
			if err != nil {
				continue
			}

			var qResp registry.QueryResponse
			json.Unmarshal([]byte(resp.Payload), &qResp)

			var targetAddr string
			for _, a := range qResp.Agents {
				if a.AgentID == qm.TargetID {
					targetAddr = a.Address
					break
				}
			}

			if targetAddr == "" {
				continue
			}

			peerConn, err := s.SwarmTLS.Dial(targetAddr)
			if err != nil {
				continue
			}

			if err := protocol.WriteMessage(peerConn, &qm.Message); err == nil {
				peerConn.Close()
				s.TaskQueue.Delete(qm.ID)
			}
		}
	}
}
