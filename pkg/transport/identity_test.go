package transport

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hamza-imran/zero-trust-swarm/pkg/pki"
)

// leafState returns a ConnectionState whose leaf certificate has the given
// Common Name and URI SANs, signed by ca (or self-signed if ca is nil).
func leafState(t *testing.T, ca *pki.CertBundle, cn string, uris ...string) tls.ConnectionState {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	for _, raw := range uris {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		template.URIs = append(template.URIs, u)
	}
	parent, signer := template, any(key)
	if ca != nil {
		parent, signer = ca.Cert, ca.PrivKey
	}
	der, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, signer)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}
}

func TestPeerAgentID(t *testing.T) {
	tests := []struct {
		name    string
		state   tls.ConnectionState
		want    string
		wantErr bool
	}{
		{name: "swarm SPIFFE ID", state: leafState(t, nil, "agent-alpha", "spiffe://swarm.local/agent/agent-alpha"), want: "agent-alpha"},
		{name: "no certificate", state: tls.ConnectionState{}, wantErr: true},
		{name: "Common Name only", state: leafState(t, nil, "agent-alpha"), wantErr: true},
		{name: "non-SPIFFE scheme", state: leafState(t, nil, "", "https://swarm.local/agent/agent-alpha"), wantErr: true},
		{name: "other trust domain", state: leafState(t, nil, "", "spiffe://evil.local/agent/agent-alpha"), wantErr: true},
		{name: "extra path segment", state: leafState(t, nil, "", "spiffe://swarm.local/agent/agent-alpha/x"), wantErr: true},
		{name: "query string", state: leafState(t, nil, "", "spiffe://swarm.local/agent/agent-alpha?x=1"), wantErr: true},
		{name: "two URI SANs", state: leafState(t, nil, "", "spiffe://swarm.local/agent/agent-alpha", "spiffe://swarm.local/agent/agent-beta"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PeerAgentID(tt.state)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("PeerAgentID = %q, want error", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("PeerAgentID = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestVerifyPeerID(t *testing.T) {
	alpha := leafState(t, nil, "agent-alpha", "spiffe://swarm.local/agent/agent-alpha")
	tests := []struct {
		name    string
		state   tls.ConnectionState
		want    string
		wantErr bool
	}{
		{name: "matching identity", state: alpha, want: "agent-alpha"},
		{name: "mismatched identity", state: alpha, want: "agent-beta", wantErr: true},
		{name: "empty expected identity", state: alpha, want: "", wantErr: true},
		{name: "spoofed Common Name", state: leafState(t, nil, "agent-beta", "spiffe://swarm.local/agent/agent-alpha"), want: "agent-beta", wantErr: true},
		{name: "Common Name without SPIFFE ID", state: leafState(t, nil, "agent-beta"), want: "agent-beta", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyPeerID(tt.state, tt.want)
			if (err != nil) != tt.wantErr {
				t.Fatalf("VerifyPeerID(%q) error = %v, wantErr %v", tt.want, err, tt.wantErr)
			}
		})
	}
}

func TestVerifyPeerRequiresSPIFFEID(t *testing.T) {
	ca, err := pki.InitCA(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	otherCA, err := pki.InitCA(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca.Cert)
	verify := verifyPeer(pool)

	tests := []struct {
		name    string
		state   tls.ConnectionState
		wantErr bool
	}{
		{name: "CA-signed with SPIFFE ID", state: leafState(t, ca, "agent-alpha", "spiffe://swarm.local/agent/agent-alpha")},
		{name: "CA-signed without SPIFFE ID", state: leafState(t, ca, "agent-alpha"), wantErr: true},
		{name: "CA-signed with foreign SPIFFE ID", state: leafState(t, ca, "", "spiffe://evil.local/agent/agent-alpha"), wantErr: true},
		{name: "other CA", state: leafState(t, otherCA, "agent-alpha", "spiffe://swarm.local/agent/agent-alpha"), wantErr: true},
		{name: "self-signed", state: leafState(t, nil, "agent-alpha", "spiffe://swarm.local/agent/agent-alpha"), wantErr: true},
		{name: "no certificate", state: tls.ConnectionState{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := verify(tt.state); (err != nil) != tt.wantErr {
				t.Fatalf("verifyPeer error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// swarmTLSFor issues a real certificate for agentID from ca.
func swarmTLSFor(t *testing.T, ca *pki.CertBundle, agentID string) *SwarmTLS {
	t.Helper()
	certPEM, keyPEM, err := pki.IssueAgentCertInMemory(agentID, ca)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca.Cert)
	return &SwarmTLS{AgentID: agentID, TLSConfig: NewTLSConfig(cert, pool), CACertPool: pool}
}

// TestDialBindsPeerIdentity dials a real mTLS listener that authenticates as
// agent-beta and checks that only a dial expecting agent-beta succeeds.
func TestDialBindsPeerIdentity(t *testing.T) {
	ca, err := pki.InitCA(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ln, err := swarmTLSFor(t, ca, "agent-beta").Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.(*tls.Conn).Handshake()
			}(conn)
		}
	}()

	alpha := swarmTLSFor(t, ca, "agent-alpha")
	tests := []struct {
		name    string
		peerID  string
		wantErr bool
	}{
		{name: "expected peer", peerID: "agent-beta"},
		{name: "peer claims to be someone else", peerID: "agent-gamma", wantErr: true},
		{name: "peer posing as registry", peerID: "swarm-registry", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := alpha.Dial(ln.Addr().String(), tt.peerID)
			if conn != nil {
				conn.Close()
			}
			if (err != nil) != tt.wantErr {
				t.Fatalf("Dial(_, %q) error = %v, wantErr %v", tt.peerID, err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(err.Error(), "refusing peer") {
				t.Fatalf("Dial(_, %q) failed for the wrong reason: %v", tt.peerID, err)
			}
		})
	}
}
