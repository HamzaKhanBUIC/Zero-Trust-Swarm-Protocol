package idp

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"time"

	"github.com/hamza-imran/zero-trust-swarm/pkg/pki"
)

// maxRequestBytes caps the size of a single identity request.
const maxRequestBytes = 4 << 10

// Server issues short-lived workload certificates, but only for identities
// on its allowlist.
type Server struct {
	CA    *pki.CertBundle
	Allow *Allowlist
}

// Serve handles identity requests on ln until ln is closed.
func (s *Server) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return
	}

	var req IdentityRequest
	if err := json.NewDecoder(io.LimitReader(conn, maxRequestBytes)).Decode(&req); err != nil {
		log.Printf("❌ Failed to decode request: %v", err)
		return
	}

	resp, err := s.issue(req)
	if err != nil {
		log.Printf("⛔ Refused identity request from %s: %v", conn.RemoteAddr(), err)
		msg := "failed to issue identity"
		if errors.Is(err, ErrDenied) {
			msg = ErrDenied.Error()
		}
		resp = &IdentityResponse{Error: msg}
	}

	if err := json.NewEncoder(conn).Encode(resp); err != nil {
		log.Printf("❌ Failed to encode response: %v", err)
		return
	}
	if resp.Error == "" {
		log.Printf("✅ Issued short-lived credentials for %s (in-memory only)", req.AgentID)
	}
}

func (s *Server) issue(req IdentityRequest) (*IdentityResponse, error) {
	if err := s.Allow.Authorize(req); err != nil {
		return nil, err
	}
	certPEM, keyPEM, err := pki.IssueAgentCertInMemory(req.AgentID, s.CA)
	if err != nil {
		return nil, err
	}
	return &IdentityResponse{
		CertPEM: string(certPEM),
		KeyPEM:  string(keyPEM),
		CAPEM:   string(s.CA.CertPEM),
	}, nil
}
