package plaidly

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func sign(t *testing.T, payload []byte, secret string) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhookSignature_Valid(t *testing.T) {
	payload := []byte(`{"event":"payment.completed"}`)
	secret := "whsec_test"
	sig := sign(t, payload, secret)
	if !VerifyWebhookSignature(payload, sig, secret) {
		t.Fatal("expected true for valid signature")
	}
}

func TestVerifyWebhookSignature_TamperedPayload(t *testing.T) {
	secret := "whsec_test"
	sig := sign(t, []byte(`{"event":"payment.completed"}`), secret)
	if VerifyWebhookSignature([]byte(`{"event":"tampered"}`), sig, secret) {
		t.Fatal("expected false for tampered payload")
	}
}

func TestVerifyWebhookSignature_WrongSecret(t *testing.T) {
	payload := []byte(`{"event":"payment.completed"}`)
	sig := sign(t, payload, "correct_secret")
	if VerifyWebhookSignature(payload, sig, "wrong_secret") {
		t.Fatal("expected false for wrong secret")
	}
}

func TestVerifyWebhookSignature_MissingPrefix(t *testing.T) {
	payload := []byte("body")
	if VerifyWebhookSignature(payload, "deadbeef", "secret") {
		t.Fatal("expected false when sha256= prefix is missing")
	}
}

func TestVerifyWebhookSignature_InvalidHex(t *testing.T) {
	payload := []byte("body")
	if VerifyWebhookSignature(payload, "sha256=zzzzzz", "secret") {
		t.Fatal("expected false for non-hex signature")
	}
}
