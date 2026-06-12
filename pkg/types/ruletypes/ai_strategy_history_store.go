package ruletypes

import (
	"context"
	"errors"
)

// ErrAIStrategyHistoryNotFound is returned when no row matches the given
// (orgID, lookup) tuple. Cross-tenant lookups map to this error so callers
// cannot infer history existence in another tenant.
var ErrAIStrategyHistoryNotFound = errors.New("ai strategy history not found")

// AIStrategyHistoryStore persists AI strategy previews — one row per
// (orgID, incidentID), partitioned by orgID so cross-tenant lookups
// return (false, nil) / no rows instead of revealing another tenant's record.
//
// Multiple incidents may share an alertFingerprint (recurrences of the same
// failure signature); ListRecent surfaces those past occurrences so the
// generator can reference prior incidents of the same failure.
type AIStrategyHistoryStore interface {
	Upsert(ctx context.Context, orgID string, record AIStrategyHistoryRecord) error
	GetLatest(ctx context.Context, orgID string, lookup AIStrategyHistoryLookupRequest) (AIStrategyHistoryRecord, bool, error)
	// ListRecent returns up to limit past records matching the lookup
	// (typically by alertFingerprint — the same-failure signature), most
	// recent first by generatedAt. A non-positive limit returns no rows.
	ListRecent(ctx context.Context, orgID string, lookup AIStrategyHistoryLookupRequest, limit int) ([]AIStrategyHistoryRecord, error)
}
