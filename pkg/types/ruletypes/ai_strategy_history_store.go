package ruletypes

import (
	"context"
	"errors"
)

// ErrAIStrategyHistoryNotFound is returned when no row matches the given
// (orgID, lookup) tuple. Cross-tenant lookups map to this error so callers
// cannot infer history existence in another tenant.
var ErrAIStrategyHistoryNotFound = errors.New("ai strategy history not found")

// AIStrategyHistoryStore persists the latest AI strategy preview per
// (orgID, incidentID) — partitioned by orgID so cross-tenant lookups
// return (false, nil) instead of revealing another tenant's record.
type AIStrategyHistoryStore interface {
	Upsert(ctx context.Context, orgID string, record AIStrategyHistoryRecord) error
	GetLatest(ctx context.Context, orgID string, lookup AIStrategyHistoryLookupRequest) (AIStrategyHistoryRecord, bool, error)
}
