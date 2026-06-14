// Package coderca_test is the Phase-A gate: an end-to-end integration test
// that proves the UJ-5 path (trigger → queued run → engine → handoff) works
// on a real SQLite store, plus fail-open and fail-closed proofs.
package coderca_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/clirunner"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/codebaseconfigstore/sqlcodebaseconfigstore"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/codebaseconfigstore/sqlcodebasercaconfigstore"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/codebaseconfigstore/sqlcodebaseservicemapstore"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/engine"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/reporesolver"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/trigger"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstoretest"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
)

// ---------------------------------------------------------------------------
// DDL helpers
// ---------------------------------------------------------------------------

// applyE2EDDL creates the union of all coderca tables needed for the e2e test.
// coderca_run already includes the 4 report columns added by migration 083
// (root_cause / proposed_fix / confidence / limitations) so no ALTER is needed.
func applyE2EDDL(ctx context.Context, ss sqlstore.SQLStore) error {
	stmts := []string{
		// ---- migration 082 tables ----
		`CREATE TABLE IF NOT EXISTS ds_codebase_repo (
			org_id                 TEXT      NOT NULL,
			repo_id                TEXT      NOT NULL,
			git_url                TEXT      NOT NULL,
			default_branch         TEXT      NOT NULL DEFAULT '',
			credential_ciphertext  TEXT      NOT NULL DEFAULT '',
			enabled                BOOLEAN   NOT NULL DEFAULT FALSE,
			branch_name            TEXT      NOT NULL DEFAULT '',
			fetched                BOOLEAN   NOT NULL DEFAULT FALSE,
			baseline_commit        TEXT      NOT NULL DEFAULT '',
			last_sync_at           TEXT      NOT NULL DEFAULT '',
			last_sync_status       TEXT      NOT NULL DEFAULT '',
			PRIMARY KEY (org_id, repo_id)
		)`,
		`CREATE TABLE IF NOT EXISTS ds_codebase_service_map (
			org_id        TEXT NOT NULL,
			service_name  TEXT NOT NULL,
			repo_id       TEXT NOT NULL,
			subpath       TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (org_id, service_name)
		)`,
		// coderca_run with 083 report columns folded in directly
		`CREATE TABLE IF NOT EXISTS coderca_run (
			run_id          TEXT    NOT NULL PRIMARY KEY,
			org_id          TEXT    NOT NULL,
			service         TEXT    NOT NULL DEFAULT '',
			dedup_key       TEXT    NOT NULL,
			status          TEXT    NOT NULL,
			baseline_commit TEXT    NOT NULL DEFAULT '',
			created_at      INTEGER NOT NULL,
			claimed_by      TEXT    NOT NULL DEFAULT '',
			lease_token     TEXT    NOT NULL DEFAULT '',
			lease_until     INTEGER NOT NULL DEFAULT 0,
			heartbeat_at    INTEGER NOT NULL DEFAULT 0,
			attempts        INTEGER NOT NULL DEFAULT 0,
			finished_at     INTEGER NOT NULL DEFAULT 0,
			result_ref      TEXT    NOT NULL DEFAULT '',
			root_cause      TEXT    NOT NULL DEFAULT '',
			proposed_fix    TEXT    NOT NULL DEFAULT '',
			confidence      TEXT    NOT NULL DEFAULT '',
			limitations     TEXT    NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS coderca_admission (
			org_id           TEXT    NOT NULL,
			dedup_key        TEXT    NOT NULL,
			last_admitted_at INTEGER NOT NULL,
			hit_count        INTEGER NOT NULL DEFAULT 0,
			last_run_ref     TEXT    NOT NULL DEFAULT '',
			PRIMARY KEY (org_id, dedup_key)
		)`,
		`CREATE TABLE IF NOT EXISTS coderca_budget (
			org_id TEXT    NOT NULL,
			day    TEXT    NOT NULL,
			used   INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (org_id, day)
		)`,
		`CREATE TABLE IF NOT EXISTS coderca_capacity (
			scope               TEXT    NOT NULL PRIMARY KEY,
			running             INTEGER NOT NULL DEFAULT 0,
			max_concurrent_runs INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS coderca_skip_stat (
			org_id TEXT    NOT NULL,
			reason TEXT    NOT NULL,
			day    TEXT    NOT NULL,
			count  INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (org_id, reason, day)
		)`,
		// ---- migration 083 tables ----
		`CREATE TABLE IF NOT EXISTS ds_codebase_config (
			org_id                        TEXT    NOT NULL PRIMARY KEY,
			enabled                       BOOLEAN NOT NULL DEFAULT FALSE,
			min_severity                  TEXT    NOT NULL DEFAULT 'high',
			cooldown_window_secs          INTEGER NOT NULL DEFAULT 21600,
			max_runs_per_day              INTEGER NOT NULL DEFAULT 20,
			max_queue_depth               INTEGER NOT NULL DEFAULT 50,
			max_concurrent_runs           INTEGER NOT NULL DEFAULT 1,
			allow_unbound_without_anomaly BOOLEAN NOT NULL DEFAULT FALSE,
			updated_at                    TEXT    NOT NULL DEFAULT ''
		)`,
	}
	for _, stmt := range stmts {
		if _, err := ss.BunDB().ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// identityEncrypt / identityDecrypt are no-op closures for tests (no secrets).
func identityEncrypt(s string) (string, error) { return s, nil }
func identityDecrypt(s string) (string, error) { return s, nil }

// ---------------------------------------------------------------------------
// Fake ports (engine interfaces)
// ---------------------------------------------------------------------------

type e2eFakeSource struct{}

func (e2eFakeSource) Prepare(_ context.Context, _ ruletypes.CodebaseRepo, _ string) (string, string, func(), error) {
	return "/tmp/fake-checkout", "base-sha-123", func() {}, nil
}

type e2eFakeCLI struct{}

func (e2eFakeCLI) Run(_ context.Context, _ clirunner.Spec) (coderca.RCAResult, coderca.RunStatus, error) {
	return coderca.RCAResult{
		BaselineCommit: "base-sha-123",
		RootCause:      "nil deref in pay handler",
		ProposedFix:    "add guard",
		Confidence:     "high",
	}, coderca.RunStatusDone, nil
}

type capturedDelivery struct {
	deliveries []engine.Delivery
}

func (c *capturedDelivery) Deliver(_ context.Context, d engine.Delivery) (string, error) {
	c.deliveries = append(c.deliveries, d)
	return "handoff-1", nil
}

type capturedAudit struct {
	events []engine.AuditEvent
}

func (c *capturedAudit) Audit(_ context.Context, e engine.AuditEvent) {
	c.events = append(c.events, e)
}

// ---------------------------------------------------------------------------
// Shared setup helper
// ---------------------------------------------------------------------------

type e2eFixture struct {
	ss         sqlstore.SQLStore
	runStore   *runstore.Store
	cfgStore   ruletypes.CodebaseRCAConfigStore
	mapStore   ruletypes.CodebaseServiceMapStore
	repoStore  ruletypes.CodebaseRepoStore
	trig       *trigger.Trigger
	eng        *engine.Engine
	deliverer  *capturedDelivery
	auditor    *capturedAudit
}

func newE2EFixture(t *testing.T) *e2eFixture {
	t.Helper()
	ctx := context.Background()

	ss := sqlitesqlstoretest.New(t)
	require.NoError(t, applyE2EDDL(ctx, ss))

	runStore := runstore.New(ss)
	cfgStore := sqlcodebasercaconfigstore.New(ss)
	mapStore := sqlcodebaseservicemapstore.New(ss)
	repoStore := sqlcodebaseconfigstore.New(ss)

	// Upsert enabled config
	cfg := ruletypes.DefaultCodebaseRCAConfig("org-1")
	cfg.Enabled = true
	require.NoError(t, cfgStore.Upsert(ctx, cfg))

	// Upsert service→repo mapping
	require.NoError(t, mapStore.Upsert(ctx, ruletypes.CodebaseServiceMap{
		OrgID:       "org-1",
		ServiceName: "pay",
		RepoID:      "repo-1",
	}))

	// Upsert repo
	require.NoError(t, repoStore.Upsert(ctx, ruletypes.CodebaseRepo{
		ContractVersion: ruletypes.CodebaseRepoContractVersion,
		OrgID:           "org-1",
		RepoID:          "repo-1",
		GitURL:          "https://example.com/x.git",
		Enabled:         true,
	}, identityEncrypt))

	trig := trigger.New(cfgStore, mapStore, runStore, slog.Default(), nil)

	deliverer := &capturedDelivery{}
	auditor := &capturedAudit{}

	eng := engine.New(
		engine.Config{
			Scope:        "global",
			InstanceID:   "test",
			Agent:        clirunner.AgentClaude,
			Model:        "m",
			MaxBudgetUSD: "0.50",
			AuthToken:    "t",
		},
		engine.Deps{
			Runs:    runStore,
			Repos:   reporesolver.New(mapStore, repoStore, identityDecrypt),
			Source:  e2eFakeSource{},
			CLI:     e2eFakeCLI{},
			Deliver: deliverer,
			Auditor: auditor,
		},
	)

	return &e2eFixture{
		ss:        ss,
		runStore:  runStore,
		cfgStore:  cfgStore,
		mapStore:  mapStore,
		repoStore: repoStore,
		trig:      trig,
		eng:       eng,
		deliverer: deliverer,
		auditor:   auditor,
	}
}

// ---------------------------------------------------------------------------
// TestUJ5EndToEnd_UnboundAnomalyAlertProducesHandoff
// ---------------------------------------------------------------------------

func TestUJ5EndToEnd_UnboundAnomalyAlertProducesHandoff(t *testing.T) {
	ctx := context.Background()
	fx := newE2EFixture(t)

	labels := map[string]string{
		"alertname":    "PayErr",
		"service.name": "pay",
		"severity":     "critical",
		"anomaly":      "true",
	}

	// Fixed "now" so both Maybe calls share the same instant → cooldown applies.
	fixedNow := time.Unix(1_700_000_000, 0)
	fx.trig = trigger.New(fx.cfgStore, fx.mapStore, fx.runStore, slog.Default(),
		func() time.Time { return fixedNow })

	// Step 1: trigger.Maybe
	fx.trig.Maybe(ctx, "org-1", labels, nil)

	// Step 2: exactly 1 queued run
	runs, err := fx.runStore.ListRuns(ctx, "org-1", runstore.ListRunsParams{})
	require.NoError(t, err)
	require.Len(t, runs, 1, "expected exactly 1 queued run after trigger")
	assert.Equal(t, coderca.RunStatusQueued, runs[0].Status)

	// Step 3: engine processes the run
	processed, err := fx.eng.ProcessNext(ctx)
	require.NoError(t, err)
	assert.True(t, processed, "engine must have processed the queued run")

	// Step 4: deliverer got 1 delivery with correct service and root cause
	require.Len(t, fx.deliverer.deliveries, 1, "expected exactly 1 delivery")
	d := fx.deliverer.deliveries[0]
	assert.Equal(t, "pay", d.Service)
	assert.True(t, strings.Contains(d.Result.RootCause, "nil deref"),
		"RootCause should contain 'nil deref', got: %q", d.Result.RootCause)

	// Step 5: run status==done + RootCause persisted + BaselineCommit=="base-sha-123"
	detail, err := fx.runStore.GetRun(ctx, "org-1", runs[0].RunID)
	require.NoError(t, err)
	assert.Equal(t, coderca.RunStatusDone, detail.Status)
	assert.True(t, strings.Contains(detail.RootCause, "nil deref"),
		"persisted RootCause should contain 'nil deref', got: %q", detail.RootCause)
	assert.Equal(t, "base-sha-123", detail.BaselineCommit)

	// Step 6: same labels, same instant → cooldown dedup, still exactly 1 run
	fx.trig.Maybe(ctx, "org-1", labels, nil)
	runs2, err := fx.runStore.ListRuns(ctx, "org-1", runstore.ListRunsParams{})
	require.NoError(t, err)
	assert.Len(t, runs2, 1, "cooldown dedup: second Maybe must not create a second run")
}

// ---------------------------------------------------------------------------
// TestUJ5FailOpen_TriggerNeverBlocksAlertPath
// ---------------------------------------------------------------------------

// panicCfgStore panics on Get — used to prove the trigger's recover() catches it.
type panicCfgStore struct{}

func (panicCfgStore) Get(_ context.Context, _ string) (ruletypes.CodebaseRCAConfig, error) {
	panic("panicCfgStore: intentional panic for fail-open test")
}
func (panicCfgStore) Upsert(_ context.Context, _ ruletypes.CodebaseRCAConfig) error { return nil }

// slowCfgStore blocks until its context is cancelled, simulating a store that
// never returns within the trigger's internal 1s timeout.
type slowCfgStore struct{}

func (slowCfgStore) Get(ctx context.Context, _ string) (ruletypes.CodebaseRCAConfig, error) {
	select {
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
	}
	return ruletypes.CodebaseRCAConfig{}, ctx.Err()
}
func (slowCfgStore) Upsert(_ context.Context, _ ruletypes.CodebaseRCAConfig) error { return nil }

// fakeMapsAlwaysMissing always returns ErrCodebaseServiceMapNotFound.
type fakeMapsAlwaysMissing struct{}

func (fakeMapsAlwaysMissing) Get(_ context.Context, _, _ string) (ruletypes.CodebaseServiceMap, error) {
	return ruletypes.CodebaseServiceMap{}, ruletypes.ErrCodebaseServiceMapNotFound
}
func (fakeMapsAlwaysMissing) Upsert(_ context.Context, _ ruletypes.CodebaseServiceMap) error {
	return nil
}
func (fakeMapsAlwaysMissing) Delete(_ context.Context, _, _ string) error { return nil }
func (fakeMapsAlwaysMissing) List(_ context.Context, _ string) ([]ruletypes.CodebaseServiceMap, error) {
	return nil, nil
}

func TestUJ5FailOpen_TriggerNeverBlocksAlertPath(t *testing.T) {
	// We call trigger.Maybe directly (rather than wiring a full dispatchhook.Hook)
	// because Hook.New requires ruletypes.SOPStore and AIStrategyGenerator which
	// are orthogonal to this test. The hook→trigger contract is already
	// unit-tested in hook_test.go (TestApplyCallsCodeRCATriggerOnUnbound).
	// trigger.Maybe IS the code path exercised by the hook's unbound branch.

	labels := map[string]string{
		"alertname":    "PayErr",
		"service.name": "pay",
		"severity":     "critical",
		"anomaly":      "true",
	}

	t.Run("case A: panicking cfgStore is swallowed by trigger recover", func(t *testing.T) {
		ctx := context.Background()
		ss := sqlitesqlstoretest.New(t)
		require.NoError(t, applyE2EDDL(ctx, ss))
		runStore := runstore.New(ss)

		trig := trigger.New(panicCfgStore{}, fakeMapsAlwaysMissing{}, runStore, slog.Default(), nil)

		// Must not panic.
		require.NotPanics(t, func() {
			trig.Maybe(ctx, "org-1", labels, nil)
		})
		// No run created (panicked before admit).
		runs, err := runStore.ListRuns(ctx, "org-1", runstore.ListRunsParams{})
		require.NoError(t, err)
		assert.Len(t, runs, 0)
	})

	t.Run("case B: slow cfgStore (2s) returns within ~1.5s due to trigger's 1s timeout", func(t *testing.T) {
		ctx := context.Background()
		ss := sqlitesqlstoretest.New(t)
		require.NoError(t, applyE2EDDL(ctx, ss))
		runStore := runstore.New(ss)

		trig := trigger.New(slowCfgStore{}, fakeMapsAlwaysMissing{}, runStore, slog.Default(), nil)

		start := time.Now()
		trig.Maybe(ctx, "org-1", labels, nil)
		elapsed := time.Since(start)

		// trigger's internal maybeTimeout is 1s; allow 500ms CI overhead.
		assert.Less(t, elapsed, 1500*time.Millisecond,
			"trigger.Maybe must return within ~1.5s when cfgStore is slow (got %v)", elapsed)

		// No run created (timed out before admit).
		runs, err := runStore.ListRuns(ctx, "org-1", runstore.ListRunsParams{})
		require.NoError(t, err)
		assert.Len(t, runs, 0)
	})
}

// ---------------------------------------------------------------------------
// TestUJ5FailClosed_NoFiringWithoutGates
// ---------------------------------------------------------------------------

func TestUJ5FailClosed_NoFiringWithoutGates(t *testing.T) {
	t.Run("(a) no anomaly label → 0 runs", func(t *testing.T) {
		ctx := context.Background()
		ss := sqlitesqlstoretest.New(t)
		require.NoError(t, applyE2EDDL(ctx, ss))
		runStore := runstore.New(ss)
		cfgStore := sqlcodebasercaconfigstore.New(ss)
		mapStore := sqlcodebaseservicemapstore.New(ss)

		cfg := ruletypes.DefaultCodebaseRCAConfig("org-1")
		cfg.Enabled = true
		require.NoError(t, cfgStore.Upsert(ctx, cfg))
		require.NoError(t, mapStore.Upsert(ctx, ruletypes.CodebaseServiceMap{
			OrgID: "org-1", ServiceName: "pay", RepoID: "repo-1",
		}))

		trig := trigger.New(cfgStore, mapStore, runStore, slog.Default(), nil)
		trig.Maybe(ctx, "org-1", map[string]string{
			"alertname":    "PayErr",
			"service.name": "pay",
			"severity":     "critical",
			// no "anomaly" key → fail-closed
		}, nil)

		runs, err := runStore.ListRuns(ctx, "org-1", runstore.ListRunsParams{})
		require.NoError(t, err)
		assert.Len(t, runs, 0, "no anomaly → no run")
	})

	t.Run("(b) Enabled=false → 0 runs", func(t *testing.T) {
		ctx := context.Background()
		ss := sqlitesqlstoretest.New(t)
		require.NoError(t, applyE2EDDL(ctx, ss))
		runStore := runstore.New(ss)
		cfgStore := sqlcodebasercaconfigstore.New(ss)
		mapStore := sqlcodebaseservicemapstore.New(ss)

		// Enabled=false (default)
		cfg := ruletypes.DefaultCodebaseRCAConfig("org-1")
		cfg.Enabled = false
		require.NoError(t, cfgStore.Upsert(ctx, cfg))
		require.NoError(t, mapStore.Upsert(ctx, ruletypes.CodebaseServiceMap{
			OrgID: "org-1", ServiceName: "pay", RepoID: "repo-1",
		}))

		trig := trigger.New(cfgStore, mapStore, runStore, slog.Default(), nil)
		trig.Maybe(ctx, "org-1", map[string]string{
			"alertname":    "PayErr",
			"service.name": "pay",
			"severity":     "critical",
			"anomaly":      "true",
		}, nil)

		runs, err := runStore.ListRuns(ctx, "org-1", runstore.ListRunsParams{})
		require.NoError(t, err)
		assert.Len(t, runs, 0, "feature disabled → no run")
	})

	t.Run("(c) no service→repo mapping → 0 runs", func(t *testing.T) {
		ctx := context.Background()
		ss := sqlitesqlstoretest.New(t)
		require.NoError(t, applyE2EDDL(ctx, ss))
		runStore := runstore.New(ss)
		cfgStore := sqlcodebasercaconfigstore.New(ss)
		mapStore := sqlcodebaseservicemapstore.New(ss)

		cfg := ruletypes.DefaultCodebaseRCAConfig("org-1")
		cfg.Enabled = true
		require.NoError(t, cfgStore.Upsert(ctx, cfg))
		// Deliberately skip mapStore.Upsert → no mapping for "pay"

		trig := trigger.New(cfgStore, mapStore, runStore, slog.Default(), nil)
		trig.Maybe(ctx, "org-1", map[string]string{
			"alertname":    "PayErr",
			"service.name": "pay",
			"severity":     "critical",
			"anomaly":      "true",
		}, nil)

		runs, err := runStore.ListRuns(ctx, "org-1", runstore.ListRunsParams{})
		require.NoError(t, err)
		assert.Len(t, runs, 0, "no mapping → no run")
	})
}
