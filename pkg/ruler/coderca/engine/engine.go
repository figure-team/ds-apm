// Package engine orchestrates one code-RCA run end to end (design §5.1):
// claim → resolve service→repo → prepare source at the pinned baseline →
// build prompt → run the CLI agent (read-only) → deliver the suggestion (HITL)
// → audit → finalize. Every external dependency is an injected port, so the
// pipeline is hermetically testable; the concrete ports (DB run store, git
// source manager, handoff deliverer, auditor) are constructed at the
// integration seam (§11) and are NOT wired from this worktree.
package engine

import (
	"context"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/clirunner"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// RunStore is the claim/finalize surface a worker needs (satisfied by
// *runstore.Store). Heartbeat is intentionally omitted: the claim sets
// LeaseTTL >= run_timeout + grace (design §6.3), so a run that finishes within
// its timeout never needs its lease extended.
type RunStore interface {
	ClaimNext(ctx context.Context, p runstore.ClaimParams) (runstore.ClaimResult, error)
	Finalize(ctx context.Context, p runstore.FinalizeParams) (bool, error)
}

// RepoResolver resolves (org, service) → the repo to analyze (+ monorepo
// subpath). The concrete impl loads service maps + repo from the config store
// and applies coderca.ResolveServiceRepo.
type RepoResolver interface {
	ResolveRepo(ctx context.Context, orgID, service string) (repo ruletypes.CodebaseRepo, subpath string, ok bool, err error)
}

// SourcePreparer fetches the repo and creates a disposable checkout pinned at
// the baseline commit. cleanup removes the checkout and is always safe to defer.
type SourcePreparer interface {
	Prepare(ctx context.Context, repo ruletypes.CodebaseRepo, subpath string) (checkoutDir, baseline string, cleanup func(), err error)
}

// CLIRunner runs the agent and returns the parsed result + status (satisfied by
// *clirunner.Runner).
type CLIRunner interface {
	Run(ctx context.Context, s clirunner.Spec) (coderca.RCAResult, coderca.RunStatus, error)
}

// Delivery is a completed RCA handed to a human reviewer (CF-3 / HITL).
type Delivery struct {
	OrgID          string
	Service        string
	RunID          string
	BaselineCommit string
	Result         coderca.RCAResult
}

// Deliverer delivers a completed RCA and returns a reference (e.g. handoff id).
// Concrete impl reuses handoff + history (seam §11).
type Deliverer interface {
	Deliver(ctx context.Context, d Delivery) (resultRef string, err error)
}

// AuditEvent is a fire-and-forget audit record for a finalized run (CF-6).
type AuditEvent struct {
	OrgID   string
	RunID   string
	Service string
	Status  coderca.RunStatus
	Detail  string
}

// Auditor records an audit event (fire-and-forget; drop-on-full upstream).
type Auditor interface {
	Audit(ctx context.Context, e AuditEvent)
}

// Config holds per-engine settings.
type Config struct {
	Scope         string          // capacity scope (default "global")
	InstanceID    string          // claimed_by
	Agent         clirunner.Agent // claude | codex
	Model         string
	MaxBudgetUSD  string // claude hard $ ceiling
	AuthToken     string // agent model-API auth (never a git credential)
	MaxConcurrent int
	LeaseTTL      time.Duration // >= run_timeout + grace (design §6.3)
}

// Deps are the engine's injected ports.
type Deps struct {
	Runs     RunStore
	Repos    RepoResolver
	Source   SourcePreparer
	CLI      CLIRunner
	Deliver  Deliverer
	Auditor  Auditor
	Evidence coderca.EvidenceCollector
	Now      func() time.Time
}

// Engine processes queued code-RCA runs one at a time.
type Engine struct {
	cfg  Config
	deps Deps
}

// New builds an Engine. Sensible defaults are filled for Scope, LeaseTTL,
// MaxConcurrent, Evidence, and Now.
func New(cfg Config, deps Deps) *Engine {
	if cfg.Scope == "" {
		cfg.Scope = "global"
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 1
	}
	if cfg.LeaseTTL <= 0 {
		cfg.LeaseTTL = 6 * time.Minute
	}
	if deps.Evidence == nil {
		deps.Evidence = coderca.NoopEvidenceCollector{}
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	return &Engine{cfg: cfg, deps: deps}
}

// ProcessNext claims one queued run and drives it to a terminal state. Returns
// processed=false when nothing was claimable (queue empty or at capacity). The
// run's own outcome is recorded in its status (and audit); err is reserved for
// infrastructure failures (claim/finalize).
//
func (e *Engine) ProcessNext(ctx context.Context) (processed bool, err error) {
	claim, err := e.deps.Runs.ClaimNext(ctx, runstore.ClaimParams{
		Scope:         e.cfg.Scope,
		ClaimedBy:     e.cfg.InstanceID,
		Now:           e.deps.Now(),
		LeaseTTL:      e.cfg.LeaseTTL,
		MaxConcurrent: e.cfg.MaxConcurrent,
	})
	if err != nil {
		return false, err
	}
	if !claim.Claimed {
		return false, nil
	}

	status, resultRef, detail, baseline, result := e.runOne(ctx, claim)

	e.deps.Auditor.Audit(ctx, AuditEvent{
		OrgID:   claim.OrgID,
		RunID:   claim.RunID,
		Service: claim.Service,
		Status:  status,
		Detail:  detail,
	})

	// Fenced terminal transition (design §6.3): only the live lease owner wins.
	_, ferr := e.deps.Runs.Finalize(ctx, runstore.FinalizeParams{
		Scope:      e.cfg.Scope,
		RunID:      claim.RunID,
		LeaseToken: claim.LeaseToken,
		Status:     status,
		ResultRef:  resultRef,
		Now:        e.deps.Now(),

		BaselineCommit: firstNonEmptyStr(result.BaselineCommit, baseline),
		RootCause:      result.RootCause,
		ProposedFix:    result.ProposedFix,
		Confidence:     result.Confidence,
		Limitations:    result.Limitations,
	})
	return true, ferr
}

// runOne executes the pipeline for a claimed run and returns its terminal
// status, the delivery ref (set only when delivered), an audit detail, the
// prepared source baseline (empty if source prep was not reached), and the
// parsed RCA result (zero value on non-done paths) for persistence.
func (e *Engine) runOne(ctx context.Context, claim runstore.ClaimResult) (coderca.RunStatus, string, string, string, coderca.RCAResult) {
	repo, subpath, ok, err := e.deps.Repos.ResolveRepo(ctx, claim.OrgID, claim.Service)
	if err != nil {
		return coderca.RunStatusFailed, "", "resolve repo: " + err.Error(), "", coderca.RCAResult{}
	}
	if !ok {
		return coderca.RunStatusFailed, "", string(coderca.SkipNoRepoMapping), "", coderca.RCAResult{}
	}

	checkout, baseline, cleanup, err := e.deps.Source.Prepare(ctx, repo, subpath)
	if cleanup != nil {
		defer cleanup() // remove the disposable checkout, even on failure/timeout
	}
	if err != nil {
		return coderca.RunStatusFailed, "", "source prepare: " + err.Error(), "", coderca.RCAResult{}
	}

	rc := coderca.RCAContext{
		OrgID:          claim.OrgID,
		Service:        claim.Service,
		BaselineCommit: baseline,
		// NOTE: signature/labels/severity are not yet persisted on the run, so
		// v1 builds a minimal context from service + baseline. Persisting the
		// full error context at admission is a deferred additive enhancement.
	}
	evidence, _ := e.deps.Evidence.Collect(ctx, rc) // best-effort; v1 Noop → none
	system, user := coderca.BuildPrompt(rc, evidence)

	result, status, _ := e.deps.CLI.Run(ctx, clirunner.Spec{
		Agent:        e.cfg.Agent,
		Model:        e.cfg.Model,
		Checkout:     checkout,
		SystemPrompt: system,
		Prompt:       user,
		MaxBudgetUSD: e.cfg.MaxBudgetUSD,
		AuthToken:    e.cfg.AuthToken,
	})
	if status != coderca.RunStatusDone {
		// timeout / failed / unparseable: never delivered (HITL gets only usable
		// results); the status itself is the audit detail. baseline is returned
		// so the caller can persist it even for non-done runs.
		return status, "", string(status), baseline, coderca.RCAResult{}
	}

	resultRef, derr := e.deps.Deliver.Deliver(ctx, Delivery{
		OrgID:          claim.OrgID,
		Service:        claim.Service,
		RunID:          claim.RunID,
		BaselineCommit: baseline,
		Result:         result,
	})
	if derr != nil {
		return coderca.RunStatusFailed, "", "deliver: " + derr.Error(), baseline, coderca.RCAResult{}
	}
	return coderca.RunStatusDone, resultRef, "delivered", baseline, result
}

// firstNonEmptyStr returns the first non-empty string from vals, or "" if all
// are empty. Used to prefer a CLI-echoed baseline over the source-prepared one.
func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// compile-time port assertions against the real implementations.
var (
	_ RunStore  = (*runstore.Store)(nil)
	_ CLIRunner = (*clirunner.Runner)(nil)
)
