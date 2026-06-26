package sqlremediationstore

import (
	"context"
	"testing"

	"github.com/SigNoz/signoz/pkg/ruler/remediationstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstoretest"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

// newTestStore builds an in-memory store with the 087 schema applied.
// Mirrors sqlsopstore/sop_test.go's setup helper — applies migration DDL
// directly against the in-memory SQLite store, then returns a ready SQLStore.
func newTestStore(t *testing.T) *SQLStore {
	t.Helper()
	ctx := context.Background()
	ss := sqlitesqlstoretest.New(t)
	stmts := []string{ // mirror 087 DDL exactly
		`CREATE TABLE IF NOT EXISTS ds_remediation_execution (
			id                 TEXT    NOT NULL PRIMARY KEY,
			org_id             TEXT    NOT NULL,
			incident_id        TEXT    NOT NULL DEFAULT '',
			alert_fingerprint  TEXT    NOT NULL DEFAULT '',
			sop_id             TEXT    NOT NULL DEFAULT '',
			sop_version        TEXT    NOT NULL DEFAULT '',
			runbook_id         TEXT    NOT NULL DEFAULT '',
			script_snapshot    TEXT    NOT NULL DEFAULT '',
			status             TEXT    NOT NULL,
			proposed_at        TEXT    NOT NULL DEFAULT '',
			approved_at        TEXT    NOT NULL DEFAULT '',
			executed_at        TEXT    NOT NULL DEFAULT '',
			terminal_at        TEXT    NOT NULL DEFAULT '',
			approved_by        TEXT    NOT NULL DEFAULT '',
			exit_code          INTEGER,
			output_snippet     TEXT    NOT NULL DEFAULT '',
			verify_result      TEXT    NOT NULL DEFAULT '',
			expires_at         TEXT    NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ds_remediation_org_incident
			ON ds_remediation_execution (org_id, incident_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ds_remediation_org_status
			ON ds_remediation_execution (org_id, status)`,
		`CREATE TABLE IF NOT EXISTS ds_remediation_config (
			org_id                 TEXT    NOT NULL PRIMARY KEY,
			execution_enabled      BOOLEAN NOT NULL DEFAULT FALSE,
			proposal_ttl_seconds   INTEGER NOT NULL DEFAULT 1800,
			exec_timeout_seconds   INTEGER NOT NULL DEFAULT 300,
			verify_window_seconds  INTEGER NOT NULL DEFAULT 600,
			max_concurrent         INTEGER NOT NULL DEFAULT 1
		)`,
	}
	for _, stmt := range stmts {
		_, err := ss.BunDB().ExecContext(ctx, stmt)
		require.NoError(t, err)
	}
	return New(ss)
}

func sampleExecution(id string) ruletypes.RemediationExecution {
	return ruletypes.RemediationExecution{
		ID:               id,
		OrgID:            "org-1",
		IncidentID:       "inc-1",
		AlertFingerprint: "fp-1",
		SOPID:            "SOP-1",
		SOPVersion:       "v1",
		RunbookID:        "rb-1",
		ScriptSnapshot:   "#!/bin/bash\necho hi\n",
		Status:           ruletypes.RemediationStatusProposed,
		ProposedAt:       "2026-06-24T00:00:00Z",
		ExpiresAt:        "2026-06-24T00:30:00Z",
	}
}

func TestCreateAndGet(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	e := sampleExecution("11111111-1111-1111-1111-111111111111")
	if err := s.Create(ctx, e); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, "org-1", e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ruletypes.RemediationStatusProposed || got.ScriptSnapshot != e.ScriptSnapshot {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
}

func TestTransitionToExecuting_SingleWinner(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	e := sampleExecution("22222222-2222-2222-2222-222222222222")
	_ = s.Create(ctx, e)

	won, err := s.TransitionToExecuting(ctx, "org-1", e.ID, "alice@x", "2026-06-24T00:05:00Z", 5)
	if err != nil || !won {
		t.Fatalf("first approve must win: won=%v err=%v", won, err)
	}
	// Second concurrent approve must lose (row no longer proposed).
	won2, err := s.TransitionToExecuting(ctx, "org-1", e.ID, "bob@x", "2026-06-24T00:06:00Z", 5)
	if err != nil || won2 {
		t.Fatalf("second approve must lose: won=%v err=%v", won2, err)
	}
	got, _ := s.Get(ctx, "org-1", e.ID)
	if got.Status != ruletypes.RemediationStatusExecuting || got.ApprovedBy != "alice@x" {
		t.Fatalf("unexpected post-state: %+v", got)
	}
}

// TestTransitionToExecuting_CapEnforced verifies that when cap=1 and one row is
// already executing, a second proposed row cannot be approved atomically.
func TestTransitionToExecuting_CapEnforced(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	e1 := sampleExecution("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	e2 := sampleExecution("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	require.NoError(t, s.Create(ctx, e1))
	require.NoError(t, s.Create(ctx, e2))

	// First approval wins (0 executing < cap 1).
	won1, err := s.TransitionToExecuting(ctx, "org-1", e1.ID, "alice@x", "2026-06-24T00:05:00Z", 1)
	require.NoError(t, err)
	require.True(t, won1, "first approve should win when cap=1 and 0 executing")

	// Second approval must lose: e1 is now executing, count=1 is not < cap 1.
	won2, err := s.TransitionToExecuting(ctx, "org-1", e2.ID, "bob@x", "2026-06-24T00:06:00Z", 1)
	require.NoError(t, err)
	require.False(t, won2, "second approve must lose when cap=1 and 1 already executing")

	// e2 must remain proposed.
	got, err := s.Get(ctx, "org-1", e2.ID)
	require.NoError(t, err)
	require.Equal(t, ruletypes.RemediationStatusProposed, got.Status)
}

func TestGetConfig_DefaultsWhenUnset(t *testing.T) {
	s := newTestStore(t)
	c, err := s.GetConfig(context.Background(), "org-unknown")
	if err != nil {
		t.Fatal(err)
	}
	if c.ExecutionEnabled || c.ProposalTTLSeconds != 1800 {
		t.Fatalf("unset org must yield safe defaults: %+v", c)
	}
}

func TestUpsertConfig_InsertThenReadBack(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	cfg := ruletypes.RemediationConfig{ExecutionEnabled: true, MaxConcurrent: 3}
	if err := s.UpsertConfig(ctx, "org-1", cfg); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetConfig(ctx, "org-1")
	if err != nil {
		t.Fatal(err)
	}
	if !got.ExecutionEnabled {
		t.Fatalf("executionEnabled must persist: %+v", got)
	}
	if got.MaxConcurrent != 3 {
		t.Fatalf("maxConcurrent must persist: %+v", got)
	}
	// Zeroed timing knobs must be backfilled with defaults, not stored as 0.
	if got.ProposalTTLSeconds != 1800 || got.ExecTimeoutSeconds != 300 {
		t.Fatalf("timing knobs must backfill to defaults: %+v", got)
	}
}

func TestUpsertConfig_OverwritesExisting(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if err := s.UpsertConfig(ctx, "org-1", ruletypes.RemediationConfig{ExecutionEnabled: true}); err != nil {
		t.Fatal(err)
	}
	// Second write flips the switch off — exercises the ON CONFLICT update path.
	if err := s.UpsertConfig(ctx, "org-1", ruletypes.RemediationConfig{ExecutionEnabled: false}); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetConfig(ctx, "org-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ExecutionEnabled {
		t.Fatalf("second upsert must turn executionEnabled off: %+v", got)
	}
}

func TestListByIncident(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	e1 := sampleExecution("33333333-3333-3333-3333-333333333331")
	e2 := sampleExecution("33333333-3333-3333-3333-333333333332")
	e2.IncidentID = "inc-2"

	require.NoError(t, s.Create(ctx, e1))
	require.NoError(t, s.Create(ctx, e2))

	list, err := s.ListByIncident(ctx, "org-1", "inc-1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, e1.ID, list[0].ID)
}

func TestListByStatus(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	e1 := sampleExecution("44444444-4444-4444-4444-444444444441")
	e2 := sampleExecution("44444444-4444-4444-4444-444444444442")

	require.NoError(t, s.Create(ctx, e1))
	require.NoError(t, s.Create(ctx, e2))

	list, err := s.ListByStatus(ctx, "org-1", ruletypes.RemediationStatusProposed)
	require.NoError(t, err)
	require.Len(t, list, 2)
}

func TestTransition_RejectFromProposed(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	e := sampleExecution("55555555-5555-5555-5555-555555555555")
	require.NoError(t, s.Create(ctx, e))

	err := s.Transition(ctx, "org-1", e.ID, ruletypes.RemediationStatusRejected, remediationstore.TransitionPatch{
		TerminalAt: "2026-06-24T00:10:00Z",
	})
	require.NoError(t, err)

	got, err := s.Get(ctx, "org-1", e.ID)
	require.NoError(t, err)
	require.Equal(t, ruletypes.RemediationStatusRejected, got.Status)
	require.Equal(t, "2026-06-24T00:10:00Z", got.TerminalAt)
}

func TestTransition_InvalidTransitionReturnsError(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	e := sampleExecution("66666666-6666-6666-6666-666666666666")
	require.NoError(t, s.Create(ctx, e))

	// proposed → succeeded is not allowed (must go via approved→executing first)
	err := s.Transition(ctx, "org-1", e.ID, ruletypes.RemediationStatusSucceeded, remediationstore.TransitionPatch{})
	require.Error(t, err)
}

func TestCountActiveByOrg(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	e1 := sampleExecution("77777777-7777-7777-7777-777777777771")
	e2 := sampleExecution("77777777-7777-7777-7777-777777777772")
	require.NoError(t, s.Create(ctx, e1))
	require.NoError(t, s.Create(ctx, e2))

	// Move e1 to executing via the atomic guard (cap=10 so it always fires).
	won, err := s.TransitionToExecuting(ctx, "org-1", e1.ID, "alice@x", "2026-06-24T00:05:00Z", 10)
	require.NoError(t, err)
	require.True(t, won)

	// Only e1 is executing; e2 is still proposed — count must be 1.
	count, err := s.CountActiveByOrg(ctx, "org-1")
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestTransition_FailsOnConcurrentChange(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Seed an execution that is already in 'succeeded' state by walking the
	// valid transition chain: proposed → executing → succeeded.
	e := sampleExecution("99999999-9999-9999-9999-999999999999")
	require.NoError(t, s.Create(ctx, e))

	won, err := s.TransitionToExecuting(ctx, "org-1", e.ID, "alice@x", "2026-06-24T00:05:00Z", 5)
	require.NoError(t, err)
	require.True(t, won)

	exitCode := 0
	require.NoError(t, s.Transition(ctx, "org-1", e.ID, ruletypes.RemediationStatusSucceeded, remediationstore.TransitionPatch{
		ExitCode:   &exitCode,
		ExecutedAt: "2026-06-24T00:06:00Z",
	}))

	// First concurrent writer moves succeeded → verified (out-of-band mutation).
	require.NoError(t, s.Transition(ctx, "org-1", e.ID, ruletypes.RemediationStatusVerified, remediationstore.TransitionPatch{
		TerminalAt: "2026-06-24T00:07:00Z",
	}))

	// Second writer (stale read) still thinks the row is 'succeeded' and tries
	// succeeded → unresolved. The guard must reject it because the row is now 'verified'.
	err = s.Transition(ctx, "org-1", e.ID, ruletypes.RemediationStatusUnresolved, remediationstore.TransitionPatch{
		TerminalAt: "2026-06-24T00:08:00Z",
	})
	require.Error(t, err, "Transition must return error when row status changed concurrently")

	// Row must remain 'verified' — the clobber was prevented.
	got, err := s.Get(ctx, "org-1", e.ID)
	require.NoError(t, err)
	require.Equal(t, ruletypes.RemediationStatusVerified, got.Status)
}

func TestExitCode_NullableRoundtrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	e := sampleExecution("88888888-8888-8888-8888-888888888888")
	require.NoError(t, s.Create(ctx, e))

	// proposed → executing via TransitionToExecuting (single-step guard)
	won, err := s.TransitionToExecuting(ctx, "org-1", e.ID, "alice@x", "2026-06-24T00:05:00Z", 5)
	require.NoError(t, err)
	require.True(t, won)

	// executing → succeeded, stamping exit code + output
	exitCode := 0
	err = s.Transition(ctx, "org-1", e.ID, ruletypes.RemediationStatusSucceeded, remediationstore.TransitionPatch{
		ExitCode:      &exitCode,
		OutputSnippet: "ok",
		ExecutedAt:    "2026-06-24T00:06:00Z",
	})
	require.NoError(t, err)

	got, err := s.Get(ctx, "org-1", e.ID)
	require.NoError(t, err)
	require.NotNil(t, got.ExitCode)
	require.Equal(t, 0, *got.ExitCode)
	require.Equal(t, "ok", got.OutputSnippet)
}
