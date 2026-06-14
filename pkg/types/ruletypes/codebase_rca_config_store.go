package ruletypes

import (
	"context"
	"errors"
)

// ErrCodebaseRCAConfigNotFound is returned when no per-org config row exists;
// callers fall back to DefaultCodebaseRCAConfig.
var ErrCodebaseRCAConfigNotFound = errors.New("codebase RCA config not found")

// CodebaseRCAConfigStore persists the per-org CF-11 toggle + thresholds.
// No secrets — encryption closures are not needed.
type CodebaseRCAConfigStore interface {
	Upsert(ctx context.Context, cfg CodebaseRCAConfig) error
	Get(ctx context.Context, orgID string) (CodebaseRCAConfig, error)
}
