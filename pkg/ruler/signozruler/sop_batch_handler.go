package signozruler

import (
	"net/http"
	"strings"

	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

func (handler *handler) CreateSOPDocumentBatch(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
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
	// Track (sopId, version) pairs already accepted in this batch. The store
	// upserts ON CONFLICT (org_id, sop_id, version), so two payload entries
	// sharing a key would silently clobber each other. We reject the later
	// one as a version conflict instead of upserting it.
	seen := make(map[string]struct{}, len(batchReq.Documents))

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

		key := strings.TrimSpace(doc.SOPID) + "\x00" + strings.TrimSpace(doc.Version)
		if _, dup := seen[key]; dup {
			result.Status = ruletypes.SOPBatchResultStatusError
			result.Error = "duplicate sopId and version in batch (version conflict)"
			failed++
			results = append(results, result)
			continue
		}
		seen[key] = struct{}{}

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
