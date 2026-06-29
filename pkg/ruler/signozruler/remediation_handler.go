package signozruler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/ruler/remediation"
	"github.com/SigNoz/signoz/pkg/ruler/remediationstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// RemediationRunner is the executor seam: the production *remediation.Executor
// satisfies it (built via the newRemediationExecutor factory), and tests inject
// a fake. Keeping it an interface decouples the handler from process spawning.
type RemediationRunner interface {
	Run(ctx context.Context, script string) remediation.ExecResult
}

// remediationListResponse is the GET /remediation wire shape.
type remediationListResponse struct {
	Remediations []ruletypes.RemediationExecution `json:"remediations"`
}

// requireOrg returns the caller's orgID or a render-ready auth error. Mirrors
// the orgID guard repeated across the SOP/runbook handlers (extractOrgID + the
// empty-string check) so the remediation handlers stay terse.
func requireOrg(req *http.Request) (string, error) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		return "", err
	}
	if orgID == "" {
		return "", errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims")
	}
	return orgID, nil
}

// remediationNotFound maps a store lookup error to a 404. The store does not
// expose a typed not-found sentinel, so any Get error here is treated as a
// missing row (the orgID guard already ran, so this is not an auth failure).
func remediationNotFound(err error) error {
	return errors.Newf(errors.TypeNotFound, errors.CodeNotFound, "remediation not found")
}

// GetRemediation handles GET /api/v2/ds/remediation/{id}.
func (h *handler) GetRemediation(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
		return
	}
	id := strings.TrimSpace(mux.Vars(req)["id"])
	e, err := h.remediationStore.Get(req.Context(), orgID, id)
	if err != nil {
		render.Error(rw, remediationNotFound(err))
		return
	}
	render.Success(rw, http.StatusOK, e)
}

// ListRemediations handles GET /api/v2/ds/remediation.
// Branches:
//   - scope=org  → ListByOrg (history view; status/sopId/limit params accepted)
//   - incidentId → ListByIncident (incident-scoped approval UI)
//   - default    → ListByStatus(proposed) (actionable default for approval UI)
func (h *handler) ListRemediations(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
		return
	}

	q := req.URL.Query()
	scope := strings.TrimSpace(q.Get("scope"))
	incidentID := strings.TrimSpace(q.Get("incidentId"))

	var list []ruletypes.RemediationExecution
	switch {
	case scope == "org":
		limit := 0
		if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
			if n, convErr := strconv.Atoi(raw); convErr == nil && n > 0 {
				limit = n
			}
		}
		list, err = h.remediationStore.ListByOrg(req.Context(), orgID, remediationstore.ListFilter{
			Status: strings.TrimSpace(q.Get("status")),
			SOPID:  strings.TrimSpace(q.Get("sopId")),
			Limit:  limit,
		})
	case incidentID != "":
		list, err = h.remediationStore.ListByIncident(req.Context(), orgID, incidentID)
	default:
		list, err = h.remediationStore.ListByStatus(req.Context(), orgID, ruletypes.RemediationStatusProposed)
	}
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "list remediations"))
		return
	}
	if list == nil {
		list = []ruletypes.RemediationExecution{}
	}
	render.Success(rw, http.StatusOK, remediationListResponse{Remediations: list})
}

// ApproveRemediation handles POST /api/v2/ds/remediation/{id}/approve.
// It enforces the master switch + concurrency cap, then uses the store's
// single-flight TransitionToExecuting guard so only the winning request starts
// the async script run (design §4.1: no double-execution).
func (h *handler) ApproveRemediation(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
		return
	}
	id := strings.TrimSpace(mux.Vars(req)["id"])

	cfg, err := h.remediationStore.GetConfig(req.Context(), orgID)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "load remediation config"))
		return
	}
	if !cfg.ExecutionEnabled {
		render.Error(rw, errors.Newf(errors.TypeForbidden, errors.CodeForbidden, "remediation execution is disabled for this org"))
		return
	}

	active, err := h.remediationStore.CountActiveByOrg(req.Context(), orgID)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "count active remediations"))
		return
	}
	if active >= cfg.MaxConcurrent {
		render.Error(rw, errors.Newf(errors.TypeTooManyRequests, errors.CodeTooManyRequests, "max concurrent remediation executions reached"))
		return
	}

	approvedBy := userDisplayNameFromCtx(req.Context())
	now := time.Now().UTC().Format(time.RFC3339)
	won, err := h.remediationStore.TransitionToExecuting(req.Context(), orgID, id, approvedBy, now, cfg.MaxConcurrent)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "transition remediation to executing"))
		return
	}
	if !won {
		// Lost the single-flight race (row was not in proposed/approved state).
		render.Error(rw, errors.Newf(errors.TypeAlreadyExists, errors.CodeAlreadyExists, "remediation is not in an approvable state"))
		return
	}

	e, err := h.remediationStore.Get(req.Context(), orgID, id)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "load remediation after transition"))
		return
	}

	execTimeout := time.Duration(cfg.ExecTimeoutSeconds) * time.Second
	runner := h.newRemediationExecutor(execTimeout)
	// Async: the HTTP response must not block on script runtime.
	go h.runRemediation(orgID, e, runner, execTimeout)

	render.Success(rw, http.StatusAccepted, e)
}

// runRemediation executes the frozen ScriptSnapshot and records the terminal-ish
// result (succeeded/failed). It runs on a detached context (the request context
// is already gone once the 202 was written) bounded to the configured timeout
// plus a small grace margin. The verifier (Task 9) later promotes
// succeeded→verified/unresolved.
func (h *handler) runRemediation(orgID string, e ruletypes.RemediationExecution, runner RemediationRunner, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout+5*time.Second)
	defer cancel()

	res := runner.Run(ctx, e.ScriptSnapshot)

	toStatus := ruletypes.RemediationStatusSucceeded
	if res.TimedOut || res.ExitCode != 0 {
		toStatus = ruletypes.RemediationStatusFailed
	}
	exit := res.ExitCode
	// OPERATOR CONTRACT: Approved runbook scripts must not print secrets to
	// stdout/stderr — OutputSnippet is stored unredacted and shown to Viewers.
	// Secret masking is a future extension point.
	_ = h.remediationStore.Transition(ctx, orgID, e.ID, toStatus, remediationstore.TransitionPatch{
		TerminalAt:    time.Now().UTC().Format(time.RFC3339),
		ExitCode:      &exit,
		OutputSnippet: truncateRemediationSnippet(res.Output),
	})
}

// truncateRemediationSnippet bounds the stored output to the audit snippet cap.
func truncateRemediationSnippet(s string) string {
	const n = ruletypes.RemediationMaxOutputSnippet
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// GetRemediationConfig handles GET /api/v2/ds/remediation/config. Admin-only
// (enforced by the route's AdminAccess guard): returns the org's auto-remediation
// master switch (ExecutionEnabled) + timing knobs, backfilled with defaults when
// no row exists. The SOP page shows this state and lets admins flip the toggle.
func (h *handler) GetRemediationConfig(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
		return
	}
	cfg, err := h.remediationStore.GetConfig(req.Context(), orgID)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "load remediation config"))
		return
	}
	render.Success(rw, http.StatusOK, cfg)
}

// UpdateRemediationConfig handles PUT /api/v2/ds/remediation/config. Admin-only:
// upserts the org's auto-remediation config. The ExecutionEnabled master switch
// is the primary knob the SOP-page toggle flips; numeric knobs are validated and
// backfilled with defaults so a toggle-only payload never zeroes the timing values.
func (h *handler) UpdateRemediationConfig(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
		return
	}
	var incoming ruletypes.RemediationConfig
	if err := binding.JSON.BindBody(req.Body, &incoming); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	incoming = incoming.WithDefaults()
	if err := ruletypes.ValidateRemediationConfig(incoming); err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "remediation config validation failed"))
		return
	}
	if err := h.remediationStore.UpsertConfig(req.Context(), orgID, incoming); err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "save remediation config"))
		return
	}
	render.Success(rw, http.StatusOK, incoming)
}

// RejectRemediation handles POST /api/v2/ds/remediation/{id}/reject. The store's
// Transition guards proposed→rejected; on success it returns the updated row.
func (h *handler) RejectRemediation(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
		return
	}
	id := strings.TrimSpace(mux.Vars(req)["id"])

	now := time.Now().UTC().Format(time.RFC3339)
	if err := h.remediationStore.Transition(req.Context(), orgID, id, ruletypes.RemediationStatusRejected, remediationstore.TransitionPatch{
		TerminalAt: now,
	}); err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "reject remediation"))
		return
	}
	e, err := h.remediationStore.Get(req.Context(), orgID, id)
	if err != nil {
		// Transition succeeded; row fetch is best-effort for response body.
		render.Success(rw, http.StatusOK, nil)
		return
	}
	render.Success(rw, http.StatusOK, e)
}
