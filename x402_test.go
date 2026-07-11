package plaidly

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testProofProvider struct {
	proof string
}

func (p *testProofProvider) ProvideProof(_ context.Context, challenge *X402Challenge) (string, error) {
	if p == nil {
		return "", nil
	}
	if challenge == nil {
		return "", nil
	}
	return p.proof, nil
}

type testFacilitator struct {
	proof string
}

func (f *testFacilitator) Facilitate(_ context.Context, _ *X402Challenge) (string, error) {
	if f == nil {
		return "", nil
	}
	return f.proof, nil
}

func TestParseX402ChallengeFromHeader(t *testing.T) {
	challenge := X402Challenge{
		Version:           "1",
		Scheme:            "exact",
		Network:           "base-mainnet",
		MaxAmountRequired: "1000000",
		Asset:             "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
		PayTo:             "0xServiceProvider",
		Memo:              "memo-abc",
	}
	payload, _ := json.Marshal(challenge)
	header := base64.StdEncoding.EncodeToString(payload)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(PaymentRequiredHeader, header)
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte("ignored"))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	parsed, err := ParseX402Challenge(resp)
	if err != nil {
		t.Fatalf("parse challenge: %v", err)
	}
	if parsed.Version != challenge.Version || parsed.PayTo != challenge.PayTo || parsed.Memo != challenge.Memo {
		t.Fatalf("parsed challenge mismatch: %+v", parsed)
	}
	if parsed.Extra == nil {
		t.Fatalf("expected extras map")
	}
}

func TestParseX402ChallengeFromBody(t *testing.T) {
	challenge := X402Challenge{
		Version: "1",
		Scheme:  "exact",
		Network: "base-mainnet",
		Asset:   "USDC",
		PayTo:   "0xProvider",
		Memo:    "memo-body",
		Extra:   map[string]any{"source": "body-fallback"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_ = json.NewEncoder(w).Encode(challenge)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	parsed, err := ParseX402Challenge(resp)
	if err != nil {
		t.Fatalf("parse challenge: %v", err)
	}
	if parsed.Asset != challenge.Asset || parsed.Memo != challenge.Memo {
		t.Fatalf("parsed challenge mismatch: %+v", parsed)
	}
}

func TestRetryRequestWithX402Proof_UsesProofProvider(t *testing.T) {
	challenge := X402Challenge{
		Scheme:  "exact",
		Amount:  "1000000",
		Asset:   "USDC",
		PayTo:   "0xProvider",
		Network: "base-mainnet",
	}
	payload, _ := json.Marshal(challenge)
	header := base64.StdEncoding.EncodeToString(payload)

	hits := 0
	hasProof := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits == 1 {
			w.Header().Set(PaymentRequiredHeader, header)
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}
		if got := r.Header.Get(PlaidlyProofHeader); got == "" {
			t.Fatalf("expected proof header")
		}
		if got := r.Header.Get(PlaidlyProofHeader); got == "provider-proof" {
			hasProof = true
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	requestFn := func(ctx context.Context, proof string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		if err != nil {
			return nil, err
		}
		if proof != "" {
			req.Header.Set(PlaidlyProofHeader, proof)
		}
		return http.DefaultClient.Do(req)
	}

	var got map[string]any
	err := RetryRequestWithX402Proof(context.Background(), requestFn, &testProofProvider{proof: "provider-proof"}, nil, &got)
	if err != nil {
		t.Fatalf("retry helper: %v", err)
	}
	if !hasProof {
		t.Fatalf("proof header missing on retry")
	}
	if hits != 2 {
		t.Fatalf("unexpected request count: %d", hits)
	}
	if got["status"] != "ok" {
		t.Fatalf("unexpected response body: %+v", got)
	}
}

func TestRetryRequestWithX402Proof_FallsBackToFacilitator(t *testing.T) {
	challenge := X402Challenge{Scheme: "exact"}
	payload, _ := json.Marshal(challenge)
	header := base64.StdEncoding.EncodeToString(payload)

	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits == 1 {
			w.Header().Set(PaymentRequiredHeader, header)
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}
		if got := r.Header.Get(PlaidlyProofHeader); got != "facilitator-proof" {
			t.Fatalf("expected facilitator proof, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	requestFn := func(ctx context.Context, proof string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		if err != nil {
			return nil, err
		}
		if proof != "" {
			req.Header.Set(PlaidlyProofHeader, proof)
		}
		return http.DefaultClient.Do(req)
	}

	var got map[string]any
	err := RetryRequestWithX402Proof(context.Background(), requestFn, nil, &testFacilitator{proof: "facilitator-proof"}, &got)
	if err != nil {
		t.Fatalf("retry helper: %v", err)
	}
	if got["status"] != "ok" {
		t.Fatalf("unexpected response body: %+v", got)
	}
	if hits != 2 {
		t.Fatalf("unexpected request count: %d", hits)
	}
}

func TestRetryRequestWithX402Proof_PropagatesNon402Errors(t *testing.T) {
	calls := 0
	defer func() {
		if calls != 1 {
			t.Errorf("expected 1 call, got %d", calls)
		}
	}()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{"session_id": "ps_ok"})
	}))
	defer srv.Close()

	requestFn := func(ctx context.Context, proof string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		if err != nil {
			return nil, err
		}
		if proof != "" {
			req.Header.Set(PlaidlyProofHeader, proof)
		}
		return http.DefaultClient.Do(req)
	}

	var got map[string]any
	err := RetryRequestWithX402Proof(context.Background(), requestFn, nil, nil, &got)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got["session_id"] != "ps_ok" {
		t.Fatalf("unexpected body: %+v", got)
	}
}
