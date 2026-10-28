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
	Token   string `json:"token,omitempty"` // bootstrap token, if the IdP requires one for AgentID
}

// IdentityResponse contains the short-lived in-memory credentials signed by
// the CA, or Error if the IdP refused the request.
type IdentityResponse struct {
	CertPEM string `json:"cert_pem,omitempty"`
	KeyPEM  string `json:"key_pem,omitempty"`
	CAPEM   string `json:"ca_pem,omitempty"`
	Error   string `json:"error,omitempty"`
}

// FetchIdentity connects to the local IdP daemon and retrieves dynamic
// credentials in-memory. It sends the bootstrap token from IDP_TOKEN, if set.
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
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, err
	}

	// Write Request
	req := IdentityRequest{AgentID: agentID, Token: os.Getenv("IDP_TOKEN")}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to encode identity request: %w", err)
	}

	// Read Response
	var resp IdentityResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode identity response: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("identity provider refused %q: %s", agentID, resp.Error)
	}

	return &resp, nil
}
