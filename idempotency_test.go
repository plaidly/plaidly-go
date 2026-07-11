package plaidly

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type idempotencyRecord struct {
	fingerprint string
	sessionID   string
	body        []byte
}

type idempotencyStore struct {
	mu      sync.Mutex
	records map[string]idempotencyRecord
	nextID  int
}

func newIdempotencyStore() *idempotencyStore {
	return &idempotencyStore{records: map[string]idempotencyRecord{}}
}

func fingerprintOf(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (s *idempotencyStore) handleCreate(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	key := r.Header.Get("Idempotency-Key")
	fp := fingerprintOf(body)

	if key != "" {
		s.mu.Lock()
		rec, exists := s.records[key]
		s.mu.Unlock()
		if exists {
			if rec.fingerprint != fp {
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    ErrorCodeConflict,
					"message": "idempotency key was already used with a different payload",
				})
				return
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(rec.body)
			return
		}
	}

	s.mu.Lock()
	s.nextID++
	sessionID := "ps_idem_" + strconvI(s.nextID)
	s.mu.Unlock()

	respBody, _ := json.Marshal(map[string]any{
		"session_id": sessionID,
		"status":     "pending",
	})
	if key != "" {
		s.mu.Lock()
		s.records[key] = idempotencyRecord{fingerprint: fp, sessionID: sessionID, body: respBody}
		s.mu.Unlock()
	}
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBody)
}

func strconvI(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func TestCreatePaymentSession_IdempotencyKeyHeaderPropagation(t *testing.T) {
	var gotHeader string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Idempotency-Key")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"session_id":"ps_1","status":"pending"}`))
	})

	_, err := c.CreatePaymentSession(context.Background(), CreatePaymentSessionRequest{Amount: 10}, WithIdempotencyKey("compono-uuid-abc"))
	if err != nil {
		t.Fatalf("CreatePaymentSession: %v", err)
	}
	if gotHeader != "compono-uuid-abc" {
		t.Errorf("Idempotency-Key header = %q, want %q", gotHeader, "compono-uuid-abc")
	}
}

func TestCreatePaymentSession_NoIdempotencyKeyByDefault(t *testing.T) {
	var sawHeader bool
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		sawHeader = r.Header.Get("Idempotency-Key") != ""
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"session_id":"ps_1","status":"pending"}`))
	})

	_, err := c.CreatePaymentSession(context.Background(), CreatePaymentSessionRequest{Amount: 10})
	if err != nil {
		t.Fatalf("CreatePaymentSession: %v", err)
	}
	if sawHeader {
		t.Error("Idempotency-Key header sent when no options were passed; backward-compat call shape must not add headers")
	}
}

func TestCreatePaymentSession_IdempotentReplayReturnsCanonicalSession(t *testing.T) {
	store := newIdempotencyStore()
	c, _ := newTestClient(t, store.handleCreate)

	req := CreatePaymentSessionRequest{Amount: 25, ExpiresIn: "15m"}
	first, err := c.CreatePaymentSession(context.Background(), req, WithIdempotencyKey("compono-payment-uuid-1"))
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	second, err := c.CreatePaymentSession(context.Background(), req, WithIdempotencyKey("compono-payment-uuid-1"))
	if err != nil {
		t.Fatalf("replayed create: %v", err)
	}

	if first.SessionId == "" || second.SessionId == "" || first.SessionId != second.SessionId {
		t.Errorf("replay returned a different session: first=%v second=%v", first.SessionId, second.SessionId)
	}
}

func TestCreatePaymentSession_IdempotencyPayloadMismatchConflicts(t *testing.T) {
	store := newIdempotencyStore()
	c, _ := newTestClient(t, store.handleCreate)

	_, err := c.CreatePaymentSession(context.Background(), CreatePaymentSessionRequest{Amount: 25}, WithIdempotencyKey("compono-payment-uuid-2"))
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = c.CreatePaymentSession(context.Background(), CreatePaymentSessionRequest{Amount: 999}, WithIdempotencyKey("compono-payment-uuid-2"))
	if err == nil {
		t.Fatal("expected conflict error for mismatched payload, got nil")
	}

	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error type = %T, want *Error", err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Errorf("StatusCode = %d, want 409", apiErr.StatusCode)
	}
	if apiErr.Code != ErrorCodeConflict {
		t.Errorf("Code = %d, want %d", apiErr.Code, ErrorCodeConflict)
	}
	if !IsIdempotencyConflict(err) {
		t.Error("IsIdempotencyConflict(err) = false, want true")
	}
}

func TestErrorCode_DecodesNumericWireFormat(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":5,"message":"idempotency key was already used with a different payload"}`))
	})

	_, err := c.CreatePaymentSession(context.Background(), CreatePaymentSessionRequest{Amount: 1})
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error type = %T, want *Error", err)
	}
	if apiErr.Code != 5 {
		t.Errorf("Code = %d, want 5 (regression: Code must decode the API's numeric wire format, not stay zero)", apiErr.Code)
	}
	if apiErr.Message == "" {
		t.Error("Message is empty, want the decoded message")
	}
}

func TestCreatePaymentSession_TimeoutReturnsPromptly(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
	}))
	defer func() {
		close(block)
		srv.Close()
	}()

	c, err := NewClient("pk_test_timeout", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = c.CreatePaymentSession(ctx, CreatePaymentSessionRequest{Amount: 1})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if elapsed > 2*time.Second {
		t.Errorf("CreatePaymentSession took %s to return after context deadline; doJSON's retry loop should return promptly once ctx is done", elapsed)
	}
}

func TestRequestPayout_IdempotencyKeyHeaderPropagation(t *testing.T) {
	var gotHeader string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Idempotency-Key")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"po_1","status":"pending"}`))
	})

	_, err := c.RequestPayout(context.Background(), RequestPayoutRequest{Amount: 10, DestinationAddress: "addr", Network: "mainnet", TokenSymbol: "USDC"}, WithIdempotencyKey("compono-payout-uuid-1"))
	if err != nil {
		t.Fatalf("RequestPayout: %v", err)
	}
	if gotHeader != "compono-payout-uuid-1" {
		t.Errorf("Idempotency-Key header = %q, want %q", gotHeader, "compono-payout-uuid-1")
	}
}
