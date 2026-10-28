// Command idp-daemon runs the central Identity Provider (IdP) daemon.
// It serves short-lived, in-memory workload certificates to agents, but only
// for identities on its allowlist. An allowlist entry can require a
// per-workload bootstrap token. It listens on loopback by default and refuses
// to listen elsewhere unless every identity requires a token.
package main

import (
	"flag"
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
	listenAddr := flag.String("listen", fmt.Sprintf("127.0.0.1:%d", idp.WorkloadAPIPort), "Address to listen on")
	allowFlag := flag.String("allowlist", os.Getenv("IDP_ALLOWLIST"),
		`Comma-separated identities this IdP may issue, each "name" or "name=token" (default $IDP_ALLOWLIST; put tokens in the env var, not on the command line)`)
	flag.Parse()

	allow, err := idp.ParseAllowlist(*allowFlag)
	if err != nil {
		log.Fatalf("Invalid identity allowlist (set --allowlist or IDP_ALLOWLIST): %v", err)
	}
	if !idp.IsLoopback(*listenAddr) && !allow.RequiresTokens() {
		log.Fatalf("Refusing to listen on non-loopback address %s: every allowlist entry needs a token (name=token)", *listenAddr)
	}

	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║   Zero-Trust Swarm — Workload Identity Daemon║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()

	// 1. Generate a fresh CA for this run
	fmt.Println("🔐 Initializing Swarm CA...")
	ca, err := pki.InitCA(certsDir)
	if err != nil {
		log.Fatalf("CA initialization failed: %v", err)
	}
	fmt.Println("  ✓ Swarm CA ready.")

	// 2. Start Local IPC Listener
	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("Failed to bind to Workload IPC address %s: %v", *listenAddr, err)
	}
	defer ln.Close()
	fmt.Printf("👂 Workload Identity API listening on %s\n\n", *listenAddr)

	// Graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("\n🛑 Shutting down Identity Daemon...")
		ln.Close()
		os.Exit(0)
	}()

	srv := &idp.Server{CA: ca, Allow: allow}
	if err := srv.Serve(ln); err != nil {
		log.Printf("Identity Daemon stopped: %v", err)
	}
}
