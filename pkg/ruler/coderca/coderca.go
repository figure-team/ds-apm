// Package coderca implements CF-11: AI codebase root-cause analysis.
//
// When an alert fires that has no matching SOP (CF-1 "unbound") and is
// anomalous (CF-7), coderca drives a CLI coding agent (claude/codex) to
// explore the offending service's source and produce a root-cause + fix
// SUGGESTION for human review (HITL — never auto-applied).
//
// Cost/volume containment is the #1 design driver (this deployment has a
// quota-runaway history). All volume enforcement is atomic and DB-backed;
// see docs/superpowers/specs/2026-06-11-cf11-code-rca-design.md for the
// full design and its adversarial-review audit trail.
//
// This package holds only new code. Shared-file wirings (dispatch-hook
// trigger, router registration, server construction) are left as documented
// seams (design §11) and are NOT edited from this worktree.
package coderca

// RunStatus is the lifecycle state of a single RCA run (coderca_run).
type RunStatus string

const (
	RunStatusQueued      RunStatus = "queued"
	RunStatusRunning     RunStatus = "running"
	RunStatusDone        RunStatus = "done"
	RunStatusFailed      RunStatus = "failed"
	RunStatusTimeout     RunStatus = "timeout"
	RunStatusUnparseable RunStatus = "unparseable"
)

// SkipReason explains why a candidate signal did not produce a run. Skip
// reasons are aggregated into counters (design §6.4) rather than one row per
// rejected alert — persisting a row per rejection would amplify writes under
// the exact flood the gates exist to survive.
type SkipReason string

const (
	SkipFeatureOff       SkipReason = "feature_off"
	SkipNoAnomaly        SkipReason = "no_anomaly"
	SkipBelowSeverity    SkipReason = "below_severity"
	SkipNoRepoMapping    SkipReason = "no_repo_mapping"
	SkipDeduped          SkipReason = "deduped"
	SkipBudgetExhausted  SkipReason = "budget_exhausted"
	SkipQueueFull        SkipReason = "queue_full"
)
