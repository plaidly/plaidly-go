package plaidly

import (
	"context"
	"crypto/sha256"
	"net/http"
	"strconv"
)

// AlgorithmSHA256LeadingZeroBitsV1 is the only proof-of-work algorithm
// version the API currently issues challenges for.
const AlgorithmSHA256LeadingZeroBitsV1 = "sha256-leading-zero-bits-v1"

// RequestEmailVerification requests a one-time email verification code, the
// first step of public/sandbox merchant registration (BDT-542). The response
// shape is identical whether or not the email satisfies the server's
// trusted-email policy, so it never reveals policy membership or account
// existence.
//
// POST /v1/merchants/email-verification
func (c *Client) RequestEmailVerification(ctx context.Context, req RequestEmailVerificationRequest) (*RequestEmailVerificationResponse, error) {
	var out RequestEmailVerificationResponse
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.RequestEmailVerification(ctx, req)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ConfirmEmailVerification confirms the one-time code sent to the address
// passed to RequestEmailVerification.
//
// POST /v1/merchants/email-verification/confirm
func (c *Client) ConfirmEmailVerification(ctx context.Context, req ConfirmEmailVerificationRequest) (*ConfirmEmailVerificationResponse, error) {
	var out ConfirmEmailVerificationResponse
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.ConfirmEmailVerification(ctx, req)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// RequestRegistrationProofOfWork requests a proof-of-work challenge, the
// second step of public/sandbox registration, after email verification.
// Required is false only when the registration intent carries a valid
// invitation token, in which case no challenge is issued and the caller can
// proceed straight to RegisterMerchant.
//
// POST /v1/merchants/registration-proof-of-work
func (c *Client) RequestRegistrationProofOfWork(ctx context.Context, req RequestRegistrationProofOfWorkRequest) (*RequestRegistrationProofOfWorkResponse, error) {
	var out RequestRegistrationProofOfWorkResponse
	err := c.doJSON(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.raw.RequestRegistrationProofOfWork(ctx, req)
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SolveProofOfWork finds a nonce such that sha256(challenge + ":" + nonce)
// has at least difficultyBits leading zero bits, matching the server's
// sha256-leading-zero-bits-v1 algorithm (the only version currently issued
// by RequestRegistrationProofOfWork). Pass the returned nonce as
// ProofOfWorkNonce, together with the challenge_id as
// ProofOfWorkChallengeId, to RegisterMerchant.
//
// The search is a plain incrementing counter starting at 0; there is no
// fixed upper bound, but leading-zero-bit counting (rather than hex string
// comparison) keeps each attempt cheap.
func SolveProofOfWork(challenge string, difficultyBits int) string {
	for i := 0; ; i++ {
		nonce := strconv.Itoa(i)
		sum := sha256.Sum256([]byte(challenge + ":" + nonce))
		if leadingZeroBits(sum[:]) >= difficultyBits {
			return nonce
		}
	}
}

// leadingZeroBits counts the number of leading zero bits (MSB-first, byte by
// byte) in sum. Mirrors plaidly-api's internal/pow.leadingZeroBits exactly.
func leadingZeroBits(sum []byte) int {
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
