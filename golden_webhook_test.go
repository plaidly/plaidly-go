package plaidly

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"
)

type webhookFixtureSet struct {
	Valid   []validWebhookFixture   `json:"valid"`
	Invalid []invalidWebhookFixture `json:"invalid"`
}

type validWebhookFixture struct {
	Name              string `json:"name"`
	Secret            string `json:"secret"`
	Timestamp         int64  `json:"timestamp"`
	Body              string `json:"body"`
	Signature         string `json:"signature"`
	VerifyAt          int64  `json:"verify_at"`
	ExpectedEventType string `json:"expected_event_type"`
	ExpectedSessionID string `json:"expected_session_id"`
}

type invalidWebhookFixture struct {
	Name          string `json:"name"`
	Reason        string `json:"reason"`
	Secret        string `json:"secret"`
	Timestamp     int64  `json:"timestamp"`
	Body          string `json:"body"`
	Signature     string `json:"signature"`
	VerifyAt      int64  `json:"verify_at"`
	ExpectedError string `json:"expected_error"`
}

func loadWebhookFixtures(t *testing.T) webhookFixtureSet {
	t.Helper()
	data, err := os.ReadFile("testdata/webhook_fixtures.json")
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}
	var set webhookFixtureSet
	if err := json.Unmarshal(data, &set); err != nil {
		t.Fatalf("parse fixtures: %v", err)
	}
	return set
}

func sentinelByName(name string) error {
	switch name {
	case "ErrMissingSignature":
		return ErrMissingSignature
	case "ErrMissingSecret":
		return ErrMissingSecret
	case "ErrInvalidSignature":
		return ErrInvalidSignature
	case "ErrSignatureExpired":
		return ErrSignatureExpired
	default:
		return nil
	}
}

func TestGoldenWebhookFixtures_Valid(t *testing.T) {
	set := loadWebhookFixtures(t)
	if len(set.Valid) == 0 {
		t.Fatal("no valid fixtures loaded")
	}
	for _, f := range set.Valid {
		t.Run(f.Name, func(t *testing.T) {
			now := time.Unix(f.VerifyAt, 0)
			ev, err := VerifyWebhookAt([]byte(f.Body), f.Signature, f.Secret, 0, now)
			if err != nil {
				t.Fatalf("expected valid signature, got error: %v", err)
			}
			if ev.EventType != f.ExpectedEventType {
				t.Errorf("EventType = %q, want %q", ev.EventType, f.ExpectedEventType)
			}
			if ev.SessionID != f.ExpectedSessionID {
				t.Errorf("SessionID = %q, want %q", ev.SessionID, f.ExpectedSessionID)
			}
			if !VerifyWebhookSignatureAt([]byte(f.Body), f.Signature, f.Secret, 0, now) {
				t.Error("VerifyWebhookSignatureAt = false, want true")
			}
		})
	}
}

func TestGoldenWebhookFixtures_Invalid(t *testing.T) {
	set := loadWebhookFixtures(t)
	if len(set.Invalid) == 0 {
		t.Fatal("no invalid fixtures loaded")
	}
	for _, f := range set.Invalid {
		t.Run(f.Name, func(t *testing.T) {
			now := time.Unix(f.VerifyAt, 0)
			_, err := VerifyWebhookAt([]byte(f.Body), f.Signature, f.Secret, 0, now)
			if err == nil {
				t.Fatalf("expected error (%s), got nil", f.Reason)
			}
			want := sentinelByName(f.ExpectedError)
			if want == nil {
				t.Fatalf("fixture references unknown expected_error %q", f.ExpectedError)
			}
			if !errors.Is(err, want) {
				t.Errorf("error = %v, want errors.Is match for %v", err, want)
			}
			if VerifyWebhookSignatureAt([]byte(f.Body), f.Signature, f.Secret, 0, now) {
				t.Error("VerifyWebhookSignatureAt = true, want false")
			}
		})
	}
}

func TestGoldenWebhookFixtures_RotationAcceptsOldSecret(t *testing.T) {
	set := loadWebhookFixtures(t)
	var rotated *validWebhookFixture
	for i := range set.Valid {
		if set.Valid[i].Name == "golden_rotated_secret_old_key_still_valid" {
			rotated = &set.Valid[i]
		}
	}
	if rotated == nil {
		t.Fatal("fixture golden_rotated_secret_old_key_still_valid not found")
	}

	now := time.Unix(rotated.VerifyAt, 0)
	secrets := []string{"whsec_test_current_after_rotation", rotated.Secret}

	ev, err := VerifyWebhookAny([]byte(rotated.Body), rotated.Signature, secrets, 0, now)
	if err != nil {
		t.Fatalf("VerifyWebhookAny with rotation window: %v", err)
	}
	if ev.SessionID != rotated.ExpectedSessionID {
		t.Errorf("SessionID = %q, want %q", ev.SessionID, rotated.ExpectedSessionID)
	}
}
