package plaidly

import (
	"context"
	"net/http"
)

// RegisterMerchant creates a new merchant account.
//
// POST /v1/merchants
func (c *Client) RegisterMerchant(ctx context.Context, req RegisterMerchantRequest) (*Merchant, error) {
	var out Merchant
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.RegisterMerchant(ctx, req)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetMe returns the authenticated merchant's profile.
//
// GET /v1/me
func (c *Client) GetMe(ctx context.Context) (*Merchant, error) {
	var out Merchant
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.GetMe(ctx)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetPaymentMethodPolicy returns the authenticated merchant's current
// declared payment-method policy version together with live-computed
// eligibility for every candidate identity (BDT-529). A merchant with no
// policy version yet, or a version with zero enabled entries, gets an empty
// effective policy; it never falls back to every globally supported method.
//
// GET /v1/me/payment_method_policy
func (c *Client) GetPaymentMethodPolicy(ctx context.Context) (*PaymentMethodPolicyState, error) {
	var out PaymentMethodPolicyState
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.GetPaymentMethodPolicy(ctx)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdatePaymentMethodPolicy replaces the authenticated merchant's full
// desired set of enabled chain/network/token identities, writing a new
// immutable policy version. Every requested identity is validated
// server-side against the canonical registry, settlement implementation,
// environment enablement, and certification gates; if any identity fails,
// the whole request is rejected and no version is written.
//
// PUT /v1/me/payment_method_policy
func (c *Client) UpdatePaymentMethodPolicy(ctx context.Context, req UpdatePaymentMethodPolicyRequest) (*PaymentMethodPolicyState, error) {
	var out PaymentMethodPolicyState
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.UpdatePaymentMethodPolicy(ctx, req)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
