package plaidly

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/plaidly/plaidly-go/generated/plaidlyapi"
)

// CreatePaymentSession creates a new payment session.
//
// POST /v1/payment_sessions
func (c *Client) CreatePaymentSession(ctx context.Context, req CreatePaymentSessionRequest, opts ...RequestOption) (*PaymentSession, error) {
	ro := buildRequestOptions(opts)
	var out PaymentSession
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.CreatePaymentSession(ctx, nil, req, ro.toEditors()...)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateDemoPaymentSession creates a sandbox-only demo payment session.
//
// POST /v1/payment_sessions/demo
func (c *Client) CreateDemoPaymentSession(ctx context.Context) (*PaymentSession, error) {
	var out PaymentSession
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.CreateDemoPaymentSession(ctx, plaidlyapi.CreateDemoSessionRequest{})
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetPaymentSession fetches a payment session by ID.
//
// GET /v1/payment_sessions/{session_id}
func (c *Client) GetPaymentSession(ctx context.Context, sessionID string) (*PaymentSession, error) {
	var out PaymentSession
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.GetPaymentSession(ctx, sessionID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// FulfillDemoPaymentSession fulfils a demo payment session (sandbox only).
//
// POST /v1/payment_sessions/{session_id}/fulfill
func (c *Client) FulfillDemoPaymentSession(ctx context.Context, sessionID string) error {
	return c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.FulfillDemoPaymentSession(ctx, sessionID)
	}, nil)
}

// SimulatePayment instantly completes a demo/sandbox session and returns the
// updated session. Demo and sandbox sessions only.
//
// POST /v1/payment_sessions/{session_id}/simulate
func (c *Client) SimulatePayment(ctx context.Context, sessionID string) (*PaymentSession, error) {
	var out PaymentSession
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.SimulatePayment(ctx, sessionID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListPaymentMethods returns the enabled chain/token combinations.
//
// GET /v1/payment_methods
func (c *Client) ListPaymentMethods(ctx context.Context) ([]PaymentMethodInfo, error) {
	var out []PaymentMethodInfo
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.ListPaymentMethods(ctx)
	}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetRates returns USD spot rates. Pass symbols to filter (e.g. "ETH", "SOL");
// omit for all. Stablecoins are reported at a fixed 1.0.
//
// GET /v1/rates?symbols=...
func (c *Client) GetRates(ctx context.Context, symbols ...string) ([]RateInfo, error) {
	var params *plaidlyapi.GetRatesParams
	if len(symbols) > 0 {
		joined := strings.Join(symbols, ",")
		params = &plaidlyapi.GetRatesParams{Symbols: &joined}
	}
	var out []RateInfo
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.GetRates(ctx, params)
	}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListSandboxFaucets returns a map of "chain:network" to faucet URL.
//
// GET /v1/sandbox/faucets
func (c *Client) ListSandboxFaucets(ctx context.Context) (map[string]string, error) {
	out := map[string]string{}
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.ListSandboxFaucets(ctx)
	}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetReceiptPDF fetches the PDF receipt for a session and returns the raw
// body. The endpoint returns application/pdf, not JSON, so the generated
// client's raw response is returned directly.
//
// GET /v1/payment_sessions/{session_id}/receipt
func (c *Client) GetReceiptPDF(ctx context.Context, sessionID string) ([]byte, error) {
	resp, err := c.raw.GetReceiptBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, &Error{
			StatusCode: resp.StatusCode,
			Code:       ErrorCodeInternal,
			Message:    fmt.Sprintf("HTTP %d", resp.StatusCode),
		}
	}
	return io.ReadAll(resp.Body)
}
