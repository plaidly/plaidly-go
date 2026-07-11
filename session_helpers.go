package plaidly

import (
	"context"
	"errors"
	"time"
)

// Polling options for session status polling helpers.
type SessionPollOptions struct {
	// PollInterval controls delay between GetPaymentSession calls.
	PollInterval time.Duration
	// MaxAttempts limits polling rounds before timing out.
	MaxAttempts int
}

// ErrSessionPollTimeout is returned when a session does not become terminal in time.
var ErrSessionPollTimeout = errors.New("plaidly: payment session did not reach terminal state in time")

// ErrPaymentNotSettled is returned when a receipt is requested for a non-settled session.
var ErrPaymentNotSettled = errors.New("plaidly: payment session is not in a settled state")

// WaitForPaymentSession polls GET /v1/payment_sessions/{session_id} until the session reaches
// a terminal state (confirmed, completed, expired, or failed), or until max attempts are hit.
func WaitForPaymentSession(ctx context.Context, client *Client, sessionID string, options SessionPollOptions) (*PaymentSession, error) {
	interval := options.PollInterval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	maxAttempts := options.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 20
	}

	for attempt := 1; attempt <= maxAttempts; attempt += 1 {
		session, err := client.GetPaymentSession(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		if IsTerminal(session.Status) {
			return session, nil
		}
		if attempt >= maxAttempts {
			return session, ErrSessionPollTimeout
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}

	return nil, ErrSessionPollTimeout
}

// GetSessionReceiptIfSettled fetches the session and only returns the PDF receipt when
// the payment has reached a success status.
func GetSessionReceiptIfSettled(ctx context.Context, client *Client, sessionID string) ([]byte, *PaymentSession, error) {
	session, err := client.GetPaymentSession(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}
	if !IsSuccess(session.Status) {
		return nil, session, ErrPaymentNotSettled
	}
	receipt, err := client.GetReceiptPDF(ctx, sessionID)
	if err != nil {
		return nil, session, err
	}
	return receipt, session, nil
}
