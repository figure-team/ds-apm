// Package delivery adapts a completed code-RCA run into a human-reviewable
// handoff (design §5.1 step 7 / §5.2 component 5). It owns the HITL framing —
// the result is a SUGGESTION, never applied — and the in-boundary formatting;
// the actual handoff transport (CF-3 handoff / incident annotation) is an
// injected HandoffSink, wired only at the integration seam (§11).
package delivery

import (
	"context"

	"github.com/SigNoz/signoz/pkg/ruler/coderca/engine"
)

// HandoffMessage is a transport-agnostic, human-reviewable RCA handoff.
type HandoffMessage struct {
	OrgID          string
	Service        string
	RunID          string
	BaselineCommit string
	Confidence     string
	Title          string
	Body           string // markdown
}

// HandoffSink delivers a HandoffMessage to a human and returns a reference
// (e.g. handoff / incident id). SEAM (§11): the concrete sink bridges to the
// CF-3 handoff / history system that lives in another branch; it is injected,
// never wired from this worktree.
type HandoffSink interface {
	Submit(ctx context.Context, msg HandoffMessage) (ref string, err error)
}

// FormatHandoff builds the HITL handoff message from a completed run. Pure.
//
// STEP-1 STUB: returns the zero message → formatting assertions fail (RED).
func FormatHandoff(d engine.Delivery) HandoffMessage {
	return HandoffMessage{}
}

// Deliverer implements engine.Deliverer by formatting the run and submitting it
// to the injected sink.
type Deliverer struct {
	sink HandoffSink
}

// New builds a Deliverer over the given sink.
func New(sink HandoffSink) *Deliverer {
	return &Deliverer{sink: sink}
}

// Deliver formats the completed run and submits it, returning the sink's ref.
//
// STEP-1 STUB: returns no ref without submitting → assertions fail (RED).
func (d *Deliverer) Deliver(ctx context.Context, del engine.Delivery) (string, error) {
	return "", nil
}

var _ engine.Deliverer = (*Deliverer)(nil)
