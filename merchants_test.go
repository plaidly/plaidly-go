package plaidly

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetPaymentMethodPolicy(t *testing.T) {
	var gotPath, gotMethod string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"version": 3, "environment": "live",
			"entries": [
				{"chain": "solana", "network": "mainnet", "token": "USDC", "kind": "spl", "enabled": true, "eligible": true, "effective": true}
			]
		}`))
	})

	policy, err := c.GetPaymentMethodPolicy(context.Background())
	if err != nil {
		t.Fatalf("GetPaymentMethodPolicy: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/v1/me/payment_method_policy" {
		t.Errorf("got %s %s, want GET /v1/me/payment_method_policy", gotMethod, gotPath)
	}
	if policy.Version != 3 || policy.Environment != "live" {
		t.Errorf("policy = %+v", policy)
	}
	if len(policy.Entries) != 1 || !policy.Entries[0].Effective {
		t.Errorf("Entries = %+v", policy.Entries)
	}
}

func TestUpdatePaymentMethodPolicy_RequestShapeAndResponse(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody UpdatePaymentMethodPolicyRequest

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"version": 4, "environment": "live",
			"entries": [
				{"chain": "ethereum", "network": "mainnet", "token": "USDC", "kind": "erc20", "enabled": true, "eligible": true, "effective": true}
			]
		}`))
	})

	policy, err := c.UpdatePaymentMethodPolicy(context.Background(), UpdatePaymentMethodPolicyRequest{
		Enabled: []PaymentMethodPolicyIdentity{
			{Chain: "ethereum", Network: "mainnet", Token: "USDC"},
		},
	})
	if err != nil {
		t.Fatalf("UpdatePaymentMethodPolicy: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v1/me/payment_method_policy" {
		t.Errorf("got %s %s, want PUT /v1/me/payment_method_policy", gotMethod, gotPath)
	}
	if len(gotBody.Enabled) != 1 || gotBody.Enabled[0].Chain != "ethereum" {
		t.Errorf("request Enabled = %+v", gotBody.Enabled)
	}
	if policy.Version != 4 {
		t.Errorf("Version = %d, want 4", policy.Version)
	}
}

func TestUpdatePaymentMethodPolicy_EmptyDisablesEverything(t *testing.T) {
	var gotBody UpdatePaymentMethodPolicyRequest
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"version": 5, "environment": "live", "entries": []}`))
	})

	policy, err := c.UpdatePaymentMethodPolicy(context.Background(), UpdatePaymentMethodPolicyRequest{
		Enabled: []PaymentMethodPolicyIdentity{},
	})
	if err != nil {
		t.Fatalf("UpdatePaymentMethodPolicy: %v", err)
	}
	if len(gotBody.Enabled) != 0 {
		t.Errorf("request Enabled = %+v, want empty", gotBody.Enabled)
	}
	if len(policy.Entries) != 0 {
		t.Errorf("Entries = %+v, want empty", policy.Entries)
	}
}
