package signozruler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// SetManagedMarkdownDisabled toggles the in-process managed_markdown rollback flag.
//
// This handler-level toggle is intentionally separate from
// ruletypes.PilotConfiguration.Enabled (the contract-level config field) for
// PoC scope: wiring the handler to read PilotConfiguration at request time
// requires a config-provider scaffold that will land in Phase 4 cockpit work.
// For now, operators flip this flag via a future admin endpoint or directly
// in tests; the contract-level field remains the canonical schema and will
// unify the toggle path once the cockpit ships.
func (handler *handler) SetManagedMarkdownDisabled(v bool) {
	handler.managedMarkdownDisabled.Store(v)
}

func (handler *handler) FetchPilotManagedMarkdownSOP(rw http.ResponseWriter, req *http.Request) {
	if handler.managedMarkdownDisabled.Load() {
		zap.L().Info("managed markdown SOP fetch rejected — administratively disabled",
			zap.String("path", req.URL.Path),
			zap.String("remote", req.RemoteAddr))
		http.Error(rw, "managed markdown SOP fetch is administratively disabled", http.StatusServiceUnavailable)
		return
	}

	var fetchReq ruletypes.PilotManagedMarkdownSOPFetchRequest
	if err := binding.JSON.BindBody(req.Body, &fetchReq); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	resp, err := ruletypes.FetchPilotManagedMarkdownSOP(fetchReq.Source, fetchReq.Fetch)
	if err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "pilot managed markdown SOP fetch validation failed"))
		return
	}
	if err := ruletypes.DispatchPilotAuditEvent(req.Context(), resp.AuditEvent); err != nil {
		zap.L().Warn("pilot managed markdown SOP audit dispatch failed",
			zap.Error(err),
			zap.String("audit_event_id", resp.AuditEvent.EventID))
	}

	render.Success(rw, http.StatusOK, resp)
}

const pilotManagedMarkdownDefaultSourceID = "src-managed-markdown-default"

// pilotManagedMarkdownDefaultSource returns the canonical live managed_markdown
// catalog entry that matches what FetchPilotManagedMarkdownSOP serves. The
// constructor returns a fresh struct on every call so handlers cannot share
// mutable package-level state.
func pilotManagedMarkdownDefaultSource() ruletypes.PilotManagedMarkdownSource {
	return ruletypes.PilotManagedMarkdownSource{
		SourceID:              pilotManagedMarkdownDefaultSourceID,
		DisplayName:           "Managed Markdown SOP Registry",
		Status:                ruletypes.PilotSOPSourceStatusHealthy,
		ServiceAccountProfile: "ds-sop-reader",
		TenantScope: ruletypes.PilotTenantScope{
			ProjectIDs:   []string{"customer-a"},
			Environments: []string{"prod"},
		},
	}
}

func (handler *handler) ListPilotSOPSources(rw http.ResponseWriter, req *http.Request) {
	resp, err := ruletypes.NewPilotManagedMarkdownCatalog([]ruletypes.PilotManagedMarkdownSource{
		pilotManagedMarkdownDefaultSource(),
	})
	if err != nil {
		zap.L().Error("pilot sop source catalog validation failed", zap.Error(err))
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(resp); err != nil {
		zap.L().Warn("pilot sop source catalog encode failed", zap.Error(err))
	}
}

func (handler *handler) GetPilotSOPSourceHealth(rw http.ResponseWriter, req *http.Request) {
	id := strings.TrimSpace(mux.Vars(req)["id"])
	if id == "" {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if id != pilotManagedMarkdownDefaultSourceID {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	checkedAt := time.Now().UTC().Format(time.RFC3339)
	resp, err := ruletypes.NewPilotManagedMarkdownHealth(pilotManagedMarkdownDefaultSource(), checkedAt)
	if err != nil {
		zap.L().Error("pilot sop source health validation failed", zap.String("sourceId", id), zap.Error(err))
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(rw).Encode(resp); err != nil {
		zap.L().Warn("pilot sop source health encode failed", zap.String("sourceId", id), zap.Error(err))
	}
}
