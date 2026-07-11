// Package telegram provides ergonomic helpers for integrating Plaidly crypto
// payments into Telegram bots.
//
// It wraps the plaidly client with bot-friendly conveniences:
//
//   - CreateInvoice builds a payment session and renders ready-to-send HTML
//     message text plus inline-keyboard button structs.
//   - VerifyWebhook validates an incoming Plaidly webhook from an *http.Request.
//   - PollSession polls a session to completion with backoff, for bots that
//     cannot receive webhooks (e.g. long-polling deployments).
//
// The keyboard structs are framework-agnostic: they carry the data needed to
// build an inline button in any Telegram bot library (telebot, telegram-bot-api,
// etc.) without taking a dependency on one.
package telegram

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	plaidly "github.com/plaidly/plaidly-go"
)

// Helper wraps a Plaidly client with Telegram-oriented conveniences.
type Helper struct {
	client *plaidly.Client
}

// New returns a Helper around an existing Plaidly client.
func New(client *plaidly.Client) *Helper {
	return &Helper{client: client}
}

// Client returns the underlying Plaidly client.
func (h *Helper) Client() *plaidly.Client { return h.client }

// InvoiceOptions configures CreateInvoice.
type InvoiceOptions struct {
	// Amount to charge, denominated in Token.
	Amount float64
	// Token symbol (e.g. "USDC", "ETH", "SOL").
	Token string
	// Chain the payment runs on (e.g. "solana", "ethereum").
	Chain string
	// Network is "mainnet" or "testnet". Defaults to "mainnet" when empty.
	Network string
	// ExpiresIn is a Go-style duration string accepted by the API (e.g. "15m",
	// "1h"). Defaults to "15m" when empty.
	ExpiresIn string
	// Metadata is attached to the session and echoed back in webhooks.
	Metadata map[string]any
	// ButtonText is the label for the pay URL button. Defaults to "Pay now".
	ButtonText     string
	IdempotencyKey string
}

// Invoice is the result of CreateInvoice: a created session plus everything a
// bot needs to present it to the user.
type Invoice struct {
	// Session is the created payment session.
	Session *plaidly.PaymentSession
	// PaymentURL is the hosted checkout link for the payer.
	PaymentURL string
	// MessageText is HTML-formatted message body (Telegram parse_mode=HTML).
	MessageText string
	// PayButton opens the hosted checkout in the user's browser.
	PayButton URLButton
	// WebAppButton opens the hosted checkout as a Telegram Web App.
	WebAppButton WebAppButton
}

// URLButton describes an inline keyboard button that opens a URL.
type URLButton struct {
	Text string
	URL  string
}

// WebAppButton describes an inline keyboard button that opens a Telegram Web App.
type WebAppButton struct {
	Text string
	URL  string
}

// ErrNoPaymentURL is returned by CreateInvoice when the API did not provide a
// hosted checkout URL for the created session.
var ErrNoPaymentURL = errors.New("plaidly/telegram: session has no payment_url")

// CreateInvoice creates a payment session and renders bot-ready output.
//
// amount is denominated in token. chain and network identify the asset. The
// returned Invoice carries the session, the hosted checkout URL, HTML message
// text, and inline-keyboard button structs.
func (h *Helper) CreateInvoice(ctx context.Context, amount float64, token, chain, network string, metadata map[string]any) (*Invoice, error) {
	return h.CreateInvoiceWithOptions(ctx, InvoiceOptions{
		Amount:   amount,
		Token:    token,
		Chain:    chain,
		Network:  network,
		Metadata: metadata,
	})
}

// CreateInvoiceWithOptions is CreateInvoice with full control over options.
func (h *Helper) CreateInvoiceWithOptions(ctx context.Context, opts InvoiceOptions) (*Invoice, error) {
	network := opts.Network
	if network == "" {
		network = "mainnet"
	}
	expiresIn := opts.ExpiresIn
	if expiresIn == "" {
		expiresIn = "15m"
	}
	buttonText := opts.ButtonText
	if buttonText == "" {
		buttonText = "Pay now"
	}

	req := plaidly.CreatePaymentSessionRequest{
		Amount:    opts.Amount,
		ExpiresIn: expiresIn,
		PaymentMethod: plaidly.PaymentMethod{
			MethodID: plaidly.MethodIDCrypto,
			Chain:    opts.Chain,
			Token:    opts.Token,
			Network:  network,
		},
	}
	if len(opts.Metadata) > 0 {
		md := map[string]any(opts.Metadata)
		req.Metadata = &md
	}

	var callOpts []plaidly.RequestOption
	if opts.IdempotencyKey != "" {
		callOpts = append(callOpts, plaidly.WithIdempotencyKey(opts.IdempotencyKey))
	}
	session, err := h.client.CreatePaymentSession(ctx, req, callOpts...)
	if err != nil {
		return nil, err
	}

	payURL := ""
	if session.PaymentUrl != nil {
		payURL = *session.PaymentUrl
	}
	if payURL == "" {
		return nil, ErrNoPaymentURL
	}

	inv := &Invoice{
		Session:      session,
		PaymentURL:   payURL,
		MessageText:  InvoiceMessage(session, opts.Amount, opts.Token, opts.Chain, network),
		PayButton:    URLButton{Text: buttonText, URL: payURL},
		WebAppButton: WebAppButton{Text: buttonText, URL: payURL},
	}
	return inv, nil
}

// InvoiceMessage renders an HTML message body describing the invoice. It is
// safe for Telegram parse_mode=HTML: all interpolated values are escaped.
func InvoiceMessage(session *plaidly.PaymentSession, amount float64, token, chain, network string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<b>Payment requested</b>\n\n")
	fmt.Fprintf(&b, "Amount: <b>%s %s</b>\n", html.EscapeString(formatAmount(amount)), html.EscapeString(token))
	fmt.Fprintf(&b, "Chain: <b>%s</b> (%s)\n", html.EscapeString(chain), html.EscapeString(network))
	if session != nil {
		fmt.Fprintf(&b, "Send to:\n<code>%s</code>\n", html.EscapeString(session.Address))
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatAmount(amount float64) string {
	s := strconv.FormatFloat(amount, 'f', -1, 64)
	return s
}

// VerifyWebhook reads, verifies, and decodes a Plaidly webhook delivery from an
// *http.Request. It uses the X-Plaidly-Signature header, secret, and tolerance.
// Pass tolerance <= 0 to use plaidly.DefaultWebhookTolerance.
//
// The request body is fully consumed. Callers that need the raw body elsewhere
// should buffer it before calling.
func VerifyWebhook(r *http.Request, secret string, tolerance time.Duration) (*plaidly.WebhookEvent, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("plaidly/telegram: read webhook body: %w", err)
	}
	sig := r.Header.Get(plaidly.SignatureHeader)
	return plaidly.VerifyWebhook(body, sig, secret, tolerance)
}

// PollOptions configures PollSession.
type PollOptions struct {
	// Interval is the initial poll interval. Defaults to 3s when <= 0.
	Interval time.Duration
	// MaxInterval caps the backed-off interval. Defaults to 30s when <= 0.
	MaxInterval time.Duration
	// MaxDuration bounds the total wait. Defaults to 15m when <= 0.
	MaxDuration time.Duration
	// Backoff multiplies Interval after each poll. Defaults to 1.5 when <= 1.
	Backoff float64
}

// ErrPollTimeout is returned by PollSession when MaxDuration elapses before the
// session reaches a terminal state.
var ErrPollTimeout = errors.New("plaidly/telegram: poll timed out before session settled")

// PollSession polls a session until it reaches a terminal state (completed,
// confirmed, expired, or failed) or the deadline passes. The poll interval
// backs off geometrically up to MaxInterval. interval seeds PollOptions.Interval
// for the common case; pass 0 to use the default.
//
// It returns the final session. A terminal-but-failed session (expired/failed)
// is returned with a nil error; inspect Status or use plaidly.IsSuccess.
// ErrPollTimeout is returned only when the deadline is hit first.
func (h *Helper) PollSession(ctx context.Context, sessionID string, interval time.Duration) (*plaidly.PaymentSession, error) {
	return h.PollSessionWithOptions(ctx, sessionID, PollOptions{Interval: interval})
}

// PollSessionWithOptions is PollSession with full backoff control.
func (h *Helper) PollSessionWithOptions(ctx context.Context, sessionID string, opts PollOptions) (*plaidly.PaymentSession, error) {
	interval := opts.Interval
	if interval <= 0 {
		interval = 3 * time.Second
	}
	maxInterval := opts.MaxInterval
	if maxInterval <= 0 {
		maxInterval = 30 * time.Second
	}
	maxDuration := opts.MaxDuration
	if maxDuration <= 0 {
		maxDuration = 15 * time.Minute
	}
	backoff := opts.Backoff
	if backoff <= 1 {
		backoff = 1.5
	}

	ctx, cancel := context.WithTimeout(ctx, maxDuration)
	defer cancel()

	for {
		session, err := h.client.GetPaymentSession(ctx, sessionID)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ErrPollTimeout
			}
			return nil, err
		}
		if plaidly.IsTerminal(session.Status) {
			return session, nil
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ErrPollTimeout
		case <-timer.C:
		}

		interval = time.Duration(float64(interval) * backoff)
		if interval > maxInterval {
			interval = maxInterval
		}
	}
}
