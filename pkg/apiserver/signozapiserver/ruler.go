package signozapiserver

import (
	"net/http"

	"github.com/SigNoz/signoz/pkg/http/handler"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/gorilla/mux"
)

func (provider *provider) addRulerRoutes(router *mux.Router) error {
	if err := router.Handle("/api/v2/rules", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.ListRules), handler.OpenAPIDef{
		ID:                  "ListRules",
		Tags:                []string{"rules"},
		Summary:             "List alert rules",
		Description:         "This endpoint lists all alert rules with their current evaluation state",
		Response:            make([]*ruletypes.Rule, 0),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/{id}", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.GetRuleByID), handler.OpenAPIDef{
		ID:                  "GetRuleByID",
		Tags:                []string{"rules"},
		Summary:             "Get alert rule by ID",
		Description:         "This endpoint returns an alert rule by ID",
		Response:            new(ruletypes.Rule),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusNotFound},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules", handler.New(provider.authZ.EditAccess(provider.rulerHandler.CreateRule), handler.OpenAPIDef{
		ID:                  "CreateRule",
		Tags:                []string{"rules"},
		Summary:             "Create alert rule",
		Description:         "This endpoint creates a new alert rule",
		Request:             new(ruletypes.PostableRule),
		RequestContentType:  "application/json",
		RequestExamples:     postableRuleExamples(),
		Response:            new(ruletypes.Rule),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusCreated,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/{id}", handler.New(provider.authZ.EditAccess(provider.rulerHandler.UpdateRuleByID), handler.OpenAPIDef{
		ID:                 "UpdateRuleByID",
		Tags:               []string{"rules"},
		Summary:            "Update alert rule",
		Description:        "This endpoint updates an alert rule by ID",
		Request:            new(ruletypes.PostableRule),
		RequestContentType: "application/json",
		RequestExamples:    postableRuleExamples(),
		SuccessStatusCode:  http.StatusNoContent,
		ErrorStatusCodes:   []int{http.StatusBadRequest, http.StatusNotFound},
		SecuritySchemes:    newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPut).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/{id}", handler.New(provider.authZ.EditAccess(provider.rulerHandler.DeleteRuleByID), handler.OpenAPIDef{
		ID:                "DeleteRuleByID",
		Tags:              []string{"rules"},
		Summary:           "Delete alert rule",
		Description:       "This endpoint deletes an alert rule by ID",
		SuccessStatusCode: http.StatusNoContent,
		ErrorStatusCodes:  []int{http.StatusNotFound},
		SecuritySchemes:   newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodDelete).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/{id}", handler.New(provider.authZ.EditAccess(provider.rulerHandler.PatchRuleByID), handler.OpenAPIDef{
		ID:                  "PatchRuleByID",
		Tags:                []string{"rules"},
		Summary:             "Patch alert rule",
		Description:         "This endpoint applies a partial update to an alert rule by ID",
		Request:             new(ruletypes.PostableRule),
		RequestContentType:  "application/json",
		RequestExamples:     postableRuleExamples(),
		Response:            new(ruletypes.Rule),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusNotFound},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPatch).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/test", handler.New(provider.authZ.EditAccess(provider.rulerHandler.TestRule), handler.OpenAPIDef{
		ID:                  "TestRule",
		Tags:                []string{"rules"},
		Summary:             "Test alert rule",
		Description:         "This endpoint fires a test notification for the given rule definition",
		Request:             new(ruletypes.PostableRule),
		RequestContentType:  "application/json",
		RequestExamples:     postableRuleExamples(),
		Response:            new(ruletypes.GettableTestRule),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/notification_template/preview", handler.New(provider.authZ.EditAccess(provider.rulerHandler.PreviewNotificationTemplate), handler.OpenAPIDef{
		ID:                  "PreviewNotificationTemplate",
		Tags:                []string{"rules"},
		Summary:             "Preview alert notification template",
		Description:         "This endpoint renders a notification message template against the rule labels and annotations without sending a notification",
		Request:             new(ruletypes.PreviewNotificationTemplateRequest),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.PreviewNotificationTemplateResponse),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/sop/preview", handler.New(provider.authZ.EditAccess(provider.rulerHandler.PreviewSOP), handler.OpenAPIDef{
		ID:                  "PreviewSOP",
		Tags:                []string{"rules"},
		Summary:             "Preview alert SOP binding",
		Description:         "This endpoint previews SOP source, binding, search, preview metadata, and service-account/API auth boundary guidance for an alert rule without fetching an external SOP document",
		Request:             new(ruletypes.PreviewSOPRequest),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.PreviewSOPResponse),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/rules/sop/pilot/managed_markdown/fetch", handler.New(provider.authZ.EditAccess(provider.rulerHandler.FetchPilotManagedMarkdownSOP), handler.OpenAPIDef{
		ID:                  "FetchPilotManagedMarkdownSOP",
		Tags:                []string{"rules"},
		Summary:             "Fetch a managed Markdown SOP for pilot validation",
		Description:         "This pilot endpoint fetches an inline managed Markdown SOP source only after the audit-required server-side contract is accepted; it does not persist sources, call external connectors, or accept browser credentials",
		Request:             new(ruletypes.PilotManagedMarkdownSOPFetchRequest),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.PilotSOPFetchResponse),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/sop/sources", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.ListPilotSOPSources), handler.OpenAPIDef{
		ID:                  "ListPilotSOPSources",
		Tags:                []string{"rules"},
		Summary:             "List SOP sources",
		Description:         "This endpoint returns the catalog of registered SOP sources",
		Response:            new(ruletypes.PilotSOPSourceCatalogResponse),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/sop/sources/{id}/health", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.GetPilotSOPSourceHealth), handler.OpenAPIDef{
		ID:                  "GetPilotSOPSourceHealth",
		Tags:                []string{"rules"},
		Summary:             "Get SOP source health",
		Description:         "This endpoint probes the health of a SOP source by ID",
		Response:            new(ruletypes.PilotSOPSourceHealthResponse),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/sop/documents", handler.New(provider.authZ.EditAccess(provider.rulerHandler.CreateSOPDocument), handler.OpenAPIDef{
		ID:                  "CreateSOPDocument",
		Tags:                []string{"rules"},
		Summary:             "Create a managed SOP document",
		Description:         "This endpoint registers an approved ds.sop_document.v1 document for SigNoz-native DS-APM AI+SOP response strategy flows",
		Request:             new(ruletypes.SOPDocument),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.SOPDocument),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusCreated,
		ErrorStatusCodes:    []int{http.StatusBadRequest},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/sop/documents/batch", handler.New(provider.authZ.EditAccess(provider.rulerHandler.CreateSOPDocumentBatch), handler.OpenAPIDef{
		ID:                  "CreateSOPDocumentBatch",
		Tags:                []string{"rules"},
		Summary:             "Batch create SOP documents",
		Description:         "Registers multiple ds.sop_document.v1 documents in a single request; each document is validated independently — valid ones are stored, invalid ones are reported with error details",
		Request:             new(ruletypes.SOPDocumentBatchRequest),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.SOPDocumentBatchResponse),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/sop/documents", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.ListSOPDocuments), handler.OpenAPIDef{
		ID:                  "ListSOPDocuments",
		Tags:                []string{"rules"},
		Summary:             "List managed SOP documents",
		Description:         "This endpoint lists registered SOP document summaries without returning markdown bodies",
		Response:            new(ruletypes.SOPDocumentListResponse),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusUnauthorized},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/sop/documents/{sopId}", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.GetSOPDocument), handler.OpenAPIDef{
		ID:                  "GetSOPDocument",
		Tags:                []string{"rules"},
		Summary:             "Get latest SOP document",
		Description:         "This endpoint returns the latest registered version of a SOP document by SOP ID",
		Response:            new(ruletypes.SOPDocument),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/sop/documents/{sopId}/versions/{version}", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.FetchSOPDocumentVersion), handler.OpenAPIDef{
		ID:                  "FetchSOPDocumentVersion",
		Tags:                []string{"rules"},
		Summary:             "Fetch exact SOP document version",
		Description:         "This endpoint returns an exact version of a registered SOP document for AI strategy generation and audit citation",
		Response:            new(ruletypes.SOPDocument),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/sop/bindings/preview", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.PreviewSOPDocumentBinding), handler.OpenAPIDef{
		ID:                  "PreviewSOPDocumentBinding",
		Tags:                []string{"rules"},
		Summary:             "Preview SOP document binding",
		Description:         "This endpoint resolves alert labels against registered SOP documents before AI strategy generation",
		Request:             new(ruletypes.SOPBindingPreviewRequest),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.SOPBindingPreviewResponse),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/ai/strategy/preview", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.PreviewAIStrategy), handler.OpenAPIDef{
		ID:                  "PreviewAIStrategy",
		Tags:                []string{"rules"},
		Summary:             "Preview SOP-grounded AI response strategy",
		Description:         "This endpoint generates a deterministic SOP-grounded AI response strategy preview from alert labels, redacted evidence refs, and a tenant-scoped SOP document",
		Request:             new(ruletypes.AIStrategyRequest),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.AIStrategy),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/ai/strategy/history/latest", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.GetLatestAIStrategyHistory), handler.OpenAPIDef{
		ID:                  "GetLatestAIStrategyHistory",
		Tags:                []string{"rules"},
		Summary:             "Get latest SOP-grounded AI response strategy",
		Description:         "This endpoint returns the latest stored ds.ai_strategy.v1 result by incidentId or alertFingerprint for Alert Detail surfaces",
		RequestQuery:        new(ruletypes.AIStrategyHistoryLookupRequest),
		Response:            new(ruletypes.AIStrategyHistoryRecord),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/ai/config", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.GetAIConfig), handler.OpenAPIDef{
		ID:                  "GetAIConfig",
		Tags:                []string{"rules"},
		Summary:             "Get per-tenant AI module configuration",
		Description:         "Returns the org's AI generator configuration (provider/transport/model/etc). The api_key field is masked as <unchanged> when set.",
		Response:            new(ruletypes.AIConfig),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/ai/config", handler.New(provider.authZ.EditAccess(provider.rulerHandler.UpdateAIConfig), handler.OpenAPIDef{
		ID:                  "UpdateAIConfig",
		Tags:                []string{"rules"},
		Summary:             "Update per-tenant AI module configuration",
		Description:         "Upserts the org's AI generator configuration. Sending api_key=<unchanged> preserves the stored key; any other value replaces it.",
		Request:             new(ruletypes.AIConfig),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.AIConfig),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleAdmin),
	})).Methods(http.MethodPut).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/ai/config/test", handler.New(provider.authZ.EditAccess(provider.rulerHandler.TestAIConfig), handler.OpenAPIDef{
		ID:                  "TestAIConfig",
		Tags:                []string{"rules"},
		Summary:             "Probe AI module configuration",
		Description:         "Instantiates a generator from the supplied (or stored) AIConfig and runs one probe Generate call. Does not mutate persisted config.",
		Request:             new(ruletypes.AIConfig),
		RequestContentType:  "application/json",
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleAdmin),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/coderca/config", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.GetCodebaseRCAConfig), handler.OpenAPIDef{
		ID:                  "GetCodebaseRCAConfig",
		Tags:                []string{"coderca"},
		Summary:             "Get code RCA config",
		Description:         "Returns the org's CF-11 code-RCA feature toggle and cost thresholds (defaults when unset).",
		Response:            new(ruletypes.CodebaseRCAConfig),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/coderca/config", handler.New(provider.authZ.EditAccess(provider.rulerHandler.UpdateCodebaseRCAConfig), handler.OpenAPIDef{
		ID:                  "UpdateCodebaseRCAConfig",
		Tags:                []string{"coderca"},
		Summary:             "Update code RCA config",
		Description:         "Upserts the org's CF-11 code-RCA feature toggle and cost thresholds.",
		Request:             new(ruletypes.CodebaseRCAConfig),
		RequestContentType:  "application/json",
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleAdmin),
	})).Methods(http.MethodPut).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/incident/report/template", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.GetIncidentReportTemplate), handler.OpenAPIDef{
		ID:                  "GetIncidentReportTemplate",
		Tags:                []string{"incident_report"},
		Summary:             "Get incident report template",
		Description:         "Returns the org's incident-report 양식 template (built-in default when unset).",
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/incident/report/template", handler.New(provider.authZ.EditAccess(provider.rulerHandler.UpdateIncidentReportTemplate), handler.OpenAPIDef{
		ID:                  "UpdateIncidentReportTemplate",
		Tags:                []string{"incident_report"},
		Summary:             "Update incident report template",
		Description:         "Upserts the org's incident-report 양식 template (Go text/template; validated before save).",
		RequestContentType:  "application/json",
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleAdmin),
	})).Methods(http.MethodPut).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/incident/report", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.GenerateIncidentReport), handler.OpenAPIDef{
		ID:                  "GenerateIncidentReport",
		Tags:                []string{"incident_report"},
		Summary:             "Generate incident report",
		Description:         "Aggregates the incident's CF-2 strategy and CF-11 finding into a Korean-SI 장애보고서, rendered with the org template.",
		RequestContentType:  "application/json",
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/coderca/repos", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.ListCodebaseRepos), handler.OpenAPIDef{
		ID:                  "ListCodebaseRepos",
		Tags:                []string{"coderca"},
		Summary:             "List codebase repos",
		Description:         "Returns all registered codebase repository entries for the org.",
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/coderca/repos", handler.New(provider.authZ.EditAccess(provider.rulerHandler.UpsertCodebaseRepo), handler.OpenAPIDef{
		ID:                  "UpsertCodebaseRepo",
		Tags:                []string{"coderca"},
		Summary:             "Upsert codebase repo",
		Description:         "Creates or updates a codebase repository entry for the org.",
		RequestContentType:  "application/json",
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleAdmin),
	})).Methods(http.MethodPut).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/coderca/repos/{repoId}", handler.New(provider.authZ.EditAccess(provider.rulerHandler.DeleteCodebaseRepo), handler.OpenAPIDef{
		ID:                  "DeleteCodebaseRepo",
		Tags:                []string{"coderca"},
		Summary:             "Delete codebase repo",
		Description:         "Removes a registered codebase repository entry by ID.",
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusNoContent,
		ErrorStatusCodes:    []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleAdmin),
	})).Methods(http.MethodDelete).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/coderca/service-maps", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.ListCodebaseServiceMaps), handler.OpenAPIDef{
		ID:                  "ListCodebaseServiceMaps",
		Tags:                []string{"coderca"},
		Summary:             "List codebase service maps",
		Description:         "Returns all service-to-repo mapping entries for the org.",
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/coderca/service-maps", handler.New(provider.authZ.EditAccess(provider.rulerHandler.UpsertCodebaseServiceMap), handler.OpenAPIDef{
		ID:                  "UpsertCodebaseServiceMap",
		Tags:                []string{"coderca"},
		Summary:             "Upsert codebase service map",
		Description:         "Creates or updates a service-to-repo mapping entry for the org.",
		RequestContentType:  "application/json",
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleAdmin),
	})).Methods(http.MethodPut).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/coderca/service-maps/{serviceName}", handler.New(provider.authZ.EditAccess(provider.rulerHandler.DeleteCodebaseServiceMap), handler.OpenAPIDef{
		ID:                  "DeleteCodebaseServiceMap",
		Tags:                []string{"coderca"},
		Summary:             "Delete codebase service map",
		Description:         "Removes a service-to-repo mapping entry by service name.",
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusNoContent,
		ErrorStatusCodes:    []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleAdmin),
	})).Methods(http.MethodDelete).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/coderca/runs", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.ListCodeRCARuns), handler.OpenAPIDef{
		ID:                  "ListCodeRCARuns",
		Tags:                []string{"coderca"},
		Summary:             "List code RCA runs",
		Description:         "Returns code RCA run records for the org, optionally filtered by service or alert fingerprint.",
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/coderca/runs/{runId}", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.GetCodeRCARun), handler.OpenAPIDef{
		ID:                  "GetCodeRCARun",
		Tags:                []string{"coderca"},
		Summary:             "Get code RCA run",
		Description:         "Returns a single code RCA run record by ID.",
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/sop/documents/{sopId}/versions/{version}/runbooks", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.ListRunbooks), handler.OpenAPIDef{
		ID:                  "ListRunbooks",
		Tags:                []string{"rules"},
		Summary:             "List runbooks for a SOP document version",
		Description:         "Returns runbooks attached to the given SOP document version, filtered by status (default: draft+approved)",
		Response:            new(ruletypes.SOPDocument),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/sop/documents/{sopId}/versions/{version}/runbooks/{runbookId}", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.GetRunbook), handler.OpenAPIDef{
		ID:                  "GetRunbook",
		Tags:                []string{"rules"},
		Summary:             "Get a single runbook",
		Description:         "Returns one runbook by ID from the given SOP document version",
		Response:            new(ruletypes.Runbook),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/sop/documents/{sopId}/versions/{version}/runbooks", handler.New(provider.authZ.EditAccess(provider.rulerHandler.CreateRunbook), handler.OpenAPIDef{
		ID:                  "CreateRunbook",
		Tags:                []string{"rules"},
		Summary:             "Create a runbook",
		Description:         "Appends a new runbook to the given SOP document version; editor role required",
		Request:             new(ruletypes.Runbook),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.Runbook),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusCreated,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/sop/documents/{sopId}/versions/{version}/runbooks/{runbookId}", handler.New(provider.authZ.EditAccess(provider.rulerHandler.UpdateRunbook), handler.OpenAPIDef{
		ID:                  "UpdateRunbook",
		Tags:                []string{"rules"},
		Summary:             "Update a runbook",
		Description:         "Replaces a runbook by ID in the given SOP document version; editor role required",
		Request:             new(ruletypes.Runbook),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.Runbook),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPut).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/sop/documents/{sopId}/versions/{version}/runbooks/{runbookId}", handler.New(provider.authZ.EditAccess(provider.rulerHandler.DeleteRunbook), handler.OpenAPIDef{
		ID:                "DeleteRunbook",
		Tags:              []string{"rules"},
		Summary:           "Delete a runbook",
		Description:       "Hard-deletes a runbook by ID from the given SOP document version; admin role required",
		SuccessStatusCode: http.StatusNoContent,
		ErrorStatusCodes:  []int{http.StatusUnauthorized, http.StatusNotFound},
		SecuritySchemes:   newSecuritySchemes(types.RoleAdmin),
	})).Methods(http.MethodDelete).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v2/ds/runbooks/draft", handler.New(provider.authZ.EditAccess(provider.rulerHandler.DraftRunbook), handler.OpenAPIDef{
		ID:                  "DraftRunbook",
		Tags:                []string{"rules"},
		Summary:             "AI-draft a runbook",
		Description:         "Generates a runbook draft from error examples and an existing SOP document version; preview only — not persisted; editor role required",
		Request:             new(ruletypes.RunbookDraftRequest),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.Runbook),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v1/downtime_schedules", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.ListDowntimeSchedules), handler.OpenAPIDef{
		ID:                  "ListDowntimeSchedules",
		Tags:                []string{"downtimeschedules"},
		Summary:             "List downtime schedules",
		Description:         "This endpoint lists all planned maintenance / downtime schedules",
		RequestQuery:        new(ruletypes.ListPlannedMaintenanceParams),
		Response:            make([]*ruletypes.PlannedMaintenance, 0),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v1/downtime_schedules/{id}", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.GetDowntimeScheduleByID), handler.OpenAPIDef{
		ID:                  "GetDowntimeScheduleByID",
		Tags:                []string{"downtimeschedules"},
		Summary:             "Get downtime schedule by ID",
		Description:         "This endpoint returns a downtime schedule by ID",
		Response:            new(ruletypes.PlannedMaintenance),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusNotFound},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v1/downtime_schedules", handler.New(provider.authZ.EditAccess(provider.rulerHandler.CreateDowntimeSchedule), handler.OpenAPIDef{
		ID:                  "CreateDowntimeSchedule",
		Tags:                []string{"downtimeschedules"},
		Summary:             "Create downtime schedule",
		Description:         "This endpoint creates a new planned maintenance / downtime schedule",
		Request:             new(ruletypes.PostablePlannedMaintenance),
		RequestContentType:  "application/json",
		Response:            new(ruletypes.PlannedMaintenance),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusCreated,
		ErrorStatusCodes:    []int{http.StatusBadRequest},
		SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v1/downtime_schedules/{id}", handler.New(provider.authZ.EditAccess(provider.rulerHandler.UpdateDowntimeScheduleByID), handler.OpenAPIDef{
		ID:                 "UpdateDowntimeScheduleByID",
		Tags:               []string{"downtimeschedules"},
		Summary:            "Update downtime schedule",
		Description:        "This endpoint updates a downtime schedule by ID",
		Request:            new(ruletypes.PostablePlannedMaintenance),
		RequestContentType: "application/json",
		SuccessStatusCode:  http.StatusNoContent,
		ErrorStatusCodes:   []int{http.StatusBadRequest, http.StatusNotFound},
		SecuritySchemes:    newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodPut).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v1/downtime_schedules/{id}", handler.New(provider.authZ.EditAccess(provider.rulerHandler.DeleteDowntimeScheduleByID), handler.OpenAPIDef{
		ID:                "DeleteDowntimeScheduleByID",
		Tags:              []string{"downtimeschedules"},
		Summary:           "Delete downtime schedule",
		Description:       "This endpoint deletes a downtime schedule by ID",
		SuccessStatusCode: http.StatusNoContent,
		ErrorStatusCodes:  []int{http.StatusNotFound},
		SecuritySchemes:   newSecuritySchemes(types.RoleEditor),
	})).Methods(http.MethodDelete).GetError(); err != nil {
		return err
	}

	return nil
}
