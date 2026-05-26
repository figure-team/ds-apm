package sqlaiconfigstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

type aiConfigStore struct {
	sqlstore sqlstore.SQLStore
}

// New returns an AIConfigStore backed by the given SQLStore. Migration 079
// must have run; the table ds_ai_config is read directly via bun ORM.
func New(store sqlstore.SQLStore) ruletypes.AIConfigStore {
	return &aiConfigStore{sqlstore: store}
}

func (s *aiConfigStore) Upsert(ctx context.Context, cfg ruletypes.AIConfig, encrypt func(string) (string, error)) error {
	storable, err := ruletypes.FromDomainAIConfig(cfg, encrypt)
	if err != nil {
		return err
	}
	return s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		_, err := s.sqlstore.BunDBCtx(ctx).
			NewInsert().
			Model(storable).
			On("CONFLICT (org_id) DO UPDATE").
			Set("provider = EXCLUDED.provider").
			Set("llm_provider = EXCLUDED.llm_provider").
			Set("transport = EXCLUDED.transport").
			Set("model = EXCLUDED.model").
			Set("api_key_ciphertext = EXCLUDED.api_key_ciphertext").
			Set("oauth_token_ciphertext = EXCLUDED.oauth_token_ciphertext").
			Set("binary_path = EXCLUDED.binary_path").
			Set("timeout_seconds = EXCLUDED.timeout_seconds").
			Set("updated_at = EXCLUDED.updated_at").
			Exec(ctx)
		return err
	})
}

func (s *aiConfigStore) Get(ctx context.Context, orgID string, decrypt func(string) (string, error)) (ruletypes.AIConfig, error) {
	storable := new(ruletypes.StorableAIConfig)
	err := s.sqlstore.BunDBCtx(ctx).
		NewSelect().
		Model(storable).
		Where("org_id = ?", orgID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return ruletypes.AIConfig{}, ruletypes.ErrAIConfigNotFound
	}
	if err != nil {
		return ruletypes.AIConfig{}, err
	}
	return storable.ToDomain(decrypt)
}
