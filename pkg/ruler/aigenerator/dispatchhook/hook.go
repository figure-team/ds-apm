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
// This package is not yet wired into pkg/alertmanager/alertmanagerserver.
// The v0.1 demo invokes the hook via the existing PreviewAIStrategy
// HTTP endpoint (which already calls the generator); a future task will
// thread the hook into the dispatcher's notify path.
package dispatchhook

import (
	"context"
	"log/slog"
	"time"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// DefaultGenerateTimeout is the upper bound the hook imposes on the AI
// generator. The dispatcher must never block on a slow provider, so this
// is intentionally aggressive.
const DefaultGenerateTimeout = time.Second

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
}

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

	genCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	strategy, err := h.generator.Generate(genCtx, ruletypes.AIStrategyRequest{
		IncidentID:       incidentID,
		AlertFingerprint: alertFingerprint,
		Labels:           labels,
		Annotations:      annotations,
		SOPDocument:      doc,
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

	merged := mergeAnnotations(annotations, ruletypes.AIStrategyIncidentAnnotations(strategy))
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

	return merged
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
