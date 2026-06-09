// Package idp implements the Workload Identity Provider (simulating the SPIFFE Workload API).
// It delivers short-lived mTLS credentials to local workloads over secure loopback IPC,
// keeping private keys entirely in-memory and avoiding disk storage.
package idp

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

const WorkloadAPIPort = 23790

// IdentityRequest is sent by an agent to request dynamic credentials.
type IdentityRequest struct {
	AgentID string `json:"agent_id"`
}

// IdentityResponse contains the short-lived in-memory credentials signed by the CA.
type IdentityResponse struct {
	CertPEM string `json:"cert_pem"`
	KeyPEM  string `json:"key_pem"`
	CAPEM   string `json:"ca_pem"`
}

// FetchIdentity connects to the local IdP daemon and retrieves dynamic credentials in-memory.
func FetchIdentity(agentID string) (*IdentityResponse, error) {
	idpHost := os.Getenv("IDP_HOST")
	if idpHost == "" {
		idpHost = fmt.Sprintf("127.0.0.1:%d", WorkloadAPIPort)
	}
	
	conn, err := net.DialTimeout("tcp", idpHost, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Identity Provider (IdP) daemon: %w. Is the idp-daemon running?", err)
	}
	defer conn.Close()

	// Write Request
	req := IdentityRequest{AgentID: agentID}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to encode identity request: %w", err)
	}

	// Read Response
	var resp IdentityResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode identity response: %w", err)
	}

	return &resp, nil
}
