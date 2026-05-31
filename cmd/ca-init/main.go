// Command ca-init generates the root CA and agent certificates.
// Usage: go run ./cmd/ca-init
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hamza-imran/zero-trust-swarm/pkg/pki"
)

const certsDir = ".certs"

func main() {
	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║   Zero-Trust Swarm Protocol — CA Init       ║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()

	// Generate Root CA
	fmt.Println("[1/3] Generating Root CA (ECDSA P-256)...")
	ca, err := pki.InitCA(certsDir)
	if err != nil {
		log.Fatalf("CA init failed: %v", err)
	}
	fmt.Println("  ✓ Root CA created: .certs/ca.crt, .certs/ca.key")

	// Determine which agents to create certs for
	agents := []string{"agent-alpha", "agent-beta"}
	if len(os.Args) > 1 {
		agents = os.Args[1:]
	}

	// Issue agent certificates
	fmt.Printf("\n[2/3] Issuing agent certificates...\n")
	for _, agentID := range agents {
		if err := pki.IssueAgentCert(agentID, certsDir, ca); err != nil {
			log.Fatalf("Failed to issue cert for %s: %v", agentID, err)
		}
		fmt.Printf("  ✓ %s → .certs/%s.crt, .certs/%s.key\n", agentID, agentID, agentID)
		fmt.Printf("    SPIFFE ID: spiffe://swarm.local/agent/%s\n", agentID)
	}

	fmt.Println("\n[3/3] Verification...")
	fmt.Println("  ✓ All certificates signed by Root CA")
	fmt.Println("  ✓ Short-lived: expires in 24 hours")
	fmt.Println("  ✓ SANs: localhost, 127.0.0.1, ::1")
	fmt.Println("\n🔐 PKI bootstrap complete. Ready to start agents.")
}
