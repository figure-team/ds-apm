package engine

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/clirunner"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// --- fakes ---

type fakeRunStore struct {
	claim     runstore.ClaimResult
	claimErr  error
	finalized runstore.FinalizeParams
	finalCnt  int
	finalOK   bool
	finalErr  error
}

func (f *fakeRunStore) ClaimNext(context.Context, runstore.ClaimParams) (runstore.ClaimResult, error) {
	return f.claim, f.claimErr
}
func (f *fakeRunStore) Finalize(_ context.Context, p runstore.FinalizeParams) (bool, error) {
	f.finalCnt++
	f.finalized = p
	return f.finalOK, f.finalErr
}

type fakeRepos struct {
	repo    ruletypes.CodebaseRepo
	subpath string
	ok      bool
	err     error
}

func (f *fakeRepos) ResolveRepo(context.Context, string, string) (ruletypes.CodebaseRepo, string, bool, error) {
	return f.repo, f.subpath, f.ok, f.err
}

type fakeSource struct {
	checkout    string
	baseline    string
	err         error
	cleanupHits int
	prepared    bool
}

func (f *fakeSource) Prepare(context.Context, ruletypes.CodebaseRepo, string) (string, string, func(), error) {
	f.prepared = true
	if f.err != nil {
		return "", "", func() {}, f.err
	}
	return f.checkout, f.baseline, func() { f.cleanupHits++ }, nil
}

type fakeCLI struct {
	result  coderca.RCAResult
	status  coderca.RunStatus
	err     error
	gotSpec clirunner.Spec
	called  bool
}

func (f *fakeCLI) Run(_ context.Context, s clirunner.Spec) (coderca.RCAResult, coderca.RunStatus, error) {
	f.called = true
	f.gotSpec = s
	return f.result, f.status, f.err
}

type fakeDeliverer struct {
	ref      string
	err      error
	called   bool
	gotDeliv Delivery
}

func (f *fakeDeliverer) Deliver(_ context.Context, d Delivery) (string, error) {
	f.called = true
	f.gotDeliv = d
	return f.ref, f.err
}

type fakeAuditor struct{ events []AuditEvent }

func (f *fakeAuditor) Audit(_ context.Context, e AuditEvent) { f.events = append(f.events, e) }

// --- harness ---

func claimedRun() runstore.ClaimResult {
	return runstore.ClaimResult{
		Claimed: true, RunID: "run-1", OrgID: "org1", LeaseToken: "tok-1",
		DedupKey: "dk", Service: "payments", Attempts: 1,
	}
}

func newEngine(t *testing.T, runs *fakeRunStore, repos *fakeRepos, src *fakeSource, cli *fakeCLI, del *fakeDeliverer, aud *fakeAuditor) *Engine {
	t.Helper()
	return New(Config{
		InstanceID: "inst-1", Agent: clirunner.AgentClaude, Model: "m", MaxBudgetUSD: "1", AuthToken: "tok",
	}, Deps{
		Runs: runs, Repos: repos, Source: src, CLI: cli, Deliver: del, Auditor: aud,
		Now: func() time.Time { return time.Unix(1_700_000_000, 0) },
	})
}

func doneCLI() *fakeCLI {
	return &fakeCLI{
		status: coderca.RunStatusDone,
		result: coderca.RCAResult{RootCause: "pool exhausted", Confidence: "medium", Raw: "raw"},
	}
}

// --- tests ---

func TestProcessNextHappyPath(t *testing.T) {
	runs := &fakeRunStore{claim: claimedRun(), finalOK: true}
	repos := &fakeRepos{repo: ruletypes.CodebaseRepo{OrgID: "org1", RepoID: "r1"}, ok: true}
	src := &fakeSource{checkout: "/co/run-1", baseline: "base-abc"}
	cli := doneCLI()
	del := &fakeDeliverer{ref: "handoff-9"}
	aud := &fakeAuditor{}

	processed, err := newEngine(t, runs, repos, src, cli, del, aud).ProcessNext(context.Background())
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if !processed {
		t.Fatal("processed = false, want true")
	}
	// finalized done, carrying the delivery ref + fencing token.
	if runs.finalCnt != 1 {
		t.Fatalf("Finalize called %d times, want 1", runs.finalCnt)
	}
	if runs.finalized.Status != coderca.RunStatusDone {
		t.Errorf("finalize status = %q, want done", runs.finalized.Status)
	}
	if runs.finalized.ResultRef != "handoff-9" {
		t.Errorf("finalize resultRef = %q, want handoff-9", runs.finalized.ResultRef)
	}
	if runs.finalized.LeaseToken != "tok-1" || runs.finalized.RunID != "run-1" {
		t.Errorf("finalize must fence with the claimed run+token: %+v", runs.finalized)
	}
	// delivered the result + baseline.
	if !del.called {
		t.Fatal("Deliver not called")
	}
	if del.gotDeliv.BaselineCommit != "base-abc" || del.gotDeliv.Result.RootCause != "pool exhausted" {
		t.Errorf("delivery wrong: %+v", del.gotDeliv)
	}
	// CLI got a checkout-scoped, read-only spec built from config.
	if cli.gotSpec.Checkout != "/co/run-1" {
		t.Errorf("cli checkout = %q, want /co/run-1", cli.gotSpec.Checkout)
	}
	if cli.gotSpec.Agent != clirunner.AgentClaude || cli.gotSpec.MaxBudgetUSD != "1" || cli.gotSpec.AuthToken != "tok" {
		t.Errorf("cli spec missing config: %+v", cli.gotSpec)
	}
	if cli.gotSpec.SystemPrompt == "" || !strings.Contains(cli.gotSpec.Prompt, "payments") {
		t.Errorf("cli prompt not built from context: sys=%q user=%q", cli.gotSpec.SystemPrompt, cli.gotSpec.Prompt)
	}
	// checkout cleaned up; outcome audited.
	if src.cleanupHits != 1 {
		t.Errorf("checkout cleanup called %d times, want 1", src.cleanupHits)
	}
	if len(aud.events) != 1 || aud.events[0].Status != coderca.RunStatusDone {
		t.Errorf("audit events = %+v, want one done", aud.events)
	}
}

func TestProcessNextNothingClaimable(t *testing.T) {
	runs := &fakeRunStore{claim: runstore.ClaimResult{Claimed: false}}
	cli := &fakeCLI{}
	processed, err := newEngine(t, runs, &fakeRepos{}, &fakeSource{}, cli, &fakeDeliverer{}, &fakeAuditor{}).
		ProcessNext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed {
		t.Error("processed = true, want false (nothing queued)")
	}
	if cli.called {
		t.Error("CLI must not run when no run is claimed")
	}
	if runs.finalCnt != 0 {
		t.Error("Finalize must not be called when no run is claimed")
	}
}

func TestProcessNextUnmappedRepoFails(t *testing.T) {
	runs := &fakeRunStore{claim: claimedRun(), finalOK: true}
	repos := &fakeRepos{ok: false} // mapping gone
	cli := &fakeCLI{}
	aud := &fakeAuditor{}

	processed, err := newEngine(t, runs, repos, &fakeSource{}, cli, &fakeDeliverer{}, aud).
		ProcessNext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !processed {
		t.Fatal("processed = false, want true")
	}
	if cli.called {
		t.Error("CLI must not run for an unmapped service")
	}
	if runs.finalized.Status != coderca.RunStatusFailed {
		t.Errorf("status = %q, want failed", runs.finalized.Status)
	}
	if len(aud.events) != 1 || !strings.Contains(aud.events[0].Detail, "no_repo_mapping") {
		t.Errorf("audit should record no_repo_mapping: %+v", aud.events)
	}
}

func TestProcessNextSourcePrepareFails(t *testing.T) {
	runs := &fakeRunStore{claim: claimedRun(), finalOK: true}
	repos := &fakeRepos{repo: ruletypes.CodebaseRepo{RepoID: "r1"}, ok: true}
	src := &fakeSource{err: errors.New("fetch failed")}
	cli := &fakeCLI{}

	_, err := newEngine(t, runs, repos, src, cli, &fakeDeliverer{}, &fakeAuditor{}).
		ProcessNext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cli.called {
		t.Error("CLI must not run when source prepare fails")
	}
	if runs.finalized.Status != coderca.RunStatusFailed {
		t.Errorf("status = %q, want failed", runs.finalized.Status)
	}
}

func TestProcessNextCLITimeoutFinalizesTimeoutNoDelivery(t *testing.T) {
	runs := &fakeRunStore{claim: claimedRun(), finalOK: true}
	repos := &fakeRepos{repo: ruletypes.CodebaseRepo{RepoID: "r1"}, ok: true}
	src := &fakeSource{checkout: "/co/run-1", baseline: "b"}
	cli := &fakeCLI{status: coderca.RunStatusTimeout, err: errors.New("timed out"), result: coderca.RCAResult{Raw: "partial"}}
	del := &fakeDeliverer{}

	_, err := newEngine(t, runs, repos, src, cli, del, &fakeAuditor{}).ProcessNext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if runs.finalized.Status != coderca.RunStatusTimeout {
		t.Errorf("status = %q, want timeout", runs.finalized.Status)
	}
	if del.called {
		t.Error("must not deliver a timed-out run")
	}
	if src.cleanupHits != 1 {
		t.Error("checkout must be cleaned up even on timeout")
	}
}

func TestProcessNextUnparseableNotDelivered(t *testing.T) {
	runs := &fakeRunStore{claim: claimedRun(), finalOK: true}
	repos := &fakeRepos{repo: ruletypes.CodebaseRepo{RepoID: "r1"}, ok: true}
	src := &fakeSource{checkout: "/co/run-1", baseline: "b"}
	cli := &fakeCLI{status: coderca.RunStatusUnparseable, result: coderca.RCAResult{Raw: "garbage"}}
	del := &fakeDeliverer{}

	_, err := newEngine(t, runs, repos, src, cli, del, &fakeAuditor{}).ProcessNext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if runs.finalized.Status != coderca.RunStatusUnparseable {
		t.Errorf("status = %q, want unparseable", runs.finalized.Status)
	}
	if del.called {
		t.Error("must not deliver an unparseable result")
	}
}

func TestProcessNextDeliveryFailureIsFailed(t *testing.T) {
	runs := &fakeRunStore{claim: claimedRun(), finalOK: true}
	repos := &fakeRepos{repo: ruletypes.CodebaseRepo{RepoID: "r1"}, ok: true}
	src := &fakeSource{checkout: "/co/run-1", baseline: "b"}
	cli := doneCLI()
	del := &fakeDeliverer{err: errors.New("handoff down")}

	_, err := newEngine(t, runs, repos, src, cli, del, &fakeAuditor{}).ProcessNext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if runs.finalized.Status != coderca.RunStatusFailed {
		t.Errorf("status = %q, want failed (delivery failed)", runs.finalized.Status)
	}
}

func TestProcessNextClaimErrorPropagates(t *testing.T) {
	runs := &fakeRunStore{claimErr: errors.New("db down")}
	processed, err := newEngine(t, runs, &fakeRepos{}, &fakeSource{}, &fakeCLI{}, &fakeDeliverer{}, &fakeAuditor{}).
		ProcessNext(context.Background())
	if err == nil {
		t.Error("claim error must propagate")
	}
	if processed {
		t.Error("processed = true on claim error, want false")
	}
}

func TestProcessNextPersistsReportOnDone(t *testing.T) {
	runs := &fakeRunStore{claim: claimedRun(), finalOK: true}
	repos := &fakeRepos{repo: ruletypes.CodebaseRepo{OrgID: "org1", RepoID: "r1"}, ok: true}
	src := &fakeSource{checkout: "/co/run-1", baseline: "src-baseline"}
	cli := &fakeCLI{
		status: coderca.RunStatusDone,
		result: coderca.RCAResult{
			BaselineCommit: "echo-commit",
			RootCause:      "rc",
			ProposedFix:    "fix",
			Confidence:     "high",
			Limitations:    "lim",
		},
	}
	del := &fakeDeliverer{ref: "handoff-42"}
	aud := &fakeAuditor{}

	processed, err := newEngine(t, runs, repos, src, cli, del, aud).ProcessNext(context.Background())
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	if !processed {
		t.Fatal("processed = false, want true")
	}
	fp := runs.finalized
	if fp.RootCause != "rc" {
		t.Errorf("RootCause = %q, want rc", fp.RootCause)
	}
	if fp.ProposedFix != "fix" {
		t.Errorf("ProposedFix = %q, want fix", fp.ProposedFix)
	}
	if fp.Confidence != "high" {
		t.Errorf("Confidence = %q, want high", fp.Confidence)
	}
	if fp.Limitations != "lim" {
		t.Errorf("Limitations = %q, want lim", fp.Limitations)
	}
	// CLI echo wins over src baseline when non-empty.
	if fp.BaselineCommit != "echo-commit" {
		t.Errorf("BaselineCommit = %q, want echo-commit (CLI echo wins)", fp.BaselineCommit)
	}
}

func TestProcessNextPersistsSourceBaselineWhenCLIEchoMissing(t *testing.T) {
	runs := &fakeRunStore{claim: claimedRun(), finalOK: true}
	repos := &fakeRepos{repo: ruletypes.CodebaseRepo{OrgID: "org1", RepoID: "r1"}, ok: true}
	src := &fakeSource{checkout: "/co/run-1", baseline: "src-baseline"}
	cli := &fakeCLI{
		status: coderca.RunStatusDone,
		result: coderca.RCAResult{
			BaselineCommit: "", // CLI did not echo baseline
			RootCause:      "rc2",
		},
	}
	del := &fakeDeliverer{ref: "handoff-43"}
	aud := &fakeAuditor{}

	_, err := newEngine(t, runs, repos, src, cli, del, aud).ProcessNext(context.Background())
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	// src baseline used as fallback when CLI echo is empty.
	if runs.finalized.BaselineCommit != "src-baseline" {
		t.Errorf("BaselineCommit = %q, want src-baseline (source baseline fallback)", runs.finalized.BaselineCommit)
	}
}

func TestProcessNextPersistsBaselineOnFailure(t *testing.T) {
	runs := &fakeRunStore{claim: claimedRun(), finalOK: true}
	repos := &fakeRepos{repo: ruletypes.CodebaseRepo{OrgID: "org1", RepoID: "r1"}, ok: true}
	src := &fakeSource{checkout: "/co/run-1", baseline: "src-baseline"}
	cli := &fakeCLI{
		status: coderca.RunStatusFailed,
		result: coderca.RCAResult{}, // zero result on failure
	}
	del := &fakeDeliverer{}
	aud := &fakeAuditor{}

	_, err := newEngine(t, runs, repos, src, cli, del, aud).ProcessNext(context.Background())
	if err != nil {
		t.Fatalf("ProcessNext: %v", err)
	}
	fp := runs.finalized
	if fp.Status != coderca.RunStatusFailed {
		t.Errorf("Status = %q, want failed", fp.Status)
	}
	if fp.RootCause != "" || fp.ProposedFix != "" || fp.Confidence != "" || fp.Limitations != "" {
		t.Errorf("report fields must be empty on failure: %+v", fp)
	}
	// baseline is known after source prep succeeded — persist it even on failure.
	if fp.BaselineCommit != "src-baseline" {
		t.Errorf("BaselineCommit = %q, want src-baseline (baseline persisted on failure)", fp.BaselineCommit)
	}
}
