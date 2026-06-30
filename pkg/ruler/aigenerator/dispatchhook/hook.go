// Package dispatchhook implements the DS-APM AI-strategy dispatch hook
// as a standalone, callable component.
//
// The hook resolves the SOP bound to a SigNoz alert's labels, runs the
// injected AIStrategyGenerator with a tight timeout, and returns the
// alert's annotation map with the AI strategy + SOP metadata merged in.
// On any error (timeout, unbound SOP, generator failure) the input
// annotations are returned unchanged so the dispatcher's existing
// behavior is preserved — the hook is intentionally additive.
//
// This hook is wired into the dispatcher's notify path via
// pkg/alertmanager/alertmanagerserver/dispatcher.go (applyAIHook) and is
// constructed in pkg/signoz/signoz.go.
package dispatchhook

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// CodeRCATrigger is the CF-11 trigger seam (design §11): called fire-and-forget
// on the unbound branch. Implementations must never panic or block beyond their
// own internal timeout.
type CodeRCATrigger interface {
	Maybe(ctx context.Context, orgID string, labels, annotations map[string]string)
}

// RemediationProposer is the fail-open seam for human-gated auto-remediation.
// Implemented by remediation.Proposer (wrapped to also resolve per-org config).
// Apply calls it only on the Bound branch; a nil proposer or a (nil,false)
// return leaves annotations unchanged.
type RemediationProposer interface {
	MaybePropose(ctx context.Context, orgID, incidentID, alertFingerprint string, labels map[string]string, doc ruletypes.SOPDocument) (map[string]string, bool)
}

// RemediationSelector is the synchronous seam for LLM-backed runbook selection
// (design §3). Select runs inline on the dispatch path so its proposal — the
// approve-URL annotations — can ride the SAME outgoing notification. The alert
// is intentionally delayed (up to the selector's internal timeout) for SOPs that
// carry approved runbooks. Implementations must never panic; a (nil,false)
// return leaves annotations unchanged (fail-open).
type RemediationSelector interface {
	Select(ctx context.Context, orgID, incidentID, alertFingerprint string, labels map[string]string, doc ruletypes.SOPDocument) (map[string]string, bool)
}

// DefaultGenerateTimeout is the upper bound the hook imposes on the AI
// generator. The dispatcher must never block on a slow provider, so this
// is intentionally aggressive.
const DefaultGenerateTimeout = time.Second

// priorIncidentLimit caps how many past occurrences of the same failure the
// hook surfaces to the generator. Kept small so the prompt stays focused.
const priorIncidentLimit = 3

// Hook resolves an alert's SOP binding, runs the AIStrategyGenerator,
// and merges the resulting public annotations back into the alert. It
// owns no goroutines and holds no per-alert state, so a single instance
// can be shared across the dispatcher.
type Hook struct {
	sopStore       ruletypes.SOPStore
	aiHistoryStore ruletypes.AIStrategyHistoryStore
	generator      ruletypes.AIStrategyGenerator
	logger         *slog.Logger
	timeout        time.Duration
	codeRCA        CodeRCATrigger
	remediation    RemediationProposer
	selector       RemediationSelector
}

// SetCodeRCATrigger injects the CF-11 trigger after construction (the trigger
// depends on stores built later in server wiring). nil-safe; optional.
func (h *Hook) SetCodeRCATrigger(t CodeRCATrigger) { h.codeRCA = t }

// SetRemediationProposer injects the proposer after construction (optional,
// nil-safe). Mirrors SetCodeRCATrigger.
func (h *Hook) SetRemediationProposer(p RemediationProposer) { h.remediation = p }

// SetRemediationSelector injects the LLM-backed selector (optional, nil-safe).
// When set, it supersedes the static first-approved Proposer on the Bound
// branch for SOPs that carry at least one approved Runbook. The selector runs
// synchronously (see applyRemediation), so wiring it delays delivery of such
// alerts until the LLM proposal is ready or its timeout fires.
func (h *Hook) SetRemediationSelector(sel RemediationSelector) { h.selector = sel }

// New constructs a Hook. logger may be nil — the hook falls back to
// slog.Default() in that case. timeout ≤ 0 falls back to
// DefaultGenerateTimeout.
func New(
	sopStore ruletypes.SOPStore,
	aiHistoryStore ruletypes.AIStrategyHistoryStore,
	generator ruletypes.AIStrategyGenerator,
	logger *slog.Logger,
	timeout time.Duration,
) *Hook {
	if logger == nil {
		logger = slog.Default()
	}
	if timeout <= 0 {
		timeout = DefaultGenerateTimeout
	}
	return &Hook{
		sopStore:       sopStore,
		aiHistoryStore: aiHistoryStore,
		generator:      generator,
		logger:         logger.With(slog.String("component", "ds-apm-ai-dispatch-hook")),
		timeout:        timeout,
	}
}

// Apply runs the SOP-binding → AI-strategy pipeline against the supplied
// alert metadata and returns the (possibly augmented) annotations map.
//
// The original annotations map is never mutated; the returned map is a
// fresh copy (when augmented) or the same reference (when unchanged) so
// callers can compare identities cheaply.
//
// Failure modes — by design — never return an error; the dispatcher
// must keep delivering the alert even if AI generation is unavailable.
//   - empty orgID                 → unchanged
//   - sopStore.List failure       → unchanged (logged at warn)
//   - SOP binding not Bound       → unchanged
//   - generator error / timeout   → unchanged (logged at warn)
//
// On success the AI strategy and the SOP annotation keys defined in
// pkg/types/alertmanagertypes/incident.go are merged in, and a history
// record is best-effort upserted (failure is logged, not propagated).
func (h *Hook) Apply(
	ctx context.Context,
	orgID string,
	incidentID string,
	alertFingerprint string,
	labels map[string]string,
	annotations map[string]string,
) map[string]string {
	if orgID == "" {
		return annotations
	}
	if h.sopStore == nil || h.generator == nil {
		return annotations
	}

	docs, err := h.sopStore.List(ctx, orgID)
	if err != nil {
		h.logger.WarnContext(ctx, "ai dispatch hook: list SOPs failed",
			slog.String("orgId", orgID), slog.String("incidentId", incidentID),
			slog.Any("err", err))
		return annotations
	}

	binding, err := ruletypes.PreviewSOPDocumentBinding(docs, ruletypes.SOPBindingPreviewRequest{
		Labels:      labels,
		Annotations: annotations,
	})
	if err != nil || binding.Status != ruletypes.SOPBindingStatusBound {
		// Unbound, forbidden, disabled, or validation failure — all
		// non-error outcomes from the dispatcher's perspective. We
		// quietly return the input annotations so existing alerts
		// continue to flow untouched.
		if err == nil && binding.Status == ruletypes.SOPBindingStatusMissing && h.codeRCA != nil {
			// CF-11 (UJ-5): only genuinely unbound (no SOP matched) alerts go to
			// the code-RCA gate. The trigger is fail-open (never panics/blocks
			// past its own timeout), so it adds no failure mode to dispatch.
			h.codeRCA.Maybe(ctx, orgID, labels, annotations)
		}
		return annotations
	}

	// Re-fetch the full SOPDocument: PreviewSOPDocumentBinding returns
	// only the summary fields. The generator needs the body markdown.
	doc, err := h.sopStore.Get(ctx, orgID, binding.SOPID, binding.Version)
	if err != nil {
		h.logger.WarnContext(ctx, "ai dispatch hook: get SOP document failed",
			slog.String("orgId", orgID), slog.String("sopId", binding.SOPID),
			slog.String("sopVersion", binding.Version), slog.Any("err", err))
		return annotations
	}

	// Reference past occurrences of the same failure signature (same
	// alertFingerprint). Best-effort context: a lookup failure must never
	// block generation, so errors are logged and we proceed with no history.
	priorIncidents := h.lookupPriorIncidents(ctx, orgID, incidentID, alertFingerprint)

	// DS-APM cost guardrail: reuse a previously generated customer notice for
	// this incident instead of paying for another LLM call. Storms and
	// re-notifications of the SAME incident (same fingerprint) hit this path.
	// Reuse only when the stored strategy still matches the bound SOP version.
	//
	// Exception: never reuse a deterministic-local draft. That format is the
	// cheap non-LLM fallback (no LLM cost is saved by reusing it), and reusing
	// it would keep re-sending boilerplate even after a real LLM becomes
	// available. Regenerate so the LLM gets a chance to replace it.
	if h.aiHistoryStore != nil {
		if rec, ok, err := h.aiHistoryStore.GetLatest(ctx, orgID, ruletypes.AIStrategyHistoryLookupRequest{
			IncidentID:       incidentID,
			AlertFingerprint: alertFingerprint,
		}); err == nil && ok &&
			strings.TrimSpace(rec.Strategy.CustomerUpdateDraft) != "" &&
			strings.TrimSpace(rec.Strategy.SOPVersion) == strings.TrimSpace(binding.Version) &&
			!rec.Strategy.IsDeterministicLocal() {
			merged := h.mergeStrategyWithSOP(annotations, rec.Strategy, doc, binding)
			merged = h.applyRemediation(ctx, orgID, incidentID, alertFingerprint, labels, doc, merged)
			return merged
		}
	}

	genCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	strategy, err := h.generator.Generate(genCtx, ruletypes.AIStrategyRequest{
		OrgID:            orgID,
		IncidentID:       incidentID,
		AlertFingerprint: alertFingerprint,
		Labels:           labels,
		Annotations:      annotations,
		SOPDocument:      doc,
		PriorIncidents:   priorIncidents,
		// EvidenceRefs is intentionally empty for v0.1: the dispatch
		// path has no evidence collector yet. The generator is expected
		// to handle the empty case (deterministic local generator does).
		EvidenceRefs: nil,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "ai dispatch hook: generate failed",
			slog.String("orgId", orgID), slog.String("incidentId", incidentID),
			slog.String("sopId", binding.SOPID), slog.Any("err", err))
		return annotations
	}

	merged := h.mergeStrategyWithSOP(annotations, strategy, doc, binding)

	if h.aiHistoryStore != nil {
		record, recErr := ruletypes.NewAIStrategyHistoryRecord(strategy)
		if recErr != nil {
			h.logger.WarnContext(ctx, "ai dispatch hook: build history record failed",
				slog.String("orgId", orgID), slog.String("incidentId", incidentID),
				slog.Any("err", recErr))
		} else if upsertErr := h.aiHistoryStore.Upsert(ctx, orgID, record); upsertErr != nil {
			h.logger.WarnContext(ctx, "ai dispatch hook: persist history record failed",
				slog.String("orgId", orgID), slog.String("incidentId", incidentID),
				slog.String("strategyId", strategy.StrategyID),
				slog.Any("err", upsertErr))
		}
	}

	merged = h.applyRemediation(ctx, orgID, incidentID, alertFingerprint, labels, doc, merged)

	return merged
}

// lookupPriorIncidents returns up to priorIncidentLimit past occurrences of the
// same failure signature (matched by alertFingerprint), excluding the current
// incident. It is best-effort: an empty fingerprint, nil store, or store error
// yields no history rather than failing the dispatch.
func (h *Hook) lookupPriorIncidents(ctx context.Context, orgID, incidentID, alertFingerprint string) []ruletypes.AIPriorIncident {
	if h.aiHistoryStore == nil || strings.TrimSpace(alertFingerprint) == "" {
		return nil
	}

	// Fetch one extra so excluding the current incident still leaves up to
	// priorIncidentLimit prior occurrences.
	records, err := h.aiHistoryStore.ListRecent(ctx, orgID,
		ruletypes.AIStrategyHistoryLookupRequest{AlertFingerprint: alertFingerprint},
		priorIncidentLimit+1)
	if err != nil {
		h.logger.WarnContext(ctx, "ai dispatch hook: list prior incidents failed",
			slog.String("orgId", orgID), slog.String("incidentId", incidentID),
			slog.Any("err", err))
		return nil
	}

	priors := make([]ruletypes.AIPriorIncident, 0, priorIncidentLimit)
	for _, record := range records {
		if strings.TrimSpace(record.IncidentID) == strings.TrimSpace(incidentID) {
			continue // the current incident is not its own prior occurrence
		}
		priors = append(priors, ruletypes.AIPriorIncidentFromHistoryRecord(record))
		if len(priors) == priorIncidentLimit {
			break
		}
	}
	return priors
}

// mergeStrategyWithSOP overlays the AI strategy annotations and the bound SOP's
// public metadata onto base, returning a fresh map. Shared by the generate path
// and the cache-reuse path so both emit identical annotations.
func (h *Hook) mergeStrategyWithSOP(base map[string]string, strategy ruletypes.AIStrategy, doc ruletypes.SOPDocument, binding ruletypes.SOPBindingPreviewResponse) map[string]string {
	merged := mergeAnnotations(base, ruletypes.AIStrategyIncidentAnnotations(strategy))
	// Public SOP metadata sourced from the bound document. These are
	// the keys consumed by alertmanagertypes.BuildIncidentInfo / the
	// Slack and webhook templates.
	if doc.DisplayURL != "" {
		merged[alertmanagertypes.IncidentAnnotationSopURL] = doc.DisplayURL
	}
	if doc.Source.SourceID != "" {
		merged[alertmanagertypes.IncidentAnnotationSopSource] = doc.Source.SourceID
	}
	if binding.Title != "" {
		merged[alertmanagertypes.IncidentAnnotationSopTitle] = binding.Title
	}
	if binding.Version != "" {
		merged[alertmanagertypes.IncidentAnnotationSopVersion] = binding.Version
	}
	if binding.Resolution != "" {
		merged[alertmanagertypes.IncidentAnnotationSopBindingID] = binding.Resolution
	}
	return merged
}

// applyRemediation routes to the LLM selector when wired (and the SOP has at
// least one approved runbook), otherwise to the static first-approved proposer.
// The selector runs synchronously: the dispatch waits for its proposal so the
// approve-URL annotations ride THIS notification (the alert is delayed up to the
// selector's internal timeout). A (nil,false) return — gate not met, LLM
// timeout, or no actionable decision — leaves annotations unchanged (fail-open).
func (h *Hook) applyRemediation(ctx context.Context, orgID, incidentID, alertFingerprint string, labels map[string]string, doc ruletypes.SOPDocument, merged map[string]string) map[string]string {
	if h.selector != nil && hasApprovedRunbook(doc) {
		if selAnn, ok := h.selector.Select(ctx, orgID, incidentID, alertFingerprint, labels, doc); ok && len(selAnn) > 0 {
			return mergeAnnotations(merged, selAnn)
		}
		return merged
	}
	if h.remediation != nil {
		if remAnn, ok := h.remediation.MaybePropose(ctx, orgID, incidentID, alertFingerprint, labels, doc); ok && len(remAnn) > 0 {
			return mergeAnnotations(merged, remAnn)
		}
	}
	return merged
}

func hasApprovedRunbook(doc ruletypes.SOPDocument) bool {
	for _, rb := range doc.Runbooks {
		if rb.Status == ruletypes.RunbookStatusApproved && strings.TrimSpace(rb.ExecutableScript) != "" {
			return true
		}
	}
	return false
}

// mergeAnnotations returns a fresh map containing base overlaid with
// overrides. base is never mutated; nil entries are tolerated.
func mergeAnnotations(base map[string]string, overrides map[string]string) map[string]string {
	merged := make(map[string]string, len(base)+len(overrides))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range overrides {
		if v == "" {
			continue
		}
		merged[k] = v
	}
	return merged
}
