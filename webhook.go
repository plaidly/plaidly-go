package plaidly

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// VerifyWebhookSignature verifies that an incoming webhook request originated from Plaidly.
//
// Plaidly sets the X-Plaidly-Signature header to "sha256=<hex>" where the
// hex string is HMAC-SHA256(secret, rawBody).
//
// Example:
//
//	ok := plaidly.VerifyWebhookSignature(body, r.Header.Get("X-Plaidly-Signature"), "whsec_...")
//	if !ok {
//	    http.Error(w, "invalid signature", http.StatusForbidden)
//	    return
//	}
func VerifyWebhookSignature(payload []byte, signature, secret string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return false
	}
	sigHex := strings.TrimPrefix(signature, prefix)
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload) //nolint:errcheck // bytes.Buffer.Write never returns an error
	expected := mac.Sum(nil)

	return hmac.Equal(sigBytes, expected)
}
