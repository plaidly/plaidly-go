package plaidly

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateCheckoutIntent_RequestShapeAndResponse(t *testing.T) {
	var gotBody CreateCheckoutIntentRequest
	var gotPath, gotIdemHeader string

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotIdemHeader = r.Header.Get("Idempotency-Key")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"intent_id": "ci_test_1",
			"status": "open",
			"amount": 50,
			"currency": "USDC",
			"methods": [
				{"chain": "solana", "network": "mainnet", "token": "USDC"},
				{"chain": "ethereum", "network": "mainnet", "token": "USDC"}
			],
			"policy_version": "v1",
			"expires_at": "2026-07-11T21:00:00Z",
			"created_at": "2026-07-11T20:45:00Z"
		}`))
	})

	req := CreateCheckoutIntentRequest{
		Amount:   50,
		Currency: "USDC",
		Methods: []CheckoutMethodOption{
			{Chain: "solana", Network: "mainnet", Token: "USDC"},
			{Chain: "ethereum", Network: "mainnet", Token: "USDC"},
		},
	}

	intent, err := c.CreateCheckoutIntent(context.Background(), req, WithIdempotencyKey("compono-checkout-uuid-1"))
	if err != nil {
		t.Fatalf("CreateCheckoutIntent: %v", err)
	}

	if gotPath != "/v1/checkout_intents" {
		t.Errorf("path = %q, want /v1/checkout_intents", gotPath)
	}
	if gotIdemHeader != "compono-checkout-uuid-1" {
		t.Errorf("Idempotency-Key header = %q, want compono-checkout-uuid-1", gotIdemHeader)
	}
	if len(gotBody.Methods) != 2 {
		t.Errorf("request carried %d methods, want 2", len(gotBody.Methods))
	}

	if intent.IntentID != "ci_test_1" {
		t.Errorf("IntentID = %q, want ci_test_1", intent.IntentID)
	}
	if intent.PolicyVersion != "v1" {
		t.Errorf("PolicyVersion = %q, want v1", intent.PolicyVersion)
	}
	if len(intent.Methods) != 2 {
		t.Errorf("Methods = %d, want 2", len(intent.Methods))
	}
	if intent.ExpiresAt.IsZero() {
		t.Error("ExpiresAt is zero, want parsed timestamp")
	}
}

func TestGetCheckoutIntent_RequestShape(t *testing.T) {
	var gotPath, gotMethod string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"intent_id":"ci_test_2","status":"open","policy_version":"v1","expires_at":"2026-07-11T21:00:00Z","created_at":"2026-07-11T20:45:00Z"}`))
	})

	intent, err := c.GetCheckoutIntent(context.Background(), "ci_test_2")
	if err != nil {
		t.Fatalf("GetCheckoutIntent: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/v1/checkout_intents/ci_test_2" {
		t.Errorf("path = %q, want /v1/checkout_intents/ci_test_2", gotPath)
	}
	if intent.IntentID != "ci_test_2" {
		t.Errorf("IntentID = %q, want ci_test_2", intent.IntentID)
	}
}

func TestSelectCheckoutMethod_RequestShapeAndResponse(t *testing.T) {
	var gotPath string
	var gotBody SelectCheckoutMethodRequest

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"intent_id": "ci_test_3",
			"session_id": "ps_bound_1",
			"chain": "solana",
			"network": "mainnet",
			"token": "USDC",
			"deposit_address": "So1anaAddr111",
			"payment_url": "https://pay.plaidly.io/ps_bound_1",
			"qr_code_url": "https://pay.plaidly.io/ps_bound_1/qr.png",
			"expires_at": "2026-07-11T21:00:00Z"
		}`))
	})

	selected, err := c.SelectCheckoutMethod(context.Background(), "ci_test_3", SelectCheckoutMethodRequest{
		Chain: "solana", Network: "mainnet", Token: "USDC",
	})
	if err != nil {
		t.Fatalf("SelectCheckoutMethod: %v", err)
	}

	if gotPath != "/v1/checkout_intents/ci_test_3/select" {
		t.Errorf("path = %q, want /v1/checkout_intents/ci_test_3/select", gotPath)
	}
	if gotBody.Chain != "solana" {
		t.Errorf("request Chain = %q, want solana", gotBody.Chain)
	}
	if selected.SessionID != "ps_bound_1" {
		t.Errorf("SessionID = %q, want ps_bound_1", selected.SessionID)
	}
	if selected.DepositAddress != "So1anaAddr111" {
		t.Errorf("DepositAddress = %q, want So1anaAddr111", selected.DepositAddress)
	}
}

func TestCreateCheckoutIntent_ConflictSurfacesTypedError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":5,"message":"idempotency key was already used with a different payload"}`))
	})

	_, err := c.CreateCheckoutIntent(context.Background(), CreateCheckoutIntentRequest{Amount: 1}, WithIdempotencyKey("dup"))
	if !IsIdempotencyConflict(err) {
		t.Errorf("IsIdempotencyConflict(err) = false, want true (err=%v)", err)
	}
}
