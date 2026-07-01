package ruletypes

import (
	"errors"
	"fmt"
	"strings"
)

// Remediation lifecycle status enum. Terminal: verified, unresolved, failed,
// rejected, expired. Non-terminal: proposed, approved, executing, succeeded.
const (
	RemediationStatusProposed   = "proposed"
	RemediationStatusApproved   = "approved"
	RemediationStatusExecuting  = "executing"
	RemediationStatusSucceeded  = "succeeded"
	RemediationStatusFailed     = "failed"
	RemediationStatusVerified   = "verified"
	RemediationStatusUnresolved = "unresolved"
	RemediationStatusRejected   = "rejected"
	RemediationStatusExpired    = "expired"

	// Source enum: where the proposed script came from.
	RemediationSourceRunbook      = "runbook"
	RemediationSourceLLMGenerated = "llm-generated"
)

// RemediationMaxScriptLen mirrors RunbookMaxScriptLen — the snapshot is a copy
// of a Runbook's ExecutableScript.
const RemediationMaxScriptLen = RunbookMaxScriptLen

// RemediationMaxOutputSnippet caps the stored stdout/stderr snippet (audit only,
// not a full log). Secret masking is a future extension point.
const RemediationMaxOutputSnippet = 8_192

// RemediationExecution is one approve→execute→verify lifecycle for a single
// incident, tied to exactly one pre-approved Runbook. ScriptSnapshot is the
// frozen copy of the Runbook script taken at propose time, so SOP edits cannot
// change what an operator approved (design §4.2).
type RemediationExecution struct {
	ID               string `json:"id"`
	OrgID            string `json:"orgId"`
	IncidentID       string `json:"incidentId"`
	AlertFingerprint string `json:"alertFingerprint"`
	SOPID            string `json:"sopId"`
	SOPVersion       string `json:"sopVersion"`
	RunbookID        string `json:"runbookId"`
	// Source records the script origin: a pre-approved Runbook ("runbook", the
	// default for legacy rows) or an LLM-proposed fallback ("llm-generated").
	// Both execute under the same org gate + single approval; Source drives only
	// the cliaudit Via tag so LLM-script runs are separable in post-hoc audit.
	Source string `json:"source,omitempty"`
	// SelectionRationale is the LLM's reason for choosing this Runbook (or for
	// proposing a fallback). Shown on the approval card and stored for audit.
	SelectionRationale string `json:"selectionRationale,omitempty"`
	ScriptSnapshot     string `json:"scriptSnapshot"`
	Status           string `json:"status"`
	ProposedAt       string `json:"proposedAt"`
	ApprovedAt       string `json:"approvedAt,omitempty"`
	ExecutedAt       string `json:"executedAt,omitempty"`
	TerminalAt       string `json:"terminalAt,omitempty"`
	ApprovedBy       string `json:"approvedBy,omitempty"`
	ExitCode         *int   `json:"exitCode,omitempty"`
	OutputSnippet    string `json:"outputSnippet,omitempty"`
	VerifyResult     string `json:"verifyResult,omitempty"`
	ExpiresAt        string `json:"expiresAt"`

	// --- 타겟 파라미터 스냅샷 (propose 시 프리즈, design §3.2 B1) ---
	// TargetID 비어있으면 로컬 실행(하위호환). 비어있지 않으면 아래 스냅샷 값으로 SSH 원격 실행.
	// 실행 시 라이브 인벤토리와 비교하지 않는다(design §3.2 New-1); 라이브에서는
	// SealedCredential 한 필드만 로드한다.
	TargetID        string `json:"targetId,omitempty"`
	TargetHost      string `json:"targetHost,omitempty"`
	TargetPort      int    `json:"targetPort,omitempty"`
	TargetUser      string `json:"targetUser,omitempty"`
	TargetHostKeyFP string `json:"targetHostKeyFp,omitempty"`
	TargetName      string `json:"targetName,omitempty"`
}

var allowedRemediationSources = map[string]struct{}{
	RemediationSourceRunbook:      {},
	RemediationSourceLLMGenerated: {},
}

var allowedRemediationStatuses = map[string]struct{}{
	RemediationStatusProposed:   {},
	RemediationStatusApproved:   {},
	RemediationStatusExecuting:  {},
	RemediationStatusSucceeded:  {},
	RemediationStatusFailed:     {},
	RemediationStatusVerified:   {},
	RemediationStatusUnresolved: {},
	RemediationStatusRejected:   {},
	RemediationStatusExpired:    {},
}

var terminalRemediationStatuses = map[string]struct{}{
	RemediationStatusVerified:   {},
	RemediationStatusUnresolved: {},
	RemediationStatusFailed:     {},
	RemediationStatusRejected:   {},
	RemediationStatusExpired:    {},
}

// allowedRemediationTransitions maps each from-status to its permitted
// to-statuses. Absent keys (terminal states) permit no transition.
//
// v1 live path: proposed → executing (atomic via TransitionToExecuting SQL guard).
// The two-step proposed→approved→executing path is retired in v1; the
// RemediationStatusApproved constant is kept for forward-compatibility but the
// approved→executing and proposed→approved edges are intentionally absent here.
// approved as a from-status has no permitted targets (it is effectively dormant).
var allowedRemediationTransitions = map[string]map[string]struct{}{
	RemediationStatusProposed: {
		RemediationStatusExecuting: {}, RemediationStatusRejected: {}, RemediationStatusExpired: {},
	},
	RemediationStatusExecuting: {
		RemediationStatusSucceeded: {}, RemediationStatusFailed: {},
	},
	RemediationStatusSucceeded: {
		RemediationStatusVerified: {}, RemediationStatusUnresolved: {},
	},
}

func (e RemediationExecution) IsTerminal() bool {
	_, ok := terminalRemediationStatuses[e.Status]
	return ok
}

// ValidateRemediationStatusTransition mirrors ValidateRunbookStatusTransition:
// validity gate first, then no-op and illegal-direction rejection.
func ValidateRemediationStatusTransition(from, to string) error {
	if _, ok := allowedRemediationStatuses[from]; !ok {
		return fmt.Errorf("status transition: from %q invalid", from)
	}
	if _, ok := allowedRemediationStatuses[to]; !ok {
		return fmt.Errorf("status transition: to %q invalid", to)
	}
	if from == to {
		return fmt.Errorf("status transition: %q → %q is a no-op", from, to)
	}
	allowed, ok := allowedRemediationTransitions[from]
	if !ok {
		return fmt.Errorf("status transition: %q is terminal", from)
	}
	if _, ok := allowed[to]; !ok {
		return fmt.Errorf("status transition: %q → %q forbidden", from, to)
	}
	return nil
}

// ValidateRemediationExecution returns nil when e is well-formed, otherwise a
// joined field-level error. Style mirrors ValidateRunbook. ScriptSnapshot is
// NOT secret-like-checked (bash legitimately references $TOKEN env vars, same
// policy as Runbook.ExecutableScript).
func ValidateRemediationExecution(e RemediationExecution) error {
	var errs []error

	if !uuidV4Pattern.MatchString(strings.TrimSpace(e.ID)) {
		errs = append(errs, fmt.Errorf("id: must be UUID v4 (got %q)", e.ID))
	}
	pilotRequireNonEmpty(&errs, "orgId", e.OrgID)
	pilotRequireNonEmpty(&errs, "incidentId", e.IncidentID)
	pilotRequireNonEmpty(&errs, "sopId", e.SOPID)
	if e.Source == RemediationSourceLLMGenerated {
		// fallback scripts have no backing runbook (design §6.1)
		if strings.TrimSpace(e.RunbookID) != "" {
			errs = append(errs, fmt.Errorf("runbookId: must be empty for llm-generated source"))
		}
	} else {
		pilotRequireNonEmpty(&errs, "runbookId", e.RunbookID)
	}
	pilotRequireAllowed(&errs, "status", e.Status, allowedRemediationStatuses)
	if strings.TrimSpace(e.Source) != "" {
		pilotRequireAllowed(&errs, "source", e.Source, allowedRemediationSources)
	}
	pilotAppendSecretLikeStringErrors(&errs, "selectionRationale", e.SelectionRationale)
	if len(e.ScriptSnapshot) > RemediationMaxScriptLen {
		errs = append(errs, fmt.Errorf("scriptSnapshot: exceeds %d-byte limit (got %d)", RemediationMaxScriptLen, len(e.ScriptSnapshot)))
	}
	if strings.ContainsRune(e.ScriptSnapshot, 0) {
		errs = append(errs, fmt.Errorf("scriptSnapshot: must not contain NUL byte"))
	}
	pilotRequireNonEmpty(&errs, "proposedAt", e.ProposedAt)
	pilotRequireNonEmpty(&errs, "expiresAt", e.ExpiresAt)

	pilotAppendSecretLikeStringErrors(&errs, "approvedBy", e.ApprovedBy)

	if strings.TrimSpace(e.TargetID) != "" {
		if !uuidV4Pattern.MatchString(strings.TrimSpace(e.TargetID)) {
			errs = append(errs, fmt.Errorf("targetId: must be UUID v4 when set (got %q)", e.TargetID))
		}
		// 프리즈 무결성: 원격 실행은 접속 파라미터 스냅샷이 반드시 있어야 한다.
		pilotRequireNonEmpty(&errs, "targetHost", e.TargetHost)
		pilotRequireNonEmpty(&errs, "targetHostKeyFp", e.TargetHostKeyFP)
	}

	return errors.Join(errs...)
}
