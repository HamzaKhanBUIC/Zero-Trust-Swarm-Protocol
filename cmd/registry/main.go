// Command registry runs the zero-trust secure swarm registry service.
// It connects to the local IdP daemon to bootstrap its own mTLS certificate,
// then listens for secure mTLS connections from agents to handle swarm discovery.
package main

import (
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
	"syscall"

	"github.com/hamza-imran/zero-trust-swarm/pkg/idp"
	"github.com/hamza-imran/zero-trust-swarm/pkg/policy"
	"github.com/hamza-imran/zero-trust-swarm/pkg/protocol"
	"github.com/hamza-imran/zero-trust-swarm/pkg/registry"
	"github.com/hamza-imran/zero-trust-swarm/pkg/transport"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:9000", "Registry service listen address")
	flag.Parse()

	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║   Zero-Trust Swarm Secure Discovery Registry ║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()

	// 1. Fetch registry's own credential dynamically from the local Workload Identity API
	fmt.Println("🔑 Bootstrapping registry credential from IdP...")
	resp, err := idp.FetchIdentity("swarm-registry")
	if err != nil {
		log.Fatalf("Bootstrapping failed: %v", err)
	}
	fmt.Println("  ✓ Dynamic in-memory credential retrieved.")

	// 2. Build custom transport configuration from retrieved credentials
	cert, err := tls.X509KeyPair([]byte(resp.CertPEM), []byte(resp.KeyPEM))
	if err != nil {
		log.Fatalf("Failed to parse X509 keypair: %v", err)
	}

	// Parse private key for payload signing
	privKeyAny := cert.PrivateKey
	_, ok := privKeyAny.(crypto.PrivateKey)
	ecdsaPrivKey, ok := privKeyAny.(*ecdsa.PrivateKey)
	if !ok {
		log.Fatalf("❌ Private key is not ECDSA")
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM([]byte(resp.CAPEM)) {
		log.Fatalf("Failed to parse CA cert from IdP")
	}

	swarmTLS := &transport.SwarmTLS{
		AgentID: "swarm-registry",
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientCAs:    caPool,
			RootCAs:      caPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS13,
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

	// Initialize Policy Engine for Registry
	fmt.Println("🛡️  Initializing Zero-Trust Policy Engine for Registry...")
	policyEngine := policy.NewEngine([]policy.Rule{
		{
			Effect:     policy.Allow,
			Principals: []string{"spiffe://swarm.local/agent/*"},
			Actions:    []protocol.MessageType{protocol.TypeRegister, protocol.TypeQuery},
		},
	})

	// 3. Start Swarm Registry service
	reg := registry.NewSwarmRegistry()
	ln, err := swarmTLS.Listen(*listen)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", *listen, err)
	}
	defer ln.Close()
	fmt.Printf("👂 Discovery Registry secure server listening on %s (mTLS strictly enforced)\n\n", *listen)

	// Graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("\n🛑 Shutting down Registry...")
		ln.Close()
		os.Exit(0)
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			break
		}
		go handleRegistryClient(conn, swarmTLS, reg, ecdsaPrivKey, policyEngine)
	}
}

func handleRegistryClient(conn net.Conn, swarmTLS *transport.SwarmTLS, reg *registry.SwarmRegistry, privKey *ecdsa.PrivateKey, engine *policy.Engine) {
	defer conn.Close()
	tlsConn := conn.(*tls.Conn)

	if err := tlsConn.Handshake(); err != nil {
		log.Printf("❌ Handshake failed: %v", err)
		return
	}

	peerID := transport.ExtractPeerID(tlsConn)

	// Read message
	msg, err := protocol.ReadMessage(conn)
	if err != nil {
		log.Printf("Read error from peer %s: %v", peerID, err)
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

	// Zero-Trust Policy Check
	if !engine.Evaluate(peerID, msg.Type) {
		log.Printf("⛔ [POLICY DENY] Action %s denied for principal %s", msg.Type, peerID)
		reply := protocol.NewMessage(protocol.TypeResult, "swarm-registry", peerID, "⛔ POLICY DENY: Unauthorized")
		reply.Sign(privKey)
		protocol.WriteMessage(conn, reply)
		return
	}

	switch msg.Type {
	case protocol.TypeRegister:
		// Message payload contains Capability/Address config
		var rec registry.AgentRecord
		if err := json.Unmarshal([]byte(msg.Payload), &rec); err != nil {
			log.Printf("❌ Failed to parse registration from %s: %v", peerID, err)
			return
		}
		reg.Register(rec.AgentID, rec.Address, rec.Capabilities)
		log.Printf("📝 Registered Agent: %s, Address: %s, Capabilities: %v", rec.AgentID, rec.Address, rec.Capabilities)

		reply := protocol.NewMessage(protocol.TypeResult, "swarm-registry", peerID, "SUCCESS")
		reply.Sign(privKey)
		_ = protocol.WriteMessage(conn, reply)

	case protocol.TypeQuery:
		// Message payload contains requested capability (if any)
		reqCap := msg.Payload
		records := reg.Query(reqCap)

		respBytes, _ := json.Marshal(registry.QueryResponse{Agents: records})
		reply := protocol.NewMessage(protocol.TypeResult, "swarm-registry", peerID, string(respBytes))
		reply.Sign(privKey)
		_ = protocol.WriteMessage(conn, reply)
		log.Printf("🔍 Queried for capability %q -> returned %d records to %s", reqCap, len(records), peerID)
	}
}
