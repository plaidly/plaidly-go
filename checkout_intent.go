package plaidly

import (
	"context"
	"net/http"
)

// Multi-method checkout intent statuses (BDT-528, deployed to production).
// A checkout intent is a PaymentSessionOrIntent whose Status is
// CheckoutIntentStatusAwaitingSelection until a method is selected, at which
// point it materializes into an ordinary PaymentSession with one of the
// regular payment-session statuses (see StatusPending etc. in client.go).
const (
	CheckoutIntentStatusAwaitingSelection = "awaiting_method_selection"
)

// CreateCheckoutIntent creates a multi-method checkout intent: it is
// POST /v1/payment_sessions with req.PaymentMethods (plural, a candidate
// set of 1-20 PaymentMethod values) set instead of the single-method
// req.PaymentMethod field. The server intersects the candidate set with
// merchant policy, environment restriction, and currently enabled/certified
// rails, and returns a checkout intent for the payer to select from via
// SelectCheckoutMethod. Supplying both PaymentMethod and PaymentMethods, or
// neither, is a 400.
//
// The returned intent's SessionId doubles as the intent_id passed to
// GetCheckoutIntent and SelectCheckoutMethod.
//
// This flow was deployed to production as part of BDT-528; it is not a
// distinct /v1/checkout_intents resource, but the paymentMethods
// (plural) extension of the existing payment-session endpoints.
//
// POST /v1/payment_sessions
func (c *Client) CreateCheckoutIntent(ctx context.Context, req CreatePaymentSessionRequest, opts ...RequestOption) (*PaymentSessionOrIntent, error) {
	ro := buildRequestOptions(opts)
	var out PaymentSessionOrIntent
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.CreatePaymentSession(ctx, nil, req, ro.toEditors()...)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCheckoutIntent fetches a checkout intent by its id (the SessionId
// returned from CreateCheckoutIntent). Before a method has been selected
// this returns the intent's candidate snapshot (Status ==
// CheckoutIntentStatusAwaitingSelection, CandidatePaymentMethods populated);
// afterwards it returns the same materialized session GetPaymentSession
// would. This is the same underlying endpoint as GetPaymentSession.
//
// GET /v1/payment_sessions/{session_id}
func (c *Client) GetCheckoutIntent(ctx context.Context, intentID string) (*PaymentSessionOrIntent, error) {
	var out PaymentSessionOrIntent
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.GetPaymentSession(ctx, intentID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SelectCheckoutMethod selects one payment method from a checkout intent's
// merchant-approved candidate set, atomically materializing it into a bound
// payment session with a deposit address. Concurrent selections for the
// same intent converge to a single method/address; a losing caller receives
// the same response as the winner rather than an error.
//
// intentID is the SessionId returned by CreateCheckoutIntent. This endpoint
// is public: the payer authorizes with knowledge of the intent id, the same
// trust model as GetCheckoutIntent.
//
// POST /v1/payment_sessions/{session_id}/select_method
func (c *Client) SelectCheckoutMethod(ctx context.Context, intentID string, method PaymentMethod) (*PaymentSession, error) {
	req := SelectPaymentMethodRequest{PaymentMethod: method}
	var out PaymentSession
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.SelectPaymentMethod(ctx, intentID, req)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
