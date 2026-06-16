package aigenerator

import (
	"context"

	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/secretbox"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// StoreAwareRunbookDrafter resolves the per-org AI config at Draft time and
// builds a drafter from it — reusing the SAME stored credential the AI module
// uses — so incident-report drafting no longer needs separate env auth. It
// falls back to the env-built drafter when the request carries no org, the
// lookup fails, or the stored config is not an LLM config.
//
// Resolution is per-call (no cache): drafting is user-triggered and infrequent,
// so a single indexed read + decrypt is cheap and avoids the cache-invalidation
// coupling StoreAware has with the AI-config update handler.
type StoreAwareRunbookDrafter struct {
	store       ruletypes.AIConfigStore
	cipher      *secretbox.Cipher
	envFallback ruletypes.RunbookDrafter
}

// NewStoreAwareRunbookDrafter wraps envFallback with per-org config resolution.
func NewStoreAwareRunbookDrafter(store ruletypes.AIConfigStore, cipher *secretbox.Cipher, envFallback ruletypes.RunbookDrafter) *StoreAwareRunbookDrafter {
	return &StoreAwareRunbookDrafter{store: store, cipher: cipher, envFallback: envFallback}
}

var _ ruletypes.RunbookDrafter = (*StoreAwareRunbookDrafter)(nil)

func (d *StoreAwareRunbookDrafter) Draft(ctx context.Context, req ruletypes.RunbookDraftRequest) (ruletypes.Runbook, error) {
	if req.OrgID == "" {
		return d.envFallback.Draft(ctx, req)
	}
	cfg, err := d.store.Get(ctx, req.OrgID, d.cipher.DecryptFunc())
	if err != nil || cfg.Provider != "llm" {
		return d.envFallback.Draft(ctx, req)
	}
	return NewRunbookDrafter(ConfigFromAIConfig(cfg)).Draft(ctx, req)
}
