package signozruler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

func TestCreateSOPDocumentBatch_HappyPath(t *testing.T) {
	h := newTestHandler()

	doc1 := validSOPDocumentRequest(t, "2026-06-01.1", ruletypes.SOPApprovalStatusApproved)
	doc2 := validSOPDocumentRequest(t, "2026-06-01.2", ruletypes.SOPApprovalStatusApproved)
	doc2.SOPID = "SOP-CART-001"
	doc2.Title = "Cart Redis timeout"

	body, err := json.Marshal(ruletypes.SOPDocumentBatchRequest{
		ContractVersion: ruletypes.SOPDocumentListContractVersion,
		Documents:       []ruletypes.SOPDocument{doc1, doc2},
	})
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents/batch", bytes.NewReader(body))
	req = withSOPTestClaims(req)
	h.CreateSOPDocumentBatch(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	var got struct {
		Data ruletypes.SOPDocumentBatchResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &got))
	require.Equal(t, ruletypes.SOPBatchResultContractVersion, got.Data.ContractVersion)
	require.Equal(t, 2, got.Data.Total)
	require.Equal(t, 2, got.Data.Succeeded)
	require.Equal(t, 0, got.Data.Failed)
	require.Len(t, got.Data.Results, 2)
	require.Equal(t, ruletypes.SOPBatchResultStatusOk, got.Data.Results[0].Status)
}

func TestCreateSOPDocumentBatch_PartialFailure(t *testing.T) {
	h := newTestHandler()

	validDoc := validSOPDocumentRequest(t, "2026-06-01.1", ruletypes.SOPApprovalStatusApproved)
	invalidDoc := validSOPDocumentRequest(t, "2026-06-01.2", ruletypes.SOPApprovalStatusApproved)
	invalidDoc.BodyMarkdown = "Rotate with access_token=hidden" // secret-like string → validation error

	body, err := json.Marshal(ruletypes.SOPDocumentBatchRequest{
		ContractVersion: ruletypes.SOPDocumentListContractVersion,
		Documents:       []ruletypes.SOPDocument{validDoc, invalidDoc},
	})
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents/batch", bytes.NewReader(body))
	req = withSOPTestClaims(req)
	h.CreateSOPDocumentBatch(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	var got struct {
		Data ruletypes.SOPDocumentBatchResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &got))
	require.Equal(t, 2, got.Data.Total)
	require.Equal(t, 1, got.Data.Succeeded)
	require.Equal(t, 1, got.Data.Failed)
	require.Equal(t, ruletypes.SOPBatchResultStatusOk, got.Data.Results[0].Status)
	require.Equal(t, ruletypes.SOPBatchResultStatusError, got.Data.Results[1].Status)
	require.NotEmpty(t, got.Data.Results[1].Error)
}

func TestBatch_PartialFailure(t *testing.T) {
	h := newTestHandler()

	// doc0: valid, first occurrence of (SOP-PAY-001, v1) -> ok
	doc0 := validSOPDocumentRequest(t, "2026-06-01.1", ruletypes.SOPApprovalStatusApproved)
	// doc1: valid content but repeats (SOP-PAY-001, v1) -> duplicate/version conflict
	doc1 := validSOPDocumentRequest(t, "2026-06-01.1", ruletypes.SOPApprovalStatusApproved)
	// doc2: distinct version but invalid content -> validation failure
	doc2 := validSOPDocumentRequest(t, "2026-06-01.2", ruletypes.SOPApprovalStatusApproved)
	doc2.BodyMarkdown = "Rotate with access_token=hidden" // secret-like -> validation error

	body, err := json.Marshal(ruletypes.SOPDocumentBatchRequest{
		ContractVersion: ruletypes.SOPDocumentListContractVersion,
		Documents:       []ruletypes.SOPDocument{doc0, doc1, doc2},
	})
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents/batch", bytes.NewReader(body))
	req = withSOPTestClaims(req)
	h.CreateSOPDocumentBatch(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	var got struct {
		Data ruletypes.SOPDocumentBatchResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &got))

	require.Equal(t, 3, got.Data.Total)
	require.Equal(t, 1, got.Data.Succeeded)
	require.Equal(t, 2, got.Data.Failed)
	require.Len(t, got.Data.Results, 3)

	require.Equal(t, ruletypes.SOPBatchResultStatusOk, got.Data.Results[0].Status)

	require.Equal(t, ruletypes.SOPBatchResultStatusError, got.Data.Results[1].Status)
	require.Contains(t, got.Data.Results[1].Error, "duplicate")

	require.Equal(t, ruletypes.SOPBatchResultStatusError, got.Data.Results[2].Status)
	require.NotEmpty(t, got.Data.Results[2].Error)
}

func TestCreateSOPDocumentBatch_RequiresClaims(t *testing.T) {
	body, err := json.Marshal(ruletypes.SOPDocumentBatchRequest{
		ContractVersion: ruletypes.SOPDocumentListContractVersion,
		Documents:       []ruletypes.SOPDocument{},
	})
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents/batch", bytes.NewReader(body))
	// No claims attached

	newTestHandler().CreateSOPDocumentBatch(rw, req)

	require.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestCreateSOPDocumentBatch_EmptyDocuments(t *testing.T) {
	h := newTestHandler()

	body, err := json.Marshal(ruletypes.SOPDocumentBatchRequest{
		ContractVersion: ruletypes.SOPDocumentListContractVersion,
		Documents:       []ruletypes.SOPDocument{},
	})
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents/batch", bytes.NewReader(body))
	req = withSOPTestClaims(req)
	h.CreateSOPDocumentBatch(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	var got struct {
		Data ruletypes.SOPDocumentBatchResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &got))
	require.Equal(t, 0, got.Data.Total)
	require.Equal(t, 0, got.Data.Succeeded)
	require.Equal(t, 0, got.Data.Failed)
}
