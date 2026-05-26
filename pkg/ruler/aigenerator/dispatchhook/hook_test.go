package dispatchhook

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/ruler/aihistorystore/aihistorystoretest"
	"github.com/SigNoz/signoz/pkg/ruler/sopstore/sopstoretest"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// stubGen returns a canned strategy without running the deterministic
// local generator, so error / timeout paths are reproducible.
type stubGen struct {
	strategy ruletypes.AIStrategy
	err      error
	delay    time.Duration
	calls    int
}

func (s *stubGen) Generate(ctx context.Context, _ ruletypes.AIStrategyRequest) (ruletypes.AIStrategy, error) {
	s.calls++
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return ruletypes.AIStrategy{}, ctx.Err()
		}
	}
	if s.err != nil {
		return ruletypes.AIStrategy{}, s.err
	}
	return s.strategy, nil
}

// dispatchSeed is a trimmed view of the v0.1 demo seed used to build a
// realistic SOP document + alert payload for the hook tests.
type dispatchSeed struct {
	SOPDocument  ruletypes.SOPDocument       `json:"sopDocument"`
	Alert        dispatchSeedAlert           `json:"alert"`
	EvidenceRefs []ruletypes.AIEvidenceRef   `json:"evidenceRefs"`
}

type dispatchSeedAlert struct {
	IncidentID  string            `json:"incidentId"`
	Fingerprint string            `json:"fingerprint"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

func loadSeed(t *testing.T) dispatchSeed {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// Reuse the existing demo seed shipped under
	// pkg/types/ruletypes/testdata/ds_ai_sop_demo_seed.json — keeping
	// a single fixture avoids drift.
	seedPath := filepath.Join(
		filepath.Dir(thisFile),
		"..", "..", "..",
		"types", "ruletypes", "testdata", "ds_ai_sop_demo_seed.json",
	)
	raw, err := os.ReadFile(seedPath)
	require.NoError(t, err)
	var seed dispatchSeed
	require.NoError(t, json.Unmarshal(raw, &seed))
	return seed
}

// seedHookFixture returns a Hook + its underlying fakes pre-loaded with
// the demo SOP for orgID. The caller can swap the generator.
func seedHookFixture(t *testing.T, orgID string, gen ruletypes.AIStrategyGenerator) (*Hook, *sopstoretest.Fake, *aihistorystoretest.Fake, dispatchSeed) {
	t.Helper()
	seed := loadSeed(t)

	sops := sopstoretest.New()
	require.NoError(t, sops.Upsert(context.Background(), orgID, seed.SOPDocument))

	hist := aihistorystoretest.New()

	hook := New(sops, hist, gen, nil, time.Second)
	return hook, sops, hist, seed
}

func TestApply_BoundSOPMergesAIAnnotationsAndPersistsHistory(t *testing.T) {
	const orgID = "customer-a"
	const headline = "Payment API 5xx SOP-grounded headline"
	const firstAction = "결제 성공률 dashboard 확인"

	gen := &stubGen{strategy: ruletypes.AIStrategy{
		ContractVersion:  ruletypes.AIStrategyContractVersion,
		StrategyID:       "strategy-test-001",
		IncidentID:       "INC-20260512-0001", // must match seed.Alert.IncidentID
		AlertFingerprint: "fp-payment-api-5xx-demo",
		Status:           ruletypes.AIStrategyStatusReady,
		Language:         "ko-KR",
		SOPID:            "SOP-PAY-001",
		SOPVersion:       "2026-05-12.1",
		Headline:         headline,
		Hypotheses: []ruletypes.AIHypothesis{
			{
				Rank:         1,
				Text:         "PG timeout가 결제 승인 큐 적체를 유발",
				Confidence:   ruletypes.AIConfidenceMedium,
				EvidenceRefs: []string{"metric:error_rate:payment-api"},
				SOPStepRefs:  []string{"step-1"},
			},
		},
		FirstActions: []ruletypes.AIFirstAction{
			{
				Text:                  firstAction,
				SOPStepRef:            "step-1",
				EvidenceRefs:          []string{"metric:error_rate:payment-api"},
				RequiresHumanApproval: true,
			},
		},
		EvidenceRefs: []ruletypes.AIEvidenceRef{
			{
				RefID:       "metric:error_rate:payment-api",
				Type:        "metric",
				Observation: "5xx rate rose from 0.2% to 12%",
				Confidence:  ruletypes.AIConfidenceHigh,
			},
		},
		Confidence: ruletypes.AIConfidenceMedium,
		Audit: ruletypes.AIStrategyAudit{
			PromptVersion:    "ds-ir-ko-v1",
			Model:            "stub-test",
			GeneratedAt:      "2026-05-12T00:00:00Z",
			RedactionApplied: true,
		},
	}}

	hook, _, hist, seed := seedHookFixture(t, orgID, gen)

	got := hook.Apply(
		context.Background(),
		orgID,
		seed.Alert.IncidentID,
		seed.Alert.Fingerprint,
		seed.Alert.Labels,
		seed.Alert.Annotations,
	)

	require.Equal(t, 1, gen.calls)
	require.Equal(t, "strategy-test-001", got[alertmanagertypes.IncidentAnnotationAIStrategyID])
	require.Equal(t, ruletypes.AIStrategyStatusReady, got[alertmanagertypes.IncidentAnnotationAIStrategyStatus])
	require.Equal(t, headline, got[alertmanagertypes.IncidentAnnotationAIHeadline])
	require.Equal(t, firstAction, got[alertmanagertypes.IncidentAnnotationAIFirstActions])
	require.Equal(t, ruletypes.AIConfidenceMedium, got[alertmanagertypes.IncidentAnnotationAIConfidence])

	// SOP metadata from the bound document.
	require.Equal(t, seed.SOPDocument.DisplayURL, got[alertmanagertypes.IncidentAnnotationSopURL])
	require.Equal(t, seed.SOPDocument.Source.SourceID, got[alertmanagertypes.IncidentAnnotationSopSource])
	require.Equal(t, seed.SOPDocument.Title, got[alertmanagertypes.IncidentAnnotationSopTitle])
	require.Equal(t, seed.SOPDocument.Version, got[alertmanagertypes.IncidentAnnotationSopVersion])
	require.Equal(t, ruletypes.SOPBindingResolutionExplicitLabel, got[alertmanagertypes.IncidentAnnotationSopBindingID])

	// Original annotations still present and untouched.
	for k, v := range seed.Alert.Annotations {
		require.Equal(t, v, got[k], "input annotation %q must survive merge", k)
	}
	// Original input map must not have been mutated.
	require.NotContains(t, seed.Alert.Annotations, alertmanagertypes.IncidentAnnotationAIStrategyID,
		"hook must not mutate the input annotation map")

	// History persisted for the org.
	rec, ok, err := hist.GetLatest(context.Background(), orgID, ruletypes.AIStrategyHistoryLookupRequest{
		IncidentID: seed.Alert.IncidentID,
	})
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, seed.Alert.IncidentID, rec.IncidentID)
	require.Equal(t, ruletypes.AIStrategyStatusReady, rec.Status)
	require.Equal(t, "strategy-test-001", rec.StrategyID)
}

func TestApply_UnboundSOPReturnsAnnotationsUnchanged(t *testing.T) {
	const orgID = "customer-a"
	gen := &stubGen{strategy: cannedStrategy("INC-x", "fp-x")}
	hook, _, hist, seed := seedHookFixture(t, orgID, gen)

	// Drop the sop_id label so the binding resolves to "missing".
	labels := cloneMap(seed.Alert.Labels)
	delete(labels, alertmanagertypes.IncidentLabelSopID)

	got := hook.Apply(
		context.Background(),
		orgID,
		"INC-x",
		"fp-x",
		labels,
		seed.Alert.Annotations,
	)

	require.Equal(t, seed.Alert.Annotations, got)
	require.Zero(t, gen.calls, "generator must not run when SOP is unbound")

	_, ok, _ := hist.GetLatest(context.Background(), orgID, ruletypes.AIStrategyHistoryLookupRequest{IncidentID: "INC-x"})
	require.False(t, ok, "history must not record when SOP is unbound")
}

func TestApply_GeneratorErrorLeavesAnnotationsUnchanged(t *testing.T) {
	const orgID = "customer-a"
	gen := &stubGen{err: errors.New("boom")}
	hook, _, hist, seed := seedHookFixture(t, orgID, gen)

	got := hook.Apply(
		context.Background(),
		orgID,
		seed.Alert.IncidentID,
		seed.Alert.Fingerprint,
		seed.Alert.Labels,
		seed.Alert.Annotations,
	)

	require.Equal(t, seed.Alert.Annotations, got)
	require.Equal(t, 1, gen.calls)

	_, ok, _ := hist.GetLatest(context.Background(), orgID, ruletypes.AIStrategyHistoryLookupRequest{IncidentID: seed.Alert.IncidentID})
	require.False(t, ok)
}

func TestApply_GeneratorTimeoutLeavesAnnotationsUnchangedAndDoesNotPanic(t *testing.T) {
	const orgID = "customer-a"
	gen := &stubGen{delay: 200 * time.Millisecond}
	hook, _, _, seed := seedHookFixture(t, orgID, gen)
	// Override the hook's timeout to something far below the stub's delay.
	hook.timeout = 5 * time.Millisecond

	require.NotPanics(t, func() {
		got := hook.Apply(
			context.Background(),
			orgID,
			seed.Alert.IncidentID,
			seed.Alert.Fingerprint,
			seed.Alert.Labels,
			seed.Alert.Annotations,
		)
		require.Equal(t, seed.Alert.Annotations, got)
	})
}

func TestApply_TenantIsolationDoesNotLeakSOPsFromOtherOrgs(t *testing.T) {
	// The SOP is upserted under "customer-a"; the caller passes
	// "customer-b" — the hook must not see the doc.
	gen := &stubGen{strategy: cannedStrategy("INC-iso", "fp-iso")}
	hook, _, hist, seed := seedHookFixture(t, "customer-a", gen)

	got := hook.Apply(
		context.Background(),
		"customer-b",
		"INC-iso",
		"fp-iso",
		seed.Alert.Labels,
		seed.Alert.Annotations,
	)

	require.Equal(t, seed.Alert.Annotations, got)
	require.Zero(t, gen.calls, "generator must not run when no SOP is bound for this tenant")

	_, ok, _ := hist.GetLatest(context.Background(), "customer-b", ruletypes.AIStrategyHistoryLookupRequest{IncidentID: "INC-iso"})
	require.False(t, ok)
}

func TestApply_EmptyOrgIDIsANoOp(t *testing.T) {
	gen := &stubGen{strategy: cannedStrategy("INC-empty", "fp-empty")}
	hook, _, _, seed := seedHookFixture(t, "customer-a", gen)

	got := hook.Apply(
		context.Background(),
		"",
		"INC-empty",
		"fp-empty",
		seed.Alert.Labels,
		seed.Alert.Annotations,
	)
	require.Equal(t, seed.Alert.Annotations, got)
	require.Zero(t, gen.calls)
}

// cannedStrategy is used when we only care that the generator is (or is
// not) invoked — it is intentionally minimal and may not pass full
// validation. The bound-SOP test uses the real local generator.
func cannedStrategy(incidentID, fingerprint string) ruletypes.AIStrategy {
	return ruletypes.AIStrategy{
		ContractVersion:  ruletypes.AIStrategyContractVersion,
		StrategyID:       "strategy-canned",
		IncidentID:       incidentID,
		AlertFingerprint: fingerprint,
		Status:           ruletypes.AIStrategyStatusReady,
		Language:         "ko-KR",
		Confidence:       ruletypes.AIConfidenceMedium,
		Headline:         "canned",
		Audit: ruletypes.AIStrategyAudit{
			PromptVersion:    "ds-ir-ko-v1",
			Model:            "stub",
			GeneratedAt:      "2026-05-20T00:00:00Z",
			RedactionApplied: true,
		},
	}
}

func cloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

