package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/hamza-imran/zero-trust-swarm/pkg/pki"
	"github.com/hamza-imran/zero-trust-swarm/pkg/registry"
)

// stateFor returns a ConnectionState whose peer authenticated as agentID.
func stateFor(t *testing.T, agentID string) tls.ConnectionState {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(pki.SPIFFEID(agentID))
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: agentID},
		URIs:         []*url.URL{u},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}
}

func TestBoundRecord(t *testing.T) {
	alpha := stateFor(t, "agent-alpha")
	tests := []struct {
		name    string
		state   tls.ConnectionState
		payload string
		wantErr bool
	}{
		{
			name:    "own identity accepted",
			state:   alpha,
			payload: `{"agent_id":"agent-alpha","address":"127.0.0.1:9101","capabilities":["math-solver"]}`,
		},
		{
			name:    "spoofed agent ID rejected",
			state:   alpha,
			payload: `{"agent_id":"agent-beta","address":"10.0.0.66:9101","capabilities":["math-solver"]}`,
			wantErr: true,
		},
		{
			name:    "registry identity rejected",
			state:   alpha,
			payload: `{"agent_id":"` + registry.ServiceID + `","address":"10.0.0.66:9000"}`,
			wantErr: true,
		},
		{
			name:    "missing agent ID rejected",
			state:   alpha,
			payload: `{"address":"127.0.0.1:9101"}`,
			wantErr: true,
		},
		{
			name:    "caller without certificate rejected",
			state:   tls.ConnectionState{},
			payload: `{"agent_id":"agent-alpha","address":"127.0.0.1:9101"}`,
			wantErr: true,
		},
		{
			name:    "malformed payload rejected",
			state:   alpha,
			payload: `not json`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, err := boundRecord(tt.state, tt.payload)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("boundRecord accepted %s", tt.payload)
				}
				return
			}
			if err != nil {
				t.Fatalf("boundRecord: %v", err)
			}
			if rec.AgentID != "agent-alpha" || rec.Address != "127.0.0.1:9101" {
				t.Fatalf("boundRecord = %+v", rec)
			}
		})
	}
}
