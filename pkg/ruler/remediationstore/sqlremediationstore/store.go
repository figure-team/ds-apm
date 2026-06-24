package sqlremediationstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/SigNoz/signoz/pkg/ruler/remediationstore"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/uptrace/bun"
)

// remediationRow is the bun-scannable flat struct that mirrors the
// ds_remediation_execution DDL (migration 087). Column tags must match exactly.
type remediationRow struct {
	bun.BaseModel    `bun:"table:ds_remediation_execution"`
	ID               string        `bun:"id"`
	OrgID            string        `bun:"org_id"`
	IncidentID       string        `bun:"incident_id"`
	AlertFingerprint string        `bun:"alert_fingerprint"`
	SOPID            string        `bun:"sop_id"`
	SOPVersion       string        `bun:"sop_version"`
	RunbookID        string        `bun:"runbook_id"`
	ScriptSnapshot   string        `bun:"script_snapshot"`
	Status           string        `bun:"status"`
	ProposedAt       string        `bun:"proposed_at"`
	ApprovedAt       string        `bun:"approved_at"`
	ExecutedAt       string        `bun:"executed_at"`
	TerminalAt       string        `bun:"terminal_at"`
	ApprovedBy       string        `bun:"approved_by"`
	ExitCode         sql.NullInt64 `bun:"exit_code"`
	OutputSnippet    string        `bun:"output_snippet"`
	VerifyResult     string        `bun:"verify_result"`
	ExpiresAt        string        `bun:"expires_at"`
}

func rowFromDomain(e ruletypes.RemediationExecution) remediationRow {
	r := remediationRow{
		ID:               e.ID,
		OrgID:            e.OrgID,
		IncidentID:       e.IncidentID,
		AlertFingerprint: e.AlertFingerprint,
		SOPID:            e.SOPID,
		SOPVersion:       e.SOPVersion,
		RunbookID:        e.RunbookID,
		ScriptSnapshot:   e.ScriptSnapshot,
		Status:           e.Status,
		ProposedAt:       e.ProposedAt,
		ApprovedAt:       e.ApprovedAt,
		ExecutedAt:       e.ExecutedAt,
		TerminalAt:       e.TerminalAt,
		ApprovedBy:       e.ApprovedBy,
		OutputSnippet:    e.OutputSnippet,
		VerifyResult:     e.VerifyResult,
		ExpiresAt:        e.ExpiresAt,
	}
	if e.ExitCode != nil {
		r.ExitCode = sql.NullInt64{Int64: int64(*e.ExitCode), Valid: true}
	}
	return r
}

func (r remediationRow) toDomain() ruletypes.RemediationExecution {
	e := ruletypes.RemediationExecution{
		ID:               r.ID,
		OrgID:            r.OrgID,
		IncidentID:       r.IncidentID,
		AlertFingerprint: r.AlertFingerprint,
		SOPID:            r.SOPID,
		SOPVersion:       r.SOPVersion,
		RunbookID:        r.RunbookID,
		ScriptSnapshot:   r.ScriptSnapshot,
		Status:           r.Status,
		ProposedAt:       r.ProposedAt,
		ApprovedAt:       r.ApprovedAt,
		ExecutedAt:       r.ExecutedAt,
		TerminalAt:       r.TerminalAt,
		ApprovedBy:       r.ApprovedBy,
		OutputSnippet:    r.OutputSnippet,
		VerifyResult:     r.VerifyResult,
		ExpiresAt:        r.ExpiresAt,
	}
	if r.ExitCode.Valid {
		v := int(r.ExitCode.Int64)
		e.ExitCode = &v
	}
	return e
}

// configRow is the bun-scannable flat struct that mirrors ds_remediation_config.
type configRow struct {
	bun.BaseModel       `bun:"table:ds_remediation_config"`
	OrgID               string `bun:"org_id"`
	ExecutionEnabled    bool   `bun:"execution_enabled"`
	ProposalTTLSeconds  int64  `bun:"proposal_ttl_seconds"`
	ExecTimeoutSeconds  int64  `bun:"exec_timeout_seconds"`
	VerifyWindowSeconds int64  `bun:"verify_window_seconds"`
	MaxConcurrent       int64  `bun:"max_concurrent"`
}

// SQLStore is the bun-backed implementation of remediationstore.Store.
type SQLStore struct {
	sqlstore sqlstore.SQLStore
}

// New returns a *SQLStore backed by the given SQLStore.
// Migration 087 must have run; tables ds_remediation_execution and
// ds_remediation_config are accessed directly via bun ORM.
func New(s sqlstore.SQLStore) *SQLStore {
	return &SQLStore{sqlstore: s}
}

// Create validates and inserts a new RemediationExecution row.
func (s *SQLStore) Create(ctx context.Context, e ruletypes.RemediationExecution) error {
	if err := ruletypes.ValidateRemediationExecution(e); err != nil {
		return err
	}
	row := rowFromDomain(e)
	_, err := s.sqlstore.BunDB().NewInsert().
		Model(&row).
		Exec(ctx)
	return err
}

// Get returns the execution by org + primary key.
func (s *SQLStore) Get(ctx context.Context, orgID, id string) (ruletypes.RemediationExecution, error) {
	var row remediationRow
	err := s.sqlstore.BunDB().NewSelect().
		Model(&row).
		Where("id = ?", id).
		Where("org_id = ?", orgID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return ruletypes.RemediationExecution{}, sql.ErrNoRows
	}
	if err != nil {
		return ruletypes.RemediationExecution{}, err
	}
	return row.toDomain(), nil
}

// ListByIncident returns all executions for the given incident.
func (s *SQLStore) ListByIncident(ctx context.Context, orgID, incidentID string) ([]ruletypes.RemediationExecution, error) {
	var rows []remediationRow
	err := s.sqlstore.BunDB().NewSelect().
		Model(&rows).
		Where("org_id = ?", orgID).
		Where("incident_id = ?", incidentID).
		OrderExpr("proposed_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ruletypes.RemediationExecution, len(rows))
	for i, r := range rows {
		out[i] = r.toDomain()
	}
	return out, nil
}

// ListByStatus returns all executions with the given status for an org.
func (s *SQLStore) ListByStatus(ctx context.Context, orgID, status string) ([]ruletypes.RemediationExecution, error) {
	var rows []remediationRow
	err := s.sqlstore.BunDB().NewSelect().
		Model(&rows).
		Where("org_id = ?", orgID).
		Where("status = ?", status).
		OrderExpr("proposed_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ruletypes.RemediationExecution, len(rows))
	for i, r := range rows {
		out[i] = r.toDomain()
	}
	return out, nil
}

// TransitionToExecuting is the single-execution guard. It atomically moves a
// row from 'proposed' to 'executing' with a conditional UPDATE.
// Returns true iff exactly 1 row was affected (this caller won the race).
// A second concurrent call on the same row returns false, nil (row already
// moved out of 'proposed' by the first winner).
func (s *SQLStore) TransitionToExecuting(ctx context.Context, orgID, id, approvedBy, approvedAt string) (bool, error) {
	res, err := s.sqlstore.BunDB().NewUpdate().
		Model((*remediationRow)(nil)).
		Set("status = ?", ruletypes.RemediationStatusExecuting).
		Set("approved_by = ?", approvedBy).
		Set("approved_at = ?", approvedAt).
		Set("executed_at = ?", approvedAt).
		Where("id = ?", id).
		Where("org_id = ?", orgID).
		Where("status = ?", ruletypes.RemediationStatusProposed).
		Exec(ctx)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// Transition applies a validated status change (from current status to toStatus)
// and stamps result fields from patch. Only non-empty patch fields are written.
func (s *SQLStore) Transition(ctx context.Context, orgID, id, toStatus string, patch remediationstore.TransitionPatch) error {
	// Fetch current status to validate the transition.
	e, err := s.Get(ctx, orgID, id)
	if err != nil {
		return err
	}
	if err := ruletypes.ValidateRemediationStatusTransition(e.Status, toStatus); err != nil {
		return err
	}

	q := s.sqlstore.BunDB().NewUpdate().
		Model((*remediationRow)(nil)).
		Set("status = ?", toStatus).
		Where("id = ?", id).
		Where("org_id = ?", orgID).
		Where("status = ?", e.Status)

	if patch.TerminalAt != "" {
		q = q.Set("terminal_at = ?", patch.TerminalAt)
	}
	if patch.ExitCode != nil {
		q = q.Set("exit_code = ?", int64(*patch.ExitCode))
	}
	if patch.OutputSnippet != "" {
		q = q.Set("output_snippet = ?", patch.OutputSnippet)
	}
	if patch.VerifyResult != "" {
		q = q.Set("verify_result = ?", patch.VerifyResult)
	}
	if patch.ExecutedAt != "" {
		q = q.Set("executed_at = ?", patch.ExecutedAt)
	}

	res, err := q.Exec(ctx)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("remediationstore: transition %s→%s: row changed concurrently (id=%s)", e.Status, toStatus, id)
	}
	return nil
}

// CountActiveByOrg returns the number of executions in 'approved' or 'executing'
// status for the given org.
func (s *SQLStore) CountActiveByOrg(ctx context.Context, orgID string) (int64, error) {
	var count int64
	err := s.sqlstore.BunDB().NewSelect().
		Model((*remediationRow)(nil)).
		ColumnExpr("COUNT(*) AS count").
		Where("org_id = ?", orgID).
		Where("status IN (?, ?)", ruletypes.RemediationStatusApproved, ruletypes.RemediationStatusExecuting).
		Scan(ctx, &count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetConfig returns the per-org remediation config, backfilled with defaults.
// Returns DefaultRemediationConfig() when no row exists (safe defaults, never an error).
func (s *SQLStore) GetConfig(ctx context.Context, orgID string) (ruletypes.RemediationConfig, error) {
	var row configRow
	err := s.sqlstore.BunDB().NewSelect().
		Model(&row).
		Where("org_id = ?", orgID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return ruletypes.DefaultRemediationConfig(), nil
	}
	if err != nil {
		return ruletypes.DefaultRemediationConfig(), err
	}
	c := ruletypes.RemediationConfig{
		ExecutionEnabled:    row.ExecutionEnabled,
		ProposalTTLSeconds:  row.ProposalTTLSeconds,
		ExecTimeoutSeconds:  row.ExecTimeoutSeconds,
		VerifyWindowSeconds: row.VerifyWindowSeconds,
		MaxConcurrent:       row.MaxConcurrent,
	}
	return c.WithDefaults(), nil
}
