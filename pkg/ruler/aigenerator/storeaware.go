// Package aigenerator exposes the AIStrategyGenerator factory.
package aigenerator

import (
	"context"
	"fmt"
	"sync"

	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/secretbox"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// StoreAware combines an in-memory cache of per-org generators with an
// AIConfigStore lookup. On Generate, it derives the org-id from
// req.Labels using a set of common label keys, looks up the config,
// builds (or reuses) a Generator, and delegates. On miss it falls back
// to a single env-derived Generator passed at construction.
//
// Concurrency: an RWMutex-guarded map. PUT to the config endpoint must
// call StoreAware.Invalidate(orgID) so the next Generate rebuilds.
//
// Note on mock provider: per-org mock provider is not supported via DB
// config. Use env DS_APM_AI_GENERATOR=mock for demo mode. If a tenant
// selects "mock" through the UI (i.e., via a stored DB config row),
// buildFromAIConfig returns an error and StoreAware falls back to
// envFallback.
type StoreAware struct {
	store       ruletypes.AIConfigStore
	cipher      *secretbox.Cipher
	envFallback ruletypes.AIStrategyGenerator
	mu          sync.RWMutex
	cache       map[string]ruletypes.AIStrategyGenerator
}

// Compile-time assertion that StoreAware implements AIStrategyGenerator.
var _ ruletypes.AIStrategyGenerator = (*StoreAware)(nil)

// NewStoreAware constructs a StoreAware generator. envFallback is used
// when no per-org config exists or when the config cannot be built (e.g.
// mock provider selected via UI).
func NewStoreAware(store ruletypes.AIConfigStore, cipher *secretbox.Cipher, envFallback ruletypes.AIStrategyGenerator) *StoreAware {
	return &StoreAware{
		store:       store,
		cipher:      cipher,
		envFallback: envFallback,
		cache:       make(map[string]ruletypes.AIStrategyGenerator),
	}
}

// Generate implements ruletypes.AIStrategyGenerator. It resolves the org
// from req.Labels, looks up the per-org config, builds (or reuses) a
// generator, and delegates. Falls back to envFallback on any error.
func (s *StoreAware) Generate(ctx context.Context, req ruletypes.AIStrategyRequest) (ruletypes.AIStrategy, error) {
	orgID := orgIDFromRequest(req)
	if orgID == "" {
		return s.envFallback.Generate(ctx, req)
	}
	gen, err := s.generatorFor(ctx, orgID)
	if err != nil || gen == nil {
		return s.envFallback.Generate(ctx, req)
	}
	return gen.Generate(ctx, req)
}

// GeneratorFor exposes the per-config lookup for callers that need to
// build a throwaway generator from a runtime-supplied AIConfig (e.g. the
// Test endpoint). It does NOT consult the store; it builds directly from
// the supplied config.
func (s *StoreAware) GeneratorFor(cfg ruletypes.AIConfig) (ruletypes.AIStrategyGenerator, error) {
	return buildFromAIConfig(cfg)
}

// Invalidate drops the cache entry for orgID so the next Generate call
// rebuilds from the latest stored config.
func (s *StoreAware) Invalidate(orgID string) {
	s.mu.Lock()
	delete(s.cache, orgID)
	s.mu.Unlock()
}

// generatorFor returns the cached generator for orgID, building it from
// the store on a cache miss.
func (s *StoreAware) generatorFor(ctx context.Context, orgID string) (ruletypes.AIStrategyGenerator, error) {
	s.mu.RLock()
	if g, ok := s.cache[orgID]; ok {
		s.mu.RUnlock()
		return g, nil
	}
	s.mu.RUnlock()

	cfg, err := s.store.Get(ctx, orgID, s.cipher.DecryptFunc())
	if err != nil {
		return nil, err
	}
	gen, err := buildFromAIConfig(cfg)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache[orgID] = gen
	s.mu.Unlock()
	return gen, nil
}

// buildFromAIConfig constructs an AIStrategyGenerator from a stored AIConfig.
// Mock provider is rejected with a descriptive error (see package comment).
func buildFromAIConfig(cfg ruletypes.AIConfig) (ruletypes.AIStrategyGenerator, error) {
	if cfg.Provider == "mock" {
		return nil, fmt.Errorf("aigenerator: per-org mock provider is not supported via DB config (use env DS_APM_AI_GENERATOR=mock for demo); falling back to envFallback")
	}
	return New(Config{
		Provider:          cfg.Provider,
		LLMProvider:       cfg.LLMProvider,
		LLMTransport:      cfg.Transport,
		LLMModel:          cfg.Model,
		LLMTimeoutSeconds: cfg.TimeoutSeconds,
		LLMAPIKey:         cfg.APIKey,
		LLMOAuthToken:     cfg.OAuthToken,
		LLMBinary:         cfg.BinaryPath,
	})
}

// orgIDFromRequest extracts the org identifier from the request labels.
// It tries common keys used across the SigNoz alert/tenant model.
func orgIDFromRequest(req ruletypes.AIStrategyRequest) string {
	if req.Labels == nil {
		return ""
	}
	// Try common keys; tenant id surfacing pattern
	// (see pkg/types/alertmanagertypes for canonical label names).
	for _, k := range []string{"org_id", "project_id", "tenant_id"} {
		if v := req.Labels[k]; v != "" {
			return v
		}
	}
	return ""
}
