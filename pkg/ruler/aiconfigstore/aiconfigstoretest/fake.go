// Package aiconfigstoretest provides an in-memory fake of
// ruletypes.AIConfigStore for tests that need a working store
// without a real database.
package aiconfigstoretest

import (
	"context"
	"strings"
	"sync"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// Fake is an in-memory AIConfigStore. Safe for concurrent use.
// Stores plaintext API keys directly — does not actually encrypt for tests.
type Fake struct {
	mu      sync.RWMutex
	configs map[string]ruletypes.AIConfig // orgID -> AIConfig (plaintext)
}

func New() *Fake {
	return &Fake{
		configs: map[string]ruletypes.AIConfig{},
	}
}

func (f *Fake) Upsert(_ context.Context, cfg ruletypes.AIConfig, encrypt func(string) (string, error)) error {
	if strings.TrimSpace(cfg.OrgID) == "" {
		return ruletypes.ErrAIConfigNotFound
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.configs[cfg.OrgID] = cfg
	return nil
}

func (f *Fake) Get(_ context.Context, orgID string, decrypt func(string) (string, error)) (ruletypes.AIConfig, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	cfg, ok := f.configs[orgID]
	if !ok {
		return ruletypes.AIConfig{}, ruletypes.ErrAIConfigNotFound
	}
	return cfg, nil
}

var _ ruletypes.AIConfigStore = (*Fake)(nil)
