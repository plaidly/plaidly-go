// Package plaidly provides a Go client for the Plaidly cryptocurrency payment API.
//
// Usage:
//
//	client := plaidly.NewClient("pk_live_...")
//	session, err := client.Sessions.Create(ctx, plaidly.CreateSessionRequest{
//	    Amount:   "100.00",
//	    Currency: "USDC",
//	    Chain:    "solana",
//	    Network:  "mainnet",
//	})
package plaidly

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.plaidly.io"

// Client is the Plaidly API client.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client

	// Sessions provides operations on payment sessions.
	Sessions *SessionsService
	// Merchants provides operations on merchant accounts.
	Merchants *MerchantsService
	// Payouts provides operations on payouts.
	Payouts *PayoutsService
	// Sandbox provides sandbox-only helpers.
	Sandbox *SandboxService
}

// Option is a functional option for configuring a Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL (useful for testing against a mock server).
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(url, "/") }
}

// WithHTTPClient replaces the default http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// NewClient creates a new Plaidly API client.
//
// apiKey is your merchant API key (passed as the X-API-Key header).
// Use Option functions to override defaults.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	c.Sessions = &SessionsService{client: c}
	c.Merchants = &MerchantsService{client: c}
	c.Payouts = &PayoutsService{client: c}
	c.Sandbox = &SandboxService{client: c}
	return c
}

// do executes an HTTP request and decodes the response into result (if non-nil).
func (c *Client) do(ctx context.Context, method, path string, body, result any) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return &Error{
			StatusCode: resp.StatusCode,
			Message:    apiErr.Message,
			Code:       apiErr.Code,
		}
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}
