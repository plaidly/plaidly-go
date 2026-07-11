package plaidly

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestWaitForPaymentSession_CompletesWhenTerminal(t *testing.T) {
	states := []string{"pending", "partial_paid", "completed"}
	callCount := 0
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/payment_sessions/ps_1" {
			t.Errorf("path = %s", r.URL.Path)
		}
		state := states[callCount]
		if callCount < len(states)-1 {
			callCount++
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": "ps_1", "merchant_id": "m", "expected_amount": 1,
			"received_amount": 0, "address": "a", "status": state,
			"metadata": map[string]any{}, "expires_at": "t", "created_at": "t",
			"updated_at": "t", "demo": false,
			"paymentMethod": map[string]any{"methodID": 0, "chain": "solana", "token": "USDC", "network": "testnet"},
		})
	})

	got, err := WaitForPaymentSession(
		context.Background(),
		c,
		"ps_1",
		SessionPollOptions{PollInterval: 0, MaxAttempts: 10},
	)
	if err != nil {
		t.Fatalf("WaitForPaymentSession: %v", err)
	}
	if got.Status != "completed" {
		t.Fatalf("status = %q", got.Status)
	}
	if callCount != 2 {
		t.Fatalf("calls = %d", callCount)
	}
}

func TestWaitForPaymentSession_TimesOutBeforeTerminal(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": "ps_2", "merchant_id": "m", "expected_amount": 1,
			"received_amount": 0, "address": "a", "status": "pending",
			"metadata": map[string]any{}, "expires_at": "t", "created_at": "t",
			"updated_at": "t", "demo": false,
			"paymentMethod": map[string]any{"methodID": 0, "chain": "solana", "token": "USDC", "network": "testnet"},
		})
	})

	got, err := WaitForPaymentSession(
		context.Background(),
		c,
		"ps_2",
		SessionPollOptions{PollInterval: 0, MaxAttempts: 2},
	)
	if !errors.Is(err, ErrSessionPollTimeout) {
		t.Fatalf("err = %v, want %v", err, ErrSessionPollTimeout)
	}
	if got == nil || got.Status != "pending" {
		t.Fatalf("session = %+v", got)
	}
}

func TestGetSessionReceiptIfSettled_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/payment_sessions/ps_3":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"session_id": "ps_3", "merchant_id": "m", "expected_amount": 1,
				"received_amount": 1, "address": "a", "status": "completed",
				"metadata": map[string]any{}, "expires_at": "t", "created_at": "t",
				"updated_at": "t", "demo": false,
				"paymentMethod": map[string]any{"methodID": 0, "chain": "solana", "token": "USDC", "network": "testnet"},
			})
		case "/v1/payment_sessions/ps_3/receipt":
			_, _ = w.Write([]byte("%PDF-1.4"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	receipt, session, err := GetSessionReceiptIfSettled(context.Background(), c, "ps_3")
	if err != nil {
		t.Fatalf("GetSessionReceiptIfSettled: %v", err)
	}
	if session == nil || session.Status != "completed" {
		t.Fatalf("session = %+v", session)
	}
	if !bytes.Equal(receipt, []byte("%PDF-1.4")) {
		t.Fatalf("unexpected receipt payload: %q", receipt)
	}
}

func TestGetSessionReceiptIfSettled_RequiresSettled(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": "ps_4", "merchant_id": "m", "expected_amount": 1,
			"received_amount": 0, "address": "a", "status": "pending",
			"metadata": map[string]any{}, "expires_at": "t", "created_at": "t",
			"updated_at": "t", "demo": false,
			"paymentMethod": map[string]any{"methodID": 0, "chain": "solana", "token": "USDC", "network": "testnet"},
		})
	})

	_, got, err := GetSessionReceiptIfSettled(context.Background(), c, "ps_4")
	if !errors.Is(err, ErrPaymentNotSettled) {
		t.Fatalf("err = %v, want %v", err, ErrPaymentNotSettled)
	}
	if got == nil || got.Status != "pending" {
		t.Fatalf("session = %+v", got)
	}
}
