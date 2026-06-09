// Command sidecar runs a localhost HTTP proxy that exposes an OpenAI-compatible REST API.
// It translates incoming Python/Node.js JSON requests into ECDSA-signed mTLS Zero-Trust Swarm messages.
package main

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hamza-imran/zero-trust-swarm/pkg/idp"
	"github.com/hamza-imran/zero-trust-swarm/pkg/protocol"
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
}

func main() {
	id := flag.String("id", "agent-sidecar", "Sidecar identity (attested by IdP)")
	listen := flag.String("listen", "127.0.0.1:8080", "HTTP port to expose OpenAI-compatible API")
	registryAddr := flag.String("registry", "127.0.0.1:9000", "Discovery registry address")
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

	s := &Sidecar{
		AgentID:      *id,
		SwarmTLS:     swarmTLS,
		PrivKey:      ecdsaPrivKey,
		RegistryAddr: *registryAddr,
	}

	http.HandleFunc("/v1/chat/completions", s.handleChatCompletions)

	fmt.Printf("[2/2]👂 Sidecar Proxy running on http://%s\n", *listen)
	log.Fatal(http.ListenAndServe(*listen, nil))
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

	// 1. The requested model is mapped directly to a Swarm Capability
	targetCapability := req.Model
	if targetCapability == "" {
		http.Error(w, "Model (capability) must be specified", http.StatusBadRequest)
		return
	}

	// The last message is the task payload
	taskPayload := req.Messages[len(req.Messages)-1].Content

	log.Printf("🤖 Received local HTTP request to invoke swarm capability: %q", targetCapability)

	// 2. Query Swarm Registry
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
	if err := json.Unmarshal([]byte(resp.Payload), &qResp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode registry response: %v", err), http.StatusInternalServerError)
		return
	}

	if len(qResp.Agents) == 0 {
		http.Error(w, fmt.Sprintf("No active swarm agents found with capability: %s", targetCapability), http.StatusNotFound)
		return
	}

	target := qResp.Agents[0]
	log.Printf("🚀 Found peer over mTLS: %s at %s", target.AgentID, target.Address)

	// 3. Dial Target Peer
	peerConn, err := s.SwarmTLS.Dial(target.Address)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect to peer %s: %v", target.AgentID, err), http.StatusInternalServerError)
		return
	}
	defer peerConn.Close()

	// 4. Wrap LLM Task & Send
	taskMsg := protocol.NewMessage(protocol.TypeTask, s.AgentID, target.AgentID, taskPayload)
	taskMsg.MaxTokens = req.MaxTokens
	taskMsg.Sign(s.PrivKey)
	protocol.WriteMessage(peerConn, taskMsg)

	// 5. Wait for Peer Result
	resultMsg, err := protocol.ReadMessage(peerConn)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read result from peer: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("🏆 Task executed successfully by %s", target.AgentID)

	// 6. Format back into OpenAI REST schema
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
