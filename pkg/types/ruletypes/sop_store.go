package ruletypes

import (
	"context"
	"errors"
)

// ErrSOPDocumentNotFound is returned by SOPStore lookups when no row matches
// the given (orgID, sopID[, version]) triple. Cross-tenant lookups (orgID
// mismatch) also map to this error so callers cannot infer whether the SOP
// exists in another tenant.
var ErrSOPDocumentNotFound = errors.New("sop document not found")

// SOPStore persists SOP documents partitioned by orgID. Every method is
// scoped by orgID; cross-tenant reads return ErrSOPDocumentNotFound.
//
// NOTE: GetLatest orders by version DESC lexicographically. Callers must
// use version strings that sort correctly under string comparison (e.g.,
// "v01", "v02", ..., "v10" with zero padding, or ISO-style "2026-05-20").
// Plain "v1", "v2", ..., "v10" will sort "v10" < "v2".
type SOPStore interface {
	Upsert(ctx context.Context, orgID string, doc SOPDocument) error
	Get(ctx context.Context, orgID, sopID, version string) (SOPDocument, error)
	GetLatest(ctx context.Context, orgID, sopID string) (SOPDocument, error)
	List(ctx context.Context, orgID string) ([]SOPDocument, error)
	Delete(ctx context.Context, orgID, sopID, version string) error

	// UpsertRunbook inserts or replaces a runbook on the SOP identified by
	// (orgID, sopID, version). Read-modify-write happens inside a single
	// transaction so concurrent callers cannot interleave a half-written
	// runbook array. Returns ErrSOPDocumentNotFound if the parent SOP is
	// missing.
	UpsertRunbook(ctx context.Context, orgID, sopID, version string, rb Runbook) error

	// DeleteRunbook removes a runbook by ID from the named SOP version.
	// Returns ErrSOPDocumentNotFound if the parent SOP is missing or if the
	// runbookID does not exist on it.
	DeleteRunbook(ctx context.Context, orgID, sopID, version, runbookID string) error
}
