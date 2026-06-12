package ruletypes

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateSOPDocumentAcceptsManagedMarkdownDocument(t *testing.T) {
	doc := validSOPDocument()

	require.NoError(t, ValidateSOPDocument(doc))
	require.Equal(t, SOPDocumentContractVersion, doc.ContractVersion)
	require.Equal(t, PilotSOPSourceKindManagedMarkdown, doc.Source.Type)
	require.True(t, strings.HasPrefix(doc.Checksum, "sha256:"))
	require.False(t, doc.SecurityContext.SecretRefVisible)
	require.False(t, doc.SecurityContext.BrowserCredentialsUsed)
	require.True(t, doc.SecurityContext.RedactionApplied)
	require.Equal(t, []string{"customer-a"}, doc.TenantScope.ProjectIDs)
	require.Equal(t, []string{"prod"}, doc.TenantScope.Environments)
}

func TestNewSOPDocumentFromManagedMarkdownNormalizesAndValidates(t *testing.T) {
	source := validPilotManagedMarkdownSource()
	source.ServiceAccountProfile = " ds-sop-reader "
	source.Documents[0].SOPID = " SOP-PAY-001 "
	source.Documents[0].Title = " Payment API 5xx response "

	doc := NewSOPDocumentFromManagedMarkdown(source, source.Documents[0], " payments ", SOPApprovalStatusApproved)

	require.Equal(t, "SOP-PAY-001", doc.SOPID)
	require.Equal(t, "Payment API 5xx response", doc.Title)
	require.Equal(t, "payments", doc.OwnerTeam)
	require.Equal(t, "ds-sop-reader", doc.SecurityContext.ServiceAccountProfile)
	require.NoError(t, ValidateSOPDocument(doc))
}

func TestValidateSOPDocumentRejectsRequiredFieldAndEnumGaps(t *testing.T) {
	missingBody := validSOPDocument()
	missingBody.BodyMarkdown = ""
	require.ErrorContains(t, ValidateSOPDocument(missingBody), "bodyMarkdown: field is required")

	missingChecksum := validSOPDocument()
	missingChecksum.Checksum = ""
	require.ErrorContains(t, ValidateSOPDocument(missingChecksum), "checksum: field is required")

	missingTenant := validSOPDocument()
	missingTenant.TenantScope.ProjectIDs = nil
	require.ErrorContains(t, ValidateSOPDocument(missingTenant), "tenantScope.projectIds: must include at least one project")

	unsupportedStatus := validSOPDocument()
	unsupportedStatus.ApprovalStatus = "half_approved"
	require.ErrorContains(t, ValidateSOPDocument(unsupportedStatus), `approvalStatus: unsupported value "half_approved"`)

	unsupportedSource := validSOPDocument()
	unsupportedSource.Source.Type = "wiki_export"
	require.ErrorContains(t, ValidateSOPDocument(unsupportedSource), `source.type: unsupported value "wiki_export"`)
}

func TestValidateSOPDocumentRejectsUnsafeMarkdownPayloads(t *testing.T) {
	secretBody := validSOPDocument()
	secretBody.BodyMarkdown = "Rotate with access_token=hidden"
	require.ErrorContains(t, ValidateSOPDocument(secretBody), "bodyMarkdown: contains secret-like value")

	htmlBody := validSOPDocument()
	htmlBody.BodyMarkdown = "<!DOCTYPE html><html><body>SOP</body></html>"
	require.ErrorContains(t, ValidateSOPDocument(htmlBody), "bodyMarkdown: payload does not look like markdown")

	oversizedBody := validSOPDocument()
	oversizedBody.BodyMarkdown = strings.Repeat("a", SOPDocumentBodyMarkdownMaxBytes+1)
	require.ErrorContains(t, ValidateSOPDocument(oversizedBody), "bodyMarkdown: exceeds max size")
}

func TestValidateSOPDocumentRejectsCredentialBearingDisplayURL(t *testing.T) {
	doc := validSOPDocument()
	doc.DisplayURL = "https://kb.example/sop/SOP-PAY-001?token=hidden"

	err := ValidateSOPDocument(doc)
	require.Error(t, err)
	require.Contains(t, err.Error(), "displayUrl")
	require.Contains(t, err.Error(), "contains secret-like value")
}

func TestValidateSOPDocumentRejectsCredentialFlags(t *testing.T) {
	t.Run("secretRefVisible", func(t *testing.T) {
		doc := validSOPDocument()
		doc.SecurityContext.SecretRefVisible = true
		require.ErrorContains(t, ValidateSOPDocument(doc), "securityContext.secretRefVisible")
	})

	t.Run("browserCredentialsUsed", func(t *testing.T) {
		doc := validSOPDocument()
		doc.SecurityContext.BrowserCredentialsUsed = true
		require.ErrorContains(t, ValidateSOPDocument(doc), "securityContext.browserCredentialsUsed")
	})

	t.Run("redactionRequired", func(t *testing.T) {
		doc := validSOPDocument()
		doc.SecurityContext.RedactionApplied = false
		require.ErrorContains(t, ValidateSOPDocument(doc), "securityContext.redactionApplied")
	})
}

func TestValidateSOPDocumentAcceptsAllApprovalStatuses(t *testing.T) {
	for _, status := range []string{
		SOPApprovalStatusDraft,
		SOPApprovalStatusApproved,
		SOPApprovalStatusDeprecated,
		SOPApprovalStatusDisabled,
	} {
		t.Run(status, func(t *testing.T) {
			doc := validSOPDocument()
			doc.ApprovalStatus = status
			require.NoError(t, ValidateSOPDocument(doc))
		})
	}
}

func TestNewSOPDocumentListResponseOmitsBodyMarkdown(t *testing.T) {
	resp := NewSOPDocumentListResponse([]SOPDocument{validSOPDocument()})

	require.Equal(t, SOPDocumentListContractVersion, resp.ContractVersion)
	require.Len(t, resp.Documents, 1)
	require.Equal(t, "SOP-PAY-001", resp.Documents[0].SOPID)

	raw, err := json.Marshal(resp)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "bodyMarkdown")
}

func TestPreviewSOPDocumentBindingResolvesExplicitLabel(t *testing.T) {
	resp, err := PreviewSOPDocumentBinding([]SOPDocument{validSOPDocument()}, SOPBindingPreviewRequest{
		Labels: map[string]string{
			"environment": "prod",
			"project_id":  "customer-a",
			"sop_id":      "SOP-PAY-001",
		},
	})

	require.NoError(t, err)
	require.Equal(t, SOPBindingContractVersion, resp.ContractVersion)
	require.Equal(t, SOPBindingStatusBound, resp.Status)
	require.Equal(t, SOPBindingResolutionExplicitLabel, resp.Resolution)
	require.Equal(t, "SOP-PAY-001", resp.SOPID)
	require.Equal(t, "Payment API 5xx response", resp.Title)
	require.NoError(t, ValidateSOPBindingPreviewResponse(resp))
}

func TestPreviewSOPDocumentBindingReportsMissingAndDisabled(t *testing.T) {
	missing, err := PreviewSOPDocumentBinding([]SOPDocument{validSOPDocument()}, SOPBindingPreviewRequest{
		Labels: map[string]string{
			"environment": "prod",
			"project_id":  "customer-a",
			"sop_id":      "SOP-UNKNOWN",
		},
	})
	require.NoError(t, err)
	require.Equal(t, SOPBindingStatusMissing, missing.Status)
	require.Equal(t, SOPBindingResolutionExplicitLabel, missing.Resolution)

	disabledDoc := validSOPDocument()
	disabledDoc.ApprovalStatus = SOPApprovalStatusDisabled
	disabled, err := PreviewSOPDocumentBinding([]SOPDocument{disabledDoc}, SOPBindingPreviewRequest{
		Labels: map[string]string{
			"environment": "prod",
			"project_id":  "customer-a",
			"sop_id":      "SOP-PAY-001",
		},
	})
	require.NoError(t, err)
	require.Equal(t, SOPBindingStatusDisabled, disabled.Status)
	require.Contains(t, disabled.Warnings, "sop document is disabled")
}

func TestPreviewSOPDocumentBindingEnforcesTenantScope(t *testing.T) {
	missingTenant, err := PreviewSOPDocumentBinding([]SOPDocument{validSOPDocument()}, SOPBindingPreviewRequest{
		Labels: map[string]string{"sop_id": "SOP-PAY-001"},
	})
	require.NoError(t, err)
	require.Equal(t, SOPBindingStatusMissing, missingTenant.Status)
	require.Contains(t, missingTenant.Warnings, SOPTenantPolicyMissingLabelsWarning)

	forbidden, err := PreviewSOPDocumentBinding([]SOPDocument{validSOPDocument()}, SOPBindingPreviewRequest{
		Labels: map[string]string{
			"environment": "stage",
			"project_id":  "customer-b",
			"sop_id":      "SOP-PAY-001",
		},
	})
	require.NoError(t, err)
	require.Equal(t, SOPBindingStatusForbidden, forbidden.Status)
	require.Contains(t, forbidden.Warnings, SOPTenantPolicyDeniedWarning)
	require.NoError(t, ValidateSOPBindingPreviewResponse(forbidden))
}

func TestValidateSOPDocument_PropagatesRunbookValidationError(t *testing.T) {
	// Start from a valid SOPDocument fixture. Embed a runbook with an
	// invalid status so the runbook validator fails. Confirm
	// ValidateSOPDocument returns an error whose message contains both
	// "runbooks[0]" (the index prefix added by the new validation block)
	// and "status" (the underlying ValidateRunbook error term).
	doc := validSOPDocument()
	doc.Runbooks = []Runbook{{
		ID:               "01928374-5566-77ab-89cd-eeff00112233",
		Title:            "Bad runbook",
		ExecutableScript: "#!/bin/bash\nhi\n",
		Status:           "weird-status",
		Confidence:       0.5,
		CreatedAt:        "2026-05-22T00:00:00Z",
		UpdatedAt:        "2026-05-22T00:00:00Z",
		UpdatedBy:        "alice",
	}}

	err := ValidateSOPDocument(doc)
	if err == nil {
		t.Fatalf("expected validation error from embedded runbook; got nil")
	}
	if !strings.Contains(err.Error(), "runbooks[0]") {
		t.Fatalf("expected error to include runbooks[0] index prefix; got %v", err)
	}
	if !strings.Contains(err.Error(), "status") {
		t.Fatalf("expected error to include status (underlying ValidateRunbook); got %v", err)
	}
}

func validSOPDocument() SOPDocument {
	source := validPilotManagedMarkdownSource()
	return NewSOPDocumentFromManagedMarkdown(source, source.Documents[0], "payments", SOPApprovalStatusApproved)
}

// sopMatchNow is a fixed reference time so staleness assertions are
// deterministic. sopFresh/sopStale are UpdatedAt values relative to it.
var (
	sopMatchNow = time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC)
	sopFresh    = "2026-06-01T00:00:00Z" // 10 days old
	sopStale    = "2025-12-01T00:00:00Z" // > 90 days old
)

func sopMatchLabels() map[string]string {
	return map[string]string{
		"project_id":   "customer-a",
		"environment":  "prod",
		"service.name": "payment-api",
		"severity":     "critical",
		"owner_team":   "payments",
	}
}

func sopMatchDoc(sopID, ownerTeam string, tags []string, version, updatedAt string) SOPDocument {
	return SOPDocument{
		ContractVersion: SOPDocumentContractVersion,
		SOPID:           sopID,
		Title:           sopID + " runbook",
		Version:         version,
		Checksum:        "sha256:0000",
		Source: SOPDocumentSource{
			Type:     PilotSOPSourceKindManagedMarkdown,
			SourceID: "src-" + sopID,
		},
		BodyMarkdown:   "## " + sopID,
		OwnerTeam:      ownerTeam,
		ApprovalStatus: SOPApprovalStatusApproved,
		TenantScope: PilotTenantScope{
			ProjectIDs:   []string{"customer-a"},
			Environments: []string{"prod"},
		},
		Tags:      tags,
		UpdatedAt: updatedAt,
		SecurityContext: PilotAuditSecurityContext{
			ServiceAccountProfile: "ds-sop-reader",
			RedactionApplied:      true,
		},
	}
}

func TestMatch_MultiLabelRanking(t *testing.T) {
	docs := []SOPDocument{
		sopMatchDoc("SOP-PAY-001", "payments", []string{"payment-api", "critical"}, "2026-05-01.1", sopFresh), // team+service+severity = 3
		sopMatchDoc("SOP-PAY-002", "payments", []string{"payment-api"}, "2026-05-02.1", sopFresh),             // team+service = 2
		sopMatchDoc("SOP-OPS-001", "payments", []string{}, "2026-05-01.1", sopFresh),                          // team only = 1 (priority 4)
		sopMatchDoc("SOP-INF-001", "infra", []string{"critical"}, "2026-05-01.1", sopFresh),                   // severity only = 1 (priority 1)
	}

	resp, err := previewSOPDocumentBindingAt(docs, SOPBindingPreviewRequest{Labels: sopMatchLabels()}, sopMatchNow)
	require.NoError(t, err)
	require.Equal(t, SOPBindingStatusBound, resp.Status)
	require.Equal(t, SOPBindingResolutionLabelMatch, resp.Resolution)
	require.Equal(t, "SOP-PAY-001", resp.SOPID)

	require.Len(t, resp.Candidates, 4)
	require.Equal(t, "SOP-PAY-001", resp.Candidates[0].SOPID)
	require.Equal(t, 3, resp.Candidates[0].Score)
	require.Equal(t, []string{"owner_team", "service.name", "severity"}, resp.Candidates[0].MatchedOn)
	require.Equal(t, "SOP-PAY-002", resp.Candidates[1].SOPID)
	require.Equal(t, 2, resp.Candidates[1].Score)
	// Same score (1), team-priority (4) outranks severity-priority (1).
	require.Equal(t, "SOP-OPS-001", resp.Candidates[2].SOPID)
	require.Equal(t, "SOP-INF-001", resp.Candidates[3].SOPID)

	require.NoError(t, ValidateSOPBindingPreviewResponse(resp))
}

func TestMatch_FallbackCandidates(t *testing.T) {
	docs := []SOPDocument{
		sopMatchDoc("SOP-PAY-002", "payments", []string{"payment-api"}, "2026-05-02.1", sopFresh), // team+service = 2
		sopMatchDoc("SOP-INF-001", "infra", []string{"critical"}, "2026-05-01.1", sopFresh),       // severity = 1
	}

	resp, err := previewSOPDocumentBindingAt(docs, SOPBindingPreviewRequest{Labels: sopMatchLabels()}, sopMatchNow)
	require.NoError(t, err)
	require.Equal(t, SOPBindingStatusMissing, resp.Status)
	require.Equal(t, SOPBindingResolutionFallback, resp.Resolution)
	require.Empty(t, resp.SOPID, "fallback must not bind a SOP")
	require.Contains(t, resp.Warnings, SOPBindingNoExactMatchWarning)

	require.Len(t, resp.Candidates, 2)
	require.Equal(t, "SOP-PAY-002", resp.Candidates[0].SOPID)
	require.Equal(t, "SOP-INF-001", resp.Candidates[1].SOPID)
}

func TestMatch_StalenessExcluded(t *testing.T) {
	// The stale doc matches all dimensions and carries a higher version, but a
	// SOP not updated within 90 days is excluded so the fresh one binds.
	docs := []SOPDocument{
		sopMatchDoc("SOP-PAY-001", "payments", []string{"payment-api", "critical"}, "2026-05-01.1", sopFresh),
		sopMatchDoc("SOP-PAY-009", "payments", []string{"payment-api", "critical"}, "2026-06-09.9", sopStale),
	}

	resp, err := previewSOPDocumentBindingAt(docs, SOPBindingPreviewRequest{Labels: sopMatchLabels()}, sopMatchNow)
	require.NoError(t, err)
	require.Equal(t, SOPBindingStatusBound, resp.Status)
	require.Equal(t, "SOP-PAY-001", resp.SOPID)
	for _, c := range resp.Candidates {
		require.NotEqual(t, "SOP-PAY-009", c.SOPID, "stale SOP must be excluded from candidates")
	}

	// When only a stale SOP would match, there is no match at all.
	onlyStale := []SOPDocument{
		sopMatchDoc("SOP-PAY-009", "payments", []string{"payment-api", "critical"}, "2026-06-09.9", sopStale),
	}
	resp, err = previewSOPDocumentBindingAt(onlyStale, SOPBindingPreviewRequest{Labels: sopMatchLabels()}, sopMatchNow)
	require.NoError(t, err)
	require.Equal(t, SOPBindingStatusMissing, resp.Status)
	require.Equal(t, SOPBindingResolutionNoMatch, resp.Resolution)
	require.Empty(t, resp.Candidates)
}
