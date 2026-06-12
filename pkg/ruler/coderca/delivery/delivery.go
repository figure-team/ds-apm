// Package delivery adapts a completed code-RCA run into a human-reviewable
// handoff (design §5.1 step 7 / §5.2 component 5). It owns the HITL framing —
// the result is a SUGGESTION, never applied — and the in-boundary formatting;
// the actual handoff transport (CF-3 handoff / incident annotation) is an
// injected HandoffSink, wired only at the integration seam (§11).
package delivery

import (
	"context"
	"fmt"
	"strings"

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
func FormatHandoff(d engine.Delivery) HandoffMessage {
	r := d.Result
	baseline := firstNonEmpty(r.BaselineCommit, d.BaselineCommit)

	var b strings.Builder
	b.WriteString("> ⚠️ AI-generated root-cause **suggestion** — it has **not** been applied. Human review is required before any change.\n\n")
	fmt.Fprintf(&b, "**Service:** %s  \n", orNA(d.Service))
	fmt.Fprintf(&b, "**Analyzed baseline commit:** `%s`  \n", orNA(baseline))
	fmt.Fprintf(&b, "**Confidence:** %s\n\n", orNA(r.Confidence))
	b.WriteString("## Root cause\n")
	b.WriteString(orNA(r.RootCause) + "\n\n")
	b.WriteString("## Suggested fix (not applied)\n")
	b.WriteString(orNA(r.ProposedFix) + "\n")
	if strings.TrimSpace(r.Limitations) != "" {
		b.WriteString("\n## Limitations\n")
		b.WriteString(r.Limitations + "\n")
	}

	return HandoffMessage{
		OrgID:          d.OrgID,
		Service:        d.Service,
		RunID:          d.RunID,
		BaselineCommit: baseline,
		Confidence:     r.Confidence,
		Title:          fmt.Sprintf("Code RCA suggestion: %s (review required)", orNA(d.Service)),
		Body:           b.String(),
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func orNA(s string) string {
	if strings.TrimSpace(s) == "" {
		return "_(not provided)_"
	}
	return s
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
func (d *Deliverer) Deliver(ctx context.Context, del engine.Delivery) (string, error) {
	ref, err := d.sink.Submit(ctx, FormatHandoff(del))
	if err != nil {
		return "", fmt.Errorf("delivery: submit handoff: %w", err)
	}
	return ref, nil
}

var _ engine.Deliverer = (*Deliverer)(nil)
