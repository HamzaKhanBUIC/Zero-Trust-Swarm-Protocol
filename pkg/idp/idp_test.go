package idp

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/hamza-imran/zero-trust-swarm/pkg/pki"
)

const testToken = "0123456789abcdef0123"

func TestParseAllowlist(t *testing.T) {
	tests := []struct {
		name           string
		in             string
		wantErr        bool
		requiresTokens bool
	}{
		{name: "names only", in: "swarm-registry,agent-alpha"},
		{name: "tokens and spaces", in: " agent-alpha=" + testToken + " , agent-beta=" + testToken, requiresTokens: true},
		{name: "mixed", in: "agent-alpha,agent-beta=" + testToken},
		{name: "empty", in: "", wantErr: true},
		{name: "only separators", in: " , ,", wantErr: true},
		{name: "path in agent ID", in: "agent/alpha", wantErr: true},
		{name: "query in agent ID", in: "agent?x=1", wantErr: true},
		{name: "empty agent ID", in: "=" + testToken, wantErr: true},
		{name: "short token", in: "agent-alpha=short", wantErr: true},
		{name: "empty token", in: "agent-alpha=", wantErr: true},
		{name: "duplicate", in: "agent-alpha,agent-alpha", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := ParseAllowlist(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseAllowlist(%q) = nil error, want error", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAllowlist(%q): %v", tt.in, err)
			}
			if got := a.RequiresTokens(); got != tt.requiresTokens {
				t.Errorf("RequiresTokens() = %v, want %v", got, tt.requiresTokens)
			}
		})
	}
}

func TestAllowlistAuthorize(t *testing.T) {
	a, err := ParseAllowlist("agent-alpha,agent-beta=" + testToken)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		req     IdentityRequest
		allowed bool
	}{
		{name: "listed identity without token", req: IdentityRequest{AgentID: "agent-alpha"}, allowed: true},
		{name: "listed identity with its token", req: IdentityRequest{AgentID: "agent-beta", Token: testToken}, allowed: true},
		{name: "unlisted identity", req: IdentityRequest{AgentID: "agent-mallory"}},
		{name: "empty identity", req: IdentityRequest{}},
		{name: "missing token", req: IdentityRequest{AgentID: "agent-beta"}},
		{name: "wrong token", req: IdentityRequest{AgentID: "agent-beta", Token: testToken + "x"}},
		{name: "case-changed identity", req: IdentityRequest{AgentID: "Agent-Alpha"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := a.Authorize(tt.req)
			if tt.allowed && err != nil {
				t.Fatalf("Authorize(%+v): %v", tt.req, err)
			}
			if !tt.allowed && !errors.Is(err, ErrDenied) {
				t.Fatalf("Authorize(%+v) = %v, want ErrDenied", tt.req, err)
			}
		})
	}
}

func TestIsLoopback(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:23790", true},
		{"[::1]:23790", true},
		{"localhost:23790", true},
		{"0.0.0.0:23790", false},
		{":23790", false},
		{"[::]:23790", false},
		{"10.0.0.5:23790", false},
		{"swarm-idp:23790", false},
		{"127.0.0.1", false},
	}
	for _, tt := range tests {
		if got := IsLoopback(tt.addr); got != tt.want {
			t.Errorf("IsLoopback(%q) = %v, want %v", tt.addr, got, tt.want)
		}
	}
}

// TestFetchIdentity runs the real client against a real server.
func TestFetchIdentity(t *testing.T) {
	ca, err := pki.InitCA(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	allow, err := ParseAllowlist("agent-alpha,agent-beta=" + testToken)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go (&Server{CA: ca, Allow: allow}).Serve(ln)
	t.Setenv("IDP_HOST", ln.Addr().String())

	roots := x509.NewCertPool()
	roots.AddCert(ca.Cert)

	tests := []struct {
		name    string
		agentID string
		token   string
		wantErr bool
	}{
		{name: "allowed identity is issued", agentID: "agent-alpha"},
		{name: "allowed identity with token is issued", agentID: "agent-beta", token: testToken},
		{name: "unlisted identity is refused", agentID: "agent-mallory", wantErr: true},
		{name: "missing token is refused", agentID: "agent-beta", wantErr: true},
		{name: "wrong token is refused", agentID: "agent-beta", token: "wrong-token-0123456789", wantErr: true},
		{name: "spoofed path is refused", agentID: "agent-alpha/../agent-beta", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("IDP_TOKEN", tt.token)
			resp, err := FetchIdentity(tt.agentID)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("FetchIdentity(%q) issued a certificate, want refusal", tt.agentID)
				}
				if !strings.Contains(err.Error(), ErrDenied.Error()) {
					t.Fatalf("FetchIdentity(%q) failed for the wrong reason: %v", tt.agentID, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("FetchIdentity(%q): %v", tt.agentID, err)
			}
			block, _ := pem.Decode([]byte(resp.CertPEM))
			if block == nil {
				t.Fatal("response has no certificate PEM")
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := cert.Verify(x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
				t.Fatalf("issued certificate does not chain to the CA: %v", err)
			}
			if len(cert.URIs) != 1 || cert.URIs[0].String() != pki.SPIFFEID(tt.agentID) {
				t.Fatalf("certificate URIs = %v, want [%s]", cert.URIs, pki.SPIFFEID(tt.agentID))
			}
		})
	}
}
