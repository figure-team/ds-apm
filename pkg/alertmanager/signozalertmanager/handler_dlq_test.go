// pkg/alertmanager/signozalertmanager/handler_dlq_test.go
package signozalertmanager_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/alertmanager/alertmanagertest"
	"github.com/SigNoz/signoz/pkg/alertmanager/signozalertmanager"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
)

func contextWithOrg(orgID string) context.Context {
	claims := authtypes.Claims{OrgID: orgID}
	return authtypes.NewContextWithClaims(context.Background(), claims)
}

func TestGetDLQEntries_ReturnsEntries(t *testing.T) {
	mockAM := alertmanagertest.NewMockAlertmanager(t)
	entries := []*alertmanagertypes.DLQEntry{
		{
			EventID:  "abc123",
			Channel:  "slack",
			FailedAt: time.Now(),
			Reason:   "timeout",
			Status:   "pending",
		},
	}
	mockAM.On("ListDLQEntries", mock.Anything, "test-org", "", "").Return(entries, nil)

	h := signozalertmanager.NewHandler(mockAM)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alertmanager/dlq/entries", nil)
	req = req.WithContext(contextWithOrg("test-org"))
	rw := httptest.NewRecorder()

	h.GetDLQEntries(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	var resp struct {
		Data []*alertmanagertypes.DLQEntry `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rw.Body).Decode(&resp))
	require.Len(t, resp.Data, 1)
	require.Equal(t, "abc123", resp.Data[0].EventID)
}

func TestGetDLQEntries_ChannelAndStatusFilter(t *testing.T) {
	mockAM := alertmanagertest.NewMockAlertmanager(t)
	mockAM.On("ListDLQEntries", mock.Anything, "test-org", "slack", "pending").
		Return([]*alertmanagertypes.DLQEntry{}, nil)

	h := signozalertmanager.NewHandler(mockAM)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alertmanager/dlq/entries?channel=slack&status=pending", nil)
	req = req.WithContext(contextWithOrg("test-org"))
	rw := httptest.NewRecorder()

	h.GetDLQEntries(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
}

func TestReplayDLQEntries_ReturnsResult(t *testing.T) {
	mockAM := alertmanagertest.NewMockAlertmanager(t)
	result := &alertmanagertypes.ReplayResult{Replayed: 1, Skipped: 0, Failed: 0}
	mockAM.On("ReplayDLQEntries", mock.Anything, "test-org", []string{"abc123"}).Return(result, nil)

	h := signozalertmanager.NewHandler(mockAM)
	body, _ := json.Marshal(map[string][]string{"event_ids": {"abc123"}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alertmanager/dlq/replay", bytes.NewReader(body))
	req = req.WithContext(contextWithOrg("test-org"))
	rw := httptest.NewRecorder()

	h.ReplayDLQEntries(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	var resp struct {
		Data *alertmanagertypes.ReplayResult `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rw.Body).Decode(&resp))
	require.Equal(t, 1, resp.Data.Replayed)
}

func TestReplayDLQEntries_EmptyEventIDs_Returns400(t *testing.T) {
	mockAM := alertmanagertest.NewMockAlertmanager(t)

	h := signozalertmanager.NewHandler(mockAM)
	body, _ := json.Marshal(map[string][]string{"event_ids": {}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alertmanager/dlq/replay", bytes.NewReader(body))
	req = req.WithContext(contextWithOrg("test-org"))
	rw := httptest.NewRecorder()

	h.ReplayDLQEntries(rw, req)

	require.Equal(t, http.StatusBadRequest, rw.Code)
}
