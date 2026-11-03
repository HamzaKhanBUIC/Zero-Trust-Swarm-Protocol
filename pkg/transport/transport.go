// Package transport provides mTLS-secured TCP connections for agent communication.
// It uses a custom protocol prefix to bypass antivirus SSL interception and implements
// elite zero-trust verification of SPIFFE identities.
package transport

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/hamza-imran/zero-trust-swarm/pkg/pki"
	"github.com/hamza-imran/zero-trust-swarm/pkg/protocol"
)

const ProtocolPrefix = "SWARM-mTLS\n"

// SwarmTLS holds the mTLS configuration for an agent.
type SwarmTLS struct {
	AgentID    string
	TLSConfig  *tls.Config
	CACertPool *x509.CertPool
}

// NewTLSConfig returns the mTLS configuration every swarm workload uses:
// TLS 1.3 only, client certificates required, and a peer is accepted only if
// its certificate chains to caPool and carries a swarm SPIFFE ID.
func NewTLSConfig(cert tls.Certificate, caPool *x509.CertPool) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		RootCAs:      caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
		// Peers are identified by SPIFFE ID, not by hostname, so Go's
		// hostname check is replaced by VerifyConnection below.
		InsecureSkipVerify: true, // #nosec G402 -- chain and SPIFFE ID are verified in VerifyConnection
		VerifyConnection:   verifyPeer(caPool),
	}
}

// verifyPeer checks the peer's chain against caPool and requires a SPIFFE ID.
// It runs on both sides of every handshake.
func verifyPeer(caPool *x509.CertPool) func(tls.ConnectionState) error {
	return func(cs tls.ConnectionState) error {
		if len(cs.PeerCertificates) == 0 {
			return errors.New("no peer certificates presented")
		}
		opts := x509.VerifyOptions{
			Roots:         caPool,
			Intermediates: x509.NewCertPool(),
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		}
		if _, err := cs.PeerCertificates[0].Verify(opts); err != nil {
			return fmt.Errorf("certificate verification failed: %w", err)
		}
		_, err := PeerAgentID(cs)
		return err
	}
}

// PeerAgentID returns the agent ID from the SPIFFE ID in the peer's leaf
// certificate. The leaf must carry exactly one URI SAN, and it must be
// spiffe://swarm.local/agent/<valid agent ID>.
func PeerAgentID(state tls.ConnectionState) (string, error) {
	if len(state.PeerCertificates) == 0 {
		return "", errors.New("no peer certificate")
	}
	uris := state.PeerCertificates[0].URIs
	if len(uris) != 1 {
		return "", fmt.Errorf("peer certificate has %d URI SANs, want exactly one SPIFFE ID", len(uris))
	}
	id, ok := pki.AgentIDFromSPIFFEID(uris[0].String())
	if !ok {
		return "", fmt.Errorf("peer URI %q is not a swarm SPIFFE ID", uris[0])
	}
	return id, nil
}

// VerifyPeerID returns an error unless the peer's certificate carries the
// SPIFFE ID of agentID.
func VerifyPeerID(state tls.ConnectionState, agentID string) error {
	got, err := PeerAgentID(state)
	if err != nil {
		return err
	}
	if got != agentID {
		return fmt.Errorf("peer is %s, want %s", pki.SPIFFEID(got), pki.SPIFFEID(agentID))
	}
	return nil
}

// swarmListener wraps a standard net.Listener to perform protocol prefix negotiation
// and wrap incoming connections in TLS.
type swarmListener struct {
	net.Listener
	config *tls.Config
}

func (l *swarmListener) Accept() (net.Conn, error) {
	rawConn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	// Read protocol prefix with a timeout to avoid hanging on rogue connections
	if err := rawConn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		rawConn.Close()
		return nil, err
	}

	// Use io.ReadFull to prevent TCP fragmentation issues
	prefixBuf := make([]byte, len(ProtocolPrefix))
	if _, err := io.ReadFull(rawConn, prefixBuf); err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("failed to fully read protocol prefix: %w", err)
	}

	if string(prefixBuf) != ProtocolPrefix {
		rawConn.Close()
		return nil, fmt.Errorf("invalid protocol prefix: %q", string(prefixBuf))
	}

	// Clear deadlines
	if err := rawConn.SetReadDeadline(time.Time{}); err != nil {
		rawConn.Close()
		return nil, err
	}

	// Wrap in TLS server connection
	tlsConn := tls.Server(rawConn, l.config)
	return tlsConn, nil
}

// Listen creates a custom swarm listener wrapped in TLS on the given address.
func (s *SwarmTLS) Listen(addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("tcp listen failed: %w", err)
	}
	return &swarmListener{Listener: ln, config: s.TLSConfig}, nil
}

// Dial connects to another agent at the given address using mTLS and protocol prefixing.
// The peer must authenticate as peerID; any other identity is rejected
// before anything is sent.
func (s *SwarmTLS) Dial(addr, peerID string) (net.Conn, error) {
	rawConn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("tcp dial failed: %w", err)
	}

	// Write protocol prefix
	if err := rawConn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		rawConn.Close()
		return nil, err
	}
	if _, err := rawConn.Write([]byte(ProtocolPrefix)); err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("failed to write protocol prefix: %w", err)
	}
	if err := rawConn.SetWriteDeadline(time.Time{}); err != nil {
		rawConn.Close()
		return nil, err
	}

	// Wrap in TLS client connection
	tlsConn := tls.Client(rawConn, s.TLSConfig)
	if err := tlsConn.Handshake(); err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("mTLS handshake failed: %w", err)
	}
	if err := VerifyPeerID(tlsConn.ConnectionState(), peerID); err != nil {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("refusing peer at %s: %w", addr, err)
	}

	return tlsConn, nil
}

// ExtractPeerID returns the peer's SPIFFE ID, or "unknown" if its
// certificate has none. There is no Common Name fallback.
func ExtractPeerID(conn *tls.Conn) string {
	id, err := PeerAgentID(conn.ConnectionState())
	if err != nil {
		return "unknown"
	}
	return pki.SPIFFEID(id)
}

// VerifyPeerSignature checks that msg was signed by the key behind the peer's
// mTLS certificate. It fails closed: a missing certificate or a non-ECDSA key
// is an error, never a skipped check.
func VerifyPeerSignature(state tls.ConnectionState, msg *protocol.Message) error {
	if len(state.PeerCertificates) == 0 {
		return fmt.Errorf("no peer certificate to verify payload signature against")
	}
	peerPubKey, ok := state.PeerCertificates[0].PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("peer key is %T, want *ecdsa.PublicKey", state.PeerCertificates[0].PublicKey)
	}
	return msg.Verify(peerPubKey)
}
