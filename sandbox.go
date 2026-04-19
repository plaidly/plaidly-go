package plaidly

import (
	"context"
	"net/http"
)

// SandboxService provides sandbox-only helpers.
type SandboxService struct {
	client *Client
}

// Faucets returns available testnet faucets.
//
// See: GET /v1/sandbox/faucets
func (s *SandboxService) Faucets(ctx context.Context) ([]Faucet, error) {
	var faucets []Faucet
	if err := s.client.do(ctx, http.MethodGet, "/v1/sandbox/faucets", nil, &faucets); err != nil {
		return nil, err
	}
	return faucets, nil
}
