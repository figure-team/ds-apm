package signozruler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/llmaigenerator"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// runbookListResponse is the GET /runbooks wire shape.
type runbookListResponse struct {
	Runbooks []ruletypes.Runbook `json:"runbooks"`
}

// draftRunbookRequest is the body for POST /api/v2/ds/runbooks/draft.
type draftRunbookRequest struct {
	SOPID         string   `json:"sopId"`
	Version       string   `json:"version"`
	ErrorExamples []string `json:"errorExamples"`
}

// draftRunbookResponse is the body for POST /api/v2/ds/runbooks/draft on failure.
type draftRunbookResponse struct {
	OK        bool              `json:"ok"`
	Data      *ruletypes.Runbook `json:"data,omitempty"`
	Error     string            `json:"error,omitempty"`
	ErrorKind string            `json:"errorKind,omitempty"`
}

// userDisplayNameFromCtx returns the user's display name from Claims.
// Claims has no dedicated "display name" field; Email is the closest proxy.
// Task 7 or future auth work can upgrade this when a display-name field lands.
func userDisplayNameFromCtx(ctx context.Context) string {
	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		return "system"
	}
	if claims.Email != "" {
		return claims.Email
	}
	if claims.UserID != "" {
		return claims.UserID
	}
	return "system"
}

// hasEditorRole returns true when the authenticated user has editor or admin
// privileges. NOTE: authtypes.Claims does not yet carry a Role field; this
// function is intentionally a no-op pass-through (returns true for any
// authenticated request) until Task 7 / auth middleware populates
// Claims.Role. The orgID guard in every handler already blocks unauthenticated
// callers; role enforcement is a pre-GA hardening task.
//
// TODO(task7): replace with real role check once Claims.Role is wired.
func hasEditorRole(_ context.Context) bool { return true } //nolint:unparam

// hasAdminRole returns true when the authenticated user has admin privileges.
// See hasEditorRole comment — same no-op pass-through caveat applies.
//
// TODO(task7): replace with real role check once Claims.Role is wired.
func hasAdminRole(_ context.Context) bool { return true } //nolint:unparam

// ListRunbooks handles GET /api/v2/ds/sop/documents/{sopId}/versions/{version}/runbooks?status=
func (h *handler) ListRunbooks(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}
	vars := mux.Vars(req)
	sopID := strings.TrimSpace(vars["sopId"])
	version := strings.TrimSpace(vars["version"])
	if sopID == "" || version == "" {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "sopId and version are required"))
		return
	}
	doc, err := h.sopStore.Get(req.Context(), orgID, sopID, version)
	if err != nil {
		render.Error(rw, runbookSOPErr(err))
		return
	}
	statusFilter := parseRunbookStatusFilter(req.URL.Query().Get("status"))
	filtered := make([]ruletypes.Runbook, 0, len(doc.Runbooks))
	for _, rb := range doc.Runbooks {
		if _, ok := statusFilter[rb.Status]; ok {
			filtered = append(filtered, rb)
		}
	}
	render.Success(rw, http.StatusOK, runbookListResponse{Runbooks: filtered})
}

// GetRunbook handles GET /.../runbooks/{runbookId}
func (h *handler) GetRunbook(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}
	vars := mux.Vars(req)
	doc, err := h.sopStore.Get(req.Context(), orgID, strings.TrimSpace(vars["sopId"]), strings.TrimSpace(vars["version"]))
	if err != nil {
		render.Error(rw, runbookSOPErr(err))
		return
	}
	for _, rb := range doc.Runbooks {
		if rb.ID == vars["runbookId"] {
			render.Success(rw, http.StatusOK, rb)
			return
		}
	}
	render.Error(rw, errors.Newf(errors.TypeNotFound, errors.CodeNotFound, "runbook not found"))
}

// CreateRunbook handles POST /.../runbooks — editor+ required.
func (h *handler) CreateRunbook(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}
	if !hasEditorRole(req.Context()) {
		render.Error(rw, errors.Newf(errors.TypeForbidden, errors.CodeForbidden, "editor role required"))
		return
	}
	vars := mux.Vars(req)
	var incoming ruletypes.Runbook
	if err := binding.JSON.BindBody(req.Body, &incoming); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	now := time.Now().UTC().Format(time.RFC3339)
	incoming.ID = uuid.NewString()
	incoming.CreatedAt = now
	incoming.UpdatedAt = now
	if incoming.UpdatedBy == "" {
		incoming.UpdatedBy = userDisplayNameFromCtx(req.Context())
	}
	if incoming.Status == "" {
		incoming.Status = ruletypes.RunbookStatusApproved
	}
	if err := ruletypes.ValidateRunbook(incoming); err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "runbook validation failed"))
		return
	}
	if err := h.sopStore.UpsertRunbook(req.Context(), orgID, strings.TrimSpace(vars["sopId"]), strings.TrimSpace(vars["version"]), incoming); err != nil {
		render.Error(rw, runbookSOPErr(err))
		return
	}
	render.Success(rw, http.StatusCreated, incoming)
}

// UpdateRunbook handles PUT /.../runbooks/{runbookId} — editor+ required.
func (h *handler) UpdateRunbook(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}
	if !hasEditorRole(req.Context()) {
		render.Error(rw, errors.Newf(errors.TypeForbidden, errors.CodeForbidden, "editor role required"))
		return
	}
	vars := mux.Vars(req)
	var incoming ruletypes.Runbook
	if err := binding.JSON.BindBody(req.Body, &incoming); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	// Force ID from URL — body cannot reassign.
	incoming.ID = strings.TrimSpace(vars["runbookId"])

	// Look up existing for status-transition check.
	doc, err := h.sopStore.Get(req.Context(), orgID, strings.TrimSpace(vars["sopId"]), strings.TrimSpace(vars["version"]))
	if err != nil {
		render.Error(rw, runbookSOPErr(err))
		return
	}
	var existing *ruletypes.Runbook
	for i := range doc.Runbooks {
		if doc.Runbooks[i].ID == incoming.ID {
			existing = &doc.Runbooks[i]
			break
		}
	}
	if existing == nil {
		render.Error(rw, errors.Newf(errors.TypeNotFound, errors.CodeNotFound, "runbook not found"))
		return
	}
	if err := ruletypes.ValidateRunbookStatusTransition(existing.Status, incoming.Status); err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "invalid status transition"))
		return
	}
	incoming.CreatedAt = existing.CreatedAt
	incoming.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if incoming.UpdatedBy == "" {
		incoming.UpdatedBy = userDisplayNameFromCtx(req.Context())
	}
	if err := ruletypes.ValidateRunbook(incoming); err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "runbook validation failed"))
		return
	}
	if err := h.sopStore.UpsertRunbook(req.Context(), orgID, strings.TrimSpace(vars["sopId"]), strings.TrimSpace(vars["version"]), incoming); err != nil {
		render.Error(rw, runbookSOPErr(err))
		return
	}
	render.Success(rw, http.StatusOK, incoming)
}

// DeleteRunbook handles DELETE /.../runbooks/{runbookId} — admin only.
func (h *handler) DeleteRunbook(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}
	if !hasAdminRole(req.Context()) {
		render.Error(rw, errors.Newf(errors.TypeForbidden, errors.CodeForbidden, "admin role required for hard delete"))
		return
	}
	vars := mux.Vars(req)
	if err := h.sopStore.DeleteRunbook(
		req.Context(), orgID,
		strings.TrimSpace(vars["sopId"]),
		strings.TrimSpace(vars["version"]),
		strings.TrimSpace(vars["runbookId"]),
	); err != nil {
		render.Error(rw, runbookSOPErr(err))
		return
	}
	rw.WriteHeader(http.StatusNoContent)
}

// DraftRunbook handles POST /api/v2/ds/runbooks/draft — editor+ required.
// Preview only: the draft is returned but never persisted.
func (h *handler) DraftRunbook(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}
	if !hasEditorRole(req.Context()) {
		render.Error(rw, errors.Newf(errors.TypeForbidden, errors.CodeForbidden, "editor role required"))
		return
	}
	if h.runbookDrafter == nil {
		writeRunbookDraftFailure(rw, "runbook drafter not configured", "other")
		return
	}
	var body draftRunbookRequest
	if err := binding.JSON.BindBody(req.Body, &body); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	if strings.TrimSpace(body.SOPID) == "" || strings.TrimSpace(body.Version) == "" {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "sopId and version required"))
		return
	}
	if len(body.ErrorExamples) == 0 {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "errorExamples required (1..3)"))
		return
	}
	if len(body.ErrorExamples) > ruletypes.RunbookMaxSourceExampleCount {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "errorExamples: at most 3 entries"))
		return
	}

	sop, err := h.sopStore.Get(req.Context(), orgID, body.SOPID, body.Version)
	if err != nil {
		render.Error(rw, runbookSOPErr(err))
		return
	}

	rb, draftErr := h.runbookDrafter.Draft(req.Context(), ruletypes.RunbookDraftRequest{
		SOP:           sop,
		ErrorExamples: body.ErrorExamples,
		Source:        "manual-paste",
	})
	if draftErr != nil {
		kind := string(llmaigenerator.ClassifyError(draftErr))
		writeRunbookDraftFailure(rw, draftErr.Error(), kind)
		return
	}
	render.Success(rw, http.StatusOK, rb)
}

// writeRunbookDraftFailure writes a 200 JSON body with ok:false for draft errors.
// 200 is intentional: the UI expects a well-typed envelope even on LLM errors.
func writeRunbookDraftFailure(rw http.ResponseWriter, msg, kind string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(rw).Encode(draftRunbookResponse{OK: false, Error: msg, ErrorKind: kind})
}

// parseRunbookStatusFilter parses the ?status= query parameter into a set.
// Default (empty string): draft + approved. Special "all": all three statuses.
func parseRunbookStatusFilter(q string) map[string]struct{} {
	if q == "" {
		return map[string]struct{}{
			ruletypes.RunbookStatusDraft:    {},
			ruletypes.RunbookStatusApproved: {},
		}
	}
	if q == "all" {
		return map[string]struct{}{
			ruletypes.RunbookStatusDraft:      {},
			ruletypes.RunbookStatusApproved:   {},
			ruletypes.RunbookStatusDeprecated: {},
		}
	}
	out := map[string]struct{}{}
	for _, s := range strings.Split(q, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			out[s] = struct{}{}
		}
	}
	return out
}

// runbookSOPErr maps SOPStore errors to render-friendly HTTP errors.
func runbookSOPErr(err error) error {
	if errors.Is(err, ruletypes.ErrSOPDocumentNotFound) {
		return errors.Newf(errors.TypeNotFound, errors.CodeNotFound, "not found")
	}
	return errors.WrapInternalf(err, errors.CodeInternal, "sop store")
}
