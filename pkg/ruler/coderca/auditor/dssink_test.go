package auditor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/types/audittypes"
)

// fakeAuditFunc captures the AuditEvent passed to it.
type fakeAuditCapture struct {
	event audittypes.AuditEvent
	calls int
}

func (f *fakeAuditCapture) record(ctx context.Context, event audittypes.AuditEvent) {
	f.calls++
	f.event = event
}

func TestDSSinkRecordsAuditEvent(t *testing.T) {
	cap := &fakeAuditCapture{}
	sink := NewDSSink(cap.record)

	fixedAt := time.Unix(1_700_000_000, 0).UTC()
	rec := AuditRecord{
		OrgID:   "org-1",
		RunID:   "run-42",
		Service: "pay",
		Status:  coderca.RunStatusDone,
		Outcome: "success",
		Detail:  "delivered",
		At:      fixedAt,
	}

	sink.Record(context.Background(), rec)

	if cap.calls != 1 {
		t.Fatalf("audit func called %d times, want 1", cap.calls)
	}

	ev := cap.event

	// EventName must be "coderca.run.updated"
	if got := ev.EventName.String(); got != "coderca.run.updated" {
		t.Errorf("EventName = %q, want %q", got, "coderca.run.updated")
	}

	// Timestamp must match rec.At
	if !ev.Timestamp.Equal(fixedAt) {
		t.Errorf("Timestamp = %v, want %v", ev.Timestamp, fixedAt)
	}

	// Body must contain run ID, org ID and status
	for _, want := range []string{"run-42", "org-1", string(coderca.RunStatusDone)} {
		if !strings.Contains(ev.Body, want) {
			t.Errorf("Body %q does not contain %q", ev.Body, want)
		}
	}
}

func TestDSSinkNilAuditIsNoOp(t *testing.T) {
	sink := NewDSSink(nil)
	rec := AuditRecord{
		OrgID: "org-1", RunID: "run-1", At: time.Now(),
	}
	// Must not panic.
	sink.Record(context.Background(), rec)
}
