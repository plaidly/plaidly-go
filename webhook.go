package plaidly

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Webhook header names set by Plaidly on every delivery.
const (
	SignatureHeader = "X-Plaidly-Signature"
	EventHeader     = "X-Plaidly-Event"
)

// DefaultWebhookTolerance is the maximum age of a webhook signature accepted by
// VerifyWebhook when no explicit tolerance is configured.
const DefaultWebhookTolerance = 5 * time.Minute

// Webhook event types delivered in the X-Plaidly-Event header and event_type body field.
const (
	EventPaymentCompleted   = "payment_session.completed"
	EventPaymentExpired     = "payment_session.expired"
	EventPaymentPartialPaid = "payment_session.partial_paid"
)

// Errors returned by the webhook verifier.
var (
	ErrMissingSignature = errors.New("plaidly: missing signature header")
	ErrMissingSecret    = errors.New("plaidly: missing webhook secret")
	ErrInvalidSignature = errors.New("plaidly: signature mismatch")
	ErrSignatureExpired = errors.New("plaidly: signature timestamp outside tolerance")
)

// WebhookEvent is the decoded body of a Plaidly webhook delivery.
type WebhookEvent struct {
	EventType string  `json:"event_type"`
	SessionID string  `json:"session_id"`
	Status    string  `json:"status"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Chain     string  `json:"chain"`
	Network   string  `json:"network"`
	Timestamp string  `json:"timestamp"`
}

// VerifyWebhook validates the X-Plaidly-Signature header against payload using
// secret, enforces the timestamp tolerance, and decodes the event body.
//
// The signature header has the form "t=<unix>,v1=<hex>", where the hex value is
// HMAC-SHA256(secret, "<t>.<rawBody>"). Comparison is constant-time. Pass
// tolerance <= 0 to use DefaultWebhookTolerance.
//
// It returns a typed error: ErrMissingSignature, ErrInvalidSignature, or
// ErrSignatureExpired.
func VerifyWebhook(payload []byte, signature, secret string, tolerance time.Duration) (*WebhookEvent, error) {
	return VerifyWebhookAt(payload, signature, secret, tolerance, time.Now())
}

// VerifyWebhookAt validates the signature at a specific instant. This is useful
// for tests and replay verification when the exact webhook timestamp is known.
func VerifyWebhookAt(
	payload []byte,
	signature,
	secret string,
	tolerance time.Duration,
	now time.Time,
) (*WebhookEvent, error) {
	if err := verifySignature(payload, signature, secret, tolerance, now); err != nil {
		return nil, err
	}
	var ev WebhookEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, fmt.Errorf("plaidly: decode webhook body: %w", err)
	}
	return &ev, nil
}

// VerifyWebhookSignature reports whether signature is a valid Plaidly signature
// for payload under secret, using DefaultWebhookTolerance. It does not decode
// the body. Use VerifyWebhook for the typed event and error.
func VerifyWebhookSignature(payload []byte, signature, secret string) bool {
	return VerifyWebhookSignatureAt(payload, signature, secret, DefaultWebhookTolerance, time.Now())
}

// VerifyWebhookSignatureAt validates a signature against payload using a fixed
// reference timestamp. Returns true if any v1 signature matches.
func VerifyWebhookSignatureAt(payload []byte, signature, secret string, tolerance time.Duration, now time.Time) bool {
	return verifySignature(payload, signature, secret, tolerance, now) == nil
}

// VerifyWebhookSignatureAny validates a signature against an old/new
// signing-key rotation window. Pass secrets in preferred order, e.g. new then old.
func VerifyWebhookSignatureAny(payload []byte, signature string, secrets []string, tolerance time.Duration, now time.Time) bool {
	for _, secret := range secrets {
		if VerifyWebhookSignatureAt(payload, signature, secret, tolerance, now) {
			return true
		}
	}
	return false
}

// VerifyWebhookAny validates and decodes a webhook against an old/new
// signing-key rotation window. Pass secrets in preferred order, e.g. new then old.
func VerifyWebhookAny(payload []byte, signature string, secrets []string, tolerance time.Duration, now time.Time) (*WebhookEvent, error) {
	for _, secret := range secrets {
		ev, err := VerifyWebhookAt(payload, signature, secret, tolerance, now)
		if err == nil {
			return ev, nil
		}
	}
	return nil, ErrInvalidSignature
}

func verifySignature(payload []byte, signature, secret string, tolerance time.Duration, now time.Time) error {
	if signature == "" {
		return ErrMissingSignature
	}
	if secret == "" {
		return ErrMissingSecret
	}
	if tolerance <= 0 {
		tolerance = DefaultWebhookTolerance
	}

	ts, sigs, ok := parseSignatureHeader(signature)
	if !ok {
		return ErrInvalidSignature
	}

	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return ErrInvalidSignature
	}

	age := now.Sub(time.Unix(tsInt, 0))
	if age < 0 {
		age = -age
	}
	if age > tolerance {
		return ErrSignatureExpired
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(payload)
	expected := mac.Sum(nil)

	for _, sig := range sigs {
		sigBytes, err := hex.DecodeString(sig)
		if err != nil || len(sigBytes) == 0 {
			continue
		}
		if hmac.Equal(sigBytes, expected) {
			return nil
		}
	}

	return ErrInvalidSignature
}

func parseSignatureHeader(header string) (ts string, v1 []string, ok bool) {
	var signatures []string
	for _, part := range strings.Split(header, ",") {
		k, v, found := strings.Cut(strings.TrimSpace(part), "=")
		if !found {
			continue
		}
		switch k {
		case "t":
			ts = v
		case "v1":
			if v != "" {
				signatures = append(signatures, v)
			}
		}
	}
	if ts == "" || len(signatures) == 0 {
		return "", nil, false
	}
	return ts, signatures, true
}
