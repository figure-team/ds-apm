package ruletypes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStorableAIStrategyHistory_Roundtrip(t *testing.T) {
	strategy := AIStrategy{
		ContractVersion:  AIStrategyContractVersion,
		StrategyID:       "strat-1",
		IncidentID:       "inc-1",
		AlertFingerprint: "fp-1",
		Status:           AIStrategyStatusUnavailable,
		Language:         "ko-KR",
		Confidence:       AIConfidenceLow,
		Limitations:      []string{"test"},
		Audit: AIStrategyAudit{
			PromptVersion:    "ds-ir-ko-v1",
			Model:            "deterministic-local",
			GeneratedAt:      "2026-05-20T09:00:00Z",
			RedactionApplied: true,
		},
	}
	record, err := NewAIStrategyHistoryRecord(strategy)
	require.NoError(t, err)

	storable, err := FromDomainAIStrategyHistoryRecord("org-1", record)
	require.NoError(t, err)
	require.Equal(t, "org-1", storable.OrgID)
	require.Equal(t, record.IncidentID, storable.IncidentID)
	require.Equal(t, record.AlertFingerprint, storable.AlertFingerprint)
	require.Equal(t, AIStrategyHistoryContractVersion, storable.ContractVersion)
	require.NotEmpty(t, storable.Payload)

	restored, err := storable.ToDomain()
	require.NoError(t, err)
	require.Equal(t, record, restored)
}

func TestStorableAIStrategyHistory_EmptyOrgIDRejected(t *testing.T) {
	record := AIStrategyHistoryRecord{
		ContractVersion: AIStrategyHistoryContractVersion,
		IncidentID:      "inc-1",
		StrategyID:      "strat-1",
		Status:          AIStrategyStatusUnavailable,
		Confidence:      AIConfidenceLow,
		GeneratedAt:     "2026-05-20T09:00:00Z",
	}
	for _, orgID := range []string{"", "   ", "\t\n"} {
		t.Run("orgID="+orgID, func(t *testing.T) {
			_, err := FromDomainAIStrategyHistoryRecord(orgID, record)
			require.Error(t, err)
		})
	}
}

func TestStorableAIStrategyHistory_EmptyContractVersionRejected(t *testing.T) {
	_, err := FromDomainAIStrategyHistoryRecord("org-1", AIStrategyHistoryRecord{
		IncidentID:  "inc-1",
		StrategyID:  "strat-1",
		Status:      AIStrategyStatusUnavailable,
		Confidence:  AIConfidenceLow,
		GeneratedAt: "2026-05-20T09:00:00Z",
	})
	require.Error(t, err)
}

func TestStorableAIStrategyHistory_ToDomain_InvalidPayloadReturnsError(t *testing.T) {
	storable := &StorableAIStrategyHistory{Payload: "not valid json"}
	_, err := storable.ToDomain()
	require.Error(t, err)
}

func TestStorableAIStrategyHistory_EmptyAlertFingerprintAllowed(t *testing.T) {
	strategy := AIStrategy{
		ContractVersion: AIStrategyContractVersion,
		StrategyID:      "strat-preview",
		IncidentID:      "inc-preview",
		Status:          AIStrategyStatusUnavailable,
		Language:        "ko-KR",
		Confidence:      AIConfidenceLow,
		Limitations:     []string{"preview"},
		Audit: AIStrategyAudit{
			PromptVersion:    "ds-ir-ko-v1",
			Model:            "deterministic-local",
			GeneratedAt:      "2026-05-20T09:00:00Z",
			RedactionApplied: true,
		},
	}
	record, err := NewAIStrategyHistoryRecord(strategy)
	require.NoError(t, err)
	require.Empty(t, record.AlertFingerprint)

	storable, err := FromDomainAIStrategyHistoryRecord("org-1", record)
	require.NoError(t, err)
	require.Equal(t, "", storable.AlertFingerprint)

	restored, err := storable.ToDomain()
	require.NoError(t, err)
	require.Equal(t, record, restored)
}

func TestStorableAIStrategyHistory_TrimsContractVersionInPayload(t *testing.T) {
	record := AIStrategyHistoryRecord{
		ContractVersion: "  " + AIStrategyHistoryContractVersion + "  ",
		IncidentID:      "inc-1",
		StrategyID:      "strat-1",
		Status:          AIStrategyStatusUnavailable,
		Confidence:      AIConfidenceLow,
		GeneratedAt:     "2026-05-20T09:00:00Z",
	}
	storable, err := FromDomainAIStrategyHistoryRecord("org-1", record)
	require.NoError(t, err)
	require.Equal(t, AIStrategyHistoryContractVersion, storable.ContractVersion)

	restored, err := storable.ToDomain()
	require.NoError(t, err)
	require.Equal(t, AIStrategyHistoryContractVersion, restored.ContractVersion, "payload must agree with flat column after trim")
}
