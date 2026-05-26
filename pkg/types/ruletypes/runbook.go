package ruletypes

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	// Status enum
	RunbookStatusDraft      = "draft"
	RunbookStatusApproved   = "approved"
	RunbookStatusDeprecated = "deprecated"

	// Size limits
	RunbookMaxTitleLen          = 200
	RunbookMaxDescriptionLen    = 50_000
	RunbookMaxScriptLen         = 65_536
	RunbookMaxSourceExampleLen  = 4_096
	RunbookMaxSourceExampleCount = 3
)

// Runbook is an executable mitigation procedure embedded inside an SOP. Each
// runbook bundles a markdown how-to with a bash script that operators run
// themselves. The system never executes these scripts in v0.1; storage is
// always plaintext (no encryption — these are not secrets, and operators
// review the content before running).
type Runbook struct {
	ID                  string   `json:"id"`                  // UUID v4, server-assigned
	Title               string   `json:"title"`               // 1..200 bytes (length checked on trimmed value for title)
	Description         string   `json:"description"`         // markdown, 0..50_000 bytes
	ExecutableScript    string   `json:"executableScript"`    // bash, 0..65_536 bytes, no NUL
	Status              string   `json:"status"`              // draft|approved|deprecated
	Confidence          float64  `json:"confidence"`          // 0.0..1.0
	AIDraftedBy         string   `json:"aiDraftedBy"`         // model name or "" if human
	SourceErrorExamples []string `json:"sourceErrorExamples"` // 0..3 entries, each <= 4096 chars
	CreatedAt           string   `json:"createdAt"`           // RFC3339
	UpdatedAt           string   `json:"updatedAt"`           // RFC3339
	UpdatedBy           string   `json:"updatedBy"`           // user displayName | "ai" | "system"
}

var allowedRunbookStatuses = map[string]struct{}{
	RunbookStatusDraft:      {},
	RunbookStatusApproved:   {},
	RunbookStatusDeprecated: {},
}

// uuidV4Pattern is intentionally lenient: hex with dashes in the standard
// 8-4-4-4-12 layout. The runbook's ID is server-assigned, so we don't need
// strict v4 byte semantics — just a sane shape.
var uuidV4Pattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidateRunbook returns nil when r is well-formed, otherwise a joined error
// describing every field-level problem at once. Style mirrors ValidateSOPDocument
// in this package: pilotRequireNonEmpty / pilotRequireAllowed / pilotAppendSecretLikeStringErrors
// helpers (defined in pilot_contract.go) for the standard checks, raw fmt.Errorf
// for size/format checks that don't have a matching helper.
//
// Notes on what is NOT secret-like-checked:
//   - ExecutableScript: bash scripts legitimately reference env vars / tokens
//     by name (e.g. kubectl ... --token=$K8S_TOKEN). Applying the secret-like
//     check here would reject valid scripts.
//   - SourceErrorExamples: raw error logs may contain user IDs / IPs / tokens
//     (spec §9). Same policy as AIStrategyHistory.audit.error — stored as-is.
func ValidateRunbook(r Runbook) error {
	var errs []error

	if !uuidV4Pattern.MatchString(strings.TrimSpace(r.ID)) {
		errs = append(errs, fmt.Errorf("id: must be UUID v4 (got %q)", r.ID))
	}
	pilotRequireNonEmpty(&errs, "title", r.Title)
	if trimmedTitle := strings.TrimSpace(r.Title); len(trimmedTitle) > RunbookMaxTitleLen {
		errs = append(errs, fmt.Errorf("title: exceeds %d-byte limit (got %d)", RunbookMaxTitleLen, len(trimmedTitle)))
	}
	if len(r.Description) > RunbookMaxDescriptionLen {
		errs = append(errs, fmt.Errorf("description: exceeds %d-byte limit (got %d)", RunbookMaxDescriptionLen, len(r.Description)))
	}
	if len(r.ExecutableScript) > RunbookMaxScriptLen {
		errs = append(errs, fmt.Errorf("executableScript: exceeds %d-byte limit (got %d)", RunbookMaxScriptLen, len(r.ExecutableScript)))
	}
	if strings.ContainsRune(r.ExecutableScript, 0) {
		errs = append(errs, fmt.Errorf("executableScript: must not contain NUL byte"))
	}
	pilotRequireAllowed(&errs, "status", r.Status, allowedRunbookStatuses)
	if r.Confidence < 0.0 || r.Confidence > 1.0 {
		errs = append(errs, fmt.Errorf("confidence: %v out of range [0.0, 1.0]", r.Confidence))
	}
	if len(r.SourceErrorExamples) > RunbookMaxSourceExampleCount {
		errs = append(errs, fmt.Errorf("sourceErrorExamples: at most %d entries (got %d)", RunbookMaxSourceExampleCount, len(r.SourceErrorExamples)))
	}
	for i, ex := range r.SourceErrorExamples {
		if len(ex) > RunbookMaxSourceExampleLen {
			errs = append(errs, fmt.Errorf("sourceErrorExamples[%d]: exceeds %d-byte limit", i, RunbookMaxSourceExampleLen))
		}
	}

	pilotAppendSecretLikeStringErrors(&errs, "title", r.Title)
	pilotAppendSecretLikeStringErrors(&errs, "description", r.Description)
	pilotAppendSecretLikeStringErrors(&errs, "aiDraftedBy", r.AIDraftedBy)
	pilotAppendSecretLikeStringErrors(&errs, "updatedBy", r.UpdatedBy)

	return errors.Join(errs...)
}

// ValidateRunbookStatusTransition rejects same-status no-ops and the
// deprecated→approved direct shortcut (must transit through draft).
// Validity gate runs first so unknown statuses don't get misreported as a
// same-status no-op (e.g. "bogus" → "bogus").
func ValidateRunbookStatusTransition(from, to string) error {
	if _, ok := allowedRunbookStatuses[from]; !ok {
		return fmt.Errorf("status transition: from %q invalid", from)
	}
	if _, ok := allowedRunbookStatuses[to]; !ok {
		return fmt.Errorf("status transition: to %q invalid", to)
	}
	if from == to {
		return fmt.Errorf("status transition: %q → %q is a no-op", from, to)
	}
	if from == RunbookStatusDeprecated && to == RunbookStatusApproved {
		return fmt.Errorf("status transition: deprecated → approved forbidden (transit through draft first)")
	}
	return nil
}
