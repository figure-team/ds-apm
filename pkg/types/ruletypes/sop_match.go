package ruletypes

import (
	"sort"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
)

const (
	// SOPBindingResolutionLabelMatch marks a binding resolved by matching a
	// combination of alert labels (service/severity/team) rather than an
	// explicit sop_id label. Additive to the v1 resolution set so the WT-ai
	// dispatch hook keeps consuming Status/SOPID/Version unchanged.
	SOPBindingResolutionLabelMatch = "label_match"
	// SOPBindingResolutionFallback marks a response that carries only
	// approximate (partial-match) candidates because no SOP matched every
	// present label dimension. Such a response is never bound.
	SOPBindingResolutionFallback = "fallback"

	// SOPStalenessMaxAge bounds how old a SOP's UpdatedAt may be before it is
	// excluded from label-combo matching (FR-CF1.5).
	SOPStalenessMaxAge = 90 * 24 * time.Hour

	// sopCandidateLimit caps how many ranked candidates a single preview
	// response carries, for both label_match and fallback resolutions.
	sopCandidateLimit = 5

	// SOPBindingNoExactMatchWarning is surfaced when no SOP matched every
	// present label dimension but approximate candidates exist. NF-5.5.1
	// forbids silently dropping near-matches, so they are returned as a
	// fallback candidate list with this warning.
	SOPBindingNoExactMatchWarning = "no exact SOP match for alert labels; surfacing approximate candidates"
	// SOPBindingStaleWarning is attached to an explicit sop_id binding whose
	// document has not been updated within SOPStalenessMaxAge. Label-combo
	// matching excludes stale documents outright; the explicit path keeps the
	// user-intended binding but flags it.
	SOPBindingStaleWarning = "sop document was not updated within 90 days"
)

// sopMatchDimensions lists the alert label keys used for label-combo grounding,
// in fixed priority order (strongest grounding signal first). This ordering is
// the deterministic rule used both for tie-breaking and for rendering the
// matchedOn list on each candidate.
var sopMatchDimensions = []string{
	alertmanagertypes.IncidentLabelOwnerTeam,   // owner_team
	alertmanagertypes.IncidentLabelServiceName, // service.name
	alertmanagertypes.IncidentLabelSeverity,    // severity
}

// sopDimensionPriority is a fixed rule-based weight used only to break ties
// between candidates that matched the same number of dimensions. It is
// intentionally NOT a tunable or semantic weight; §9.1 keeps grounding to label
// combinations.
func sopDimensionPriority(dim string) int {
	switch dim {
	case alertmanagertypes.IncidentLabelOwnerTeam:
		return 4
	case alertmanagertypes.IncidentLabelServiceName:
		return 2
	case alertmanagertypes.IncidentLabelSeverity:
		return 1
	default:
		return 0
	}
}

// SOPBindingCandidate is one ranked SOP suggestion produced by label-combo
// matching. It carries the same summary fields a bound response exposes, plus
// the match score and the dimensions that matched, so downstream consumers can
// render or re-rank without re-fetching documents.
type SOPBindingCandidate struct {
	SOPID     string   `json:"sopId"`
	Version   string   `json:"version"`
	Title     string   `json:"title"`
	SourceID  string   `json:"sourceId,omitempty"`
	OwnerTeam string   `json:"ownerTeam,omitempty"`
	Score     int      `json:"score"`
	MatchedOn []string `json:"matchedOn,omitempty"`
}

type sopMatchEntry struct {
	doc       SOPDocument
	matchedOn []string
	score     int
	priority  int
}

// previewSOPDocumentBindingAt is the time-injectable core of
// PreviewSOPDocumentBinding. Tests drive staleness deterministically by passing
// an explicit now.
func previewSOPDocumentBindingAt(docs []SOPDocument, req SOPBindingPreviewRequest, now time.Time) (SOPBindingPreviewResponse, error) {
	if strings.TrimSpace(req.Labels[alertmanagertypes.IncidentLabelSopID]) != "" {
		return previewExplicitSOPBinding(docs, req, now)
	}
	return previewLabelComboBinding(docs, req, now)
}

// previewExplicitSOPBinding preserves the v1 explicit-label behaviour: an alert
// carrying sop_id resolves to that exact document (highest priority,
// backward compatible). The only additive change is a staleness warning.
func previewExplicitSOPBinding(docs []SOPDocument, req SOPBindingPreviewRequest, now time.Time) (SOPBindingPreviewResponse, error) {
	sopID := strings.TrimSpace(req.Labels[alertmanagertypes.IncidentLabelSopID])

	doc, ok := latestSOPDocumentByID(docs, sopID)
	if !ok {
		return SOPBindingPreviewResponse{
			ContractVersion: SOPBindingContractVersion,
			Status:          SOPBindingStatusMissing,
			Resolution:      SOPBindingResolutionExplicitLabel,
			SOPID:           sopID,
			Warnings:        []string{"sop document was not found"},
		}, nil
	}

	tenant := PilotTenantFromLabels(req.Labels)
	if !PilotTenantIsComplete(tenant) {
		return SOPBindingPreviewResponse{
			ContractVersion: SOPBindingContractVersion,
			Status:          SOPBindingStatusMissing,
			Resolution:      SOPBindingResolutionExplicitLabel,
			SOPID:           sopID,
			Warnings:        []string{SOPTenantPolicyMissingLabelsWarning},
		}, nil
	}
	if !PilotTenantScopeAllows(doc.TenantScope, tenant) {
		return SOPBindingPreviewResponse{
			ContractVersion: SOPBindingContractVersion,
			Status:          SOPBindingStatusForbidden,
			Resolution:      SOPBindingResolutionExplicitLabel,
			SOPID:           sopID,
			Warnings:        []string{SOPTenantPolicyDeniedWarning},
		}, nil
	}

	status := SOPBindingStatusBound
	warnings := []string{}
	if doc.ApprovalStatus == SOPApprovalStatusDisabled {
		status = SOPBindingStatusDisabled
		warnings = append(warnings, "sop document is disabled")
	}
	if sopIsStale(doc, now) {
		warnings = append(warnings, SOPBindingStaleWarning)
	}

	resp := SOPBindingPreviewResponse{
		ContractVersion: SOPBindingContractVersion,
		Status:          status,
		Resolution:      SOPBindingResolutionExplicitLabel,
		SOPID:           doc.SOPID,
		Version:         doc.Version,
		Title:           doc.Title,
		SourceID:        doc.Source.SourceID,
		Warnings:        warnings,
	}

	return resp, ValidateSOPBindingPreviewResponse(resp)
}

// previewLabelComboBinding ranks SOPs against the service/severity/team labels
// when no explicit sop_id is present. A candidate that matches every present
// dimension binds (resolution label_match); otherwise approximate candidates
// are surfaced as a fallback (resolution fallback, not bound).
func previewLabelComboBinding(docs []SOPDocument, req SOPBindingPreviewRequest, now time.Time) (SOPBindingPreviewResponse, error) {
	tenant := PilotTenantFromLabels(req.Labels)
	if !PilotTenantIsComplete(tenant) {
		return SOPBindingPreviewResponse{
			ContractVersion: SOPBindingContractVersion,
			Status:          SOPBindingStatusMissing,
			Resolution:      SOPBindingResolutionNoMatch,
			Warnings:        []string{SOPTenantPolicyMissingLabelsWarning},
		}, nil
	}

	present := presentMatchDimensions(req.Labels)
	if len(present) == 0 {
		return SOPBindingPreviewResponse{
			ContractVersion: SOPBindingContractVersion,
			Status:          SOPBindingStatusMissing,
			Resolution:      SOPBindingResolutionNoMatch,
			Warnings:        []string{"sop_id label is not set"},
		}, nil
	}

	entries := collectSOPMatches(docs, req.Labels, tenant, now)
	if len(entries) == 0 {
		return SOPBindingPreviewResponse{
			ContractVersion: SOPBindingContractVersion,
			Status:          SOPBindingStatusMissing,
			Resolution:      SOPBindingResolutionNoMatch,
			Warnings:        []string{"no SOP document matched the alert labels"},
		}, nil
	}

	sortSOPMatches(entries)
	candidates := sopCandidatesFromEntries(entries)

	if top, ok := firstExactMatch(entries, len(present)); ok {
		status := SOPBindingStatusBound
		warnings := []string{}
		if top.doc.ApprovalStatus == SOPApprovalStatusDisabled {
			status = SOPBindingStatusDisabled
			warnings = append(warnings, "sop document is disabled")
		}
		resp := SOPBindingPreviewResponse{
			ContractVersion: SOPBindingContractVersion,
			Status:          status,
			Resolution:      SOPBindingResolutionLabelMatch,
			SOPID:           top.doc.SOPID,
			Version:         top.doc.Version,
			Title:           top.doc.Title,
			SourceID:        top.doc.Source.SourceID,
			Warnings:        warnings,
			Candidates:      candidates,
		}
		return resp, ValidateSOPBindingPreviewResponse(resp)
	}

	resp := SOPBindingPreviewResponse{
		ContractVersion: SOPBindingContractVersion,
		Status:          SOPBindingStatusMissing,
		Resolution:      SOPBindingResolutionFallback,
		Warnings:        []string{SOPBindingNoExactMatchWarning},
		Candidates:      candidates,
	}
	return resp, ValidateSOPBindingPreviewResponse(resp)
}

// collectSOPMatches reduces docs to the latest non-stale version per SOPID that
// is allowed for the tenant and matches at least one present label dimension.
func collectSOPMatches(docs []SOPDocument, labels map[string]string, tenant PilotAuditTenant, now time.Time) []sopMatchEntry {
	latest := map[string]SOPDocument{}
	for _, doc := range docs {
		sopID := strings.TrimSpace(doc.SOPID)
		if sopID == "" {
			continue
		}
		if !PilotTenantScopeAllows(doc.TenantScope, tenant) {
			continue
		}
		if sopIsStale(doc, now) {
			continue
		}
		if cur, ok := latest[sopID]; !ok || strings.TrimSpace(doc.Version) > strings.TrimSpace(cur.Version) {
			latest[sopID] = doc
		}
	}

	entries := make([]sopMatchEntry, 0, len(latest))
	for _, doc := range latest {
		matched := matchedDimensions(doc, labels)
		if len(matched) == 0 {
			continue
		}
		priority := 0
		for _, dim := range matched {
			priority += sopDimensionPriority(dim)
		}
		entries = append(entries, sopMatchEntry{
			doc:       doc,
			matchedOn: matched,
			score:     len(matched),
			priority:  priority,
		})
	}
	return entries
}

func firstExactMatch(entries []sopMatchEntry, presentDimCount int) (sopMatchEntry, bool) {
	for _, e := range entries {
		if e.score == presentDimCount {
			return e, true
		}
	}
	return sopMatchEntry{}, false
}

func sortSOPMatches(entries []sopMatchEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.score != b.score {
			return a.score > b.score
		}
		if a.priority != b.priority {
			return a.priority > b.priority
		}
		av, bv := strings.TrimSpace(a.doc.Version), strings.TrimSpace(b.doc.Version)
		if av != bv {
			return av > bv
		}
		return strings.TrimSpace(a.doc.SOPID) < strings.TrimSpace(b.doc.SOPID)
	})
}

func sopCandidatesFromEntries(entries []sopMatchEntry) []SOPBindingCandidate {
	limit := len(entries)
	if limit > sopCandidateLimit {
		limit = sopCandidateLimit
	}
	candidates := make([]SOPBindingCandidate, 0, limit)
	for _, e := range entries[:limit] {
		candidates = append(candidates, SOPBindingCandidate{
			SOPID:     e.doc.SOPID,
			Version:   e.doc.Version,
			Title:     e.doc.Title,
			SourceID:  e.doc.Source.SourceID,
			OwnerTeam: e.doc.OwnerTeam,
			Score:     e.score,
			MatchedOn: e.matchedOn,
		})
	}
	return candidates
}

func presentMatchDimensions(labels map[string]string) []string {
	present := make([]string, 0, len(sopMatchDimensions))
	for _, dim := range sopMatchDimensions {
		if strings.TrimSpace(labels[dim]) != "" {
			present = append(present, dim)
		}
	}
	return present
}

// matchedDimensions returns, in fixed priority order, the present alert
// dimensions that the document matches.
func matchedDimensions(doc SOPDocument, labels map[string]string) []string {
	matched := make([]string, 0, len(sopMatchDimensions))
	for _, dim := range sopMatchDimensions {
		value := strings.TrimSpace(labels[dim])
		if value == "" {
			continue
		}
		if sopMatchesDimension(doc, dim, value) {
			matched = append(matched, dim)
		}
	}
	return matched
}

func sopMatchesDimension(doc SOPDocument, dim, value string) bool {
	switch dim {
	case alertmanagertypes.IncidentLabelOwnerTeam:
		return strings.EqualFold(strings.TrimSpace(doc.OwnerTeam), value)
	case alertmanagertypes.IncidentLabelServiceName, alertmanagertypes.IncidentLabelSeverity:
		return sopTagsMatch(doc.Tags, dim, value)
	default:
		return false
	}
}

// sopTagsMatch matches an alert dimension value against a SOP's tags. A bare
// tag equal to the value matches (e.g. tag "payment-api" for service.name), as
// does a key:value / key=value tag whose key aliases the dimension
// (e.g. "service:payment-api", "severity=critical").
func sopTagsMatch(tags []string, dim, value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if strings.EqualFold(tag, value) {
			return true
		}
		if key, val, ok := splitTagKeyValue(tag); ok {
			if strings.EqualFold(val, value) && tagKeyMatchesDimension(key, dim) {
				return true
			}
		}
	}
	return false
}

func splitTagKeyValue(tag string) (string, string, bool) {
	for _, sep := range []string{":", "="} {
		if i := strings.Index(tag, sep); i > 0 && i < len(tag)-1 {
			return strings.TrimSpace(tag[:i]), strings.TrimSpace(tag[i+1:]), true
		}
	}
	return "", "", false
}

func tagKeyMatchesDimension(key, dim string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	switch dim {
	case alertmanagertypes.IncidentLabelServiceName:
		return key == "service" || key == "service.name" || key == "service_name"
	case alertmanagertypes.IncidentLabelSeverity:
		return key == "severity" || key == "sev"
	default:
		return false
	}
}

func sopIsStale(doc SOPDocument, now time.Time) bool {
	ts := strings.TrimSpace(doc.UpdatedAt)
	if ts == "" {
		return false
	}
	updated, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		// Lenient: an unparseable timestamp is not treated as stale so a
		// formatting quirk never silently drops an otherwise-valid SOP.
		return false
	}
	return now.Sub(updated) > SOPStalenessMaxAge
}
