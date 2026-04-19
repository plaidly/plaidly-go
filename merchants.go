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
