package protocol

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

func TestMessage_SignAndVerify(t *testing.T) {
	// Generate a test ECDSA keypair
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	msg := NewMessage(TypeTask, "agent-1", "agent-2", "Test Payload")

	// Verify before signing should fail
	if err := msg.Verify(&privKey.PublicKey); err == nil {
		t.Error("Verify() on unsigned message should fail, but it succeeded")
	}

	// Sign the message
	err = msg.Sign(privKey)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	if msg.Signature == "" {
		t.Error("Sign() did not populate the Signature field")
	}

	// Verify the signed message
	err = msg.Verify(&privKey.PublicKey)
	if err != nil {
		t.Errorf("Verify() failed for valid signature: %v", err)
	}

	// Tamper with the payload
	msg.Payload = "Tampered Payload"
	err = msg.Verify(&privKey.PublicKey)
	if err == nil {
		t.Error("Verify() succeeded after payload was tampered with, expected failure")
	}

	// Test with a different key
	privKey2, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	// Restore payload
	msg.Payload = "Test Payload"
	// Sign with Key 1
	msg.Sign(privKey)
	// Verify with Key 2 should fail
	err = msg.Verify(&privKey2.PublicKey)
	if err == nil {
		t.Error("Verify() succeeded with wrong public key, expected failure")
	}
}
