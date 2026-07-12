package plaidly

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := NewClient("pk_test_123", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, srv
}

func TestNewClient_RequiresAPIKey(t *testing.T) {
	if _, err := NewClient(""); err == nil {
		t.Fatal("expected error for empty api key")
	}
}

func TestCreatePaymentSession(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/payment_sessions" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "pk_test_123" {
			t.Errorf("X-API-Key = %q", got)
		}
		var req CreatePaymentSessionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Amount != 25.5 || req.ExpiresIn != "15m" {
			t.Errorf("body = %+v", req)
		}
		if req.PaymentMethod.Chain != "solana" || req.PaymentMethod.Token != "USDC" {
			t.Errorf("paymentMethod = %+v", req.PaymentMethod)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": "ps_1", "merchant_id": "m_1", "expected_amount": 25.5,
			"received_amount": 0, "address": "Sol111", "status": "pending",
			"metadata": map[string]any{}, "expires_at": "t", "created_at": "t",
			"updated_at": "t", "demo": false, "payment_url": "https://pay.plaidly.io/ps_1",
			"paymentMethod": map[string]any{"methodID": 0, "chain": "solana", "token": "USDC", "network": "mainnet"},
		})
	})

	got, err := c.CreatePaymentSession(context.Background(), CreatePaymentSessionRequest{
		Amount:    25.5,
		ExpiresIn: "15m",
		PaymentMethod: &PaymentMethod{
			MethodID: MethodIDCrypto, Chain: "solana", Token: "USDC", Network: "mainnet",
		},
	})
	if err != nil {
		t.Fatalf("CreatePaymentSession: %v", err)
	}
	if got.SessionId != "ps_1" || got.Address != "Sol111" {
		t.Errorf("session = %+v", got)
	}
	if got.PaymentUrl == nil || *got.PaymentUrl != "https://pay.plaidly.io/ps_1" {
		t.Errorf("payment_url = %v", got.PaymentUrl)
	}
}

func TestGetPaymentSession(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/payment_sessions/ps_42" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": "ps_42", "merchant_id": "m", "expected_amount": 1,
			"received_amount": 1, "address": "a", "status": "completed",
			"metadata": map[string]any{}, "expires_at": "t", "created_at": "t",
			"updated_at": "t", "demo": true,
			"paymentMethod": map[string]any{"methodID": 0, "chain": "ethereum", "token": "ETH", "network": "testnet"},
		})
	})
	got, err := c.GetPaymentSession(context.Background(), "ps_42")
	if err != nil {
		t.Fatalf("GetPaymentSession: %v", err)
	}
	if got.Status != "completed" || !got.Demo {
		t.Errorf("session = %+v", got)
	}
}

func TestCreateDemoPaymentSession(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/payment_sessions/demo" || r.Method != http.MethodPost {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": "demo_1", "merchant_id": "m", "expected_amount": 5,
			"received_amount": 0, "address": "a", "status": "pending",
			"metadata": map[string]any{}, "expires_at": "t", "created_at": "t",
			"updated_at": "t", "demo": true,
			"paymentMethod": map[string]any{"methodID": 0, "chain": "solana", "token": "SOL", "network": "testnet"},
		})
	})
	got, err := c.CreateDemoPaymentSession(context.Background())
	if err != nil {
		t.Fatalf("CreateDemoPaymentSession: %v", err)
	}
	if got.SessionId != "demo_1" || !got.Demo {
		t.Errorf("session = %+v", got)
	}
}

func TestSimulatePayment(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/payment_sessions/demo_1/simulate" || r.Method != http.MethodPost {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": "demo_1", "merchant_id": "m", "expected_amount": 5,
			"received_amount": 5, "address": "a", "status": "completed",
			"metadata": map[string]any{}, "expires_at": "t", "created_at": "t",
			"updated_at": "t", "demo": true,
			"paymentMethod": map[string]any{"methodID": 0, "chain": "solana", "token": "SOL", "network": "testnet"},
		})
	})
	got, err := c.SimulatePayment(context.Background(), "demo_1")
	if err != nil {
		t.Fatalf("SimulatePayment: %v", err)
	}
	if !IsSuccess(got.Status) {
		t.Errorf("status = %q, want success", got.Status)
	}
}

func TestListPaymentMethods(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/payment_methods" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"chain": "solana", "network": "mainnet", "token": "USDC", "display_name": "USD Coin", "decimals": 6, "kind": "spl", "min_amount": 0.01},
			{"chain": "ethereum", "network": "mainnet", "token": "ETH", "display_name": "Ether", "decimals": 18, "kind": "native"},
		})
	})
	got, err := c.ListPaymentMethods(context.Background())
	if err != nil {
		t.Fatalf("ListPaymentMethods: %v", err)
	}
	if len(got) != 2 || got[0].Token != "USDC" || got[1].Kind != "native" {
		t.Errorf("methods = %+v", got)
	}
}

func TestGetRates(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/rates" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("symbols"); got != "ETH,SOL" {
			t.Errorf("symbols = %q", got)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"symbol": "ETH", "usd": 3500.5, "updated_at": "t"},
			{"symbol": "SOL", "usd": 150.25, "updated_at": "t"},
		})
	})
	got, err := c.GetRates(context.Background(), "ETH", "SOL")
	if err != nil {
		t.Fatalf("GetRates: %v", err)
	}
	if len(got) != 2 || got[0].Symbol != "ETH" || got[0].Usd != 3500.5 {
		t.Errorf("rates = %+v", got)
	}
}

func TestListSandboxFaucets(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sandbox/faucets" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"solana:testnet":   "https://faucet.solana.com",
			"ethereum:testnet": "https://holesky-faucet.pk910.de",
		})
	})
	got, err := c.ListSandboxFaucets(context.Background())
	if err != nil {
		t.Fatalf("ListSandboxFaucets: %v", err)
	}
	if got["solana:testnet"] != "https://faucet.solana.com" {
		t.Errorf("faucets = %+v", got)
	}
}

func TestRegisterMerchant(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/merchants" || r.Method != http.MethodPost {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "m_1", "name": "Acme", "api_key": "pk_live_x",
			"webhook_secret": "whsec_y", "created_at": "t",
		})
	})
	got, err := c.RegisterMerchant(context.Background(), RegisterMerchantRequest{Name: "Acme"})
	if err != nil {
		t.Fatalf("RegisterMerchant: %v", err)
	}
	if got.ApiKey != "pk_live_x" || got.WebhookSecret == nil || *got.WebhookSecret != "whsec_y" {
		t.Errorf("merchant = %+v", got)
	}
}

func TestAPIError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": "bad_request", "message": "amount too low"})
	})
	_, err := c.GetPaymentSession(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("err type = %T", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest || apiErr.Message != "amount too low" {
		t.Errorf("apiErr = %+v", apiErr)
	}
}

func TestRetryOn5xx(t *testing.T) {
	var calls int32
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": "unavailable", "message": "try later"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": "ps_ok", "merchant_id": "m", "expected_amount": 1,
			"received_amount": 0, "address": "a", "status": "pending",
			"metadata": map[string]any{}, "expires_at": "t", "created_at": "t",
			"updated_at": "t", "demo": false,
			"paymentMethod": map[string]any{"methodID": 0, "chain": "solana", "token": "SOL", "network": "mainnet"},
		})
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	got, err := c.GetPaymentSession(ctx, "ps_ok")
	if err != nil {
		t.Fatalf("GetPaymentSession: %v", err)
	}
	if got.SessionId != "ps_ok" {
		t.Errorf("session = %+v", got)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestNoRetryOn4xx(t *testing.T) {
	var calls int32
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": "bad", "message": "nope"})
	})
	_, _ = c.GetPaymentSession(context.Background(), "x")
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("calls = %d, want 1 (no retry on 4xx)", calls)
	}
}
