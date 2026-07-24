package signozruler

import (
	"net/http"
	"strings"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/gorilla/mux"
)

func (handler *handler) CreateSOPDocument(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
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
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
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
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
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
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
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
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
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

func (handler *handler) PreviewSOP(rw http.ResponseWriter, req *http.Request) {
	var previewReq ruletypes.PreviewSOPRequest
	if err := binding.JSON.BindBody(req.Body, &previewReq); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	render.Success(rw, http.StatusOK, ruletypes.PreviewSOP(previewReq))
}
