package plaidly

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"testing"
)

func TestRequestEmailVerification_RequestShapeAndResponse(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody RequestEmailVerificationRequest

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"registration_intent_id": "ri_1",
			"expires_at": "2026-07-12T00:00:00Z"
		}`))
	})

	resp, err := c.RequestEmailVerification(context.Background(), RequestEmailVerificationRequest{
		Email: "agent@example.com",
	})
	if err != nil {
		t.Fatalf("RequestEmailVerification: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/v1/merchants/email-verification" {
		t.Errorf("path = %q, want /v1/merchants/email-verification", gotPath)
	}
	if gotBody.Email != "agent@example.com" {
		t.Errorf("request Email = %q, want agent@example.com", gotBody.Email)
	}
	if resp.RegistrationIntentId != "ri_1" {
		t.Errorf("RegistrationIntentId = %q, want ri_1", resp.RegistrationIntentId)
	}
}

func TestConfirmEmailVerification_RequestShapeAndResponse(t *testing.T) {
	var gotPath string
	var gotBody ConfirmEmailVerificationRequest

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"registration_intent_id": "ri_1", "verified": true}`))
	})

	resp, err := c.ConfirmEmailVerification(context.Background(), ConfirmEmailVerificationRequest{
		RegistrationIntentId: "ri_1",
		Code:                 "123456",
	})
	if err != nil {
		t.Fatalf("ConfirmEmailVerification: %v", err)
	}
	if gotPath != "/v1/merchants/email-verification/confirm" {
		t.Errorf("path = %q, want /v1/merchants/email-verification/confirm", gotPath)
	}
	if gotBody.Code != "123456" {
		t.Errorf("request Code = %q, want 123456", gotBody.Code)
	}
	if !resp.Verified {
		t.Error("Verified = false, want true")
	}
}

func TestRequestRegistrationProofOfWork_RequestShapeAndResponse(t *testing.T) {
	var gotPath string
	var gotBody RequestRegistrationProofOfWorkRequest

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"required": true,
			"challenge_id": "pow_1",
			"algorithm_version": "sha256-leading-zero-bits-v1",
			"challenge": "deadbeef",
			"difficulty": 8,
			"expires_at": "2026-07-12T00:00:00Z"
		}`))
	})

	resp, err := c.RequestRegistrationProofOfWork(context.Background(), RequestRegistrationProofOfWorkRequest{
		RegistrationIntentId: "ri_1",
	})
	if err != nil {
		t.Fatalf("RequestRegistrationProofOfWork: %v", err)
	}
	if gotPath != "/v1/merchants/registration-proof-of-work" {
		t.Errorf("path = %q, want /v1/merchants/registration-proof-of-work", gotPath)
	}
	if gotBody.RegistrationIntentId != "ri_1" {
		t.Errorf("request RegistrationIntentId = %q, want ri_1", gotBody.RegistrationIntentId)
	}
	if !resp.Required {
		t.Error("Required = false, want true")
	}
	if resp.AlgorithmVersion == nil || *resp.AlgorithmVersion != AlgorithmSHA256LeadingZeroBitsV1 {
		t.Errorf("AlgorithmVersion = %v, want %q", resp.AlgorithmVersion, AlgorithmSHA256LeadingZeroBitsV1)
	}
	if resp.Difficulty == nil || *resp.Difficulty != 8 {
		t.Errorf("Difficulty = %v, want 8", resp.Difficulty)
	}
}

func TestRequestRegistrationProofOfWork_InvitationExempt(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"required": false}`))
	})

	resp, err := c.RequestRegistrationProofOfWork(context.Background(), RequestRegistrationProofOfWorkRequest{
		RegistrationIntentId: "ri_2",
	})
	if err != nil {
		t.Fatalf("RequestRegistrationProofOfWork: %v", err)
	}
	if resp.Required {
		t.Error("Required = true, want false for an invitation-exempt intent")
	}
	if resp.Challenge != nil {
		t.Errorf("Challenge = %v, want nil when not required", resp.Challenge)
	}
}

// referenceLeadingZeroBits mirrors plaidly-api's internal/pow.leadingZeroBits
// (and pow.Verify's use of it) exactly, independently of this package's own
// leadingZeroBits helper, so the test proves SolveProofOfWork's output would
// actually be accepted by the server rather than just by its own logic.
func referenceLeadingZeroBits(sum []byte) int {
	count := 0
	for _, b := range sum {
		if b == 0 {
			count += 8
			continue
		}
		for mask := byte(0x80); mask > 0; mask >>= 1 {
			if b&mask != 0 {
				return count
			}
			count++
		}
	}
	return count
}

func referenceVerify(challenge, nonce string, difficultyBits int) bool {
	if challenge == "" || nonce == "" {
		return false
	}
	sum := sha256.Sum256([]byte(challenge + ":" + nonce))
	return referenceLeadingZeroBits(sum[:]) >= difficultyBits
}

func TestSolveProofOfWork_ProducesAnAcceptableNonce(t *testing.T) {
	for _, tc := range []struct {
		name       string
		challenge  string
		difficulty int
	}{
		{"low difficulty", "deadbeefcafebabe", 4},
		{"medium difficulty", "0011223344556677", 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nonce := SolveProofOfWork(tc.challenge, tc.difficulty)
			if nonce == "" {
				t.Fatal("SolveProofOfWork returned an empty nonce")
			}
			if !referenceVerify(tc.challenge, nonce, tc.difficulty) {
				t.Errorf("reference pow.Verify-equivalent check rejected nonce %q for challenge %q at difficulty %d", nonce, tc.challenge, tc.difficulty)
			}
		})
	}
}

func TestSolveProofOfWork_ThenRegisterMerchant(t *testing.T) {
	const (
		challenge  = "sandboxregistrationchallenge"
		difficulty = 6
	)
	var gotBody RegisterMerchantRequest

	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": "m_1", "name": "Agent Store", "api_key": "pk_test", "created_at": "2026-07-12T00:00:00Z"}`))
	})

	nonce := SolveProofOfWork(challenge, difficulty)
	if !referenceVerify(challenge, nonce, difficulty) {
		t.Fatalf("SolveProofOfWork produced an unverifiable nonce %q", nonce)
	}

	registrationIntentID := "ri_solved"
	challengeID := "pow_solved"
	_, err := c.RegisterMerchant(context.Background(), RegisterMerchantRequest{
		Name:                   "Agent Store",
		RegistrationIntentId:   &registrationIntentID,
		ProofOfWorkChallengeId: &challengeID,
		ProofOfWorkNonce:       &nonce,
	})
	if err != nil {
		t.Fatalf("RegisterMerchant: %v", err)
	}
	if gotBody.ProofOfWorkNonce == nil || *gotBody.ProofOfWorkNonce != nonce {
		t.Errorf("request ProofOfWorkNonce = %v, want %q", gotBody.ProofOfWorkNonce, nonce)
	}
	if gotBody.RegistrationIntentId == nil || *gotBody.RegistrationIntentId != registrationIntentID {
		t.Errorf("request RegistrationIntentId = %v, want %q", gotBody.RegistrationIntentId, registrationIntentID)
	}
}
