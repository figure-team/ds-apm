package remediation

import (
	"context"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/remediationstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// ---------------------------------------------------------------------------
// Fake Store
// ---------------------------------------------------------------------------

type vFakeStore struct {
	proposed    []ruletypes.RemediationExecution
	succeeded   []ruletypes.RemediationExecution
	transitions map[string]string // id -> toStatus
}

func (s *vFakeStore) Create(_ context.Context, _ ruletypes.RemediationExecution) error {
	return nil
}
func (s *vFakeStore) Get(_ context.Context, _, _ string) (ruletypes.RemediationExecution, error) {
	return ruletypes.RemediationExecution{}, nil
}
func (s *vFakeStore) ListByIncident(_ context.Context, _, _ string) ([]ruletypes.RemediationExecution, error) {
	return nil, nil
}
func (s *vFakeStore) ListByStatus(_ context.Context, _ string, status string) ([]ruletypes.RemediationExecution, error) {
	switch status {
	case ruletypes.RemediationStatusProposed:
		return s.proposed, nil
	case ruletypes.RemediationStatusSucceeded:
		return s.succeeded, nil
	}
	return nil, nil
}
func (s *vFakeStore) TransitionToExecuting(_ context.Context, _, _, _, _ string, _ int64) (bool, error) {
	return false, nil
}
func (s *vFakeStore) Transition(_ context.Context, _ string, id, toStatus string, _ remediationstore.TransitionPatch) error {
	s.transitions[id] = toStatus
	return nil
}
func (s *vFakeStore) CountActiveByOrg(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
func (s *vFakeStore) GetConfig(_ context.Context, _ string) (ruletypes.RemediationConfig, error) {
	return ruletypes.RemediationConfig{}, nil
}
func (s *vFakeStore) UpsertConfig(_ context.Context, _ string, _ ruletypes.RemediationConfig) error {
	return nil
}

func (s *vFakeStore) ListByOrg(_ context.Context, _ string, _ remediationstore.ListFilter) ([]ruletypes.RemediationExecution, error) {
	return nil, nil
}

func (s *vFakeStore) ListActiveByFingerprint(_ context.Context, _, _ string) ([]ruletypes.RemediationExecution, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Fake AlertStateLookup
// ---------------------------------------------------------------------------

type fakeAlerts struct{ firing map[string]bool }

func (f fakeAlerts) IsFiring(_ context.Context, _, fp string) (bool, error) {
	return f.firing[fp], nil
}

// ---------------------------------------------------------------------------
// Fixed clock
// ---------------------------------------------------------------------------

func atNow() time.Time { return time.Date(2026, 6, 24, 1, 0, 0, 0, time.UTC) }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestTick_ExpiresStaleProposed: stale proposal (ExpiresAt < now) → Expired;
// fresh (ExpiresAt > now) → untouched.
func TestTick_ExpiresStaleProposed(t *testing.T) {
	s := &vFakeStore{transitions: map[string]string{}}
	s.proposed = []ruletypes.RemediationExecution{
		{ID: "old", OrgID: "o", Status: ruletypes.RemediationStatusProposed, ExpiresAt: "2026-06-24T00:30:00Z"},   // < now
		{ID: "fresh", OrgID: "o", Status: ruletypes.RemediationStatusProposed, ExpiresAt: "2026-06-24T02:00:00Z"}, // > now
	}
	v := NewVerifier(s, fakeAlerts{firing: map[string]bool{}}, atNow)
	if err := v.Tick(context.Background(), "o"); err != nil {
		t.Fatalf("Tick returned error: %v", err)
	}
	if s.transitions["old"] != ruletypes.RemediationStatusExpired {
		t.Fatalf("stale proposal must expire, got %q", s.transitions["old"])
	}
	if _, has := s.transitions["fresh"]; has {
		t.Fatal("fresh proposal must not expire")
	}
}

// TestTick_VerifiesWhenAlertResolved: succeeded + IsFiring=false → Verified.
func TestTick_VerifiesWhenAlertResolved(t *testing.T) {
	s := &vFakeStore{transitions: map[string]string{}}
	s.succeeded = []ruletypes.RemediationExecution{
		{
			ID: "rem", OrgID: "o", Status: ruletypes.RemediationStatusSucceeded,
			AlertFingerprint: "fp",
			ExecutedAt:       "2026-06-24T00:58:00Z",
		},
	}
	v := NewVerifier(s, fakeAlerts{firing: map[string]bool{"fp": false}}, atNow) // resolved
	if err := v.Tick(context.Background(), "o"); err != nil {
		t.Fatalf("Tick returned error: %v", err)
	}
	if s.transitions["rem"] != ruletypes.RemediationStatusVerified {
		t.Fatalf("resolved alert → verified, got %q", s.transitions["rem"])
	}
}

// TestTick_UnresolvedWhenWindowElapsedStillFiring: succeeded, ExecutedAt 20m ago,
// verifyWindow=10m, IsFiring=true → Unresolved.
func TestTick_UnresolvedWhenWindowElapsedStillFiring(t *testing.T) {
	s := &vFakeStore{transitions: map[string]string{}}
	s.succeeded = []ruletypes.RemediationExecution{
		{
			ID: "rem", OrgID: "o", Status: ruletypes.RemediationStatusSucceeded,
			AlertFingerprint: "fp",
			ExecutedAt:       "2026-06-24T00:40:00Z", // 20 min ago > 10 min window
		},
	}
	v := NewVerifier(s, fakeAlerts{firing: map[string]bool{"fp": true}}, atNow)
	v.verifyWindow = 10 * time.Minute
	if err := v.Tick(context.Background(), "o"); err != nil {
		t.Fatalf("Tick returned error: %v", err)
	}
	if s.transitions["rem"] != ruletypes.RemediationStatusUnresolved {
		t.Fatalf("window elapsed + firing → unresolved, got %q", s.transitions["rem"])
	}
}
