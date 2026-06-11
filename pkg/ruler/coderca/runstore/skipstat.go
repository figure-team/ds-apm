package runstore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
)

// bumpSkipStat increments the aggregated per-(org, reason, day) skip counter.
// One row per (org, reason, day) — under a flood this is a repeated UPDATE of a
// single row, not one row per rejected alert (design §6.4). Uses BunDBCtx so it
// joins an in-flight transaction (e.g. inside Admit) or runs standalone.
func (s *Store) bumpSkipStat(ctx context.Context, orgID string, reason coderca.SkipReason, day string) error {
	_, err := s.sqlstore.BunDBCtx(ctx).ExecContext(ctx,
		`INSERT INTO coderca_skip_stat (org_id, reason, day, count) VALUES (?, ?, ?, 1)
		 ON CONFLICT (org_id, reason, day) DO UPDATE SET count = count + 1`,
		orgID, string(reason), day,
	)
	return err
}

// RecordSkip records a skip decided outside Admit (trigger-gate reasons such as
// feature_off / no_anomaly / below_severity / no_repo_mapping). Single
// statement; no explicit transaction needed.
func (s *Store) RecordSkip(ctx context.Context, orgID string, reason coderca.SkipReason, now time.Time) error {
	return s.bumpSkipStat(ctx, orgID, reason, now.UTC().Format("2006-01-02"))
}

// SkipStat reads the aggregated skip count for (org, reason, day); 0 if absent.
func (s *Store) SkipStat(ctx context.Context, orgID string, reason coderca.SkipReason, day string) (int, error) {
	var count int
	err := s.sqlstore.BunDBCtx(ctx).NewRaw(
		"SELECT count FROM coderca_skip_stat WHERE org_id = ? AND reason = ? AND day = ?",
		orgID, string(reason), day,
	).Scan(ctx, &count)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}
