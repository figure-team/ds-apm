// Package auditor adapts a finalized code-RCA run into a stable audit record
// (design §5.1 step 8 / CF-6). It owns the in-boundary mapping (stable event
// name, outcome classification, timestamp); the actual audit transport
// (auditor.Audit, drop-on-full) is an injected Sink, wired only at the
// integration seam (§11).
package auditor

import (
	"context"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/engine"
)

// EventName is the stable audit event name for a finalized run, so consumers
// can filter CF-11 runs without knowing coderca internals.
const EventName = "coderca.run_finalized"

// AuditRecord is the transport-agnostic audit record for a finalized run.
type AuditRecord struct {
	EventName string
	OrgID     string
	RunID     string
	Service   string
	Status    coderca.RunStatus
	Outcome   string // "success" | "failure"
	Detail    string
	At        time.Time
}

// Sink records an AuditRecord. Fire-and-forget (no return) to match the
// drop-on-full upstream auditor. SEAM (§11): the concrete sink bridges to CF-6
// auditor.Audit in another branch; it is injected, never wired here.
type Sink interface {
	Record(ctx context.Context, rec AuditRecord)
}

// Outcome classifies a terminal run status into a coarse success/failure for
// audit consumers.
//
func Outcome(status coderca.RunStatus) string {
	if status == coderca.RunStatusDone {
		return "success"
	}
	return "failure"
}

// Auditor implements engine.Auditor by mapping the event to an AuditRecord and
// recording it via the injected sink.
type Auditor struct {
	sink Sink
	now  func() time.Time
}

// New builds an Auditor over the given sink and clock (defaults to time.Now).
func New(sink Sink, now func() time.Time) *Auditor {
	if now == nil {
		now = time.Now
	}
	return &Auditor{sink: sink, now: now}
}

// Audit maps the engine event to an AuditRecord and records it (fire-and-forget).
//
func (a *Auditor) Audit(ctx context.Context, e engine.AuditEvent) {
	a.sink.Record(ctx, AuditRecord{
		EventName: EventName,
		OrgID:     e.OrgID,
		RunID:     e.RunID,
		Service:   e.Service,
		Status:    e.Status,
		Outcome:   Outcome(e.Status),
		Detail:    e.Detail,
		At:        a.now(),
	})
}

var _ engine.Auditor = (*Auditor)(nil)
