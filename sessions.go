package plaidly

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// CreatePaymentSession creates a new payment session.
//
// POST /v1/payment_sessions
func (c *Client) CreatePaymentSession(ctx context.Context, req CreatePaymentSessionRequest) (*PaymentSession, error) {
	var out PaymentSession
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.CreatePaymentSession(ctx, req)
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
		return c.raw.CreateDemoPaymentSession(ctx)
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
			Code:       "RECEIPT_FETCH_FAILED",
			Message:    fmt.Sprintf("HTTP %d", resp.StatusCode),
		}
	}
	return io.ReadAll(resp.Body)
}
