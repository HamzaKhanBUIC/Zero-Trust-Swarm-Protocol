package transport

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/hamza-imran/zero-trust-swarm/pkg/protocol"
)

// peerState builds a ConnectionState whose peer certificate carries pub,
// self-signed by priv.
func peerState(t *testing.T, pub, priv any) tls.ConnectionState {
	t.Helper()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-agent"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}
	return tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}
}

func newECDSAKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

func signedMessage(t *testing.T, signer *ecdsa.PrivateKey) *protocol.Message {
	t.Helper()
	msg := protocol.NewMessage(protocol.TypeTask, "spiffe://swarm.local/agent/a", "spiffe://swarm.local/agent/b", "hello")
	if err := msg.Sign(signer); err != nil {
		t.Fatalf("sign: %v", err)
	}
	return msg
}

func TestVerifyPeerSignatureValid(t *testing.T) {
	key := newECDSAKey(t)
	state := peerState(t, &key.PublicKey, key)
	if err := VerifyPeerSignature(state, signedMessage(t, key)); err != nil {
		t.Fatalf("expected valid signature to pass, got: %v", err)
	}
}

func TestVerifyPeerSignatureRejectsMissingCert(t *testing.T) {
	key := newECDSAKey(t)
	if err := VerifyPeerSignature(tls.ConnectionState{}, signedMessage(t, key)); err == nil {
		t.Fatal("expected error when peer presented no certificate")
	}
}

func TestVerifyPeerSignatureRejectsNonECDSAKey(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}
	state := peerState(t, pub, crypto.Signer(priv))
	if err := VerifyPeerSignature(state, signedMessage(t, newECDSAKey(t))); err == nil {
		t.Fatal("expected error for a non-ECDSA peer key instead of skipping the check")
	}
}

func TestVerifyPeerSignatureRejectsWrongSigner(t *testing.T) {
	peerKey := newECDSAKey(t)
	state := peerState(t, &peerKey.PublicKey, peerKey)
	if err := VerifyPeerSignature(state, signedMessage(t, newECDSAKey(t))); err == nil {
		t.Fatal("expected error when the payload was signed by a different key")
	}
}

func TestVerifyPeerSignatureRejectsUnsignedMessage(t *testing.T) {
	key := newECDSAKey(t)
	state := peerState(t, &key.PublicKey, key)
	msg := protocol.NewMessage(protocol.TypeTask, "a", "b", "hello")
	if err := VerifyPeerSignature(state, msg); err == nil {
		t.Fatal("expected error for an unsigned message")
	}
}
