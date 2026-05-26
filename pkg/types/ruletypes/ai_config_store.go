package ruletypes

import (
	"context"
	"errors"
)

var ErrAIConfigNotFound = errors.New("ai config not found")

type AIConfigStore interface {
	Upsert(ctx context.Context, cfg AIConfig, encrypt func(string) (string, error)) error
	Get(ctx context.Context, orgID string, decrypt func(string) (string, error)) (AIConfig, error)
}
