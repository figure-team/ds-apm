package auditor

import (
	"context"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/engine"
)

type fakeSink struct {
	got   AuditRecord
	calls int
}

func (f *fakeSink) Record(_ context.Context, rec AuditRecord) {
	f.calls++
	f.got = rec
}

func fixedNow() time.Time { return time.Unix(1_700_000_000, 0).UTC() }

func TestOutcome(t *testing.T) {
	tests := []struct {
		status coderca.RunStatus
		want   string
	}{
		{coderca.RunStatusDone, "success"},
		{coderca.RunStatusFailed, "failure"},
		{coderca.RunStatusTimeout, "failure"},
		{coderca.RunStatusUnparseable, "failure"},
		{coderca.RunStatusQueued, "failure"},
		{coderca.RunStatusRunning, "failure"},
	}
	for _, tc := range tests {
		if got := Outcome(tc.status); got != tc.want {
			t.Errorf("Outcome(%q) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestAuditRecordsToSink(t *testing.T) {
	sink := &fakeSink{}
	New(sink, fixedNow).Audit(context.Background(), engine.AuditEvent{
		OrgID: "org1", RunID: "run-1", Service: "payments",
		Status: coderca.RunStatusDone, Detail: "delivered",
	})

	if sink.calls != 1 {
		t.Fatalf("sink.Record called %d times, want 1", sink.calls)
	}
	rec := sink.got
	if rec.EventName != EventName {
		t.Errorf("EventName = %q, want %q", rec.EventName, EventName)
	}
	if rec.OrgID != "org1" || rec.RunID != "run-1" || rec.Service != "payments" {
		t.Errorf("identifiers not mapped: %+v", rec)
	}
	if rec.Status != coderca.RunStatusDone {
		t.Errorf("Status = %q, want done", rec.Status)
	}
	if rec.Outcome != "success" {
		t.Errorf("Outcome = %q, want success", rec.Outcome)
	}
	if rec.Detail != "delivered" {
		t.Errorf("Detail = %q, want delivered", rec.Detail)
	}
	if !rec.At.Equal(fixedNow()) {
		t.Errorf("At = %v, want %v (injected clock)", rec.At, fixedNow())
	}
}

func TestAuditFailureOutcome(t *testing.T) {
	sink := &fakeSink{}
	New(sink, fixedNow).Audit(context.Background(), engine.AuditEvent{
		OrgID: "org1", RunID: "run-2", Service: "orders",
		Status: coderca.RunStatusTimeout, Detail: "timeout",
	})
	if sink.got.Outcome != "failure" {
		t.Errorf("Outcome = %q, want failure for a timed-out run", sink.got.Outcome)
	}
}
