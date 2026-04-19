package plaidly

import (
	"context"
	"net/http"
)

// CreateWallet creates a new wallet.
//
// POST /v1/wallets
func (c *Client) CreateWallet(ctx context.Context, req CreateWalletRequest) (*Wallet, error) {
	var out Wallet
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.CreateWallet(ctx, req)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListWallets lists all wallets for the authenticated merchant.
//
// GET /v1/wallets
func (c *Client) ListWallets(ctx context.Context) ([]Wallet, error) {
	var out []Wallet
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.ListWallets(ctx)
	}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetWallet fetches a wallet by ID.
//
// GET /v1/wallets/{wallet_id}
func (c *Client) GetWallet(ctx context.Context, walletID string) (*Wallet, error) {
	var out Wallet
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.GetWallet(ctx, walletID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListTransactions lists transactions for a wallet.
//
// GET /v1/wallets/{wallet_id}/transactions
func (c *Client) ListTransactions(ctx context.Context, walletID string) ([]Transaction, error) {
	var out []Transaction
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.ListTransactions(ctx, walletID)
	}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
