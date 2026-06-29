package remediation

import (
	"context"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/remediationstore"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// ---------------------------------------------------------------------------
// Test-local fake store for Selector tests.
// (propose_test.go's fakeStore lacks cfg/active/lastCreated/createCount fields;
// we add a separate memStore so the two fakes coexist without collision.)
// ---------------------------------------------------------------------------

type memStore struct {
	cfg         ruletypes.RemediationConfig
	active      []ruletypes.RemediationExecution
	lastCreated ruletypes.RemediationExecution
	createCount int
	createErr   error
}

func newMemStore() *memStore { return &memStore{} }

func (m *memStore) Create(_ context.Context, e ruletypes.RemediationExecution) error {
	if m.createErr != nil {
		return m.createErr
	}
	if err := ruletypes.ValidateRemediationExecution(e); err != nil {
		return err
	}
	m.lastCreated = e
	m.createCount++
	return nil
}

func (m *memStore) Get(_ context.Context, _, _ string) (ruletypes.RemediationExecution, error) {
	return ruletypes.RemediationExecution{}, nil
}

func (m *memStore) ListByIncident(_ context.Context, _, _ string) ([]ruletypes.RemediationExecution, error) {
	return nil, nil
}

func (m *memStore) ListByStatus(_ context.Context, _, _ string) ([]ruletypes.RemediationExecution, error) {
	return nil, nil
}

func (m *memStore) TransitionToExecuting(_ context.Context, _, _, _, _ string, _ int64) (bool, error) {
	return false, nil
}

func (m *memStore) Transition(_ context.Context, _, _, _ string, _ remediationstore.TransitionPatch) error {
	return nil
}

func (m *memStore) CountActiveByOrg(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (m *memStore) GetConfig(_ context.Context, _ string) (ruletypes.RemediationConfig, error) {
	return m.cfg, nil
}

func (m *memStore) UpsertConfig(_ context.Context, _ string, _ ruletypes.RemediationConfig) error {
	return nil
}

func (m *memStore) ListActiveByFingerprint(_ context.Context, _, _ string) ([]ruletypes.RemediationExecution, error) {
	return m.active, nil
}

// ---------------------------------------------------------------------------
// remediationIDAnnotationKey returns the annotation key for remediation id.
// ---------------------------------------------------------------------------

func remediationIDAnnotationKey() string {
	return alertmanagertypes.IncidentAnnotationRemediationID
}

// ---------------------------------------------------------------------------
// Fake LLM provider / resolver
// ---------------------------------------------------------------------------

// fakeProvider is a test double for LLMProvider.
// calls is a *int pointer so tests can observe the count even when
// fakeResolver holds a copy of the struct value.
type fakeProvider struct {
	resp  string
	err   error
	calls *int // incremented on every Complete invocation; nil == don't track
}

func (f *fakeProvider) Complete(_ context.Context, _, _ string) (string, error) {
	if f.calls != nil {
		*f.calls++
	}
	return f.resp, f.err
}

type fakeResolver struct{ p *fakeProvider }

func (f fakeResolver) Resolve(_ context.Context, _ string) (LLMProvider, string, error) {
	return f.p, "test-model", nil
}

// newFakeProvider allocates a fakeProvider with a live call counter.
func newFakeProvider(resp string, err error) *fakeProvider {
	calls := 0
	return &fakeProvider{resp: resp, err: err, calls: &calls}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// fixedNow is already defined in propose_test.go — do not redeclare.
// Use it directly in tests below.

func docWithRunbooks(rbs ...ruletypes.Runbook) ruletypes.SOPDocument {
	return ruletypes.SOPDocument{
		SOPID: "SOP-1", Version: "v1", Title: "t", BodyMarkdown: "b", Runbooks: rbs,
	}
}

func approvedRB(id, script string) ruletypes.Runbook {
	return ruletypes.Runbook{ID: id, Title: id, ExecutableScript: script, Status: ruletypes.RunbookStatusApproved, Confidence: 0.6}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestSelect_SelectedRunbook_CreatesExecution(t *testing.T) {
	store := newMemStore()
	store.cfg = ruletypes.RemediationConfig{ExecutionEnabled: true, ProposalTTLSeconds: 600, MaxConcurrent: 1}
	fp := newFakeProvider(`{"outcome":"selected","chosenRunbookId":"rb-2","confidence":"high","rationale":"맞음"}`, nil)
	sel := NewSelector(store, fakeResolver{fp}, "https://x", time.Second, fixedNow, nil)

	doc := docWithRunbooks(approvedRB("rb-1", "echo a"), approvedRB("rb-2", "echo b"))
	ann, ok := sel.Select(context.Background(), "org-1", "inc-1", "fp-1", map[string]string{"ruleId": "R1"}, doc)
	if !ok {
		t.Fatalf("expected a proposal")
	}
	if ann[remediationIDAnnotationKey()] == "" {
		t.Fatalf("expected remediation id annotation")
	}
	created := store.lastCreated
	if created.RunbookID != "rb-2" || created.ScriptSnapshot != "echo b" || created.Source != ruletypes.RemediationSourceRunbook {
		t.Fatalf("wrong execution snapshot: %+v", created)
	}
	if *fp.calls != 1 {
		t.Fatalf("expected provider called exactly 1 time, got %d", *fp.calls)
	}
}

func TestSelect_Fallback_CreatesLLMGeneratedExecution(t *testing.T) {
	store := newMemStore()
	store.cfg = ruletypes.RemediationConfig{ExecutionEnabled: true, ProposalTTLSeconds: 600, MaxConcurrent: 1}
	resp := `{"outcome":"fallback","confidence":"medium","rationale":"적합 없음","fallbackScript":"kubectl rollout restart deploy/x","fallbackSummary":"재시작"}`
	fp := newFakeProvider(resp, nil)
	sel := NewSelector(store, fakeResolver{fp}, "https://x", time.Second, fixedNow, nil)

	doc := docWithRunbooks(approvedRB("rb-1", "echo a"))
	_, ok := sel.Select(context.Background(), "org-1", "inc-1", "fp-1", nil, doc)
	if !ok {
		t.Fatalf("expected fallback proposal")
	}
	if store.lastCreated.Source != ruletypes.RemediationSourceLLMGenerated {
		t.Fatalf("expected llm-generated source, got %q", store.lastCreated.Source)
	}
	if store.lastCreated.ScriptSnapshot != "kubectl rollout restart deploy/x" {
		t.Fatalf("fallback script not snapshotted")
	}
	if *fp.calls != 1 {
		t.Fatalf("expected provider called exactly 1 time, got %d", *fp.calls)
	}
}

func TestSelect_None_NoExecution(t *testing.T) {
	store := newMemStore()
	store.cfg = ruletypes.RemediationConfig{ExecutionEnabled: true, ProposalTTLSeconds: 600, MaxConcurrent: 1}
	fp := newFakeProvider(`{"outcome":"none","confidence":"low","rationale":"불명"}`, nil)
	sel := NewSelector(store, fakeResolver{fp}, "https://x", time.Second, fixedNow, nil)
	_, ok := sel.Select(context.Background(), "org-1", "inc-1", "fp-1", nil, docWithRunbooks(approvedRB("rb-1", "echo a")))
	if ok || store.createCount != 0 {
		t.Fatalf("none outcome must not create an execution")
	}
}

func TestSelect_LLMError_FailOpen(t *testing.T) {
	store := newMemStore()
	store.cfg = ruletypes.RemediationConfig{ExecutionEnabled: true, ProposalTTLSeconds: 600, MaxConcurrent: 1}
	fp := newFakeProvider("", context.DeadlineExceeded)
	sel := NewSelector(store, fakeResolver{fp}, "https://x", time.Second, fixedNow, nil)
	_, ok := sel.Select(context.Background(), "org-1", "inc-1", "fp-1", nil, docWithRunbooks(approvedRB("rb-1", "echo a")))
	if ok || store.createCount != 0 {
		t.Fatalf("LLM error must fail open (no execution)")
	}
}

func TestSelect_ExistingActiveProposal_Skips(t *testing.T) {
	store := newMemStore()
	store.cfg = ruletypes.RemediationConfig{ExecutionEnabled: true, ProposalTTLSeconds: 600, MaxConcurrent: 1}
	store.active = []ruletypes.RemediationExecution{{ID: "x", Status: ruletypes.RemediationStatusProposed}}
	fp := newFakeProvider(`{"outcome":"selected","chosenRunbookId":"rb-1","confidence":"high","rationale":"x"}`, nil)
	sel := NewSelector(store, fakeResolver{fp}, "https://x", time.Second, fixedNow, nil)
	_, ok := sel.Select(context.Background(), "org-1", "inc-1", "fp-1", nil, docWithRunbooks(approvedRB("rb-1", "echo a")))
	if ok || store.createCount != 0 {
		t.Fatalf("existing active proposal must short-circuit (cache reuse)")
	}
	if *fp.calls != 0 {
		t.Fatalf("LLM must not be called on cache hit, got %d call(s)", *fp.calls)
	}
}

func TestSelect_ExecutionDisabled_NoOp(t *testing.T) {
	store := newMemStore()
	store.cfg = ruletypes.RemediationConfig{ExecutionEnabled: false}
	fp := newFakeProvider(`{"outcome":"selected","chosenRunbookId":"rb-1","confidence":"high","rationale":"x"}`, nil)
	sel := NewSelector(store, fakeResolver{fp}, "https://x", time.Second, fixedNow, nil)
	_, ok := sel.Select(context.Background(), "org-1", "inc-1", "fp-1", nil, docWithRunbooks(approvedRB("rb-1", "echo a")))
	if ok {
		t.Fatalf("execution disabled must no-op")
	}
	if *fp.calls != 0 {
		t.Fatalf("LLM must not be called when execution is disabled, got %d call(s)", *fp.calls)
	}
}

// TestSelect_ZeroApprovedRunbooks_NoLLM verifies the cost-guardrail invariant:
// when all runbooks are non-approved (or have empty scripts), the zero-approved
// short-circuit path MUST return false without invoking the LLM.
func TestSelect_ZeroApprovedRunbooks_NoLLM(t *testing.T) {
	store := newMemStore()
	store.cfg = ruletypes.RemediationConfig{ExecutionEnabled: true, ProposalTTLSeconds: 600, MaxConcurrent: 1}
	fp := newFakeProvider(`{"outcome":"selected","chosenRunbookId":"rb-1","confidence":"high","rationale":"x"}`, nil)
	sel := NewSelector(store, fakeResolver{fp}, "https://x", time.Second, fixedNow, nil)

	// Build a doc whose runbooks are all non-approved (Status=draft) or have empty scripts.
	draftRB := ruletypes.Runbook{ID: "rb-draft", Title: "draft", ExecutableScript: "echo a", Status: ruletypes.RunbookStatusDraft, Confidence: 0.9}
	emptyScriptRB := ruletypes.Runbook{ID: "rb-empty", Title: "empty", ExecutableScript: "   ", Status: ruletypes.RunbookStatusApproved, Confidence: 0.9}
	doc := docWithRunbooks(draftRB, emptyScriptRB)

	_, ok := sel.Select(context.Background(), "org-1", "inc-1", "fp-1", nil, doc)
	if ok {
		t.Fatalf("zero approved runbooks must return false")
	}
	if store.createCount != 0 {
		t.Fatalf("zero approved runbooks must not create an execution, got %d", store.createCount)
	}
	if *fp.calls != 0 {
		t.Fatalf("zero approved runbooks short-circuit must not call the LLM, got %d call(s)", *fp.calls)
	}
}

// Ensure fixedNow is used (suppress "declared and not used" if only referenced
// in test bodies — Go does not require this but it silences linters).
var _ func() time.Time = fixedNow
