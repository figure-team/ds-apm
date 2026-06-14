package runstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
)

// ErrRunNotFound is returned by GetRun for a missing or other-org run.
var ErrRunNotFound = errors.New("coderca run not found")

// RunSummary is one row of the run-history list (no report body).
type RunSummary struct {
	RunID          string            `json:"runId"`
	OrgID          string            `json:"orgId"`
	Service        string            `json:"service"`
	Status         coderca.RunStatus `json:"status"`
	BaselineCommit string            `json:"baselineCommit"`
	CreatedAt      int64             `json:"createdAt"`  // unix seconds
	FinishedAt     int64             `json:"finishedAt"` // 0 = not finished
	Attempts       int               `json:"attempts"`
	ResultRef      string            `json:"resultRef"`
}

// RunDetail is a run with its persisted RCA report.
type RunDetail struct {
	RunSummary
	RootCause   string `json:"rootCause"`
	ProposedFix string `json:"proposedFix"`
	Confidence  string `json:"confidence"`
	Limitations string `json:"limitations"`
}

// ListRunsParams filters the run-history list. Zero values = no filter.
type ListRunsParams struct {
	Status  string
	Service string
	Limit   int // default 50, max 200
	Offset  int
}

// ListRuns returns the org's runs, newest first.
func (s *Store) ListRuns(ctx context.Context, orgID string, p ListRunsParams) ([]RunSummary, error) {
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	q := `SELECT run_id, org_id, service, status, baseline_commit, created_at, finished_at, attempts, result_ref
	      FROM coderca_run WHERE org_id = ?`
	args := []interface{}{orgID}
	if p.Status != "" {
		q += " AND status = ?"
		args = append(args, p.Status)
	}
	if p.Service != "" {
		q += " AND service = ?"
		args = append(args, p.Service)
	}
	q += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, p.Offset)

	rows, err := s.sqlstore.BunDBCtx(ctx).QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	out := make([]RunSummary, 0, limit)
	for rows.Next() {
		var r RunSummary
		var status string
		if err := rows.Scan(&r.RunID, &r.OrgID, &r.Service, &status, &r.BaselineCommit,
			&r.CreatedAt, &r.FinishedAt, &r.Attempts, &r.ResultRef); err != nil {
			return nil, err
		}
		r.Status = coderca.RunStatus(status)
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetRun returns one run with its report. Tenant-isolated: a run belonging to
// another org returns ErrRunNotFound (existence is not leaked).
func (s *Store) GetRun(ctx context.Context, orgID, runID string) (RunDetail, error) {
	var d RunDetail
	var status string
	err := s.sqlstore.BunDBCtx(ctx).QueryRowContext(ctx,
		`SELECT run_id, org_id, service, status, baseline_commit, created_at, finished_at, attempts, result_ref,
		        root_cause, proposed_fix, confidence, limitations
		 FROM coderca_run WHERE org_id = ? AND run_id = ?`,
		orgID, runID,
	).Scan(&d.RunID, &d.OrgID, &d.Service, &status, &d.BaselineCommit,
		&d.CreatedAt, &d.FinishedAt, &d.Attempts, &d.ResultRef,
		&d.RootCause, &d.ProposedFix, &d.Confidence, &d.Limitations)
	if errors.Is(err, sql.ErrNoRows) {
		return RunDetail{}, ErrRunNotFound
	}
	if err != nil {
		return RunDetail{}, err
	}
	d.Status = coderca.RunStatus(status)
	return d, nil
}
