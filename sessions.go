package plaidly

import (
	"context"
	"fmt"
	"net/http"
)

// SessionsService handles operations on payment sessions.
type SessionsService struct {
	client *Client
}

// Create creates a new payment session.
//
// See: POST /v1/sessions
func (s *SessionsService) Create(ctx context.Context, req CreateSessionRequest) (*Session, error) {
	var session Session
	if err := s.client.do(ctx, http.MethodPost, "/v1/sessions", req, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// Get fetches a payment session by ID.
//
// See: GET /v1/sessions/{id}
func (s *SessionsService) Get(ctx context.Context, id string) (*Session, error) {
	var session Session
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sessions/%s", id), nil, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// List returns all sessions for the authenticated merchant.
//
// See: GET /v1/sessions
func (s *SessionsService) List(ctx context.Context) (*ListSessionsResponse, error) {
	var resp ListSessionsResponse
	if err := s.client.do(ctx, http.MethodGet, "/v1/sessions", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Simulate triggers a simulated payment for a sandbox session.
//
// See: POST /v1/sessions/{id}/simulate
func (s *SessionsService) Simulate(ctx context.Context, id string, req SimulatePaymentRequest) error {
	return s.client.do(ctx, http.MethodPost, fmt.Sprintf("/v1/sessions/%s/simulate", id), req, nil)
}
