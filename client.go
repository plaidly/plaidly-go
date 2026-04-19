// Package plaidly provides a Go client for the Plaidly cryptocurrency payment API.
//
// All API types and the raw HTTP client are generated from the Plaidly
// OpenAPI 3.1 specification (see spec/openapi.yaml, regenerate with `make
// generate`). The Client type in this file is a hand-written wrapper that
// adds the X-API-Key header, retries on transient 5xx failures, and
// translates non-2xx responses into a typed *Error.
//
// Usage:
//
//	client, err := plaidly.NewClient("pk_live_...")
//	if err != nil { /* ... */ }
//	session, err := client.CreatePaymentSession(ctx, plaidlyapi.CreatePaymentSessionRequest{
//	    Amount:     100.00,
//	    ExpiresIn:  "15m",
//	    PaymentMethod: plaidlyapi.PaymentMethod{
//	        MethodID: 0, Chain: "solana", Token: "USDC", Network: "mainnet",
//	    },
//	})
package plaidly

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/plaidly/plaidly-go/generated/plaidlyapi"
)

const defaultBaseURL = "https://api.plaidly.io"

// Re-export generated types so callers only need to import this package.
type (
	CreatePaymentSessionRequest = plaidlyapi.CreatePaymentSessionRequest
	CreateWalletRequest         = plaidlyapi.CreateWalletRequest
	Merchant                    = plaidlyapi.Merchant
	PaymentMethod               = plaidlyapi.PaymentMethod
	PaymentSession              = plaidlyapi.PaymentSession
	Payout                      = plaidlyapi.Payout
	Receipt                     = plaidlyapi.Receipt
	RegisterMerchantRequest     = plaidlyapi.RegisterMerchantRequest
	RequestPayoutRequest        = plaidlyapi.RequestPayoutRequest
	Transaction                 = plaidlyapi.Transaction
	User                        = plaidlyapi.User
	Wallet                      = plaidlyapi.Wallet
)

// Client is the high-level Plaidly API client.
// Methods on Client are wrappers around the generated plaidlyapi.Client that
// decode responses into typed values and surface API errors as *Error.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
	raw     *plaidlyapi.Client
}

// Option is a functional option for configuring a Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(url, "/") }
}

// WithHTTPClient replaces the default http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// NewClient creates a new Plaidly API client.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("plaidly: apiKey is required")
	}
	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	raw, err := plaidlyapi.NewClient(
		c.baseURL,
		plaidlyapi.WithHTTPClient(c.http),
		plaidlyapi.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("X-API-Key", c.apiKey)
			req.Header.Set("Accept", "application/json")
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}
	c.raw = raw
	return c, nil
}

// Raw returns the underlying generated client. Use this to call endpoints
// that are not yet exposed by the higher-level Client methods.
func (c *Client) Raw() *plaidlyapi.Client { return c.raw }

// doJSON executes fn (a generated client call) with up to 3 attempts on
// transient 5xx failures, decodes the 2xx JSON body into out (if non-nil),
// and returns a typed *Error on non-2xx responses.
func (c *Client) doJSON(ctx context.Context, fn func(context.Context) (*http.Response, error), out any) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(1<<attempt) * 500 * time.Millisecond):
			}
		}
		resp, err := fn(ctx)
		if err != nil {
			lastErr = err
			continue
		}
		apiErr, decoded, decodeErr := decodeResponse(resp, out)
		if apiErr != nil && apiErr.StatusCode >= 500 {
			lastErr = apiErr
			continue
		}
		if apiErr != nil {
			return apiErr
		}
		if decodeErr != nil {
			return decodeErr
		}
		_ = decoded
		return nil
	}
	return lastErr
}

// decodeResponse inspects an HTTP response; returns a non-nil *Error for
// non-2xx, or decodes the body into out for 2xx JSON responses.
func decodeResponse(resp *http.Response, out any) (*Error, bool, error) {
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		var apiErr struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		}
		_ = json.Unmarshal(body, &apiErr)
		return &Error{
			StatusCode: resp.StatusCode,
			Message:    apiErr.Message,
			Code:       apiErr.Code,
		}, false, nil
	}

	if out == nil || resp.StatusCode == http.StatusNoContent {
		// Drain body to allow connection reuse.
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, false, nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, false, fmt.Errorf("plaidly: decode response: %w", err)
	}
	return nil, true, nil
}

// readAllBody drains resp.Body and returns a fresh reader, so the retry
// loop can re-inspect a body without double-closing the response.
func readAllBody(resp *http.Response) []byte {
	if resp == nil || resp.Body == nil {
		return nil
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(b))
	return b
}

var _ = readAllBody // kept for future streaming helpers
