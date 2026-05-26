package aigenerator

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/aiconfigstoretest"
	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/secretbox"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

// countingGen is a stub AIStrategyGenerator that records how many times
// Generate was called and returns a fixed strategy.
type countingGen struct {
	calls atomic.Int64
	out   ruletypes.AIStrategy
}

func (g *countingGen) Generate(_ context.Context, _ ruletypes.AIStrategyRequest) (ruletypes.AIStrategy, error) {
	g.calls.Add(1)
	return g.out, nil
}

// newTestStoreAware builds a StoreAware with a plaintext cipher (for tests)
// and the supplied fake store and fallback.
func newTestStoreAware(store ruletypes.AIConfigStore, fallback ruletypes.AIStrategyGenerator) *StoreAware {
	return NewStoreAware(store, secretbox.PlaintextCipher(), fallback)
}

// validLocalConfig returns an AIConfig that NewStoreAware can build into a
// real generator (provider=local needs no extra fields).
func validLocalConfig(orgID string) ruletypes.AIConfig {
	return ruletypes.AIConfig{
		ContractVersion: ruletypes.AIConfigContractVersion,
		OrgID:           orgID,
		Provider:        "local",
		UpdatedAt:       "2026-05-21T00:00:00Z",
	}
}

func TestStoreAware_FallsBackWhenOrgMissing(t *testing.T) {
	fallback := &countingGen{out: ruletypes.AIStrategy{StrategyID: "fallback"}}
	sa := newTestStoreAware(aiconfigstoretest.New(), fallback)

	// Request has no org-identifying labels.
	req := ruletypes.AIStrategyRequest{
		IncidentID: "INC-1",
		Labels:     map[string]string{"alertname": "HighCPU"},
	}
	got, err := sa.Generate(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "fallback", got.StrategyID)
	require.Equal(t, int64(1), fallback.calls.Load())
}

func TestStoreAware_FallsBackWhenStoreReturnsNotFound(t *testing.T) {
	fallback := &countingGen{out: ruletypes.AIStrategy{StrategyID: "fallback"}}
	// Empty store — no config for org-1.
	sa := newTestStoreAware(aiconfigstoretest.New(), fallback)

	req := ruletypes.AIStrategyRequest{
		IncidentID: "INC-1",
		Labels:     map[string]string{"org_id": "org-1"},
	}
	got, err := sa.Generate(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "fallback", got.StrategyID)
	require.Equal(t, int64(1), fallback.calls.Load())
}

func TestStoreAware_UsesPerOrgGenerator(t *testing.T) {
	fallback := &countingGen{out: ruletypes.AIStrategy{StrategyID: "fallback"}}
	store := aiconfigstoretest.New()

	// Seed config for org-1 — provider=local produces a real strategy.
	cfg := validLocalConfig("org-1")
	require.NoError(t, store.Upsert(context.Background(), cfg, func(s string) (string, error) { return s, nil }))

	sa := newTestStoreAware(store, fallback)

	req := ruletypes.AIStrategyRequest{
		IncidentID: "INC-1",
		Labels:     map[string]string{"org_id": "org-1", "alertname": "X", "service.name": "Y"},
	}
	got, err := sa.Generate(context.Background(), req)
	require.NoError(t, err)
	// Local generator returns a ready or non-empty strategy; it is NOT the
	// fallback stub so StrategyID will differ.
	require.NotEqual(t, "fallback", got.StrategyID, "expected per-org generator, not fallback")
	// Fallback should never have been called.
	require.Equal(t, int64(0), fallback.calls.Load())
}

func TestStoreAware_CachesPerOrg(t *testing.T) {
	fallback := &countingGen{out: ruletypes.AIStrategy{StrategyID: "fallback"}}
	store := aiconfigstoretest.New()

	cfg := validLocalConfig("org-1")
	require.NoError(t, store.Upsert(context.Background(), cfg, func(s string) (string, error) { return s, nil }))

	sa := newTestStoreAware(store, fallback)

	req := ruletypes.AIStrategyRequest{
		IncidentID: "INC-1",
		Labels:     map[string]string{"org_id": "org-1", "alertname": "X", "service.name": "Y"},
	}

	// First call populates cache.
	_, err := sa.Generate(context.Background(), req)
	require.NoError(t, err)

	// Remove the config from the store — a second call must still succeed
	// because the generator is cached. We can't easily remove from the fake,
	// so instead verify the cache has exactly one entry.
	sa.mu.RLock()
	cacheLen := len(sa.cache)
	sa.mu.RUnlock()
	require.Equal(t, 1, cacheLen, "cache should hold one entry after first Generate")

	// Second call — store is not consulted again (cache hit).
	_, err = sa.Generate(context.Background(), req)
	require.NoError(t, err)

	// Still only one cache entry.
	sa.mu.RLock()
	cacheLen = len(sa.cache)
	sa.mu.RUnlock()
	require.Equal(t, 1, cacheLen)
}

func TestStoreAware_InvalidateForcesRebuild(t *testing.T) {
	fallback := &countingGen{out: ruletypes.AIStrategy{StrategyID: "fallback"}}
	store := aiconfigstoretest.New()

	cfg := validLocalConfig("org-1")
	require.NoError(t, store.Upsert(context.Background(), cfg, func(s string) (string, error) { return s, nil }))

	sa := newTestStoreAware(store, fallback)

	req := ruletypes.AIStrategyRequest{
		IncidentID: "INC-1",
		Labels:     map[string]string{"org_id": "org-1", "alertname": "X", "service.name": "Y"},
	}

	// First Generate — populates cache.
	_, err := sa.Generate(context.Background(), req)
	require.NoError(t, err)

	// Verify cache is populated.
	sa.mu.RLock()
	_, inCache := sa.cache["org-1"]
	sa.mu.RUnlock()
	require.True(t, inCache)

	// Invalidate.
	sa.Invalidate("org-1")

	// Verify cache is empty.
	sa.mu.RLock()
	_, inCache = sa.cache["org-1"]
	sa.mu.RUnlock()
	require.False(t, inCache, "cache entry should be gone after Invalidate")

	// Second Generate — must go back to store (store still has config).
	_, err = sa.Generate(context.Background(), req)
	require.NoError(t, err)

	// Cache repopulated.
	sa.mu.RLock()
	_, inCache = sa.cache["org-1"]
	sa.mu.RUnlock()
	require.True(t, inCache)
}

func TestStoreAware_MockProviderViaConfigErrors(t *testing.T) {
	// buildFromAIConfig should return an error for provider=mock (DB-sourced).
	cfg := ruletypes.AIConfig{
		ContractVersion: ruletypes.AIConfigContractVersion,
		OrgID:           "org-1",
		Provider:        "mock",
		UpdatedAt:       "2026-05-21T00:00:00Z",
	}
	_, err := buildFromAIConfig(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mock provider is not supported via DB config")
}
