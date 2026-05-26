package sqlaiconfigstore

import (
	"context"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstoretest"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

// identity encrypt/decrypt for tests — no real encryption needed.
func identityEncrypt(s string) (string, error) { return s, nil }
func identityDecrypt(s string) (string, error) { return s, nil }

func applyAIConfigDDL(ctx context.Context, ss sqlstore.SQLStore) error {
	stmts := []string{
		`CREATE TABLE ds_ai_config (
			org_id                  TEXT      NOT NULL PRIMARY KEY,
			provider                TEXT      NOT NULL,
			llm_provider            TEXT      NOT NULL DEFAULT '',
			transport               TEXT      NOT NULL DEFAULT '',
			model                   TEXT      NOT NULL DEFAULT '',
			api_key_ciphertext      TEXT      NOT NULL DEFAULT '',
			binary_path             TEXT      NOT NULL DEFAULT '',
			timeout_seconds         INTEGER   NOT NULL DEFAULT 0,
			updated_at              TEXT      NOT NULL,
			oauth_token_ciphertext  TEXT      NOT NULL DEFAULT ''
		)`,
	}
	for _, stmt := range stmts {
		if _, err := ss.BunDB().ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func newTestStoreRaw(t *testing.T) ruletypes.AIConfigStore {
	t.Helper()
	ctx := context.Background()
	ss := sqlitesqlstoretest.New(t)
	require.NoError(t, applyAIConfigDDL(ctx, ss))
	return New(ss)
}

func makeAIConfig(orgID, provider string) ruletypes.AIConfig {
	return ruletypes.AIConfig{
		ContractVersion: ruletypes.AIConfigContractVersion,
		OrgID:           orgID,
		Provider:        provider,
		LLMProvider:     "",
		Transport:       "",
		Model:           "test-model",
		APIKey:          "secret-key-" + orgID,
		BinaryPath:      "",
		TimeoutSeconds:  30,
		UpdatedAt:       time.Now().UTC().Format(time.RFC3339),
	}
}

func makeLLMConfig(orgID string) ruletypes.AIConfig {
	return ruletypes.AIConfig{
		ContractVersion: ruletypes.AIConfigContractVersion,
		OrgID:           orgID,
		Provider:        "llm",
		LLMProvider:     "claude",
		Transport:       "api",
		Model:           "claude-3-sonnet",
		APIKey:          "sk-ant-" + orgID,
		BinaryPath:      "",
		TimeoutSeconds:  60,
		UpdatedAt:       time.Now().UTC().Format(time.RFC3339),
	}
}

func TestAIConfigStore_UpsertGet(t *testing.T) {
	ctx := context.Background()
	store := newTestStoreRaw(t)

	cfg := makeAIConfig("org-1", "local")

	require.NoError(t, store.Upsert(ctx, cfg, identityEncrypt))

	got, err := store.Get(ctx, "org-1", identityDecrypt)
	require.NoError(t, err)
	require.Equal(t, cfg.OrgID, got.OrgID)
	require.Equal(t, cfg.Provider, got.Provider)
	require.Equal(t, cfg.Model, got.Model)
	require.Equal(t, cfg.APIKey, got.APIKey)
	require.Equal(t, cfg.TimeoutSeconds, got.TimeoutSeconds)
}

func TestAIConfigStore_GetNotFound(t *testing.T) {
	ctx := context.Background()
	store := newTestStoreRaw(t)

	_, err := store.Get(ctx, "org-missing", identityDecrypt)
	require.ErrorIs(t, err, ruletypes.ErrAIConfigNotFound)
}

func TestAIConfigStore_CrossTenantIsolation(t *testing.T) {
	ctx := context.Background()
	store := newTestStoreRaw(t)

	cfgA := makeAIConfig("org-A", "local")
	require.NoError(t, store.Upsert(ctx, cfgA, identityEncrypt))

	_, err := store.Get(ctx, "org-B", identityDecrypt)
	require.ErrorIs(t, err, ruletypes.ErrAIConfigNotFound, "C2 regression: cross-tenant config visible")
}

func TestAIConfigStore_UpsertOverwrite(t *testing.T) {
	ctx := context.Background()
	store := newTestStoreRaw(t)

	first := makeAIConfig("org-1", "local")
	first.Model = "first-model"
	require.NoError(t, store.Upsert(ctx, first, identityEncrypt))

	second := makeAIConfig("org-1", "llm")
	second.Model = "second-model"
	second.LLMProvider = "claude"
	second.Transport = "api"
	require.NoError(t, store.Upsert(ctx, second, identityEncrypt))

	got, err := store.Get(ctx, "org-1", identityDecrypt)
	require.NoError(t, err)
	require.Equal(t, "llm", got.Provider)
	require.Equal(t, "second-model", got.Model)
	require.Equal(t, "claude", got.LLMProvider)
}

func TestAIConfigStore_APIKeyRoundtrip(t *testing.T) {
	ctx := context.Background()
	store := newTestStoreRaw(t)

	cfg := makeLLMConfig("org-roundtrip")
	originalKey := cfg.APIKey

	require.NoError(t, store.Upsert(ctx, cfg, identityEncrypt))

	got, err := store.Get(ctx, "org-roundtrip", identityDecrypt)
	require.NoError(t, err)
	require.Equal(t, originalKey, got.APIKey, "plaintext API key must survive encrypt→store→decrypt roundtrip")
}

func TestSQLAIConfigStore_OAuthTokenRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newTestStoreRaw(t)

	cfg := ruletypes.AIConfig{
		ContractVersion: ruletypes.AIConfigContractVersion,
		OrgID:           "org-oauth",
		Provider:        "llm",
		LLMProvider:     "claude",
		Transport:       "cli",
		OAuthToken:      "tok-xyz",
		UpdatedAt:       "2026-05-21T00:00:00Z",
	}

	require.NoError(t, store.Upsert(ctx, cfg, identityEncrypt), "Upsert(insert)")

	// Second write to exercise the ON CONFLICT path — the row already exists,
	// so the update branch runs. If the new column isn't in the SET list,
	// the value will be discarded here.
	cfg.OAuthToken = "tok-updated"
	require.NoError(t, store.Upsert(ctx, cfg, identityEncrypt), "Upsert(update)")

	got, err := store.Get(ctx, "org-oauth", identityDecrypt)
	require.NoError(t, err, "Get")
	require.Equal(t, "tok-updated", got.OAuthToken, "OAuthToken must survive second-write ON CONFLICT path")
}
