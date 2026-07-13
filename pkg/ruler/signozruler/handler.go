package signozruler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/ruler"
	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/secretbox"
	codercarunstore "github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
	sqltemplatestore "github.com/SigNoz/signoz/pkg/ruler/incidentreport/sqltemplatestore"
	"github.com/SigNoz/signoz/pkg/ruler/remediation"
	"github.com/SigNoz/signoz/pkg/ruler/remediationstore"
	"github.com/SigNoz/signoz/pkg/ruler/remediationtargetstore"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
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
// factory into the handler. Kept as a post-construction setter (rather than a
// NewHandler arg) to avoid churning the long NewHandler signature; the
// apiserver provider calls this when the remediation feature is enabled.
func (h *handler) SetRemediationDeps(store remediationstore.Store, targetStore remediationtargetstore.Store, newExec func(time.Duration) RemediationRunner) {
	h.remediationStore = store
	h.remediationTargetStore = targetStore
	h.newRemediationExecutor = newExec
}

// NewHandler constructs a ruler HTTP handler. aiGenerator is the
// AIStrategyGenerator implementation injected by the caller; use
// aigenerator.New to build it from env-driven config.
// aiConfigStore, aiCipher, and aiRebuilder wire in the AI config CRUD
// endpoints; pass nil for each if those endpoints are not needed.
func NewHandler(
	ruler ruler.Ruler,
	sopStore ruletypes.SOPStore,
	aiHistoryStore ruletypes.AIStrategyHistoryStore,
	aiGenerator ruletypes.AIStrategyGenerator,
	aiConfigStore ruletypes.AIConfigStore,
	aiCipher *secretbox.Cipher,
	aiRebuilder aiGeneratorRebuilder,
	runbookDrafter ruletypes.RunbookDrafter,
	codebaseRepoStore ruletypes.CodebaseRepoStore,
	codebaseMapStore ruletypes.CodebaseServiceMapStore,
	codercaCfgStore ruletypes.CodebaseRCAConfigStore,
	codercaRunStore *codercarunstore.Store,
	aiCipherInsecure bool,
	reportTemplateStore *sqltemplatestore.Store,
	remediationStore remediationstore.Store,
	remediationTargetStore remediationtargetstore.Store,
	newRemediationExecutor func(time.Duration) RemediationRunner,
	remediationHealth *remediation.HealthChecker,
) ruler.Handler {
	return &handler{
		ruler:                  ruler,
		sopStore:               sopStore,
		aiHistoryStore:         aiHistoryStore,
		aiGenerator:            aiGenerator,
		aiConfigStore:          aiConfigStore,
		aiCipher:               aiCipher,
		aiRebuilder:            aiRebuilder,
		runbookDrafter:         runbookDrafter,
		codebaseRepoStore:      codebaseRepoStore,
		codebaseMapStore:       codebaseMapStore,
		codercaCfgStore:        codercaCfgStore,
		codercaRunStore:        codercaRunStore,
		aiCipherInsecure:       aiCipherInsecure,
		reportTemplateStore:    reportTemplateStore,
		remediationStore:       remediationStore,
		remediationTargetStore: remediationTargetStore,
		newRemediationExecutor: newRemediationExecutor,
		remediationHealth:      remediationHealth,
	}
}

// extractOrgID returns the OrgID from the SigNoz auth claims attached to
// ctx. Mirrors the TestRule handler (see :181 of this file) — the
// upstream auth error is preserved so callers can render it directly
// instead of synthesizing a fresh one.
func extractOrgID(ctx context.Context) (string, error) {
	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		return "", err
	}
	return claims.OrgID, nil
}

func (handler *handler) ListRules(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	rules, err := handler.ruler.ListRuleStates(ctx)
	if err != nil {
		render.Error(rw, err)
		return
	}

	view := make([]*ruletypes.Rule, 0, len(rules.Rules))
	for _, rule := range rules.Rules {
		view = append(view, ruletypes.NewRule(rule))
	}

	render.Success(rw, http.StatusOK, view)
}

func (handler *handler) GetRuleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	rule, err := handler.ruler.GetRule(ctx, id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, ruletypes.NewRule(rule))
}

func (handler *handler) CreateRule(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	rule, err := handler.ruler.CreateRule(ctx, string(body))
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusCreated, ruletypes.NewRule(rule))
}

func (handler *handler) UpdateRuleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	err = handler.ruler.EditRule(ctx, string(body), id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusNoContent, nil)
}

func (handler *handler) DeleteRuleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	err = handler.ruler.DeleteRule(ctx, id.StringValue())
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusNoContent, nil)
}

func (handler *handler) PatchRuleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	rule, err := handler.ruler.PatchRule(ctx, string(body), id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, ruletypes.NewRule(rule))
}

func (handler *handler) TestRule(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Minute)
	defer cancel()

	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		render.Error(rw, err)
		return
	}

	orgID, err := valuer.NewUUID(claims.OrgID)
	if err != nil {
		render.Error(rw, err)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	alertCount, err := handler.ruler.TestNotification(ctx, orgID, string(body))
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, ruletypes.GettableTestRule{AlertCount: alertCount, Message: "notification sent"})
}

func (handler *handler) PreviewNotificationTemplate(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	var previewReq ruletypes.PreviewNotificationTemplateRequest
	if err := binding.JSON.BindBody(req.Body, &previewReq); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	preview, err := ruletypes.PreviewNotificationTemplate(ctx, previewReq)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, preview)
}

func (handler *handler) PreviewSOP(rw http.ResponseWriter, req *http.Request) {
	var previewReq ruletypes.PreviewSOPRequest
	if err := binding.JSON.BindBody(req.Body, &previewReq); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	render.Success(rw, http.StatusOK, ruletypes.PreviewSOP(previewReq))
}

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

func (handler *handler) ListDowntimeSchedules(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		render.Error(rw, err)
		return
	}

	var params ruletypes.ListPlannedMaintenanceParams
	if err := binding.Query.BindQuery(req.URL.Query(), &params); err != nil {
		render.Error(rw, err)
		return
	}

	schedules, err := handler.ruler.MaintenanceStore().ListPlannedMaintenance(ctx, claims.OrgID)
	if err != nil {
		render.Error(rw, err)
		return
	}

	if params.Active != nil {
		activeSchedules := make([]*ruletypes.PlannedMaintenance, 0)
		for _, schedule := range schedules {
			now := time.Now().In(time.FixedZone(schedule.Schedule.Timezone, 0))
			if schedule.IsActive(now) == *params.Active {
				activeSchedules = append(activeSchedules, schedule)
			}
		}
		schedules = activeSchedules
	}

	if params.Recurring != nil {
		recurringSchedules := make([]*ruletypes.PlannedMaintenance, 0)
		for _, schedule := range schedules {
			if schedule.IsRecurring() == *params.Recurring {
				recurringSchedules = append(recurringSchedules, schedule)
			}
		}
		schedules = recurringSchedules
	}

	render.Success(rw, http.StatusOK, schedules)
}

func (handler *handler) GetDowntimeScheduleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	schedule, err := handler.ruler.MaintenanceStore().GetPlannedMaintenanceByID(ctx, id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, schedule)
}

func (handler *handler) CreateDowntimeSchedule(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	schedule := new(ruletypes.PostablePlannedMaintenance)
	if err := binding.JSON.BindBody(req.Body, schedule); err != nil {
		render.Error(rw, err)
		return
	}

	if err := schedule.Validate(); err != nil {
		render.Error(rw, err)
		return
	}

	created, err := handler.ruler.MaintenanceStore().CreatePlannedMaintenance(ctx, schedule)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusCreated, created)
}

func (handler *handler) UpdateDowntimeScheduleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	schedule := new(ruletypes.PostablePlannedMaintenance)
	if err := binding.JSON.BindBody(req.Body, schedule); err != nil {
		render.Error(rw, err)
		return
	}

	if err := schedule.Validate(); err != nil {
		render.Error(rw, err)
		return
	}

	err = handler.ruler.MaintenanceStore().UpdatePlannedMaintenance(ctx, schedule, id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusNoContent, nil)
}

func (handler *handler) DeleteDowntimeScheduleByID(rw http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
	defer cancel()

	id, err := valuer.NewUUID(mux.Vars(req)["id"])
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "id is not a valid uuid-v7"))
		return
	}

	err = handler.ruler.MaintenanceStore().DeletePlannedMaintenance(ctx, id)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusNoContent, nil)
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

func (handler *handler) CreateSOPDocument(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	var doc ruletypes.SOPDocument
	if err := binding.JSON.BindBody(req.Body, &doc); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	if err := ruletypes.ValidateSOPDocument(doc); err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "SOP document validation failed"))
		return
	}

	if err := handler.sopStore.Upsert(req.Context(), orgID, doc); err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "persist SOP document"))
		return
	}

	render.Success(rw, http.StatusCreated, doc)
}

func (handler *handler) ListSOPDocuments(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	docs, err := handler.sopStore.List(req.Context(), orgID)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "list SOP documents"))
		return
	}
	render.Success(rw, http.StatusOK, ruletypes.NewSOPDocumentListResponse(docs))
}

func (handler *handler) GetSOPDocument(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	sopID := strings.TrimSpace(mux.Vars(req)["sopId"])
	if sopID == "" {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	doc, err := handler.sopStore.GetLatest(req.Context(), orgID, sopID)
	if errors.Is(err, ruletypes.ErrSOPDocumentNotFound) {
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "fetch SOP document"))
		return
	}
	render.Success(rw, http.StatusOK, doc)
}

func (handler *handler) FetchSOPDocumentVersion(rw http.ResponseWriter, req *http.Request) {
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
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	doc, err := handler.sopStore.Get(req.Context(), orgID, sopID, version)
	if errors.Is(err, ruletypes.ErrSOPDocumentNotFound) {
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "fetch SOP document version"))
		return
	}
	render.Success(rw, http.StatusOK, doc)
}

func (handler *handler) PreviewSOPDocumentBinding(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	var previewReq ruletypes.SOPBindingPreviewRequest
	if err := binding.JSON.BindBody(req.Body, &previewReq); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	docs, err := handler.sopStore.List(req.Context(), orgID)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "list SOP documents for preview"))
		return
	}

	resp, err := ruletypes.PreviewSOPDocumentBinding(docs, previewReq)
	if err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "SOP binding preview validation failed"))
		return
	}
	if resp.Status == ruletypes.SOPBindingStatusForbidden {
		render.Error(rw, errors.New(errors.TypeForbidden, errors.CodeForbidden, ruletypes.SOPTenantPolicyDeniedWarning))
		return
	}

	render.Success(rw, http.StatusOK, resp)
}

func (handler *handler) PreviewAIStrategy(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	var strategyReq ruletypes.AIStrategyRequest
	if err := binding.JSON.BindBody(req.Body, &strategyReq); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	strategy, err := handler.aiGenerator.Generate(req.Context(), strategyReq)
	if err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "AI strategy preview validation failed"))
		return
	}

	record, recErr := ruletypes.NewAIStrategyHistoryRecord(strategy)
	if recErr == nil {
		if upsertErr := handler.aiHistoryStore.Upsert(req.Context(), orgID, record); upsertErr != nil {
			zap.L().Warn("ai history persist failed",
				zap.String("orgId", orgID),
				zap.String("strategyId", strategy.StrategyID),
				zap.Error(upsertErr),
			) //nolint:depguard
		}
	}

	render.Success(rw, http.StatusOK, strategy)
}

func (handler *handler) GetLatestAIStrategyHistory(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	lookupReq := ruletypes.AIStrategyHistoryLookupRequest{
		IncidentID:       req.URL.Query().Get("incidentId"),
		AlertFingerprint: req.URL.Query().Get("alertFingerprint"),
	}
	if err := ruletypes.ValidateAIStrategyHistoryLookup(lookupReq); err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "AI strategy history lookup validation failed"))
		return
	}

	record, ok, err := handler.aiHistoryStore.GetLatest(req.Context(), orgID, lookupReq)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "fetch AI strategy history"))
		return
	}
	if !ok {
		render.Error(rw, errors.NewNotFoundf(errors.CodeNotFound, "AI strategy history was not found"))
		return
	}

	render.Success(rw, http.StatusOK, record)
}
