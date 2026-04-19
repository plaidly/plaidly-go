package plaidly

import (
	"context"
	"net/http"
)

// MerchantsService handles operations on merchant accounts.
type MerchantsService struct {
	client *Client
}

// Register creates a new merchant account.
//
// See: POST /v1/merchants
func (s *MerchantsService) Register(ctx context.Context, req CreateMerchantRequest) (*Merchant, error) {
	var merchant Merchant
	if err := s.client.do(ctx, http.MethodPost, "/v1/merchants", req, &merchant); err != nil {
		return nil, err
	}
	return &merchant, nil
}

// Me returns the authenticated merchant's profile.
//
// See: GET /v1/me
func (s *MerchantsService) Me(ctx context.Context) (*Merchant, error) {
	var merchant Merchant
	if err := s.client.do(ctx, http.MethodGet, "/v1/me", nil, &merchant); err != nil {
		return nil, err
	}
	return &merchant, nil
}
