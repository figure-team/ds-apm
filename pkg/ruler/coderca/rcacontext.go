package coderca

import (
	"context"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// RCAContext is the v1 input to a code-RCA run (design §7). It carries only what
// the dispatch hook actually has — labels/annotations + a derived error
// signature — plus the resolved baseline the checkout is pinned at. Richer
// log/trace evidence arrives later via EvidenceCollector without changing this
// shape.
type RCAContext struct {
	OrgID          string
	Service        string
	Severity       string
	Environment    string
	Fingerprint    string
	ErrorSignature string
	BaselineCommit string
	Labels         map[string]string
	Annotations    map[string]string
}

// EvidenceCollector gathers supporting evidence (logs / traces / metrics) for a
// run's error context. v1 ships NoopEvidenceCollector; a SigNoz-querying
// collector is a later, additive implementation (design §7) that needs no
// change to RCAContext or the prompt builder.
type EvidenceCollector interface {
	Collect(ctx context.Context, rc RCAContext) ([]ruletypes.AIEvidenceRef, error)
}

// NoopEvidenceCollector returns no evidence — the v1 default, so the prompt is
// built from labels/annotations + signature alone.
type NoopEvidenceCollector struct{}

// Collect implements EvidenceCollector.
func (NoopEvidenceCollector) Collect(context.Context, RCAContext) ([]ruletypes.AIEvidenceRef, error) {
	return nil, nil
}

var _ EvidenceCollector = NoopEvidenceCollector{}
