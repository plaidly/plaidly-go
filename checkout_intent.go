package plaidly

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

const (
	CheckoutIntentStatusOpen     = "open"
	CheckoutIntentStatusSelected = "selected"
	CheckoutIntentStatusExpired  = "expired"
	CheckoutIntentStatusCanceled = "canceled"
)

const (
	EventCheckoutIntentMethodSelected = "checkout_intent.method_selected"
	EventCheckoutIntentExpired        = "checkout_intent.expired"
)

type CheckoutMethodOption struct {
	Chain   string `json:"chain"`
	Network string `json:"network"`
	Token   string `json:"token"`
}

type CreateCheckoutIntentRequest struct {
	Amount    float64                 `json:"amount"`
	Currency  string                  `json:"currency,omitempty"`
	ExpiresIn string                  `json:"expires_in,omitempty"`
	Methods   []CheckoutMethodOption  `json:"methods"`
	Metadata  *map[string]interface{} `json:"metadata,omitempty"`
}

type CheckoutIntent struct {
	IntentID      string                 `json:"intent_id"`
	Status        string                 `json:"status"`
	Amount        float64                `json:"amount"`
	Currency      string                 `json:"currency"`
	Methods       []CheckoutMethodOption `json:"methods"`
	PolicyVersion string                 `json:"policy_version"`
	ExpiresAt     time.Time              `json:"expires_at"`
	CreatedAt     time.Time              `json:"created_at"`
	SelectedIndex *int                   `json:"selected_index,omitempty"`
}

type SelectCheckoutMethodRequest struct {
	Chain   string `json:"chain"`
	Network string `json:"network"`
	Token   string `json:"token"`
}

type SelectedCheckoutMethod struct {
	IntentID       string    `json:"intent_id"`
	SessionID      string    `json:"session_id"`
	Chain          string    `json:"chain"`
	Network        string    `json:"network"`
	Token          string    `json:"token"`
	DepositAddress string    `json:"deposit_address"`
	PaymentURL     string    `json:"payment_url"`
	QRCodeURL      string    `json:"qr_code_url,omitempty"`
	DeepLink       string    `json:"deep_link,omitempty"`
	ExpiresAt      time.Time `json:"expires_at"`
}

func (c *Client) CreateCheckoutIntent(ctx context.Context, req CreateCheckoutIntentRequest, opts ...RequestOption) (*CheckoutIntent, error) {
	ro := buildRequestOptions(opts)
	var out CheckoutIntent
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.doRequest(ctx, http.MethodPost, "/v1/checkout_intents", req, ro)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetCheckoutIntent(ctx context.Context, intentID string, opts ...RequestOption) (*CheckoutIntent, error) {
	ro := buildRequestOptions(opts)
	var out CheckoutIntent
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.doRequest(ctx, http.MethodGet, "/v1/checkout_intents/"+url.PathEscape(intentID), nil, ro)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SelectCheckoutMethod(ctx context.Context, intentID string, req SelectCheckoutMethodRequest, opts ...RequestOption) (*SelectedCheckoutMethod, error) {
	ro := buildRequestOptions(opts)
	var out SelectedCheckoutMethod
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.doRequest(ctx, http.MethodPost, "/v1/checkout_intents/"+url.PathEscape(intentID)+"/select", req, ro)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
