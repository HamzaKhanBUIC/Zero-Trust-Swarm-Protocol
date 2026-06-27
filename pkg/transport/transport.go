// Package transport provides mTLS-secured TCP connections for agent communication.
// It uses a custom protocol prefix to bypass antivirus SSL interception and implements
// elite zero-trust verification of SPIFFE identities.
package transport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"time"
)

const ProtocolPrefix = "SWARM-mTLS\n"

// SwarmTLS holds the mTLS configuration for an agent.
type SwarmTLS struct {
	AgentID    string
	TLSConfig *tls.Config
	CACertPool *x509.CertPool
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
func (s *SwarmTLS) Dial(addr string) (net.Conn, error) {
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

	return tlsConn, nil
}

// ExtractPeerID extracts the SPIFFE identity or CommonName from a TLS connection.
func ExtractPeerID(conn *tls.Conn) string {
	state := conn.ConnectionState()
	if len(state.PeerCertificates) > 0 {
		peer := state.PeerCertificates[0]
		if len(peer.URIs) > 0 {
			return peer.URIs[0].String()
		}
		return peer.Subject.CommonName
	}
	return "unknown"
}
