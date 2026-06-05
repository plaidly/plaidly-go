package telegram

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	plaidly "github.com/plaidly/plaidly-go"
)

func newHelper(t *testing.T, h http.HandlerFunc) *Helper {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := plaidly.NewClient("pk_test", plaidly.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return New(c)
}

func sessionJSON(id, status string, paymentURL string) map[string]any {
	m := map[string]any{
		"session_id": id, "merchant_id": "m", "expected_amount": 12.5,
		"received_amount": 0, "address": "Addr123", "status": status,
		"metadata": map[string]any{}, "expires_at": "t", "created_at": "t",
		"updated_at": "t", "demo": false,
		"paymentMethod": map[string]any{"methodID": 0, "chain": "solana", "token": "USDC", "network": "mainnet"},
	}
	if paymentURL != "" {
		m["payment_url"] = paymentURL
	}
	return m
}

func TestCreateInvoice(t *testing.T) {
	h := newHelper(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/payment_sessions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var req plaidly.CreatePaymentSessionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Amount != 12.5 || req.ExpiresIn != "15m" {
			t.Errorf("req = %+v", req)
		}
		if req.PaymentMethod.MethodID != plaidly.MethodIDCrypto {
			t.Errorf("methodID = %v", req.PaymentMethod.MethodID)
		}
		if req.Metadata == nil || (*req.Metadata)["order_id"] != "A-1" {
			t.Errorf("metadata = %v", req.Metadata)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(sessionJSON("ps_1", "pending", "https://pay.plaidly.io/ps_1"))
	})

	inv, err := h.CreateInvoice(context.Background(), 12.5, "USDC", "solana", "mainnet", map[string]any{"order_id": "A-1"})
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
	if inv.PaymentURL != "https://pay.plaidly.io/ps_1" {
		t.Errorf("PaymentURL = %q", inv.PaymentURL)
	}
	if inv.PayButton.URL != inv.PaymentURL || inv.WebAppButton.URL != inv.PaymentURL {
		t.Errorf("buttons = %+v / %+v", inv.PayButton, inv.WebAppButton)
	}
	if inv.PayButton.Text != "Pay now" {
		t.Errorf("button text = %q", inv.PayButton.Text)
	}
	for _, want := range []string{"12.5", "USDC", "solana", "mainnet", "<code>Addr123</code>"} {
		if !strings.Contains(inv.MessageText, want) {
			t.Errorf("message missing %q:\n%s", want, inv.MessageText)
		}
	}
}

func TestCreateInvoice_DefaultsNetwork(t *testing.T) {
	h := newHelper(t, func(w http.ResponseWriter, r *http.Request) {
		var req plaidly.CreatePaymentSessionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.PaymentMethod.Network != "mainnet" {
			t.Errorf("network = %q, want mainnet default", req.PaymentMethod.Network)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(sessionJSON("ps_2", "pending", "https://pay/ps_2"))
	})
	if _, err := h.CreateInvoice(context.Background(), 1, "SOL", "solana", "", nil); err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
}

func TestCreateInvoice_NoPaymentURL(t *testing.T) {
	h := newHelper(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(sessionJSON("ps_3", "pending", ""))
	})
	_, err := h.CreateInvoice(context.Background(), 1, "SOL", "solana", "mainnet", nil)
	if err != ErrNoPaymentURL {
		t.Fatalf("err = %v, want ErrNoPaymentURL", err)
	}
}

func TestInvoiceMessage_EscapesHTML(t *testing.T) {
	sess := &plaidly.PaymentSession{Address: "<script>"}
	msg := InvoiceMessage(sess, 1, "T&T", "c", "n")
	if strings.Contains(msg, "<script>") {
		t.Errorf("unescaped address in message:\n%s", msg)
	}
	if !strings.Contains(msg, "T&amp;T") {
		t.Errorf("token not escaped:\n%s", msg)
	}
}

func TestVerifyWebhook_FromRequest(t *testing.T) {
	secret := "whsec_test_secret"
	body := []byte(`{"event_type":"payment_session.completed","session_id":"ps_1","status":"completed","amount":12.5,"currency":"USDC","chain":"solana","network":"mainnet","timestamp":"now"}`)
	ts := fmt.Sprintf("%d", time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	sig := "t=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	req.Header.Set(plaidly.SignatureHeader, sig)
	req.Header.Set(plaidly.EventHeader, plaidly.EventPaymentCompleted)

	ev, err := VerifyWebhook(req, secret, plaidly.DefaultWebhookTolerance)
	if err != nil {
		t.Fatalf("VerifyWebhook: %v", err)
	}
	if ev.SessionID != "ps_1" || !plaidly.IsSuccess(ev.Status) {
		t.Errorf("event = %+v", ev)
	}
}

func TestVerifyWebhook_BadSignature(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{}`))
	req.Header.Set(plaidly.SignatureHeader, "t=1700000000,v1=deadbeef")
	if _, err := VerifyWebhook(req, "secret", plaidly.DefaultWebhookTolerance); err == nil {
		t.Fatal("expected error for bad signature")
	}
}

func TestPollSession_CompletesAfterBackoff(t *testing.T) {
	var calls int32
	h := newHelper(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		status := "pending"
		if n >= 3 {
			status = "completed"
		}
		_ = json.NewEncoder(w).Encode(sessionJSON("ps_1", status, ""))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sess, err := h.PollSessionWithOptions(ctx, "ps_1", PollOptions{
		Interval: 10 * time.Millisecond, MaxInterval: 20 * time.Millisecond, MaxDuration: 4 * time.Second,
	})
	if err != nil {
		t.Fatalf("PollSession: %v", err)
	}
	if !plaidly.IsSuccess(sess.Status) {
		t.Errorf("status = %q", sess.Status)
	}
	if atomic.LoadInt32(&calls) < 3 {
		t.Errorf("calls = %d, want >= 3", calls)
	}
}

func TestPollSession_Timeout(t *testing.T) {
	h := newHelper(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(sessionJSON("ps_1", "pending", ""))
	})
	ctx := context.Background()
	_, err := h.PollSessionWithOptions(ctx, "ps_1", PollOptions{
		Interval: 10 * time.Millisecond, MaxDuration: 60 * time.Millisecond,
	})
	if err != ErrPollTimeout {
		t.Fatalf("err = %v, want ErrPollTimeout", err)
	}
}

func TestPollSession_TerminalFailure(t *testing.T) {
	h := newHelper(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(sessionJSON("ps_1", "expired", ""))
	})
	sess, err := h.PollSession(context.Background(), "ps_1", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("PollSession: %v", err)
	}
	if sess.Status != plaidly.StatusExpired || plaidly.IsSuccess(sess.Status) {
		t.Errorf("status = %q", sess.Status)
	}
}
