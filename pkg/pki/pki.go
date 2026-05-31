package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// CertBundle holds a certificate and its private key.
type CertBundle struct {
	Cert    *x509.Certificate
	CertPEM []byte
	KeyPEM  []byte
	PrivKey *ecdsa.PrivateKey
}

// InitCA generates a new Elliptic Curve root CA certificate and key.
// ECDSA P-256 is used for modern, fast, and compact certificates.
func InitCA(certsDir string) (*CertBundle, error) {
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create certs dir: %w", err)
	}

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA key: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Zero-Trust Swarm Protocol"},
			CommonName:   "Swarm Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		MaxPathLen:            1,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA cert: %w", err)
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA cert: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyBytes, _ := x509.MarshalECPrivateKey(privKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	// Write to disk
	if err := os.WriteFile(filepath.Join(certsDir, "ca.crt"), certPEM, 0644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(certsDir, "ca.key"), keyPEM, 0600); err != nil {
		return nil, err
	}

	return &CertBundle{Cert: cert, CertPEM: certPEM, KeyPEM: keyPEM, PrivKey: privKey}, nil
}

// IssueAgentCert creates a short-lived agent certificate signed by the CA.
// Each agent gets a SPIFFE identity: spiffe://swarm.local/agent/<agentID>
func IssueAgentCert(agentID, certsDir string, ca *CertBundle) error {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate agent key: %w", err)
	}

	spiffeURI, _ := url.Parse(fmt.Sprintf("spiffe://swarm.local/agent/%s", agentID))

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Organization: []string{"Zero-Trust Swarm Protocol"},
			CommonName:   agentID,
		},
		URIs:        []*url.URL{spiffeURI},
		DNSNames:    []string{"localhost", agentID},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour), // Short-lived: 24 hours
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, ca.Cert, &privKey.PublicKey, ca.PrivKey)
	if err != nil {
		return fmt.Errorf("failed to sign agent cert: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyBytes, _ := x509.MarshalECPrivateKey(privKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	if err := os.WriteFile(filepath.Join(certsDir, agentID+".crt"), certPEM, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(certsDir, agentID+".key"), keyPEM, 0600); err != nil {
		return err
	}

	return nil
}

// IssueAgentCertInMemory generates a short-lived agent certificate and private key in-memory.
func IssueAgentCertInMemory(agentID string, ca *CertBundle) (certPEM []byte, keyPEM []byte, err error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate agent key: %w", err)
	}

	spiffeURI, _ := url.Parse(fmt.Sprintf("spiffe://swarm.local/agent/%s", agentID))

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Organization: []string{"Zero-Trust Swarm Protocol"},
			CommonName:   agentID,
		},
		URIs:        []*url.URL{spiffeURI},
		DNSNames:    []string{"localhost", agentID},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(1 * time.Hour), // Short-lived: 1 hour for dynamic rotation
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, ca.Cert, &privKey.PublicKey, ca.PrivKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign agent cert: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyBytes, _ := x509.MarshalECPrivateKey(privKey)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	return certPEM, keyPEM, nil
}

