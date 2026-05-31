// Command idp-daemon runs the secure central Identity Provider (IdP) daemon.
// It serves temporary, in-memory workload certificates to agents over local loopback IPC,
// implementing a true zero-trust workload attestation & certificate rotation flow.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/hamza-imran/zero-trust-swarm/pkg/idp"
	"github.com/hamza-imran/zero-trust-swarm/pkg/pki"
)

const certsDir = ".certs"

func main() {
	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║   Zero-Trust Swarm — Workload Identity Daemon║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()

	// 1. Initialize or Load CA
	fmt.Println("🔐 Initializing Swarm CA...")
	ca, err := pki.InitCA(certsDir)
	if err != nil {
		log.Fatalf("CA initialization failed: %v", err)
	}
	fmt.Println("  ✓ Swarm CA loaded successfully.")

	// 2. Start Local IPC Listener
	addr := fmt.Sprintf("127.0.0.1:%d", idp.WorkloadAPIPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to bind to Workload IPC address %s: %v", addr, err)
	}
	defer ln.Close()
	fmt.Printf("👂 Workload Identity API listening on %s (strict local-only loopback)\n\n", addr)

	// Graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("\n🛑 Shutting down Identity Daemon...")
		ln.Close()
		os.Exit(0)
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			break
		}
		go handleWorkloadIdentityRequest(conn, ca)
	}
}

func handleWorkloadIdentityRequest(conn net.Conn, ca *pki.CertBundle) {
	defer conn.Close()

	// Read dynamic identity request
	var req idp.IdentityRequest
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		log.Printf("❌ Failed to decode request: %v", err)
		return
	}

	log.Printf("🔑 Received Identity Request for agent: %s", req.AgentID)

	// Issue ephemeral (1-hour) workload credentials in-memory
	certPEM, keyPEM, err := pki.IssueAgentCertInMemory(req.AgentID, ca)
	if err != nil {
		log.Printf("❌ Failed to issue cert for %s: %v", req.AgentID, err)
		return
	}

	// Prepare Response
	resp := idp.IdentityResponse{
		CertPEM: string(certPEM),
		KeyPEM:  string(keyPEM),
		CAPEM:   string(ca.CertPEM),
	}

	// Return response
	if err := json.NewEncoder(conn).Encode(resp); err != nil {
		log.Printf("❌ Failed to encode response: %v", err)
		return
	}

	log.Printf("✅ Dynamically issued short-lived credentials for %s (in-memory only)", req.AgentID)
}
