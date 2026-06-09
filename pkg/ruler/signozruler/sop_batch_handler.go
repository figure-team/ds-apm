package signozruler

import (
	"net/http"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

func (handler *handler) CreateSOPDocumentBatch(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	var batchReq ruletypes.SOPDocumentBatchRequest
	if err := binding.JSON.BindBody(req.Body, &batchReq); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	results := make([]ruletypes.SOPDocumentBatchResult, 0, len(batchReq.Documents))
	succeeded := 0
	failed := 0

	for _, doc := range batchReq.Documents {
		result := ruletypes.SOPDocumentBatchResult{
			SOPID:   doc.SOPID,
			Version: doc.Version,
		}

		if err := ruletypes.ValidateSOPDocument(doc); err != nil {
			result.Status = ruletypes.SOPBatchResultStatusError
			result.Error = err.Error()
			failed++
			results = append(results, result)
			continue
		}

		if err := handler.sopStore.Upsert(req.Context(), orgID, doc); err != nil {
			result.Status = ruletypes.SOPBatchResultStatusError
			result.Error = "failed to persist document"
			failed++
			results = append(results, result)
			continue
		}

		result.Status = ruletypes.SOPBatchResultStatusOk
		succeeded++
		results = append(results, result)
	}

	render.Success(rw, http.StatusOK, ruletypes.SOPDocumentBatchResponse{
		ContractVersion: ruletypes.SOPBatchResultContractVersion,
		Total:           len(batchReq.Documents),
		Succeeded:       succeeded,
		Failed:          failed,
		Results:         results,
	})
}
