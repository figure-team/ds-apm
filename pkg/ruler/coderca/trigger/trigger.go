// Package trigger is the CF-11 dispatch-side gate (design §10): a fire-and-
// forget facade the dispatch hook calls on the unbound branch. It never
// returns an error and never panics — any failure inside the gate must leave
// the alert path untouched (FR-CF11.6).
package trigger

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// maybeTimeout bounds the trigger's worst-case dispatch-path cost: the cheap
// pre-checks are in-memory; the only DB work is one config Get, one map Get,
// and the single Admit transaction.
const maybeTimeout = time.Second

// AnomalyLabel is the v1 anomaly signal: an explicit label/annotation on the
// alert (design §10, fail-closed). CF-7's anomaly rule stamps it at firing.
const AnomalyLabel = "anomaly"

// Admitter is the runstore admission port (satisfied by *runstore.Store).
type Admitter interface {
	Admit(ctx context.Context, p runstore.AdmitParams) (runstore.AdmitResult, error)
	RecordSkip(ctx context.Context, orgID string, reason coderca.SkipReason, now time.Time) error
}

// Trigger gates and enqueues code-RCA runs from the dispatch path.
type Trigger struct {
	cfgs   ruletypes.CodebaseRCAConfigStore
	maps   ruletypes.CodebaseServiceMapStore
	runs   Admitter
	logger *slog.Logger
	now    func() time.Time
}

// New builds a Trigger. logger may be nil (slog.Default); now may be nil.
func New(cfgs ruletypes.CodebaseRCAConfigStore, maps ruletypes.CodebaseServiceMapStore, runs Admitter, logger *slog.Logger, now func() time.Time) *Trigger {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return &Trigger{cfgs: cfgs, maps: maps, runs: runs, logger: logger.With(slog.String("component", "ds-apm-coderca-trigger")), now: now}
}

// Maybe evaluates the gate chain for an unbound alert and, when every gate
// passes, atomically admits a queued run. It NEVER returns an error or panics;
// it returns quickly (bounded by maybeTimeout). Gate order (design §6.1/§10):
// feature_on → anomaly(fail-closed) → severity → service→repo → Admit.
func (t *Trigger) Maybe(ctx context.Context, orgID string, labels, annotations map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			t.logger.ErrorContext(ctx, "coderca trigger: recovered panic", slog.Any("panic", r))
		}
	}()
	if t == nil || orgID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, maybeTimeout)
	defer cancel()

	cfg, err := t.cfgs.Get(ctx, orgID)
	if err != nil {
		// not-found → 기본값(Enabled=false) = feature off. 그 외 에러 → fail-closed.
		return
	}
	if !cfg.Enabled {
		return // feature_off: DB 기록 없음 (§6.1 — pre-check는 DB 미접촉)
	}

	if !anomalous(labels, annotations) {
		if cfg.AllowUnboundWithoutAnomaly {
			t.logger.WarnContext(ctx, "coderca trigger: admitting WITHOUT anomaly signal (allow_unbound_without_anomaly is ON)", slog.String("orgId", orgID))
		} else {
			return // fail-closed (§10)
		}
	}

	if !ruletypes.SeverityAtLeast(labels["severity"], cfg.MinSeverity) {
		return
	}

	service := strings.TrimSpace(labels["service.name"])
	if service == "" {
		return
	}
	if _, err := t.maps.Get(ctx, orgID, service); err != nil {
		// 미매핑(또는 조회 실패) → skip. no_repo_mapping만 집계 (§6.4).
		_ = t.runs.RecordSkip(ctx, orgID, coderca.SkipNoRepoMapping, t.now())
		return
	}

	sig := coderca.ErrorSignature(labels)
	res, err := t.runs.Admit(ctx, runstore.AdmitParams{
		OrgID:          orgID,
		Service:        service,
		DedupKey:       coderca.DedupKey(orgID, service, sig),
		Now:            t.now(),
		CooldownWindow: time.Duration(cfg.CooldownWindowSecs) * time.Second,
		MaxRunsPerDay:  cfg.MaxRunsPerDay,
		MaxQueueDepth:  cfg.MaxQueueDepth,
	})
	if err != nil {
		t.logger.WarnContext(ctx, "coderca trigger: admit failed", slog.String("orgId", orgID), slog.Any("err", err))
		return
	}
	if res.Admitted {
		t.logger.InfoContext(ctx, "coderca trigger: run queued", slog.String("orgId", orgID), slog.String("service", service), slog.String("runId", res.RunID))
	}
}

// anomalous reports whether the alert carries the explicit anomaly signal.
func anomalous(labels, annotations map[string]string) bool {
	for _, m := range []map[string]string{labels, annotations} {
		v := strings.ToLower(strings.TrimSpace(m[AnomalyLabel]))
		if v == "true" || v == "1" {
			return true
		}
	}
	return false
}
