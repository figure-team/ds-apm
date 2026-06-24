package remediationstore

import (
	"context"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// Store persists remediation executions and per-org config.
type Store interface {
	Create(ctx context.Context, e ruletypes.RemediationExecution) error
	Get(ctx context.Context, orgID, id string) (ruletypes.RemediationExecution, error)
	ListByIncident(ctx context.Context, orgID, incidentID string) ([]ruletypes.RemediationExecution, error)
	ListByStatus(ctx context.Context, orgID, status string) ([]ruletypes.RemediationExecution, error)
	// TransitionToExecuting atomically moves a proposed row to executing,
	// stamping approver+time. Returns true iff this call won the race (affected
	// rows == 1). Guards against double-execution (design §4.1).
	TransitionToExecuting(ctx context.Context, orgID, id, approvedBy, approvedAt string) (bool, error)
	// Transition applies a validated status change, stamping result fields.
	// Used by reject/executor-result/verifier. patch carries only the fields
	// relevant to the target status (exit code, output, verify result, terminal time).
	Transition(ctx context.Context, orgID, id, toStatus string, patch TransitionPatch) error
	CountActiveByOrg(ctx context.Context, orgID string) (int64, error) // approved+executing
	GetConfig(ctx context.Context, orgID string) (ruletypes.RemediationConfig, error)
}

// TransitionPatch carries optional result fields for a status transition. Empty
// fields are not written.
type TransitionPatch struct {
	TerminalAt    string
	ExitCode      *int
	OutputSnippet string
	VerifyResult  string
	ExecutedAt    string
}
