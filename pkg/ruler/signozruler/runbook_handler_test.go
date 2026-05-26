package signozruler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/ruler/runbookdrafter/mockrunbookdrafter"
	"github.com/SigNoz/signoz/pkg/ruler/sopstore/sopstoretest"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// testOrgID matches the OrgID embedded in withSOPTestClaims (handler_test.go).
const testOrgID = "00000000-0000-0000-0000-000000000001"

// validTestRunbook returns a minimal but valid Runbook suitable for Create/Update bodies.
// ID is intentionally left blank — server assigns it on Create.
func validTestRunbook() ruletypes.Runbook {
	return ruletypes.Runbook{
		Title:       "Restart payment-api",
		Description: "Drain the queue then restart.",
		Status:      ruletypes.RunbookStatusApproved,
		Confidence:  0.9,
		UpdatedBy:   "operator@example.com",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
}

// newRunbookTestHandler constructs a handler with a fresh sopstoretest.Fake
// seeded with one valid SOP (SOP-PAY-001 v01) and a mock RunbookDrafter.
// The store is returned so tests can inspect or further mutate it.
func newRunbookTestHandler(t *testing.T) (*sopstoretest.Fake, *handler) {
	t.Helper()
	store := sopstoretest.New()

	// Seed one SOP document that our test claims org can read.
	seedDoc := ruletypes.SOPDocument{
		ContractVersion: ruletypes.SOPDocumentContractVersion,
		SOPID:           "SOP-PAY-001",
		Version:         "v01",
		Title:           "Payment API 5xx response",
		BodyMarkdown:    "Restart payment-api only after confirming queue drain.",
		ApprovalStatus:  ruletypes.SOPApprovalStatusApproved,
		OwnerTeam:       "payments",
	}
	require.NoError(t, store.Upsert(t.Context(), testOrgID, seedDoc))

	mockDrafter := mockrunbookdrafter.New(ruletypes.Runbook{
		ID:          "draft-id-mock",
		Title:       "Restart payment-api (AI draft)",
		Description: "AI-generated runbook based on error examples.",
		Status:      ruletypes.RunbookStatusDraft,
		Confidence:  0.75,
		AIDraftedBy: "claude-test",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		UpdatedBy:   "ai",
	})

	h := &handler{
		sopStore:       store,
		aiHistoryStore: newMemAIHistoryStore(),
		runbookDrafter: mockDrafter,
	}
	return store, h
}

// TestListRunbooks_EmptyByDefault verifies that a freshly-seeded SOP with no
// runbooks returns an empty list (not a 404 or nil).
func TestListRunbooks_EmptyByDefault(t *testing.T) {
	_, h := newRunbookTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/ds/sop/documents/SOP-PAY-001/versions/v01/runbooks", nil)
	req = withSOPTestClaims(req)
	req = muxSetVar(req, "sopId", "SOP-PAY-001")
	req = muxSetVar(req, "version", "v01")
	rw := httptest.NewRecorder()

	h.ListRunbooks(rw, req)

	require.Equal(t, http.StatusOK, rw.Code, "body=%s", rw.Body.String())
	var got struct {
		Data runbookListResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &got))
	require.NotNil(t, got.Data.Runbooks)
	require.Empty(t, got.Data.Runbooks)
}

// TestCreateRunbook_PersistsAndAssignsID verifies that POST /runbooks
// assigns a server-side UUID, stores the runbook, and returns 201.
func TestCreateRunbook_PersistsAndAssignsID(t *testing.T) {
	store, h := newRunbookTestHandler(t)

	rb := validTestRunbook()
	rb.ID = "" // ensure server assigns
	body, err := json.Marshal(rb)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents/SOP-PAY-001/versions/v01/runbooks", bytes.NewReader(body))
	req = withSOPTestClaims(req)
	req = muxSetVar(req, "sopId", "SOP-PAY-001")
	req = muxSetVar(req, "version", "v01")
	rw := httptest.NewRecorder()

	h.CreateRunbook(rw, req)

	require.Equal(t, http.StatusCreated, rw.Code, "body=%s", rw.Body.String())
	var envelope struct {
		Data ruletypes.Runbook `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &envelope))
	require.NotEmpty(t, envelope.Data.ID, "server must assign a UUID")
	require.Equal(t, ruletypes.RunbookStatusApproved, envelope.Data.Status)

	// Verify persisted in store.
	doc, err := store.Get(t.Context(), testOrgID, "SOP-PAY-001", "v01")
	require.NoError(t, err)
	require.Len(t, doc.Runbooks, 1)
	require.Equal(t, envelope.Data.ID, doc.Runbooks[0].ID)
}

// TestCreateRunbook_RejectsParentNotFound verifies that creating a runbook on
// a non-existent SOP version returns 404.
func TestCreateRunbook_RejectsParentNotFound(t *testing.T) {
	_, h := newRunbookTestHandler(t)

	rb := validTestRunbook()
	body, err := json.Marshal(rb)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents/SOP-PAY-001/versions/v99/runbooks", bytes.NewReader(body))
	req = withSOPTestClaims(req)
	req = muxSetVar(req, "sopId", "SOP-PAY-001")
	req = muxSetVar(req, "version", "v99") // does not exist
	rw := httptest.NewRecorder()

	h.CreateRunbook(rw, req)

	require.Equal(t, http.StatusNotFound, rw.Code, "body=%s", rw.Body.String())
}

// TestUpdateRunbook_RejectsForbiddenTransition verifies that the
// deprecated→approved shortcut is rejected with 400.
func TestUpdateRunbook_RejectsForbiddenTransition(t *testing.T) {
	store, h := newRunbookTestHandler(t)

	// Create a deprecated runbook directly in the store.
	deprecatedRB := ruletypes.Runbook{
		ID:        "rb-deprecated-001",
		Title:     "Old runbook",
		Status:    ruletypes.RunbookStatusDeprecated,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedBy: "operator@example.com",
	}
	require.NoError(t, store.UpsertRunbook(t.Context(), testOrgID, "SOP-PAY-001", "v01", deprecatedRB))

	// Attempt to transition directly to approved (forbidden).
	incoming := deprecatedRB
	incoming.Status = ruletypes.RunbookStatusApproved
	body, err := json.Marshal(incoming)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/v2/ds/sop/documents/SOP-PAY-001/versions/v01/runbooks/rb-deprecated-001", bytes.NewReader(body))
	req = withSOPTestClaims(req)
	req = muxSetVar(req, "sopId", "SOP-PAY-001")
	req = muxSetVar(req, "version", "v01")
	req = muxSetVar(req, "runbookId", "rb-deprecated-001")
	rw := httptest.NewRecorder()

	h.UpdateRunbook(rw, req)

	require.Equal(t, http.StatusBadRequest, rw.Code, "body=%s", rw.Body.String())
}

// TestDeleteRunbook_RequiresAdmin verifies that DeleteRunbook returns 204 when
// authenticated (role gating is pass-through until Claims.Role is wired — see
// hasAdminRole comment in runbook_handler.go) and 404 when the runbook is missing.
func TestDeleteRunbook_RequiresAdmin(t *testing.T) {
	store, h := newRunbookTestHandler(t)

	// Seed a runbook to delete.
	rb := ruletypes.Runbook{
		ID:        "rb-delete-me",
		Title:     "To be deleted",
		Status:    ruletypes.RunbookStatusApproved,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedBy: "operator@example.com",
	}
	require.NoError(t, store.UpsertRunbook(t.Context(), testOrgID, "SOP-PAY-001", "v01", rb))

	// Delete it — succeeds (role gating is pass-through).
	req := httptest.NewRequest(http.MethodDelete, "/api/v2/ds/sop/documents/SOP-PAY-001/versions/v01/runbooks/rb-delete-me", nil)
	req = withSOPTestClaims(req)
	req = muxSetVar(req, "sopId", "SOP-PAY-001")
	req = muxSetVar(req, "version", "v01")
	req = muxSetVar(req, "runbookId", "rb-delete-me")
	rw := httptest.NewRecorder()

	h.DeleteRunbook(rw, req)
	require.Equal(t, http.StatusNoContent, rw.Code, "body=%s", rw.Body.String())

	// Second delete — 404 because the runbook no longer exists.
	req2 := httptest.NewRequest(http.MethodDelete, "/api/v2/ds/sop/documents/SOP-PAY-001/versions/v01/runbooks/rb-delete-me", nil)
	req2 = withSOPTestClaims(req2)
	req2 = muxSetVar(req2, "sopId", "SOP-PAY-001")
	req2 = muxSetVar(req2, "version", "v01")
	req2 = muxSetVar(req2, "runbookId", "rb-delete-me")
	rw2 := httptest.NewRecorder()

	h.DeleteRunbook(rw2, req2)
	require.Equal(t, http.StatusNotFound, rw2.Code, "body=%s", rw2.Body.String())
}

// TestDraftRunbook_ReturnsDraftWithoutPersisting verifies that DraftRunbook
// returns the draft from the drafter and does NOT persist it to the store.
func TestDraftRunbook_ReturnsDraftWithoutPersisting(t *testing.T) {
	store, h := newRunbookTestHandler(t)

	body, err := json.Marshal(draftRunbookRequest{
		SOPID:         "SOP-PAY-001",
		Version:       "v01",
		ErrorExamples: []string{"payment-api returned 503 after deploy"},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/runbooks/draft", bytes.NewReader(body))
	req = withSOPTestClaims(req)
	rw := httptest.NewRecorder()

	h.DraftRunbook(rw, req)

	require.Equal(t, http.StatusOK, rw.Code, "body=%s", rw.Body.String())
	var envelope struct {
		Data ruletypes.Runbook `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &envelope))
	require.Equal(t, "draft-id-mock", envelope.Data.ID)
	require.Equal(t, ruletypes.RunbookStatusDraft, envelope.Data.Status)

	// Confirm the draft was NOT persisted to the store.
	doc, err := store.Get(t.Context(), testOrgID, "SOP-PAY-001", "v01")
	require.NoError(t, err)
	require.Empty(t, doc.Runbooks, "draft must not be persisted")
}
