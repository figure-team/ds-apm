package remediation

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/SigNoz/signoz/pkg/ruler/cliaudit"
	"github.com/SigNoz/signoz/pkg/ruler/remediationstore"
	"github.com/SigNoz/signoz/pkg/ruler/remediationtargetstore"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// LLMProvider is the minimal completion seam (mirrors llmaigenerator.Provider)
// kept local so this package does not import the generator package.
type LLMProvider interface {
	Complete(ctx context.Context, system, user string) (string, error)
}

// ProviderResolver resolves the per-org LLM provider + model. The production
// implementation lives in package aigenerator (store-aware, reusing the same
// AIConfig the rest of the product uses).
type ProviderResolver interface {
	Resolve(ctx context.Context, orgID string) (LLMProvider, string, error)
}

// Selector picks the best approved Runbook for an incident (or proposes a
// fallback script) using a per-org LLM, then creates a proposed
// RemediationExecution. Every failure path is fail-open (returns nil,false) so
// it never blocks alert delivery — it runs off the dispatch path entirely.
type Selector struct {
	store       remediationstore.Store
	targetStore remediationtargetstore.Store // nil = 로컬 전용
	resolver    ProviderResolver
	baseURL     string
	timeout     time.Duration
	now         func() time.Time
	logger      *slog.Logger
}

// NewSelector constructs a Selector. now/logger may be nil.
// targetStore may be nil (local-only mode — no target resolution/freeze).
func NewSelector(store remediationstore.Store, targetStore remediationtargetstore.Store, resolver ProviderResolver, baseURL string, timeout time.Duration, now func() time.Time, logger *slog.Logger) *Selector {
	if now == nil {
		now = time.Now
	}
	if logger == nil {
		logger = slog.Default()
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Selector{
		store:       store,
		targetStore: targetStore,
		resolver:    resolver,
		baseURL:     strings.TrimRight(baseURL, "/"),
		timeout:     timeout,
		now:         now,
		logger:      logger.With(slog.String("component", "ds-apm-remediation-selector")),
	}
}

// Select runs the full decision pipeline synchronously. The trigger (dispatch
// hook) wraps it in a fire-and-forget goroutine with panic recovery.
func (s *Selector) Select(
	ctx context.Context,
	orgID, incidentID, alertFingerprint string,
	labels map[string]string,
	doc ruletypes.SOPDocument,
) (map[string]string, bool) {
	if s == nil || s.store == nil || s.resolver == nil {
		return nil, false
	}

	cfg, err := s.store.GetConfig(ctx, orgID)
	if err != nil || !cfg.ExecutionEnabled {
		return nil, false
	}

	// Collect approved runbooks with a non-empty script.
	approved := approvedRunbooks(doc)
	if len(approved) == 0 {
		return nil, false // nothing to select; legacy 0-runbook behaviour
	}

	// Cost guardrail: reuse a still-active proposal for the same failure
	// signature instead of paying for another LLM call.
	// Note: a store error here does NOT short-circuit — it deliberately falls
	// through to attempt selection. A transient cache-read failure must not
	// permanently suppress proposals; the system remains fail-open overall.
	if active, err := s.store.ListActiveByFingerprint(ctx, orgID, alertFingerprint); err == nil && len(active) > 0 {
		return nil, false
	}

	provider, model, err := s.resolver.Resolve(ctx, orgID)
	if err != nil || provider == nil {
		s.logger.WarnContext(ctx, "selector: resolve provider failed", slog.String("orgId", orgID), slog.Any("err", err))
		return nil, false
	}

	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	system, user := RenderSelectionPrompt(SelectionInput{
		OrgID:            orgID,
		IncidentID:       incidentID,
		AlertFingerprint: alertFingerprint,
		Labels:           labels,
		SOP:              doc,
		Runbooks:         approved,
	})

	start := s.now()
	raw, err := provider.Complete(callCtx, system, user)
	rec := cliaudit.Record{
		Via:         "remediation-select",
		Model:       model,
		Source:      "", // selection call; execution source tagged later
		Fingerprint: alertFingerprint,
		DurationMS:  s.now().Sub(start).Milliseconds(),
		Outcome:     "ok",
	}
	if err != nil {
		rec.Outcome = "failed"
		rec.Err = truncate(strings.TrimSpace(err.Error()), 256)
		cliaudit.Default().Log(rec)
		s.logger.WarnContext(ctx, "selector: llm complete failed", slog.String("orgId", orgID), slog.Any("err", err))
		return nil, false
	}
	rec.OutputBytes = len(raw)
	cliaudit.Default().Log(rec)

	ids := approvedRunbookIDSet(approved)
	decision, err := ParseSelectionResponse(raw, ids)
	if err != nil || !decision.IsActionable() {
		if err != nil {
			s.logger.WarnContext(ctx, "selector: parse failed", slog.String("orgId", orgID), slog.Any("err", err))
		}
		return nil, false
	}

	return s.createExecution(ctx, orgID, incidentID, alertFingerprint, labels, doc, approved, decision, cfg)
}

// createExecution snapshots the chosen/fallback script into a proposed
// RemediationExecution and returns the approval-card annotations.
func (s *Selector) createExecution(
	ctx context.Context,
	orgID, incidentID, alertFingerprint string,
	labels map[string]string,
	doc ruletypes.SOPDocument,
	approved []ruletypes.Runbook,
	decision ruletypes.RunbookSelectionDecision,
	cfg ruletypes.RemediationConfig,
) (map[string]string, bool) {
	var (
		runbookID string
		script    string
		source    string
		summary   string
	)
	switch decision.Outcome {
	case ruletypes.RunbookSelectionOutcomeSelected:
		rb, ok := findRunbook(approved, decision.ChosenRunbookID)
		if !ok {
			return nil, false
		}
		runbookID = rb.ID
		script = rb.ExecutableScript
		source = ruletypes.RemediationSourceRunbook
		summary = strings.TrimSpace(rb.Title)
	case ruletypes.RunbookSelectionOutcomeFallback:
		// 정적 게이트 (fail-closed): 명백히 파괴적인 LLM 스크립트는 승인 카드에
		// 올리지 않는다. Select 전체의 fail-open 계약(알림 비차단)은 유지 —
		// 여기서의 거부는 "제안 없음"일 뿐 알림 흐름을 막지 않는다.
		if gateErr := CheckLLMScript(decision.FallbackScript); gateErr != nil {
			s.logger.WarnContext(ctx, "selector: fallback script blocked by static gate",
				slog.String("orgId", orgID), slog.Any("err", gateErr))
			return nil, false
		}
		runbookID = "" // no backing runbook
		script = decision.FallbackScript
		source = ruletypes.RemediationSourceLLMGenerated
		summary = strings.TrimSpace(decision.FallbackSummary)
	default:
		return nil, false
	}

	now := s.now().UTC()
	e := ruletypes.RemediationExecution{
		ID:                 uuid.NewString(),
		OrgID:              orgID,
		IncidentID:         incidentID,
		AlertFingerprint:   alertFingerprint,
		SOPID:              doc.SOPID,
		SOPVersion:         doc.Version,
		RunbookID:          runbookID,
		ScriptSnapshot:     script, // verbatim snapshot — safety invariant
		Status:             ruletypes.RemediationStatusProposed,
		Source:             source,
		SelectionRationale: truncate(strings.TrimSpace(decision.Rationale), ruletypes.RunbookMaxDescriptionLen),
		ProposedAt:         now.Format(time.RFC3339),
		ExpiresAt:          now.Add(time.Duration(cfg.ProposalTTLSeconds) * time.Second).Format(time.RFC3339),
	}
	freezeTargetSnapshot(ctx, s.targetStore, orgID, labels, &e)
	if err := s.store.Create(ctx, e); err != nil {
		s.logger.WarnContext(ctx, "selector: create execution failed", slog.String("orgId", orgID), slog.Any("err", err))
		return nil, false
	}

	if summary == "" {
		summary = "자동 대응 스크립트"
	}
	ann := map[string]string{
		alertmanagertypes.IncidentAnnotationRemediationID:            e.ID,
		alertmanagertypes.IncidentAnnotationRemediationScriptSummary: summary + " (승인 시 웹 UI에서 실행)",
		alertmanagertypes.IncidentAnnotationRemediationApproveURL:    s.approveURL(e.ID),
	}
	return ann, true
}

// approveURL mirrors Proposer.approveURL (same deep link contract: the
// chromeless /remediation/approve/:id page).
func (s *Selector) approveURL(remediationID string) string {
	p := &Proposer{baseURL: s.baseURL}
	return p.approveURL(remediationID)
}

func approvedRunbooks(doc ruletypes.SOPDocument) []ruletypes.Runbook {
	out := make([]ruletypes.Runbook, 0, len(doc.Runbooks))
	for _, rb := range doc.Runbooks {
		if rb.Status == ruletypes.RunbookStatusApproved && strings.TrimSpace(rb.ExecutableScript) != "" {
			out = append(out, rb)
		}
	}
	return out
}

func approvedRunbookIDSet(rbs []ruletypes.Runbook) map[string]struct{} {
	m := make(map[string]struct{}, len(rbs))
	for _, rb := range rbs {
		m[rb.ID] = struct{}{}
	}
	return m
}

func findRunbook(rbs []ruletypes.Runbook, id string) (ruletypes.Runbook, bool) {
	for _, rb := range rbs {
		if rb.ID == id {
			return rb, true
		}
	}
	return ruletypes.Runbook{}, false
}
