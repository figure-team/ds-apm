package signozruler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/remediation"
	"github.com/SigNoz/signoz/pkg/ruler/remediationstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// fakeRemediationStore is an in-memory remediationstore.Store for handler tests.
// TransitionToExecuting honours the proposed→executing single-flight guard so we
// can exercise both the winning and losing approve paths.
type fakeRemediationStore struct {
	mu   sync.Mutex
	rows map[string]ruletypes.RemediationExecution
	cfg  ruletypes.RemediationConfig
}

func newFakeRemediationStore() *fakeRemediationStore {
	return &fakeRemediationStore{
		rows: map[string]ruletypes.RemediationExecution{},
		cfg: ruletypes.RemediationConfig{
			ExecutionEnabled:   true,
			ExecTimeoutSeconds: 1,
			// High enough that the concurrency cap never fires in these tests, so
			// the lost-race case exercises the TransitionToExecuting guard (→409)
			// rather than the cap (→429).
			MaxConcurrent: 5,
		},
	}
}

func (f *fakeRemediationStore) seed(e ruletypes.RemediationExecution) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[e.ID] = e
}

func (f *fakeRemediationStore) get(id string) ruletypes.RemediationExecution {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rows[id]
}

func (f *fakeRemediationStore) Create(_ context.Context, e ruletypes.RemediationExecution) error {
	f.seed(e)
	return nil
}

func (f *fakeRemediationStore) Get(_ context.Context, _, id string) (ruletypes.RemediationExecution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.rows[id]
	if !ok {
		return ruletypes.RemediationExecution{}, context.Canceled // any error → handler maps to 404
	}
	return e, nil
}

func (f *fakeRemediationStore) ListByIncident(_ context.Context, _, incidentID string) ([]ruletypes.RemediationExecution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []ruletypes.RemediationExecution{}
	for _, e := range f.rows {
		if e.IncidentID == incidentID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeRemediationStore) ListByStatus(_ context.Context, _, status string) ([]ruletypes.RemediationExecution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []ruletypes.RemediationExecution{}
	for _, e := range f.rows {
		if e.Status == status {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeRemediationStore) TransitionToExecuting(_ context.Context, _, id, approvedBy, approvedAt string, _ int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.rows[id]
	if !ok {
		return false, nil
	}
	// Single-flight guard: only a proposed (or approved) row may start executing.
	if e.Status != ruletypes.RemediationStatusProposed && e.Status != ruletypes.RemediationStatusApproved {
		return false, nil
	}
	e.Status = ruletypes.RemediationStatusExecuting
	e.ApprovedBy = approvedBy
	e.ApprovedAt = approvedAt
	f.rows[id] = e
	return true, nil
}

func (f *fakeRemediationStore) Transition(_ context.Context, _, id, toStatus string, patch remediationstore.TransitionPatch) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.rows[id]
	if !ok {
		return context.Canceled
	}
	e.Status = toStatus
	if patch.TerminalAt != "" {
		e.TerminalAt = patch.TerminalAt
	}
	if patch.ExitCode != nil {
		e.ExitCode = patch.ExitCode
	}
	if patch.OutputSnippet != "" {
		e.OutputSnippet = patch.OutputSnippet
	}
	f.rows[id] = e
	return nil
}

func (f *fakeRemediationStore) CountActiveByOrg(_ context.Context, _ string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for _, e := range f.rows {
		if e.Status == ruletypes.RemediationStatusApproved || e.Status == ruletypes.RemediationStatusExecuting {
			n++
		}
	}
	return n, nil
}

func (f *fakeRemediationStore) GetConfig(_ context.Context, _ string) (ruletypes.RemediationConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cfg, nil
}

func (f *fakeRemediationStore) UpsertConfig(_ context.Context, _ string, cfg ruletypes.RemediationConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cfg = cfg.WithDefaults()
	return nil
}

// fakeRunner records the script it was asked to run and signals completion so
// tests can synchronise on the async execution goroutine.
type fakeRunner struct {
	mu         sync.Mutex
	calls      int
	lastScript string
	exitCode   int
	done       chan struct{}
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{done: make(chan struct{}, 1)}
}

func (r *fakeRunner) Run(_ context.Context, script string) remediation.ExecResult {
	r.mu.Lock()
	r.calls++
	r.lastScript = script
	exit := r.exitCode
	r.mu.Unlock()
	r.done <- struct{}{}
	return remediation.ExecResult{ExitCode: exit, Output: "ok"}
}

// waitDone blocks until Run has fired once (or fails the test on timeout).
func (r *fakeRunner) waitDone(t *testing.T) {
	t.Helper()
	select {
	case <-r.done:
	case <-time.After(2 * time.Second):
		t.Fatal("executor goroutine did not run within 2s")
	}
}

// newRemediationHandler builds a handler wired to a fake store + a captured fake
// runner returned through the newRemediationExecutor factory.
func newRemediationHandler(t *testing.T) (*handler, *fakeRemediationStore, *fakeRunner) {
	t.Helper()
	store := newFakeRemediationStore()
	runner := newFakeRunner()
	h := &handler{}
	h.SetRemediationDeps(store, func(time.Duration) RemediationRunner { return runner })
	return h, store, runner
}

// newAuthedReq builds a request carrying SOP test claims (org-scoped) and the
// given mux path vars, mirroring the runbook handler test harness.
func newAuthedReq(t *testing.T, method, target, _ string, vars map[string]string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	req = withSOPTestClaims(req)
	for k, v := range vars {
		req = muxSetVar(req, k, v)
	}
	return req
}

func TestGetRemediationConfig_ReturnsConfig(t *testing.T) {
	h, _, _ := newRemediationHandler(t)

	rw := httptest.NewRecorder()
	req := newAuthedReq(t, http.MethodGet, "/api/v2/ds/remediation/config", testOrgID, nil)
	h.GetRemediationConfig(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (body=%s)", rw.Code, rw.Body.String())
	}
	if !strings.Contains(rw.Body.String(), "executionEnabled") {
		t.Fatalf("response missing executionEnabled: %s", rw.Body.String())
	}
}

func TestUpdateRemediationConfig_PersistsToggle(t *testing.T) {
	h, store, _ := newRemediationHandler(t)
	// Fake seeds ExecutionEnabled:true; PUT flips it off.
	req := httptest.NewRequest(http.MethodPut, "/api/v2/ds/remediation/config",
		strings.NewReader(`{"executionEnabled": false}`))
	req = withSOPTestClaims(req)

	rw := httptest.NewRecorder()
	h.UpdateRemediationConfig(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (body=%s)", rw.Code, rw.Body.String())
	}
	got, _ := store.GetConfig(context.Background(), testOrgID)
	if got.ExecutionEnabled {
		t.Fatalf("PUT must persist executionEnabled=false, got %+v", got)
	}
	// Toggle-only payload must not zero the timing knobs.
	if got.ProposalTTLSeconds != 1800 {
		t.Fatalf("timing knobs must backfill, got %+v", got)
	}
}

func TestUpdateRemediationConfig_RejectsNegative(t *testing.T) {
	h, _, _ := newRemediationHandler(t)
	req := httptest.NewRequest(http.MethodPut, "/api/v2/ds/remediation/config",
		strings.NewReader(`{"executionEnabled": true, "maxConcurrent": -3}`))
	req = withSOPTestClaims(req)

	rw := httptest.NewRecorder()
	h.UpdateRemediationConfig(rw, req)

	if rw.Code != http.StatusBadRequest {
		t.Fatalf("want 400 on negative knob, got %d (body=%s)", rw.Code, rw.Body.String())
	}
}

func TestApproveRemediation_Wins_StartsExecution(t *testing.T) {
	h, store, exec := newRemediationHandler(t)
	store.seed(ruletypes.RemediationExecution{
		ID:             "rem-1",
		OrgID:          testOrgID,
		Status:         ruletypes.RemediationStatusProposed,
		ScriptSnapshot: "echo hi",
		IncidentID:     "inc-1",
	})

	rw := httptest.NewRecorder()
	req := newAuthedReq(t, http.MethodPost, "/api/v2/ds/remediation/rem-1/approve", testOrgID, map[string]string{"id": "rem-1"})
	h.ApproveRemediation(rw, req)

	if rw.Code != http.StatusAccepted {
		t.Fatalf("want 202, got %d (body=%s)", rw.Code, rw.Body.String())
	}
	exec.waitDone(t)
	if exec.lastScript != "echo hi" {
		t.Fatalf("executor got wrong script: %q", exec.lastScript)
	}
}

func TestApproveRemediation_Loses_NoDoubleExec(t *testing.T) {
	h, store, exec := newRemediationHandler(t)
	store.seed(ruletypes.RemediationExecution{
		ID:     "rem-2",
		OrgID:  testOrgID,
		Status: ruletypes.RemediationStatusExecuting, // already not proposed → lose the race
	})

	rw := httptest.NewRecorder()
	req := newAuthedReq(t, http.MethodPost, "/api/v2/ds/remediation/rem-2/approve", testOrgID, map[string]string{"id": "rem-2"})
	h.ApproveRemediation(rw, req)

	if rw.Code != http.StatusConflict {
		t.Fatalf("want 409 on lost race, got %d (body=%s)", rw.Code, rw.Body.String())
	}
	if exec.calls != 0 {
		t.Fatal("must not execute on lost race")
	}
}

func TestRejectRemediation_OK(t *testing.T) {
	h, store, _ := newRemediationHandler(t)
	store.seed(ruletypes.RemediationExecution{
		ID:     "rem-3",
		OrgID:  testOrgID,
		Status: ruletypes.RemediationStatusProposed,
	})

	rw := httptest.NewRecorder()
	req := newAuthedReq(t, http.MethodPost, "/api/v2/ds/remediation/rem-3/reject", testOrgID, map[string]string{"id": "rem-3"})
	h.RejectRemediation(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (body=%s)", rw.Code, rw.Body.String())
	}
	if store.get("rem-3").Status != ruletypes.RemediationStatusRejected {
		t.Fatal("status must be rejected")
	}
}
