package ruletypes

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	SOPDocumentContractVersion = "ds.sop_document.v1"

	SOPApprovalStatusDraft      = "draft"
	SOPApprovalStatusApproved   = "approved"
	SOPApprovalStatusDeprecated = "deprecated"
	SOPApprovalStatusDisabled   = "disabled"

	SOPDocumentBodyMarkdownMaxBytes = PilotSOPFetchBodyMarkdownMaxBytes

	SOPDocumentListContractVersion = "ds.sop_document_list.v1"
	SOPBindingContractVersion      = "ds.sop_binding.v1"

	SOPBindingStatusBound     = "bound"
	SOPBindingStatusMissing   = "missing"
	SOPBindingStatusDisabled  = "disabled"
	SOPBindingStatusForbidden = "forbidden"

	SOPBindingResolutionExplicitLabel = "explicit_label"
	SOPBindingResolutionNoMatch       = "no_match"
)

type SOPDocument struct {
	ContractVersion string            `json:"contractVersion"`
	SOPID           string            `json:"sopId"`
	Title           string            `json:"title"`
	Version         string            `json:"version"`
	Checksum        string            `json:"checksum"`
	Source          SOPDocumentSource `json:"source"`
	BodyMarkdown    string            `json:"bodyMarkdown"`
	// CustomerUpdateTemplate / VendorRequestTemplate are optional org-approved
	// comms templates. When present, the AI generator fills their slots rather
	// than free-writing the customer/vendor draft, so external-facing wording
	// stays consistent and within approved bounds (CF-2 comms grounding).
	// Stored in the SOP payload blob — additive, no migration, no contract bump.
	CustomerUpdateTemplate string                    `json:"customerUpdateTemplate,omitempty"`
	VendorRequestTemplate  string                    `json:"vendorRequestTemplate,omitempty"`
	DisplayURL             string                    `json:"displayUrl,omitempty"`
	OwnerTeam              string                    `json:"ownerTeam"`
	ApprovalStatus         string                    `json:"approvalStatus"`
	TenantScope            PilotTenantScope          `json:"tenantScope"`
	Tags                   []string                  `json:"tags,omitempty"`
	Runbooks               []Runbook                 `json:"runbooks,omitempty"`
	UpdatedAt              string                    `json:"updatedAt"`
	SecurityContext        PilotAuditSecurityContext `json:"securityContext"`
}

type SOPDocumentSource struct {
	Type     string `json:"type"`
	SourceID string `json:"sourceId"`
}

type SOPDocumentSummary struct {
	ContractVersion string            `json:"contractVersion"`
	SOPID           string            `json:"sopId"`
	Title           string            `json:"title"`
	Version         string            `json:"version"`
	Checksum        string            `json:"checksum"`
	Source          SOPDocumentSource `json:"source"`
	DisplayURL      string            `json:"displayUrl,omitempty"`
	OwnerTeam       string            `json:"ownerTeam"`
	ApprovalStatus  string            `json:"approvalStatus"`
	TenantScope     PilotTenantScope  `json:"tenantScope"`
	Tags            []string          `json:"tags,omitempty"`
	UpdatedAt       string            `json:"updatedAt"`
}

type SOPDocumentListResponse struct {
	ContractVersion string               `json:"contractVersion"`
	Documents       []SOPDocumentSummary `json:"documents"`
}

type SOPBindingPreviewRequest struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type SOPBindingPreviewResponse struct {
	ContractVersion string                `json:"contractVersion"`
	Status          string                `json:"status"`
	Resolution      string                `json:"resolution"`
	SOPID           string                `json:"sopId,omitempty"`
	Version         string                `json:"version,omitempty"`
	Title           string                `json:"title,omitempty"`
	SourceID        string                `json:"sourceId,omitempty"`
	Warnings        []string              `json:"warnings,omitempty"`
	Candidates      []SOPBindingCandidate `json:"candidates,omitempty"`
}

func NewSOPDocumentFromManagedMarkdown(source PilotManagedMarkdownSource, doc PilotManagedMarkdownDocument, ownerTeam string, approvalStatus string) SOPDocument {
	return SOPDocument{
		ContractVersion: SOPDocumentContractVersion,
		SOPID:           strings.TrimSpace(doc.SOPID),
		Title:           strings.TrimSpace(doc.Title),
		Version:         strings.TrimSpace(doc.Version),
		Checksum:        checksumForSOPDocument(doc),
		Source: SOPDocumentSource{
			Type:     PilotSOPSourceKindManagedMarkdown,
			SourceID: strings.TrimSpace(source.SourceID),
		},
		BodyMarkdown:   doc.BodyMarkdown,
		DisplayURL:     strings.TrimSpace(doc.DisplayURL),
		OwnerTeam:      strings.TrimSpace(ownerTeam),
		ApprovalStatus: strings.TrimSpace(approvalStatus),
		TenantScope:    normalizePilotTenantScope(source.TenantScope),
		Tags:           doc.Tags,
		UpdatedAt:      strings.TrimSpace(doc.UpdatedAt),
		SecurityContext: PilotAuditSecurityContext{
			ServiceAccountProfile:  strings.TrimSpace(source.ServiceAccountProfile),
			SecretRefVisible:       false,
			BrowserCredentialsUsed: false,
			RedactionApplied:       true,
		},
	}
}

func NewSOPDocumentListResponse(docs []SOPDocument) SOPDocumentListResponse {
	resp := SOPDocumentListResponse{
		ContractVersion: SOPDocumentListContractVersion,
		Documents:       make([]SOPDocumentSummary, 0, len(docs)),
	}
	for _, doc := range docs {
		resp.Documents = append(resp.Documents, NewSOPDocumentSummary(doc))
	}

	return resp
}

func NewSOPDocumentSummary(doc SOPDocument) SOPDocumentSummary {
	return SOPDocumentSummary{
		ContractVersion: doc.ContractVersion,
		SOPID:           doc.SOPID,
		Title:           doc.Title,
		Version:         doc.Version,
		Checksum:        doc.Checksum,
		Source:          doc.Source,
		DisplayURL:      doc.DisplayURL,
		OwnerTeam:       doc.OwnerTeam,
		ApprovalStatus:  doc.ApprovalStatus,
		TenantScope:     normalizePilotTenantScope(doc.TenantScope),
		Tags:            doc.Tags,
		UpdatedAt:       doc.UpdatedAt,
	}
}

// PreviewSOPDocumentBinding resolves the best SOP for an alert. An explicit
// sop_id label keeps the v1 single-match behaviour (highest priority); without
// it, the service/severity/team labels are ranked into candidates (see
// sop_match.go). Output fields consumed by the WT-ai dispatch hook
// (Status/SOPID/Version) are unchanged; candidate ranking is additive.
func PreviewSOPDocumentBinding(docs []SOPDocument, req SOPBindingPreviewRequest) (SOPBindingPreviewResponse, error) {
	return previewSOPDocumentBindingAt(docs, req, time.Now().UTC())
}

func ValidateSOPDocument(doc SOPDocument) error {
	var errs []error

	pilotRequireContractVersion(&errs, "contractVersion", doc.ContractVersion, SOPDocumentContractVersion)
	pilotRequireNonEmpty(&errs, "sopId", doc.SOPID)
	pilotRequireNonEmpty(&errs, "title", doc.Title)
	pilotRequireNonEmpty(&errs, "version", doc.Version)
	pilotRequireNonEmpty(&errs, "checksum", doc.Checksum)
	if strings.TrimSpace(doc.Checksum) != "" && !strings.HasPrefix(strings.TrimSpace(doc.Checksum), "sha256:") {
		errs = append(errs, fmt.Errorf("checksum: must use sha256:<digest> format"))
	}
	pilotRequireAllowed(&errs, "source.type", doc.Source.Type, allowedSOPDocumentSourceTypes)
	pilotRequireNonEmpty(&errs, "source.sourceId", doc.Source.SourceID)
	pilotRequireNonEmpty(&errs, "bodyMarkdown", doc.BodyMarkdown)
	if len(doc.BodyMarkdown) > SOPDocumentBodyMarkdownMaxBytes {
		errs = append(errs, fmt.Errorf("bodyMarkdown: exceeds max size of %d bytes", SOPDocumentBodyMarkdownMaxBytes))
	}
	if pilotBodyMarkdownLooksNonMarkdown(doc.BodyMarkdown) {
		errs = append(errs, fmt.Errorf("bodyMarkdown: payload does not look like markdown"))
	}
	pilotRequireNonEmpty(&errs, "ownerTeam", doc.OwnerTeam)
	pilotRequireAllowed(&errs, "approvalStatus", doc.ApprovalStatus, allowedSOPApprovalStatuses)
	validatePilotTenantScope(&errs, "tenantScope", doc.TenantScope)
	pilotRequireNonEmpty(&errs, "updatedAt", doc.UpdatedAt)
	pilotRequireNonEmpty(&errs, "securityContext.serviceAccountProfile", doc.SecurityContext.ServiceAccountProfile)
	if doc.SecurityContext.SecretRefVisible {
		errs = append(errs, fmt.Errorf("securityContext.secretRefVisible: must be false for SOP document responses"))
	}
	if doc.SecurityContext.BrowserCredentialsUsed {
		errs = append(errs, fmt.Errorf("securityContext.browserCredentialsUsed: must be false for SOP document responses"))
	}
	if !doc.SecurityContext.RedactionApplied {
		errs = append(errs, fmt.Errorf("securityContext.redactionApplied: must be true before SOP document responses feed AI or browser surfaces"))
	}
	if strings.TrimSpace(doc.DisplayURL) != "" {
		if _, warning, ok := safeDisplayURL(doc.DisplayURL); !ok {
			errs = append(errs, fmt.Errorf("displayUrl: %s", warning))
		}
	}

	pilotAppendSecretLikeStringErrors(&errs, "sopId", doc.SOPID)
	pilotAppendSecretLikeStringErrors(&errs, "title", doc.Title)
	pilotAppendSecretLikeStringErrors(&errs, "version", doc.Version)
	pilotAppendSecretLikeStringErrors(&errs, "checksum", doc.Checksum)
	pilotAppendSecretLikeStringErrors(&errs, "source.sourceId", doc.Source.SourceID)
	pilotAppendSecretLikeStringErrors(&errs, "bodyMarkdown", doc.BodyMarkdown)
	pilotAppendSecretLikeStringErrors(&errs, "displayUrl", doc.DisplayURL)
	pilotAppendSecretLikeStringErrors(&errs, "ownerTeam", doc.OwnerTeam)
	pilotAppendSecretLikeStringErrors(&errs, "updatedAt", doc.UpdatedAt)
	pilotAppendSecretLikeStringErrors(&errs, "securityContext.serviceAccountProfile", doc.SecurityContext.ServiceAccountProfile)
	for i, tag := range doc.Tags {
		pilotAppendSecretLikeStringErrors(&errs, fmt.Sprintf("tags[%d]", i), tag)
	}

	for i, rb := range doc.Runbooks {
		if err := ValidateRunbook(rb); err != nil {
			errs = append(errs, fmt.Errorf("runbooks[%d]: %w", i, err))
		}
	}

	return errors.Join(errs...)
}

func ValidateSOPBindingPreviewResponse(resp SOPBindingPreviewResponse) error {
	var errs []error

	pilotRequireContractVersion(&errs, "contractVersion", resp.ContractVersion, SOPBindingContractVersion)
	pilotRequireAllowed(&errs, "status", resp.Status, allowedSOPBindingStatuses)
	pilotRequireAllowed(&errs, "resolution", resp.Resolution, allowedSOPBindingResolutions)
	if resp.Status == SOPBindingStatusBound || resp.Status == SOPBindingStatusDisabled {
		pilotRequireNonEmpty(&errs, "sopId", resp.SOPID)
		pilotRequireNonEmpty(&errs, "version", resp.Version)
		pilotRequireNonEmpty(&errs, "title", resp.Title)
		pilotRequireNonEmpty(&errs, "sourceId", resp.SourceID)
	}
	if resp.Status == SOPBindingStatusForbidden {
		pilotRequireNonEmpty(&errs, "sopId", resp.SOPID)
	}

	pilotAppendSecretLikeStringErrors(&errs, "sopId", resp.SOPID)
	pilotAppendSecretLikeStringErrors(&errs, "version", resp.Version)
	pilotAppendSecretLikeStringErrors(&errs, "title", resp.Title)
	pilotAppendSecretLikeStringErrors(&errs, "sourceId", resp.SourceID)
	for i, warning := range resp.Warnings {
		pilotAppendSecretLikeStringErrors(&errs, fmt.Sprintf("warnings[%d]", i), warning)
	}
	for i, candidate := range resp.Candidates {
		pilotAppendSecretLikeStringErrors(&errs, fmt.Sprintf("candidates[%d].sopId", i), candidate.SOPID)
		pilotAppendSecretLikeStringErrors(&errs, fmt.Sprintf("candidates[%d].version", i), candidate.Version)
		pilotAppendSecretLikeStringErrors(&errs, fmt.Sprintf("candidates[%d].title", i), candidate.Title)
		pilotAppendSecretLikeStringErrors(&errs, fmt.Sprintf("candidates[%d].sourceId", i), candidate.SourceID)
		pilotAppendSecretLikeStringErrors(&errs, fmt.Sprintf("candidates[%d].ownerTeam", i), candidate.OwnerTeam)
	}

	return errors.Join(errs...)
}

var allowedSOPDocumentSourceTypes = map[string]struct{}{
	PilotSOPSourceKindManagedMarkdown: {},
	PilotSOPSourceKindGitMarkdown:     {},
	PilotSOPSourceKindConfluence:      {},
	PilotSOPSourceKindNotion:          {},
	PilotSOPSourceKindSharePoint:      {},
	PilotSOPSourceKindURLRegistry:     {},
	PilotSOPSourceKindCustomConnector: {},
}

var allowedSOPApprovalStatuses = map[string]struct{}{
	SOPApprovalStatusDraft:      {},
	SOPApprovalStatusApproved:   {},
	SOPApprovalStatusDeprecated: {},
	SOPApprovalStatusDisabled:   {},
}

var allowedSOPBindingStatuses = map[string]struct{}{
	SOPBindingStatusBound:     {},
	SOPBindingStatusMissing:   {},
	SOPBindingStatusDisabled:  {},
	SOPBindingStatusForbidden: {},
}

var allowedSOPBindingResolutions = map[string]struct{}{
	SOPBindingResolutionExplicitLabel: {},
	SOPBindingResolutionNoMatch:       {},
	SOPBindingResolutionLabelMatch:    {},
	SOPBindingResolutionFallback:      {},
}

func checksumForSOPDocument(doc PilotManagedMarkdownDocument) string {
	if strings.TrimSpace(doc.BodyMarkdown) == "" {
		return "sha256:unavailable"
	}

	sum := sha256.Sum256([]byte(doc.BodyMarkdown))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func latestSOPDocumentByID(docs []SOPDocument, sopID string) (SOPDocument, bool) {
	sopID = strings.TrimSpace(sopID)
	var latest SOPDocument
	found := false
	for _, doc := range docs {
		if strings.TrimSpace(doc.SOPID) != sopID {
			continue
		}
		if !found || strings.TrimSpace(doc.Version) > strings.TrimSpace(latest.Version) {
			latest = doc
			found = true
		}
	}

	return latest, found
}

// isSOPApproved reports whether a document is in the approved state — the only
// approval status eligible to ground AI strategies or bind to an alert.
func isSOPApproved(doc SOPDocument) bool {
	return strings.TrimSpace(doc.ApprovalStatus) == SOPApprovalStatusApproved
}

// latestApprovedSOPDocumentByID returns the highest-version APPROVED document
// for sopID. A newer draft/deprecated/disabled version never shadows an older
// approved one — binding policy recognises approved SOPs only.
func latestApprovedSOPDocumentByID(docs []SOPDocument, sopID string) (SOPDocument, bool) {
	sopID = strings.TrimSpace(sopID)
	var latest SOPDocument
	found := false
	for _, doc := range docs {
		if strings.TrimSpace(doc.SOPID) != sopID {
			continue
		}
		if !isSOPApproved(doc) {
			continue
		}
		if !found || strings.TrimSpace(doc.Version) > strings.TrimSpace(latest.Version) {
			latest = doc
			found = true
		}
	}

	return latest, found
}

const (
	SOPBatchResultContractVersion = "ds.sop_batch_result.v1"
	SOPBatchResultStatusOk        = "ok"
	SOPBatchResultStatusError     = "error"
)

type SOPDocumentBatchRequest struct {
	ContractVersion string        `json:"contractVersion"`
	Documents       []SOPDocument `json:"documents"`
}

type SOPDocumentBatchResponse struct {
	ContractVersion string                   `json:"contractVersion"`
	Total           int                      `json:"total"`
	Succeeded       int                      `json:"succeeded"`
	Failed          int                      `json:"failed"`
	Results         []SOPDocumentBatchResult `json:"results"`
}

type SOPDocumentBatchResult struct {
	SOPID   string `json:"sopId"`
	Version string `json:"version"`
	Status  string `json:"status"` // "ok" | "error"
	Error   string `json:"error,omitempty"`
}
