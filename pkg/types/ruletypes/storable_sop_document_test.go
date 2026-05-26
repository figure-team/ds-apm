package ruletypes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStorableSOPDocument_Roundtrip(t *testing.T) {
	orig := SOPDocument{
		ContractVersion: SOPDocumentContractVersion,
		SOPID:           "SOP-PAY-DEGRADE",
		Version:         "v3.2",
		Title:           "결제 시스템 지연 대응",
		BodyMarkdown:    "## 1단계\n헬스체크\n## 2단계\n우회 라우팅",
		UpdatedAt:       "2026-05-20T09:00:00Z",
		TenantScope: PilotTenantScope{
			ProjectIDs:   []string{"payments-prod"},
			Environments: []string{"prod"},
		},
	}

	storable, err := FromDomainSOPDocument("org-1", orig)
	require.NoError(t, err)
	require.Equal(t, "org-1", storable.OrgID)
	require.Equal(t, "SOP-PAY-DEGRADE", storable.SOPID)
	require.Equal(t, "v3.2", storable.Version)
	require.Equal(t, SOPDocumentContractVersion, storable.ContractVersion)
	require.Equal(t, "결제 시스템 지연 대응", storable.Title)
	require.Equal(t, "2026-05-20T09:00:00Z", storable.UpdatedAt)
	require.NotEmpty(t, storable.Payload)

	restored, err := storable.ToDomain()
	require.NoError(t, err)
	require.Equal(t, orig, restored)
}

func TestStorableSOPDocument_EmptyOrgIDRejected(t *testing.T) {
	cases := []string{"", "   ", "\t\n"}
	for _, orgID := range cases {
		t.Run("orgID="+orgID, func(t *testing.T) {
			_, err := FromDomainSOPDocument(orgID, SOPDocument{SOPID: "S1", Version: "v1", ContractVersion: SOPDocumentContractVersion})
			require.Error(t, err)
		})
	}
}

func TestStorableSOPDocument_ToDomain_InvalidPayloadReturnsError(t *testing.T) {
	storable := &StorableSOPDocument{Payload: "not valid json"}
	_, err := storable.ToDomain()
	require.Error(t, err)
}

func TestStorableSOPDocument_EmptyContractVersionRejected(t *testing.T) {
	_, err := FromDomainSOPDocument("org-1", SOPDocument{SOPID: "S1", Version: "v1"})
	require.Error(t, err)
}

func TestStorableSOPDocument_PreservesAllDomainFields(t *testing.T) {
	// Verify that *any* field on SOPDocument survives the roundtrip via the
	// payload JSON column — protects against silent data loss when new
	// SOPDocument fields are added in the future.
	orig := SOPDocument{
		ContractVersion: SOPDocumentContractVersion,
		SOPID:           "S1",
		Version:         "v1",
		Title:           "T",
		BodyMarkdown:    "body",
		UpdatedAt:       "2026-05-20T09:00:00Z",
		Tags:            []string{"a", "b"},
		OwnerTeam:       "platform",
		ApprovalStatus:  "approved",
		DisplayURL:      "https://wiki/sop/1",
	}
	storable, err := FromDomainSOPDocument("org-1", orig)
	require.NoError(t, err)
	restored, err := storable.ToDomain()
	require.NoError(t, err)
	require.Equal(t, orig, restored)
}

func TestStorableSOPDocument_TrimsContractVersionInPayload(t *testing.T) {
	doc := SOPDocument{
		ContractVersion: "  " + SOPDocumentContractVersion + "  ",
		SOPID:           "S1",
		Version:         "v1",
		Title:           "T",
		UpdatedAt:       "2026-05-20T09:00:00Z",
	}
	storable, err := FromDomainSOPDocument("org-1", doc)
	require.NoError(t, err)
	require.Equal(t, SOPDocumentContractVersion, storable.ContractVersion)

	restored, err := storable.ToDomain()
	require.NoError(t, err)
	require.Equal(t, SOPDocumentContractVersion, restored.ContractVersion, "payload must agree with flat column after trim")
}

func TestStorableSOPDocument_RoundTripWithRunbooks(t *testing.T) {
	// Construct a valid SOPDocument inline with a Runbooks slice.
	// Confirm FromDomainSOPDocument → ToDomain preserves it.
	orig := SOPDocument{
		ContractVersion: SOPDocumentContractVersion,
		SOPID:           "SOP-RB-TEST",
		Version:         "v1.0",
		Title:           "Runbook Test SOP",
		BodyMarkdown:    "## Mitigation\nRun the included runbooks.",
		UpdatedAt:       "2026-05-20T10:00:00Z",
		OwnerTeam:       "platform",
		ApprovalStatus:  "approved",
		TenantScope: PilotTenantScope{
			ProjectIDs:   []string{"test-project"},
			Environments: []string{"prod"},
		},
		Runbooks: []Runbook{
			{
				ID:               "550e8400-e29b-41d4-a716-446655440000",
				Title:            "Restart Service",
				Description:      "Restart the failed service.",
				ExecutableScript: "#!/bin/bash\nsystemctl restart myservice",
				Status:           "approved",
				Confidence:       1.0,
				AIDraftedBy:      "",
				CreatedAt:        "2026-05-20T10:00:00Z",
				UpdatedAt:        "2026-05-20T10:00:00Z",
				UpdatedBy:        "alice",
			},
		},
	}

	storable, err := FromDomainSOPDocument("org-1", orig)
	require.NoError(t, err)
	require.NotEmpty(t, storable.Payload)

	restored, err := storable.ToDomain()
	require.NoError(t, err)
	require.Equal(t, orig, restored)
	require.Len(t, restored.Runbooks, 1)
	require.Equal(t, "Restart Service", restored.Runbooks[0].Title)
	require.Equal(t, "alice", restored.Runbooks[0].UpdatedBy)
}

func TestStorableSOPDocument_BackwardCompat_NoRunbooksField(t *testing.T) {
	// Construct a StorableSOPDocument whose Payload JSON omits the
	// "runbooks" key entirely (simulating a payload written before this
	// feature). ToDomain must return without error and the resulting
	// SOPDocument.Runbooks must be nil (not []Runbook{}).
	payloadJSON := `{
		"contractVersion": "ds.sop_document.v1",
		"sopId": "SOP-LEGACY",
		"title": "Legacy SOP",
		"version": "v1",
		"checksum": "sha256:abc123",
		"source": {"type": "managed_markdown", "sourceId": "source-1"},
		"bodyMarkdown": "Legacy content",
		"ownerTeam": "ops",
		"approvalStatus": "approved",
		"tenantScope": {"projectIds": ["proj-1"], "environments": ["prod"]},
		"updatedAt": "2026-05-20T09:00:00Z",
		"securityContext": {
			"serviceAccountProfile": "sa@example.com",
			"secretRefVisible": false,
			"browserCredentialsUsed": false,
			"redactionApplied": true
		}
	}`

	storable := &StorableSOPDocument{
		Payload: payloadJSON,
	}

	restored, err := storable.ToDomain()
	require.NoError(t, err)
	require.Nil(t, restored.Runbooks, "legacy payload without runbooks should deserialize to nil, not empty slice")
}
