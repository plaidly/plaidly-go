package plaidly

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateCheckoutIntent_RequestShapeAndResponse(t *testing.T) {
	var gotBody CreatePaymentSessionRequest
	var gotPath, gotMethod, gotIdemHeader string

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotIdemHeader = r.Header.Get("Idempotency-Key")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"session_id": "ps_intent_1",
			"merchant_id": "m_1",
			"expected_amount": 50,
			"received_amount": 0,
			"status": "awaiting_method_selection",
			"currency": "USDC",
			"candidate_payment_methods": [
				{"methodID": 0, "chain": "solana", "network": "mainnet", "token": "USDC"},
				{"methodID": 0, "chain": "ethereum", "network": "mainnet", "token": "USDC"}
			],
			"policy_version": "v1",
			"metadata": {},
			"expires_at": "2026-07-11T21:00:00Z",
			"created_at": "2026-07-11T20:45:00Z",
			"updated_at": "2026-07-11T20:45:00Z",
			"demo": false
		}`))
	})

	req := CreatePaymentSessionRequest{
		Amount:    50,
		ExpiresIn: "15m",
		PaymentMethods: &[]PaymentMethod{
			{MethodID: MethodIDCrypto, Chain: "solana", Network: "mainnet", Token: "USDC"},
			{MethodID: MethodIDCrypto, Chain: "ethereum", Network: "mainnet", Token: "USDC"},
		},
	}

	intent, err := c.CreateCheckoutIntent(context.Background(), req, WithIdempotencyKey("compono-checkout-uuid-1"))
	if err != nil {
		t.Fatalf("CreateCheckoutIntent: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/v1/payment_sessions" {
		t.Errorf("path = %q, want /v1/payment_sessions", gotPath)
	}
	if gotIdemHeader != "compono-checkout-uuid-1" {
		t.Errorf("Idempotency-Key header = %q, want compono-checkout-uuid-1", gotIdemHeader)
	}
	if gotBody.PaymentMethod != nil {
		t.Errorf("request carried a singular PaymentMethod, want none")
	}
	if gotBody.PaymentMethods == nil || len(*gotBody.PaymentMethods) != 2 {
		t.Errorf("request carried %v methods, want 2", gotBody.PaymentMethods)
	}

	if intent.SessionId != "ps_intent_1" {
		t.Errorf("SessionId = %q, want ps_intent_1", intent.SessionId)
	}
	if intent.Status != CheckoutIntentStatusAwaitingSelection {
		t.Errorf("Status = %q, want %q", intent.Status, CheckoutIntentStatusAwaitingSelection)
	}
	if intent.PolicyVersion == nil || *intent.PolicyVersion != "v1" {
		t.Errorf("PolicyVersion = %v, want v1", intent.PolicyVersion)
	}
	if intent.CandidatePaymentMethods == nil || len(*intent.CandidatePaymentMethods) != 2 {
		t.Errorf("CandidatePaymentMethods = %v, want 2 entries", intent.CandidatePaymentMethods)
	}
}

func TestGetCheckoutIntent_RequestShape(t *testing.T) {
	var gotPath, gotMethod string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"session_id": "ps_intent_2",
			"merchant_id": "m_1",
			"expected_amount": 50,
			"received_amount": 0,
			"status": "awaiting_method_selection",
			"metadata": {},
			"expires_at": "2026-07-11T21:00:00Z",
			"created_at": "2026-07-11T20:45:00Z",
			"updated_at": "2026-07-11T20:45:00Z",
			"demo": false
		}`))
	})

	intent, err := c.GetCheckoutIntent(context.Background(), "ps_intent_2")
	if err != nil {
		t.Fatalf("GetCheckoutIntent: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/v1/payment_sessions/ps_intent_2" {
		t.Errorf("path = %q, want /v1/payment_sessions/ps_intent_2", gotPath)
	}
	if intent.SessionId != "ps_intent_2" {
		t.Errorf("SessionId = %q, want ps_intent_2", intent.SessionId)
	}
}

func TestSelectCheckoutMethod_RequestShapeAndResponse(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody SelectPaymentMethodRequest

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"session_id": "ps_intent_3",
			"merchant_id": "m_1",
			"expected_amount": 50,
			"received_amount": 0,
			"address": "So1anaAddr111",
			"status": "pending",
			"payment_url": "https://pay.plaidly.io/ps_intent_3",
			"qr_data": "solana:So1anaAddr111?amount=50",
			"paymentMethod": {"methodID": 0, "chain": "solana", "network": "mainnet", "token": "USDC"},
			"policy_version": "v1",
			"method_selected_at": "2026-07-11T20:46:00Z",
			"metadata": {},
			"expires_at": "2026-07-11T21:00:00Z",
			"created_at": "2026-07-11T20:45:00Z",
			"updated_at": "2026-07-11T20:46:00Z",
			"demo": false
		}`))
	})

	selected, err := c.SelectCheckoutMethod(context.Background(), "ps_intent_3", PaymentMethod{
		MethodID: MethodIDCrypto, Chain: "solana", Network: "mainnet", Token: "USDC",
	})
	if err != nil {
		t.Fatalf("SelectCheckoutMethod: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/v1/payment_sessions/ps_intent_3/select_method" {
		t.Errorf("path = %q, want /v1/payment_sessions/ps_intent_3/select_method", gotPath)
	}
	if gotBody.PaymentMethod.Chain != "solana" {
		t.Errorf("request PaymentMethod.Chain = %q, want solana", gotBody.PaymentMethod.Chain)
	}
	if selected.SessionId != "ps_intent_3" {
		t.Errorf("SessionId = %q, want ps_intent_3", selected.SessionId)
	}
	if selected.Address != "So1anaAddr111" {
		t.Errorf("Address = %q, want So1anaAddr111", selected.Address)
	}
	if selected.PaymentMethod.Chain != "solana" {
		t.Errorf("PaymentMethod = %+v, want chain solana", selected.PaymentMethod)
	}
}

func TestCreateCheckoutIntent_ConflictSurfacesTypedError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":5,"message":"idempotency key was already used with a different payload"}`))
	})

	req := CreatePaymentSessionRequest{
		Amount:    1,
		ExpiresIn: "15m",
		PaymentMethods: &[]PaymentMethod{
			{MethodID: MethodIDCrypto, Chain: "solana", Network: "mainnet", Token: "USDC"},
		},
	}
	_, err := c.CreateCheckoutIntent(context.Background(), req, WithIdempotencyKey("dup"))
	if !IsIdempotencyConflict(err) {
		t.Errorf("IsIdempotencyConflict(err) = false, want true (err=%v)", err)
	}
}
