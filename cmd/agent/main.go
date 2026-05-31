// Command agent runs an AI agent that communicates over mutual TLS (mTLS).
// It bootstraps its identity dynamically from the local IdP daemon, registers
// its capabilities with the Discovery Registry, and can handle tasks concurrently.
package main

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hamza-imran/zero-trust-swarm/pkg/gossip"
	"github.com/hamza-imran/zero-trust-swarm/pkg/idp"
	"github.com/hamza-imran/zero-trust-swarm/pkg/policy"
	"github.com/hamza-imran/zero-trust-swarm/pkg/protocol"
	"github.com/hamza-imran/zero-trust-swarm/pkg/registry"
	"github.com/hamza-imran/zero-trust-swarm/pkg/telemetry"
	"github.com/hamza-imran/zero-trust-swarm/pkg/transport"

	"go.opentelemetry.io/otel"
)

func main() {
	id := flag.String("id", "", "Agent identity (attested by IdP)")
	listen := flag.String("listen", "", "Address to listen on (e.g. 127.0.0.1:9100)")
	registryAddr := flag.String("registry", "127.0.0.1:9000", "Discovery registry address")
	capsFlag := flag.String("caps", "", "Comma-separated capabilities of this agent")
	targetCap := flag.String("target-cap", "", "Query registry for an agent with this capability and send a task")
	taskPayload := flag.String("task", "Analyze this dataset for security vulnerabilities", "Task payload to send")
	flag.Parse()

	if *id == "" {
		fmt.Println("Usage: agent --id <agent-id> [--listen <host:port>] [--registry <host:port>] [--caps <cap1,cap2>] [--target-cap <capability>]")
		os.Exit(1)
	}

	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║   Zero-Trust Swarm Protocol — Autonomous Agent║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Printf("  Agent ID:      %s\n", *id)
	if *capsFlag != "" {
		fmt.Printf("  Capabilities:  %s\n", *capsFlag)
	}
	fmt.Println()

	// 1. Fetch secure in-memory identity from IdP daemon (no disk keys!)
	fmt.Println("[1/4] Connecting to Workload Identity Provider (IdP)...")
	identity, err := idp.FetchIdentity(*id)
	if err != nil {
		log.Fatalf("❌ Failed to fetch dynamic credentials: %v", err)
	}
	fmt.Println("  ✓ Dynamic in-memory credentials fetched successfully.")

	// 2. Build custom transport configuration from dynamic credentials
	cert, err := tls.X509KeyPair([]byte(identity.CertPEM), []byte(identity.KeyPEM))
	if err != nil {
		log.Fatalf("❌ Failed to parse X509 keypair: %v", err)
	}

	// Parse private key for payload signing
	privKeyAny := cert.PrivateKey
	_, ok := privKeyAny.(crypto.PrivateKey) // wait crypto.PrivateKey is an interface, let's cast directly to *ecdsa.PrivateKey
	ecdsaPrivKey, ok := privKeyAny.(*ecdsa.PrivateKey)
	if !ok {
		log.Fatalf("❌ Private key is not ECDSA")
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM([]byte(identity.CAPEM)) {
		log.Fatalf("❌ Failed to parse CA cert from IdP")
	}

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
	fmt.Println("🔐 Mutual TLS (mTLS 1.3) engine initialized in-memory.")

	// Initialize Policy Engine
	fmt.Println("🛡️  Initializing Zero-Trust Policy Engine...")
	policyEngine := policy.NewEngine([]policy.Rule{
		{
			Effect:     policy.Allow,
			Principals: []string{"spiffe://swarm.local/agent/*"}, // Only allow registered swarm agents
			Actions:    []protocol.MessageType{protocol.TypeTask, protocol.TypePing, protocol.TypeResult, protocol.TypeGossip},
		},
	})

	// Initialize OpenTelemetry
	fmt.Println("📡 Initializing OpenTelemetry Distributed Tracing...")
	tp, err := telemetry.InitTracer(*id)
	if err != nil {
		log.Fatalf("❌ Failed to init telemetry: %v", err)
	}
	defer tp.Shutdown(context.Background())

	// Initialize Gossip Engine
	gEngine := gossip.NewGossipEngine(*id, *listen, swarmTLS, ecdsaPrivKey)

	// 3. Start Secure mTLS Listener (if --listen is specified)
	if *listen != "" {
		go runListener(swarmTLS, *listen, ecdsaPrivKey, policyEngine, gEngine)

		// Initialize our own empty or capable state
		caps := []string{}
		if *capsFlag != "" {
			caps = strings.Split(*capsFlag, ",")
			for i := range caps {
				caps[i] = strings.TrimSpace(caps[i])
			}
		}
		
		gEngine.UpdateLocal(registry.AgentRecord{
			AgentID:      *id,
			Address:      *listen,
			Capabilities: caps,
		})
		gEngine.Start() // Start periodic background gossip loop

		// 4. Start heartbeat/registration loop with secure discovery registry
		if *registryAddr != "" && *capsFlag != "" {
			go runHeartbeatLoop(swarmTLS, *registryAddr, *listen, caps, ecdsaPrivKey)
		}
	}

	// 5. Query and execute a task on a peer (if --target-cap is specified)
	if *targetCap != "" {
		// Wait a moment for any startup/registration to settle in the demo
		time.Sleep(2 * time.Second)
		executeTaskOnPeer(swarmTLS, *registryAddr, *targetCap, *taskPayload, ecdsaPrivKey, gEngine)
	}

	// Wait for termination signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	fmt.Println("\n🛑 Shutting down Agent...")
}

func runListener(swarmTLS *transport.SwarmTLS, addr string, privKey *ecdsa.PrivateKey, engine *policy.Engine, gEngine *gossip.GossipEngine) {
	ln, err := swarmTLS.Listen(addr)
	if err != nil {
		log.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close()
	fmt.Printf("[2/4]👂 Listening for swarm peer connections on %s\n", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleConnection(swarmTLS, conn, privKey, engine, gEngine)
	}
}

func handleConnection(swarmTLS *transport.SwarmTLS, conn net.Conn, privKey *ecdsa.PrivateKey, engine *policy.Engine, gEngine *gossip.GossipEngine) {
	defer conn.Close()
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		log.Printf("❌ Failed to cast connection to TLS connection")
		return
	}

	if err := tlsConn.Handshake(); err != nil {
		log.Printf("❌ TLS handshake failed: %v", err)
		return
	}

	peerID := transport.ExtractPeerID(tlsConn)

	// Read incoming message
	msg, err := protocol.ReadMessage(conn)
	if err != nil {
		log.Printf("Read error: %v", err)
		return
	}
	
	// Cryptographic Integrity Check
	if len(tlsConn.ConnectionState().PeerCertificates) > 0 {
		peerPubKey, ok := tlsConn.ConnectionState().PeerCertificates[0].PublicKey.(*ecdsa.PublicKey)
		if ok {
			if err := msg.Verify(peerPubKey); err != nil {
				log.Printf("❌ [INTEGRITY FAILURE] Payload signature invalid from %s: %v", peerID, err)
				return
			}
		}
	}

	fmt.Printf("\n📨 [mTLS RECEIVED] from %s: type=%s, payload=%q\n", peerID, msg.Type, msg.Payload)

	// OpenTelemetry Extraction
	ctx := context.Background()
	if msg.TraceID != "" && msg.SpanID != "" {
		ctx = telemetry.ExtractContext(ctx, msg.TraceID, msg.SpanID)
	}
	tracer := otel.Tracer("swarm-agent")
	_, span := tracer.Start(ctx, fmt.Sprintf("HandleConnection/%s", msg.Type))
	defer span.End()

	// Zero-Trust Policy Check
	if !engine.Evaluate(peerID, msg.Type) {
		log.Printf("⛔ [POLICY DENY] Action %s denied for principal %s", msg.Type, peerID)
		reply := protocol.NewMessage(protocol.TypeResult, swarmTLS.AgentID, msg.From, "⛔ POLICY DENY: Unauthorized")
		reply.Sign(privKey)
		protocol.WriteMessage(conn, reply)
		return
	}
	fmt.Printf("✅ [POLICY ALLOW] Action %s allowed for principal %s\n", msg.Type, peerID)

	// Perform task processing or respond
	var reply *protocol.Message
	switch msg.Type {
	case protocol.TypeTask:
		// Process task
		resultPayload := fmt.Sprintf("[AGENT %s SUCCESS] Successfully executed task: %q", swarmTLS.AgentID, msg.Payload)
		reply = protocol.NewMessage(protocol.TypeResult, swarmTLS.AgentID, msg.From, resultPayload)
	case protocol.TypeGossip:
		var incoming map[string]registry.AgentRecord
		if err := json.Unmarshal([]byte(msg.Payload), &incoming); err == nil {
			gEngine.MergeState(incoming)
		}
		reply = protocol.NewMessage(protocol.TypeResult, swarmTLS.AgentID, msg.From, "GOSSIP_ACK")
	case protocol.TypePing:
		reply = protocol.NewMessage(protocol.TypePong, swarmTLS.AgentID, msg.From, "PONG")
	default:
		reply = protocol.NewMessage(protocol.TypeResult, swarmTLS.AgentID, msg.From, "Received standard message.")
	}

	reply.Sign(privKey)
	if err := protocol.WriteMessage(conn, reply); err != nil {
		log.Printf("Write error: %v", err)
	}
	fmt.Printf("📤 [mTLS RESPONSE] sent to %s\n", peerID)
}

func runHeartbeatLoop(swarmTLS *transport.SwarmTLS, regAddr, listenAddr string, caps []string, privKey *ecdsa.PrivateKey) {
	fmt.Printf("[3/4]📝 Starting secure registry heartbeat loop with registry %s...\n", regAddr)
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	// Initial registration
	registerWithRegistry(swarmTLS, regAddr, listenAddr, caps, privKey)

	for range ticker.C {
		registerWithRegistry(swarmTLS, regAddr, listenAddr, caps, privKey)
	}
}

func registerWithRegistry(swarmTLS *transport.SwarmTLS, regAddr, listenAddr string, caps []string, privKey *ecdsa.PrivateKey) {
	conn, err := swarmTLS.Dial(regAddr)
	if err != nil {
		log.Printf("⚠️ Registry heartbeat failed (could not connect): %v", err)
		return
	}
	defer conn.Close()

	rec := registry.AgentRecord{
		AgentID:      swarmTLS.AgentID,
		Address:      listenAddr,
		Capabilities: caps,
	}
	recBytes, _ := json.Marshal(rec)

	msg := protocol.NewMessage(protocol.TypeRegister, swarmTLS.AgentID, "swarm-registry", string(recBytes))
	msg.Sign(privKey) // Cryptographic Integrity Check
	if err := protocol.WriteMessage(conn, msg); err != nil {
		log.Printf("⚠️ Registry heartbeat failed (write error): %v", err)
		return
	}

	resp, err := protocol.ReadMessage(conn)
	if err != nil || resp.Payload != "SUCCESS" {
		log.Printf("⚠️ Registry registration rejected: %v", err)
	}
}

func executeTaskOnPeer(swarmTLS *transport.SwarmTLS, regAddr, capName, task string, privKey *ecdsa.PrivateKey, gEngine *gossip.GossipEngine) {
	fmt.Printf("[4/4]🔍 Querying Registry for capability: %q\n", capName)

	conn, err := swarmTLS.Dial(regAddr)
	if err != nil {
		log.Fatalf("❌ Failed to connect to registry: %v", err)
	}
	defer conn.Close()

	// Send query message
	queryMsg := protocol.NewMessage(protocol.TypeQuery, swarmTLS.AgentID, "swarm-registry", capName)
	queryMsg.Sign(privKey)
	if err := protocol.WriteMessage(conn, queryMsg); err != nil {
		log.Fatalf("❌ Failed to send query to registry: %v", err)
	}

	// Read response
	resp, err := protocol.ReadMessage(conn)
	if err != nil {
		log.Fatalf("❌ Failed to read query response: %v", err)
	}

	var qResp registry.QueryResponse
	if err := json.Unmarshal([]byte(resp.Payload), &qResp); err != nil {
		log.Fatalf("❌ Failed to parse query response: %v", err)
	}

	if len(qResp.Agents) == 0 {
		fmt.Printf("⚠️ No active agents found matching capability: %q\n", capName)
		return
	}

	// Select the first agent
	target := qResp.Agents[0]
	fmt.Printf("🚀 Found peer: %s at %s. Dispatching task over secure mTLS...\n", target.AgentID, target.Address)

	// Add discovered peer to our gossip state
	gEngine.UpdateLocal(*target)

	// Dial target agent
	peerConn, err := swarmTLS.Dial(target.Address)
	if err != nil {
		log.Fatalf("❌ Failed to connect to peer %s: %v", target.AgentID, err)
	}
	defer peerConn.Close()

	// OpenTelemetry Injection
	tracer := otel.Tracer("swarm-agent")
	_, span := tracer.Start(context.Background(), "ExecuteTaskOnPeer")
	defer span.End()

	taskMsg := protocol.NewMessage(protocol.TypeTask, swarmTLS.AgentID, target.AgentID, task)
	taskMsg.TraceID = span.SpanContext().TraceID().String()
	taskMsg.SpanID = span.SpanContext().SpanID().String()
	taskMsg.Sign(privKey) // Sign the task payload
	if err := protocol.WriteMessage(peerConn, taskMsg); err != nil {
		log.Fatalf("❌ Failed to send task to peer: %v", err)
	}
	fmt.Printf("📤 Sent task to peer: %q\n", task)

	// Read result
	resultMsg, err := protocol.ReadMessage(peerConn)
	if err != nil {
		log.Fatalf("❌ Failed to read task result from peer: %v", err)
	}

	fmt.Println("\n════════════════════════════════════════════════════════")
	fmt.Printf("  🏆 TASK EXECUTED SUCCESSFULLY OVER ZERO-TRUST SWARM!\n")
	fmt.Printf("  Peer ID:   %s\n", target.AgentID)
	fmt.Printf("  Response:  %s\n", resultMsg.Payload)
	fmt.Println("════════════════════════════════════════════════════════")
}
