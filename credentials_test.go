package plaidly

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetMerchantCredentialStatus(t *testing.T) {
	var gotPath, gotMethod string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"api_key_prefix": "plaidly_sk_live_...a1b2",
			"sandbox": true,
			"webhook_secret_configured": true,
			"webhook_secret_mask": "plaidly_whsec_...c3d4",
			"created_at": "2026-01-01T00:00:00Z"
		}`))
	})

	status, err := c.GetMerchantCredentialStatus(context.Background())
	if err != nil {
		t.Fatalf("GetMerchantCredentialStatus: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/v1/me/credentials" {
		t.Errorf("got %s %s, want GET /v1/me/credentials", gotMethod, gotPath)
	}
	if status.ApiKeyPrefix != "plaidly_sk_live_...a1b2" || !status.WebhookSecretConfigured {
		t.Errorf("status = %+v", status)
	}
}

func TestRotateMerchantCredential_RequestShapeAndSecretReturnedOnce(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody MerchantCredentialOperationRequest
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "merch_1", "name": "Acme", "created_at": "2026-01-01T00:00:00Z",
			"api_key": "plaidly_sk_live_freshvalue"
		}`))
	})

	merchant, err := c.RotateMerchantCredential(context.Background(), CredentialTypeAPIKey)
	if err != nil {
		t.Fatalf("RotateMerchantCredential: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/me/credentials/rotate" {
		t.Errorf("got %s %s, want POST /v1/me/credentials/rotate", gotMethod, gotPath)
	}
	if gotBody.CredentialType != CredentialTypeAPIKey {
		t.Errorf("CredentialType = %q, want api_key", gotBody.CredentialType)
	}
	if merchant.ApiKey != "plaidly_sk_live_freshvalue" {
		t.Errorf("ApiKey = %q, want fresh secret in response", merchant.ApiKey)
	}
}

func TestRotateMerchantCredential_IdempotencyKeyHeader(t *testing.T) {
	var gotHeader string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Idempotency-Key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "merch_1", "name": "Acme", "created_at": "2026-01-01T00:00:00Z"}`))
	})

	_, err := c.RotateMerchantCredential(context.Background(), CredentialTypeWebhookSecret, WithIdempotencyKey("rot-1"))
	if err != nil {
		t.Fatalf("RotateMerchantCredential: %v", err)
	}
	if gotHeader != "rot-1" {
		t.Errorf("Idempotency-Key header = %q, want rot-1", gotHeader)
	}
}

func TestRevokeMerchantCredential(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody MerchantCredentialOperationRequest
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "merch_1", "name": "Acme", "created_at": "2026-01-01T00:00:00Z"}`))
	})

	_, err := c.RevokeMerchantCredential(context.Background(), CredentialTypeWebhookSecret)
	if err != nil {
		t.Fatalf("RevokeMerchantCredential: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/me/credentials/revoke" {
		t.Errorf("got %s %s, want POST /v1/me/credentials/revoke", gotMethod, gotPath)
	}
	if gotBody.CredentialType != CredentialTypeWebhookSecret {
		t.Errorf("CredentialType = %q, want webhook_secret", gotBody.CredentialType)
	}
}

func TestUpdateMerchantWebhook(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody UpdateMerchantWebhookRequest
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "merch_1", "name": "Acme", "created_at": "2026-01-01T00:00:00Z", "webhook_url": "https://example.com/hook"}`))
	})

	merchant, err := c.UpdateMerchantWebhook(context.Background(), "https://example.com/hook")
	if err != nil {
		t.Fatalf("UpdateMerchantWebhook: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v1/me/webhook" {
		t.Errorf("got %s %s, want PUT /v1/me/webhook", gotMethod, gotPath)
	}
	if gotBody.WebhookUrl != "https://example.com/hook" {
		t.Errorf("WebhookUrl = %q", gotBody.WebhookUrl)
	}
	if merchant.WebhookUrl == nil || *merchant.WebhookUrl != "https://example.com/hook" {
		t.Errorf("response WebhookUrl = %+v", merchant.WebhookUrl)
	}
}

func TestTestMerchantWebhookDelivery(t *testing.T) {
	var gotPath, gotMethod string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"delivered": true, "status_code": 200, "message": "ok"}`))
	})

	result, err := c.TestMerchantWebhookDelivery(context.Background())
	if err != nil {
		t.Fatalf("TestMerchantWebhookDelivery: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/me/webhook/test" {
		t.Errorf("got %s %s, want POST /v1/me/webhook/test", gotMethod, gotPath)
	}
	if !result.Delivered {
		t.Errorf("Delivered = false, want true")
	}
}
