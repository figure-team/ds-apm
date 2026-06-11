package ruletypes

import (
	"context"
	"errors"
)

// ErrCodebaseServiceMapNotFound is returned when no mapping exists for a
// (org_id, service_name).
var ErrCodebaseServiceMapNotFound = errors.New("codebase service map not found")

// CodebaseServiceMapStore persists CF-11 service→repo mappings. All methods are
// org-scoped (design §8). No secrets — a mapping carries no credential.
type CodebaseServiceMapStore interface {
	Upsert(ctx context.Context, m CodebaseServiceMap) error
	Get(ctx context.Context, orgID, serviceName string) (CodebaseServiceMap, error)
	List(ctx context.Context, orgID string) ([]CodebaseServiceMap, error)
}
