package signozruler

import (
	"context"
	"net/http"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	codercarunstore "github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
	"github.com/SigNoz/signoz/pkg/ruler/incidentreport"
)

// codeRCAAdapter maps the coderca run store onto incidentreport.CodeRCAReader.
// MVP correlation: most recent DONE run for the service (created_at DESC).
type codeRCAAdapter struct {
	runs *codercarunstore.Store
}

func (a codeRCAAdapter) LatestFinding(ctx context.Context, orgID, service string) (incidentreport.CodeRCAFinding, bool, error) {
	if a.runs == nil {
		return incidentreport.CodeRCAFinding{}, false, nil
	}
	runs, err := a.runs.ListRuns(ctx, orgID, codercarunstore.ListRunsParams{
		Service: service,
		Status:  string(coderca.RunStatusDone),
		Limit:   1,
	})
	if err != nil {
		return incidentreport.CodeRCAFinding{}, false, err
	}
	if len(runs) == 0 {
		return incidentreport.CodeRCAFinding{}, false, nil
	}
	d, err := a.runs.GetRun(ctx, orgID, runs[0].RunID)
	if err != nil {
		return incidentreport.CodeRCAFinding{}, false, err
	}
	return incidentreport.CodeRCAFinding{
		RunID:          d.RunID,
		RootCause:      d.RootCause,
		ProposedFix:    d.ProposedFix,
		Confidence:     d.Confidence,
		Limitations:    d.Limitations,
		BaselineCommit: d.BaselineCommit,
		CreatedAt:      d.CreatedAt,
		FinishedAt:     d.FinishedAt,
	}, true, nil
}

// incidentReportTemplateResponse carries the org's managed template (or the
// built-in default when none is set, flagged by IsDefault).
type incidentReportTemplateResponse struct {
	Template  string `json:"template"`
	IsDefault bool   `json:"isDefault"`
}

type incidentReportTemplateRequest struct {
	Template string `json:"template"`
}

// GetIncidentReportTemplate returns the org's report 양식 template, falling back
// to the built-in default when the org has not set one.
func (handler *handler) GetIncidentReportTemplate(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if handler.reportTemplateStore == nil {
		render.Success(rw, http.StatusOK, incidentReportTemplateResponse{Template: incidentreport.DefaultReportTemplate, IsDefault: true})
		return
	}
	tmpl, ok, err := handler.reportTemplateStore.Get(req.Context(), orgID)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "get incident report template"))
		return
	}
	if !ok || tmpl == "" {
		render.Success(rw, http.StatusOK, incidentReportTemplateResponse{Template: incidentreport.DefaultReportTemplate, IsDefault: true})
		return
	}
	render.Success(rw, http.StatusOK, incidentReportTemplateResponse{Template: tmpl, IsDefault: false})
}

// UpdateIncidentReportTemplate sets the org's report template. The template is
// validated by rendering it against a probe report so a malformed 양식 is
// rejected before it is stored.
func (handler *handler) UpdateIncidentReportTemplate(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if handler.reportTemplateStore == nil {
		render.Error(rw, errors.Newf(errors.TypeInternal, errors.CodeInternal, "incident report template store not configured"))
		return
	}
	var body incidentReportTemplateRequest
	if err := binding.JSON.BindBody(req.Body, &body); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	// Validate: a non-empty template must parse+execute against a probe.
	if body.Template != "" {
		if _, rErr := (incidentreport.IncidentReport{IncidentID: "probe"}).Render(body.Template); rErr != nil {
			render.Error(rw, errors.WrapInvalidInputf(rErr, errors.CodeInvalidInput, "invalid report template"))
			return
		}
	}
	if err := handler.reportTemplateStore.Upsert(req.Context(), orgID, body.Template); err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "save incident report template"))
		return
	}
	render.Success(rw, http.StatusOK, incidentReportTemplateResponse{Template: body.Template, IsDefault: body.Template == ""})
}

type generateIncidentReportRequest struct {
	IncidentID       string `json:"incidentId"`
	AlertFingerprint string `json:"alertFingerprint"`
	Service          string `json:"service"`
	Severity         string `json:"severity"`
}

type generateIncidentReportResponse struct {
	Markdown string                        `json:"markdown"`
	Report   incidentreport.IncidentReport `json:"report"`
}

// GenerateIncidentReport aggregates the incident's CF-2 strategy + CF-11 finding
// into a report and renders it with the org's managed template (or the default).
func (handler *handler) GenerateIncidentReport(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	var body generateIncidentReportRequest
	if err := binding.JSON.BindBody(req.Body, &body); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	gen := incidentreport.NewGenerator(handler.aiHistoryStore, codeRCAAdapter{runs: handler.codercaRunStore})
	report, err := gen.Build(req.Context(), incidentreport.Params{
		OrgID:            orgID,
		IncidentID:       body.IncidentID,
		AlertFingerprint: body.AlertFingerprint,
		Service:          body.Service,
		Severity:         body.Severity,
		Now:              time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "build incident report"))
		return
	}

	tmpl := ""
	if handler.reportTemplateStore != nil {
		if t, ok, gErr := handler.reportTemplateStore.Get(req.Context(), orgID); gErr == nil && ok {
			tmpl = t
		}
	}
	// Render with the org template (empty → default). A stored template that
	// fails is surfaced, not silently swapped, so the user fixes their 양식.
	md, err := report.Render(tmpl)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "render incident report — stored template may be invalid"))
		return
	}
	render.Success(rw, http.StatusOK, generateIncidentReportResponse{Markdown: md, Report: report})
}
