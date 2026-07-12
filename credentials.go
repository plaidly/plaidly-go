package plaidly

import (
	"context"
	"net/http"
)

// GetMerchantCredentialStatus returns non-sensitive metadata about the
// authenticated merchant's current API key and webhook secret: masked
// previews, creation/rotation timestamps, and rotation-overlap grace period
// expiry. It never returns a usable credential value.
//
// GET /v1/me/credentials
func (c *Client) GetMerchantCredentialStatus(ctx context.Context) (*MerchantCredentialStatus, error) {
	var out MerchantCredentialStatus
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.GetMerchantCredentialStatus(ctx)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// RotateMerchantCredential rotates the authenticated merchant's API key or
// webhook secret and keeps the previous credential valid for a short overlap
// window. The returned Merchant carries the new credential value in ApiKey
// or WebhookSecret (response-only fields) exactly once; callers must capture
// it immediately, it is never returned again.
//
// POST /v1/me/credentials/rotate
func (c *Client) RotateMerchantCredential(ctx context.Context, credentialType MerchantCredentialType, opts ...RequestOption) (*Merchant, error) {
	ro := buildRequestOptions(opts)
	var out Merchant
	body := MerchantCredentialOperationRequest{CredentialType: credentialType}
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.RotateMerchantCredential(ctx, body, ro.toEditors()...)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// RevokeMerchantCredential revokes the previous credential created by the
// most recent rotation, ending the overlap window early.
//
// POST /v1/me/credentials/revoke
func (c *Client) RevokeMerchantCredential(ctx context.Context, credentialType MerchantCredentialType, opts ...RequestOption) (*Merchant, error) {
	ro := buildRequestOptions(opts)
	var out Merchant
	body := MerchantCredentialOperationRequest{CredentialType: credentialType}
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.RevokeMerchantCredential(ctx, body, ro.toEditors()...)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateMerchantWebhook replaces the authenticated merchant's webhook URL
// after SSRF validation (blocks localhost/metadata/internal/private-range
// hosts).
//
// PUT /v1/me/webhook
func (c *Client) UpdateMerchantWebhook(ctx context.Context, webhookURL string, opts ...RequestOption) (*Merchant, error) {
	ro := buildRequestOptions(opts)
	var out Merchant
	body := UpdateMerchantWebhookRequest{WebhookUrl: webhookURL}
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.UpdateMerchantWebhook(ctx, body, ro.toEditors()...)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// TestMerchantWebhookDelivery sends a synthetic event through the same
// signed, SSRF-guarded delivery path used for real webhook events, so the
// merchant can confirm their endpoint receives and verifies deliveries.
//
// POST /v1/me/webhook/test
func (c *Client) TestMerchantWebhookDelivery(ctx context.Context) (*WebhookTestDeliveryResult, error) {
	var out WebhookTestDeliveryResult
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.TestMerchantWebhookDelivery(ctx)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
