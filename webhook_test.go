package plaidly

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"
)

const (
	goldenSecret    = "whsec_test_secret"
	goldenTimestamp = int64(1700000000)
	goldenBody      = `{"event_type":"payment_session.completed","session_id":"ps_abc123","status":"completed","amount":10,"currency":"USDC","chain":"solana","network":"mainnet","timestamp":"1700000000"}`
	goldenHeader    = "t=1700000000,v1=eaaa2c9a7389633fa6f1aefc642eb0a5cb265b0491c96f0c49182dc509d783e0"
)

func signWebhook(payload []byte, secret string, ts int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	tsStr := fmt.Sprintf("%d", ts)
	mac.Write([]byte(tsStr))
	mac.Write([]byte("."))
	mac.Write(payload)
	return "t=" + tsStr + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature_GoldenVector(t *testing.T) {
	now := time.Unix(goldenTimestamp, 0)
	if err := verifySignature([]byte(goldenBody), goldenHeader, goldenSecret, DefaultWebhookTolerance, now); err != nil {
		t.Fatalf("golden vector failed: %v", err)
	}
}

func TestVerifyWebhook_DecodesEvent(t *testing.T) {
	payload := []byte(goldenBody)
	sig := signWebhook(payload, goldenSecret, time.Now().Unix())
	ev, err := VerifyWebhook(payload, sig, goldenSecret, DefaultWebhookTolerance)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ev.EventType != EventPaymentCompleted {
		t.Errorf("event_type = %q, want %q", ev.EventType, EventPaymentCompleted)
	}
	if ev.SessionID != "ps_abc123" {
		t.Errorf("session_id = %q, want ps_abc123", ev.SessionID)
	}
	if ev.Status != StatusCompleted {
		t.Errorf("status = %q, want completed", ev.Status)
	}
	if ev.Amount != 10 {
		t.Errorf("amount = %v, want 10", ev.Amount)
	}
	if ev.Chain != "solana" || ev.Network != "mainnet" || ev.Currency != "USDC" {
		t.Errorf("chain/network/currency = %q/%q/%q", ev.Chain, ev.Network, ev.Currency)
	}
}

func TestVerifyWebhook_MissingSignature(t *testing.T) {
	_, err := VerifyWebhook([]byte(goldenBody), "", goldenSecret, DefaultWebhookTolerance)
	if !errors.Is(err, ErrMissingSignature) {
		t.Fatalf("err = %v, want ErrMissingSignature", err)
	}
}

func TestVerifyWebhook_TamperedPayload(t *testing.T) {
	sig := signWebhook([]byte(goldenBody), goldenSecret, time.Now().Unix())
	_, err := VerifyWebhook([]byte(`{"event_type":"tampered"}`), sig, goldenSecret, DefaultWebhookTolerance)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("err = %v, want ErrInvalidSignature", err)
	}
}

func TestVerifyWebhook_WrongSecret(t *testing.T) {
	payload := []byte(goldenBody)
	sig := signWebhook(payload, goldenSecret, time.Now().Unix())
	_, err := VerifyWebhook(payload, sig, "whsec_wrong", DefaultWebhookTolerance)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("err = %v, want ErrInvalidSignature", err)
	}
}

func TestVerifyWebhook_ExpiredTimestamp(t *testing.T) {
	payload := []byte(goldenBody)
	old := time.Now().Add(-10 * time.Minute).Unix()
	sig := signWebhook(payload, goldenSecret, old)
	_, err := VerifyWebhook(payload, sig, goldenSecret, 5*time.Minute)
	if !errors.Is(err, ErrSignatureExpired) {
		t.Fatalf("err = %v, want ErrSignatureExpired", err)
	}
}

func TestVerifyWebhook_MultipleCandidates(t *testing.T) {
	timestamp := time.Now().Unix()
	valid := signWebhook([]byte(goldenBody), goldenSecret, timestamp)
	badHex := "t=" + fmt.Sprintf("%d", timestamp) + ",v1=zzzz"
	sig := badHex + "," + valid
	if err := verifySignature(
		[]byte(goldenBody),
		sig,
		goldenSecret,
		DefaultWebhookTolerance,
		time.Unix(timestamp, 0),
	); err != nil {
		t.Fatalf("verifySignature with mixed candidates: %v", err)
	}
}

func TestVerifyWebhookAt_InjectedTimestamp(t *testing.T) {
	timestamp := time.Now().Unix()
	sig := signWebhook([]byte(goldenBody), goldenSecret, timestamp)
	ev, err := VerifyWebhookAt([]byte(goldenBody), sig, goldenSecret, DefaultWebhookTolerance, time.Unix(timestamp, 0))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ev.EventType != EventPaymentCompleted {
		t.Errorf("event_type = %q", ev.EventType)
	}
}

func TestVerifyWebhook_MissingSecret(t *testing.T) {
	_, err := VerifyWebhookAt([]byte(goldenBody), goldenHeader, "", DefaultWebhookTolerance, time.Unix(goldenTimestamp, 0))
	if !errors.Is(err, ErrMissingSecret) {
		t.Fatalf("err = %v, want ErrMissingSecret", err)
	}
}

func TestVerifyWebhookSignatureAt_Bool(t *testing.T) {
	timestamp := time.Now().Unix()
	sig := signWebhook([]byte(goldenBody), goldenSecret, timestamp)
	if !VerifyWebhookSignatureAt([]byte(goldenBody), sig, goldenSecret, DefaultWebhookTolerance, time.Unix(timestamp, 0)) {
		t.Fatal("expected true")
	}
	if VerifyWebhookSignatureAt([]byte(goldenBody), sig, "wrong", DefaultWebhookTolerance, time.Unix(timestamp, 0)) {
		t.Fatal("expected false for wrong secret")
	}
}

func TestVerifyWebhook_FutureTimestampWithinTolerance(t *testing.T) {
	payload := []byte(goldenBody)
	future := time.Now().Add(2 * time.Minute).Unix()
	sig := signWebhook(payload, goldenSecret, future)
	if _, err := VerifyWebhook(payload, sig, goldenSecret, 5*time.Minute); err != nil {
		t.Fatalf("future-but-within-tolerance rejected: %v", err)
	}
}

func TestVerifyWebhook_MalformedHeader(t *testing.T) {
	nowTs := time.Now().Unix()
	for _, h := range []string{"v1=abcd", "t=123", "garbage", fmt.Sprintf("t=%d,v1=zzzz", nowTs)} {
		if _, err := VerifyWebhook([]byte(goldenBody), h, goldenSecret, DefaultWebhookTolerance); !errors.Is(err, ErrInvalidSignature) {
			t.Errorf("header %q: err = %v, want ErrInvalidSignature", h, err)
		}
	}
}

func TestVerifyWebhookSignature_Bool(t *testing.T) {
	payload := []byte(goldenBody)
	sig := signWebhook(payload, goldenSecret, time.Now().Unix())
	if !VerifyWebhookSignature(payload, sig, goldenSecret) {
		t.Error("expected true for valid signature")
	}
	if VerifyWebhookSignature(payload, sig, "whsec_wrong") {
		t.Error("expected false for wrong secret")
	}
}

func TestVerifyWebhookSignatureAny_RotationWindow(t *testing.T) {
	payload := []byte(goldenBody)
	ts := time.Now().Unix()
	sig := signWebhook(payload, "whsec_old", ts)
	if !VerifyWebhookSignatureAny(payload, sig, []string{"whsec_new", "whsec_old"}, DefaultWebhookTolerance, time.Unix(ts, 0)) {
		t.Fatal("expected old secret to pass during rotation window")
	}
	if VerifyWebhookSignatureAny(payload, sig, []string{"whsec_new"}, DefaultWebhookTolerance, time.Unix(ts, 0)) {
		t.Fatal("expected new secret only to fail")
	}
}

func TestVerifyWebhookAny_RotationWindow(t *testing.T) {
	payload := []byte(goldenBody)
	ts := time.Now().Unix()
	sig := signWebhook(payload, "whsec_old", ts)
	ev, err := VerifyWebhookAny(payload, sig, []string{"whsec_new", "whsec_old"}, DefaultWebhookTolerance, time.Unix(ts, 0))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ev.SessionID != "ps_abc123" {
		t.Fatalf("session_id = %q", ev.SessionID)
	}
}
