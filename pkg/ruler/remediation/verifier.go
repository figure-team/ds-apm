package remediation

import (
	"context"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/remediationstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// AlertStateLookup reports whether an alert (by fingerprint) is currently
// firing. Implemented in wiring over the alertmanager's incident/alert state
// (EndsAt-based). Observation only — not a causal guarantee (design §9).
type AlertStateLookup interface {
	IsFiring(ctx context.Context, orgID, alertFingerprint string) (bool, error)
}

// Verifier runs periodic passes to expire stale proposals and promote succeeded
// executions to verified or unresolved by observing alert resolution.
type Verifier struct {
	store        remediationstore.Store
	alerts       AlertStateLookup
	now          func() time.Time
	verifyWindow time.Duration
}

// NewVerifier constructs a Verifier. now may be nil (falls back to time.Now).
// Default verifyWindow is 10 minutes.
func NewVerifier(store remediationstore.Store, alerts AlertStateLookup, now func() time.Time) *Verifier {
	if now == nil {
		now = time.Now
	}
	return &Verifier{
		store:        store,
		alerts:       alerts,
		now:          now,
		verifyWindow: 10 * time.Minute,
	}
}

// Tick runs one verification pass for an org:
//  1. Expire proposed executions whose ExpiresAt is in the past.
//  2. For each succeeded execution: if the alert is no longer firing → Verified;
//     if still firing and the verifyWindow has elapsed → Unresolved.
//
// IsFiring errors are swallowed per-row (best-effort; retried next tick).
// Store errors from ListByStatus are returned immediately.
func (v *Verifier) Tick(ctx context.Context, orgID string) error {
	now := v.now().UTC()

	// --- 1. Expire stale proposals ---
	proposed, err := v.store.ListByStatus(ctx, orgID, ruletypes.RemediationStatusProposed)
	if err != nil {
		return err
	}
	for _, e := range proposed {
		exp, perr := time.Parse(time.RFC3339, e.ExpiresAt)
		if perr != nil {
			// Unparseable ExpiresAt: skip rather than accidentally expiring.
			continue
		}
		if now.After(exp) {
			_ = v.store.Transition(ctx, orgID, e.ID, ruletypes.RemediationStatusExpired,
				remediationstore.TransitionPatch{TerminalAt: now.Format(time.RFC3339)})
		}
	}

	// --- 2. Verify succeeded executions ---
	succeeded, err := v.store.ListByStatus(ctx, orgID, ruletypes.RemediationStatusSucceeded)
	if err != nil {
		return err
	}
	for _, e := range succeeded {
		firing, ferr := v.alerts.IsFiring(ctx, orgID, e.AlertFingerprint)
		if ferr != nil {
			continue // best-effort: retry next tick
		}
		if !firing {
			_ = v.store.Transition(ctx, orgID, e.ID, ruletypes.RemediationStatusVerified,
				remediationstore.TransitionPatch{
					TerminalAt:   now.Format(time.RFC3339),
					VerifyResult: ruletypes.RemediationStatusVerified,
				})
			continue
		}
		// Still firing — give up only after verifyWindow has elapsed.
		execAt, perr := time.Parse(time.RFC3339, e.ExecutedAt)
		if perr != nil {
			continue // unparseable ExecutedAt: skip
		}
		if now.After(execAt.Add(v.verifyWindow)) {
			_ = v.store.Transition(ctx, orgID, e.ID, ruletypes.RemediationStatusUnresolved,
				remediationstore.TransitionPatch{
					TerminalAt:   now.Format(time.RFC3339),
					VerifyResult: ruletypes.RemediationStatusUnresolved,
				})
		}
	}
	return nil
}

// Run loops Tick over all orgs every interval until ctx is cancelled.
// orgs is called each tick so the org list can change at runtime.
func (v *Verifier) Run(ctx context.Context, interval time.Duration, orgs func(context.Context) []string) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			for _, orgID := range orgs(ctx) {
				_ = v.Tick(ctx, orgID)
			}
		}
	}
}
