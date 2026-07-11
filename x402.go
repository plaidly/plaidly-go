package plaidly

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Header names used by the x402 flow.
const (
	PaymentRequiredHeader  = "PAYMENT-REQUIRED"
	PaymentSignatureHeader = "PAYMENT-SIGNATURE"
	PlaidlyProofHeader     = "X-Plaidly-Payment"
)

// X402Challenge captures typed payment requirements from an x402 HTTP-402 response.
type X402Challenge struct {
	Version           string         `json:"version,omitempty"`
	Scheme            string         `json:"scheme,omitempty"`
	Network           string         `json:"network,omitempty"`
	MaxAmountRequired string         `json:"maxAmountRequired,omitempty"`
	Amount            string         `json:"amount,omitempty"`
	Asset             string         `json:"asset,omitempty"`
	PayTo             string         `json:"payTo,omitempty"`
	Recipient         string         `json:"recipient,omitempty"`
	Memo              string         `json:"memo,omitempty"`
	Resource          string         `json:"resource,omitempty"`
	Description       string         `json:"description,omitempty"`
	Extra             map[string]any `json:"-"`
}

// X402ProofProvider builds a proof payload for a parsed challenge.
type X402ProofProvider interface {
	ProvideProof(ctx context.Context, challenge *X402Challenge) (string, error)
}

// X402Facilitator builds proof via a third-party facilitator for a parsed challenge.
type X402Facilitator interface {
	Facilitate(ctx context.Context, challenge *X402Challenge) (string, error)
}

var (
	ErrMissingX402Challenge = errors.New("plaidly: missing x402 challenge")
	ErrInvalidX402Challenge = errors.New("plaidly: invalid x402 challenge")
	ErrX402ProofUnavailable = errors.New("plaidly: no x402 proof provider or facilitator provided")
)

// ParseX402Challenge reads an x402 challenge from a 402 response.
// It accepts either the x402 header (base64-encoded JSON) or JSON in the body.
func ParseX402Challenge(resp *http.Response) (*X402Challenge, error) {
	if resp == nil {
		return nil, ErrMissingX402Challenge
	}
	if resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}

	if headerVal := firstNonEmpty(
		strings.TrimSpace(resp.Header.Get(PaymentRequiredHeader)),
		strings.TrimSpace(resp.Header.Get("X-PAYMENT-REQUIRED")),
	); headerVal != "" {
		challenge, err := parseX402ChallengeFromHeader(headerVal)
		if err == nil {
			return challenge, nil
		}
	}

	if resp.Body == nil {
		return nil, ErrMissingX402Challenge
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("plaidly: read x402 challenge body: %w", err)
	}
	body = []byte(strings.TrimSpace(string(body)))
	if len(body) == 0 {
		return nil, ErrMissingX402Challenge
	}

	return parseX402ChallengeJSON(body)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func parseX402ChallengeFromHeader(headerVal string) (*X402Challenge, error) {
	payload, err := decodeX402Payload(headerVal)
	if err != nil {
		return nil, err
	}
	return parseX402ChallengeJSON(payload)
}

func parseX402ChallengeJSON(payload []byte) (*X402Challenge, error) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidX402Challenge, err)
	}

	var challenge X402Challenge
	if err := json.Unmarshal(payload, &challenge); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidX402Challenge, err)
	}

	if len(raw) == 0 {
		return nil, ErrMissingX402Challenge
	}

	challenge.Extra = map[string]any{}
	for k, v := range raw {
		if isKnownX402Field(k) {
			continue
		}
		challenge.Extra[k] = v
	}
	return &challenge, nil
}

func isKnownX402Field(field string) bool {
	switch field {
	case "version", "scheme", "network", "maxAmountRequired", "amount", "asset", "payTo", "recipient", "memo", "resource", "description":
		return true
	default:
		return false
	}
}

func decodeX402Payload(value string) ([]byte, error) {
	value = strings.TrimSpace(strings.Trim(value, "\""))
	if value == "" {
		return nil, ErrMissingX402Challenge
	}
	if strings.HasPrefix(value, "{") || strings.HasPrefix(value, "[") {
		return []byte(value), nil
	}

	decoders := []*base64.Encoding{
		base64.RawStdEncoding,
		base64.StdEncoding,
		base64.RawURLEncoding,
		base64.URLEncoding,
	}
	for _, dec := range decoders {
		decoded, err := dec.DecodeString(value)
		if err == nil {
			return decoded, nil
		}
	}

	return nil, fmt.Errorf("%w: %v", ErrInvalidX402Challenge, errors.New("failed to decode challenge payload"))
}

// RetryRequestWithX402Proof retries an HTTP request after resolving an x402 challenge.
// `doRequest` is called once without proof and, on HTTP 402 with a parseable challenge,
// called again with the generated proof value.
func RetryRequestWithX402Proof(
	ctx context.Context,
	doRequest func(ctx context.Context, proof string) (*http.Response, error),
	proofProvider X402ProofProvider,
	facilitator X402Facilitator,
	out any,
) error {
	if doRequest == nil {
		return errors.New("plaidly: doRequest callback is nil")
	}

	resp, err := doRequest(ctx, "")
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusPaymentRequired {
		apiErr, _, decodeErr := decodeResponse(resp, out)
		if apiErr != nil {
			return apiErr
		}
		return decodeErr
	}

	challenge, err := ParseX402Challenge(resp)
	if err != nil {
		return err
	}

	proof, err := resolveX402Proof(ctx, challenge, proofProvider, facilitator)
	if err != nil {
		return err
	}

	resp, err = doRequest(ctx, proof)
	if err != nil {
		return err
	}
	// decodeResponse handles 4xx/5xx responses too, returning a typed error.
	apiErr, _, decodeErr := decodeResponse(resp, out)
	if apiErr != nil {
		return apiErr
	}
	return decodeErr
}

func resolveX402Proof(
	ctx context.Context,
	challenge *X402Challenge,
	proofProvider X402ProofProvider,
	facilitator X402Facilitator,
) (string, error) {
	if proofProvider != nil {
		proof, err := proofProvider.ProvideProof(ctx, challenge)
		if err == nil {
			return proof, nil
		}
		if facilitator != nil {
			proof, err := facilitator.Facilitate(ctx, challenge)
			if err == nil {
				return proof, nil
			}
		}
		return "", err
	}
	if facilitator != nil {
		return facilitator.Facilitate(ctx, challenge)
	}
	return "", ErrX402ProofUnavailable
}
