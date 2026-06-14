// Package sqlcodebasercaconfigstore is the SQL-backed implementation of
// ruletypes.CodebaseRCAConfigStore (CF-11 per-org feature toggle + thresholds).
package sqlcodebasercaconfigstore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/uptrace/bun"
)

// storableCodebaseRCAConfig is the flat DB row for ds_codebase_config.
// Column tags must match the DDL in migration 083.
type storableCodebaseRCAConfig struct {
	bun.BaseModel              `bun:"table:ds_codebase_config"`
	OrgID                      string `bun:"org_id,pk"`
	Enabled                    bool   `bun:"enabled"`
	MinSeverity                string `bun:"min_severity"`
	CooldownWindowSecs         int    `bun:"cooldown_window_secs"`
	MaxRunsPerDay              int    `bun:"max_runs_per_day"`
	MaxQueueDepth              int    `bun:"max_queue_depth"`
	MaxConcurrentRuns          int    `bun:"max_concurrent_runs"`
	AllowUnboundWithoutAnomaly bool   `bun:"allow_unbound_without_anomaly"`
	UpdatedAt                  string `bun:"updated_at"`
}

func (r *storableCodebaseRCAConfig) toDomain() ruletypes.CodebaseRCAConfig {
	return ruletypes.CodebaseRCAConfig{
		ContractVersion:            ruletypes.CodebaseRCAConfigContractVersion,
		OrgID:                      r.OrgID,
		Enabled:                    r.Enabled,
		MinSeverity:                r.MinSeverity,
		CooldownWindowSecs:         r.CooldownWindowSecs,
		MaxRunsPerDay:              r.MaxRunsPerDay,
		MaxQueueDepth:              r.MaxQueueDepth,
		MaxConcurrentRuns:          r.MaxConcurrentRuns,
		AllowUnboundWithoutAnomaly: r.AllowUnboundWithoutAnomaly,
		UpdatedAt:                  r.UpdatedAt,
	}
}

func fromDomain(cfg ruletypes.CodebaseRCAConfig) *storableCodebaseRCAConfig {
	updatedAt := cfg.UpdatedAt
	if updatedAt == "" {
		updatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return &storableCodebaseRCAConfig{
		OrgID:                      cfg.OrgID,
		Enabled:                    cfg.Enabled,
		MinSeverity:                cfg.MinSeverity,
		CooldownWindowSecs:         cfg.CooldownWindowSecs,
		MaxRunsPerDay:              cfg.MaxRunsPerDay,
		MaxQueueDepth:              cfg.MaxQueueDepth,
		MaxConcurrentRuns:          cfg.MaxConcurrentRuns,
		AllowUnboundWithoutAnomaly: cfg.AllowUnboundWithoutAnomaly,
		UpdatedAt:                  updatedAt,
	}
}

// Store is the SQL-backed CodebaseRCAConfigStore. Migration 083 must have run
// (table ds_codebase_config).
type Store struct {
	sqlstore sqlstore.SQLStore
}

// Compile-time assertion: Store satisfies the interface.
var _ ruletypes.CodebaseRCAConfigStore = (*Store)(nil)

// New returns a CodebaseRCAConfigStore backed by the given SQLStore.
func New(store sqlstore.SQLStore) *Store {
	return &Store{sqlstore: store}
}

// Upsert inserts or updates the per-org config row.
func (s *Store) Upsert(ctx context.Context, cfg ruletypes.CodebaseRCAConfig) error {
	row := fromDomain(cfg)
	return s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		_, err := s.sqlstore.BunDBCtx(ctx).
			NewInsert().
			Model(row).
			On("CONFLICT (org_id) DO UPDATE").
			Set("enabled = EXCLUDED.enabled").
			Set("min_severity = EXCLUDED.min_severity").
			Set("cooldown_window_secs = EXCLUDED.cooldown_window_secs").
			Set("max_runs_per_day = EXCLUDED.max_runs_per_day").
			Set("max_queue_depth = EXCLUDED.max_queue_depth").
			Set("max_concurrent_runs = EXCLUDED.max_concurrent_runs").
			Set("allow_unbound_without_anomaly = EXCLUDED.allow_unbound_without_anomaly").
			Set("updated_at = EXCLUDED.updated_at").
			Exec(ctx)
		return err
	})
}

// Get returns the per-org config. Returns ErrCodebaseRCAConfigNotFound when no
// row exists for the given orgID.
func (s *Store) Get(ctx context.Context, orgID string) (ruletypes.CodebaseRCAConfig, error) {
	row := new(storableCodebaseRCAConfig)
	err := s.sqlstore.BunDBCtx(ctx).
		NewSelect().
		Model(row).
		Where("org_id = ?", orgID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return ruletypes.CodebaseRCAConfig{}, ruletypes.ErrCodebaseRCAConfigNotFound
	}
	if err != nil {
		return ruletypes.CodebaseRCAConfig{}, err
	}
	return row.toDomain(), nil
}
