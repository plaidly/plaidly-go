package plaidly

import (
	"context"
	"net/http"
)

// RequestPayout requests a new payout.
//
// POST /v1/payouts
func (c *Client) RequestPayout(ctx context.Context, req RequestPayoutRequest, opts ...RequestOption) (*Payout, error) {
	ro := buildRequestOptions(opts)
	var out Payout
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.RequestPayout(ctx, req, ro.toEditors()...)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetPayout fetches a payout by ID.
//
// GET /v1/payouts/{payout_id}
func (c *Client) GetPayout(ctx context.Context, payoutID string) (*Payout, error) {
	var out Payout
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.GetPayout(ctx, payoutID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
