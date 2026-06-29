package aigenerator

import (
	"context"
	"fmt"
	"strings"

	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/secretbox"
	"github.com/SigNoz/signoz/pkg/ruler/remediation"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// StoreAwareSelectorProvider resolves a per-org LLM provider for the remediation
// Selector, reusing the SAME stored AIConfig the strategy generator uses. It is
// resolved per-call (no cache): selection is incident-triggered and infrequent.
type StoreAwareSelectorProvider struct {
	store  ruletypes.AIConfigStore
	cipher *secretbox.Cipher
}

var _ remediation.ProviderResolver = (*StoreAwareSelectorProvider)(nil)

// NewStoreAwareSelectorProvider constructs the resolver.
func NewStoreAwareSelectorProvider(store ruletypes.AIConfigStore, cipher *secretbox.Cipher) *StoreAwareSelectorProvider {
	return &StoreAwareSelectorProvider{store: store, cipher: cipher}
}

// Resolve returns the per-org LLM provider + model. It errors (so the Selector
// fails open) when the org is empty, the store/cipher is unset, the config
// lookup fails, or the org is not configured for an LLM provider.
func (r *StoreAwareSelectorProvider) Resolve(ctx context.Context, orgID string) (remediation.LLMProvider, string, error) {
	if strings.TrimSpace(orgID) == "" || r.store == nil || r.cipher == nil {
		return nil, "", fmt.Errorf("selector provider: org/store/cipher unset")
	}
	cfg, err := r.store.Get(ctx, orgID, r.cipher.DecryptFunc())
	if err != nil {
		return nil, "", err
	}
	if cfg.Provider != "llm" {
		return nil, "", fmt.Errorf("selector provider: org %s not configured for llm (provider=%q)", orgID, cfg.Provider)
	}
	gcfg := ConfigFromAIConfig(cfg)
	provider, err := buildLLMProvider(gcfg)
	if err != nil {
		return nil, "", err
	}
	model := gcfg.LLMModel
	if model == "" {
		model = defaultModelFor(gcfg.LLMProvider)
	}
	return provider, model, nil
}
