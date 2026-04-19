package plaidly

import (
	"context"
	"fmt"
	"net/http"
)

// PayoutsService handles payout operations.
type PayoutsService struct {
	client *Client
}

// Create requests a new payout.
//
// See: POST /v1/payouts
func (s *PayoutsService) Create(ctx context.Context, req CreatePayoutRequest) (*Payout, error) {
	var payout Payout
	if err := s.client.do(ctx, http.MethodPost, "/v1/payouts", req, &payout); err != nil {
		return nil, err
	}
	return &payout, nil
}

// Get fetches a payout by ID.
//
// See: GET /v1/payouts/{id}
func (s *PayoutsService) Get(ctx context.Context, id string) (*Payout, error) {
	var payout Payout
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/v1/payouts/%s", id), nil, &payout); err != nil {
		return nil, err
	}
	return &payout, nil
}
