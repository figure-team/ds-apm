package signozruler

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler"
	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/secretbox"
	codercarunstore "github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
	sqltemplatestore "github.com/SigNoz/signoz/pkg/ruler/incidentreport/sqltemplatestore"
	"github.com/SigNoz/signoz/pkg/ruler/remediation"
	"github.com/SigNoz/signoz/pkg/ruler/remediationstore"
	"github.com/SigNoz/signoz/pkg/ruler/remediationtargetstore"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

type handler struct {
	ruler                   ruler.Ruler
	managedMarkdownDisabled atomic.Bool
	sopStore                ruletypes.SOPStore
	aiHistoryStore          ruletypes.AIStrategyHistoryStore
	aiGenerator             ruletypes.AIStrategyGenerator
	// AI config CRUD fields (added for AI Module Settings page).
	aiConfigStore ruletypes.AIConfigStore
	aiCipher      *secretbox.Cipher
	aiRebuilder   aiGeneratorRebuilder
	// Runbook drafter (Task 7 wires this into NewHandler; Task 6 added the field).
	runbookDrafter ruletypes.RunbookDrafter
	// CF-11 code RCA settings + run history (coderca_handler.go).
	codebaseRepoStore ruletypes.CodebaseRepoStore
	codebaseMapStore  ruletypes.CodebaseServiceMapStore
	codercaCfgStore   ruletypes.CodebaseRCAConfigStore
	codercaRunStore   *codercarunstore.Store
	aiCipherInsecure  bool
	// Incident report 양식 template store (incident_report_handler.go).
	reportTemplateStore *sqltemplatestore.Store
	// Remediation execution (remediation_handler.go). remediationStore persists
	// the approve→execute→verify lifecycle; remediationTargetStore resolves a
	// frozen TargetID to its live SealedCredential at execute time (design §3.1,
	// §3.2); newRemediationExecutor is a factory that builds a per-run executor
	// bound to the org's configured timeout (the factory seam keeps the executor
	// fake-able in tests). All three are wired via SetRemediationDeps; nil until
	// then (remediationTargetStore stays nil in production until Task 13 wires it,
	// which is fine — no remote execution is stamped without it, see runRemediation).
	remediationStore       remediationstore.Store
	remediationTargetStore remediationtargetstore.Store
	newRemediationExecutor func(timeout time.Duration) RemediationRunner
	// remediationHealth merges per-target health into the targets list and is
	// poked after create/update. Concrete pointer on purpose — an interface
	// here would revive the typed-nil trap; all methods are nil-receiver safe
	// so nil (unwired) simply reads as fail-open unknown (spec §2.4).
	remediationHealth *remediation.HealthChecker
}

// SetRemediationDeps wires the remediation store, target store, and executor
// factory into the handler after construction. Deps carries the same three
// fields for callers that have them up front; this setter stays because the
// apiserver provider only resolves them once the remediation feature is
// enabled, which is after the handler exists.
func (h *handler) SetRemediationDeps(store remediationstore.Store, targetStore remediationtargetstore.Store, newExec func(time.Duration) RemediationRunner) {
	h.remediationStore = store
	h.remediationTargetStore = targetStore
	h.newRemediationExecutor = newExec
}

// Deps carries everything NewHandler needs to build a handler. It replaced an
// 18-argument positional signature: several of the fields share an interface
// shape (the AI/codebase stores in particular), so a transposed argument would
// still compile. Named fields make the wiring in pkg/signoz/handler.go legible
// and let new dependencies land without touching the call sites that do not
// care about them.
//
// A zero value is meaningful: every field is optional and the handlers that
// need one guard against nil, so callers only set what they wire.
type Deps struct {
	Ruler ruler.Ruler

	SOPStore ruletypes.SOPStore

	// AIHistoryStore/AIGenerator back the AI strategy preview + history endpoints.
	AIHistoryStore ruletypes.AIStrategyHistoryStore
	AIGenerator    ruletypes.AIStrategyGenerator

	// AIConfigStore, AICipher, and AIRebuilder wire in the AI config CRUD
	// endpoints; leave them nil if those endpoints are not needed.
	AIConfigStore    ruletypes.AIConfigStore
	AICipher         *secretbox.Cipher
	AIRebuilder      aiGeneratorRebuilder
	AICipherInsecure bool

	RunbookDrafter ruletypes.RunbookDrafter

	// CF-11 code RCA settings + run history.
	CodebaseRepoStore ruletypes.CodebaseRepoStore
	CodebaseMapStore  ruletypes.CodebaseServiceMapStore
	CodeRCACfgStore   ruletypes.CodebaseRCAConfigStore
	CodeRCARunStore   *codercarunstore.Store

	ReportTemplateStore *sqltemplatestore.Store

	// Remediation execution. SetRemediationDeps sets the same three fields
	// post-construction when the apiserver enables the feature later.
	RemediationStore       remediationstore.Store
	RemediationTargetStore remediationtargetstore.Store
	NewRemediationExecutor func(timeout time.Duration) RemediationRunner
	RemediationHealth      *remediation.HealthChecker
}

// NewHandler constructs a ruler HTTP handler from deps. AIGenerator is the
// AIStrategyGenerator implementation injected by the caller; use
// aigenerator.New to build it from env-driven config.
func NewHandler(deps Deps) ruler.Handler {
	return &handler{
		ruler:                  deps.Ruler,
		sopStore:               deps.SOPStore,
		aiHistoryStore:         deps.AIHistoryStore,
		aiGenerator:            deps.AIGenerator,
		aiConfigStore:          deps.AIConfigStore,
		aiCipher:               deps.AICipher,
		aiRebuilder:            deps.AIRebuilder,
		runbookDrafter:         deps.RunbookDrafter,
		codebaseRepoStore:      deps.CodebaseRepoStore,
		codebaseMapStore:       deps.CodebaseMapStore,
		codercaCfgStore:        deps.CodeRCACfgStore,
		codercaRunStore:        deps.CodeRCARunStore,
		aiCipherInsecure:       deps.AICipherInsecure,
		reportTemplateStore:    deps.ReportTemplateStore,
		remediationStore:       deps.RemediationStore,
		remediationTargetStore: deps.RemediationTargetStore,
		newRemediationExecutor: deps.NewRemediationExecutor,
		remediationHealth:      deps.RemediationHealth,
	}
}

// extractOrgID returns the OrgID from the SigNoz auth claims attached to
// ctx. Mirrors the TestRule handler (rule_handler.go) — the
// upstream auth error is preserved so callers can render it directly
// instead of synthesizing a fresh one.
func extractOrgID(ctx context.Context) (string, error) {
	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		return "", err
	}
	return claims.OrgID, nil
}
