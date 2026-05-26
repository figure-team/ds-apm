// Package mockrunbookdrafter provides a fixed-response RunbookDrafter for
// handler tests that need a working drafter without a real LLM.
package mockrunbookdrafter

import (
	"context"
	"errors"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// Mock is a RunbookDrafter that returns the same Runbook on every call.
type Mock struct {
	runbook ruletypes.Runbook
	err     error
}

// New returns a Mock that produces the supplied Runbook on every Draft call.
func New(r ruletypes.Runbook) *Mock { return &Mock{runbook: r} }

// NewError returns a Mock whose Draft always fails with the supplied message.
func NewError(msg string) *Mock { return &Mock{err: errors.New(msg)} }

func (m *Mock) Draft(_ context.Context, _ ruletypes.RunbookDraftRequest) (ruletypes.Runbook, error) {
	if m.err != nil {
		return ruletypes.Runbook{}, m.err
	}
	return m.runbook, nil
}

var _ ruletypes.RunbookDrafter = (*Mock)(nil)
