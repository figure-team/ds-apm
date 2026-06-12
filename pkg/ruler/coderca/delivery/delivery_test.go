package delivery

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/engine"
)

type fakeSink struct {
	got    HandoffMessage
	ref    string
	err    error
	called bool
}

func (f *fakeSink) Submit(_ context.Context, msg HandoffMessage) (string, error) {
	f.called = true
	f.got = msg
	return f.ref, f.err
}

func sampleDelivery() engine.Delivery {
	return engine.Delivery{
		OrgID:          "org1",
		Service:        "payments",
		RunID:          "run-1",
		BaselineCommit: "a1b2c3",
		Result: coderca.RCAResult{
			BaselineCommit: "a1b2c3",
			RootCause:      "connection pool exhausted under load",
			ProposedFix:    "reuse a pooled *sql.DB and close rows",
			Confidence:     "medium",
			Limitations:    "static analysis only",
		},
	}
}

func TestFormatHandoffHITLFraming(t *testing.T) {
	msg := FormatHandoff(sampleDelivery())
	low := strings.ToLower(msg.Body)

	for _, want := range []string{
		"suggestion",  // it is a suggestion
		"not applied", // never auto-applied
		"review",      // human review required
		"a1b2c3",      // baseline commit echoed
		"connection pool exhausted under load", // root cause
		"reuse a pooled *sql.db and close rows", // proposed fix
		"medium",                // confidence
		"static analysis only",  // limitations
	} {
		if !strings.Contains(low, want) {
			t.Errorf("handoff body missing %q\n---\n%s", want, msg.Body)
		}
	}
	if !strings.Contains(msg.Title, "payments") {
		t.Errorf("title should name the service: %q", msg.Title)
	}
	if msg.BaselineCommit != "a1b2c3" {
		t.Errorf("BaselineCommit = %q, want a1b2c3", msg.BaselineCommit)
	}
}

func TestFormatHandoffGracefulWithSparseResult(t *testing.T) {
	d := engine.Delivery{
		OrgID: "org1", Service: "orders", RunID: "run-2", BaselineCommit: "deadbeef",
		Result: coderca.RCAResult{RootCause: "nil deref in Submit"},
	}
	msg := FormatHandoff(d)
	if msg.Body == "" {
		t.Fatal("body must not be empty for a sparse result")
	}
	if !strings.Contains(msg.Body, "nil deref in Submit") {
		t.Errorf("body missing root cause: %q", msg.Body)
	}
	// HITL framing is present even when fix/limitations are absent.
	if !strings.Contains(strings.ToLower(msg.Body), "not applied") {
		t.Errorf("HITL framing lost for sparse result: %q", msg.Body)
	}
	// Baseline falls back to the run's baseline when the result echoed none.
	if !strings.Contains(msg.Body, "deadbeef") {
		t.Errorf("body missing baseline fallback: %q", msg.Body)
	}
}

func TestDelivererSubmitsAndReturnsRef(t *testing.T) {
	sink := &fakeSink{ref: "handoff-7"}
	ref, err := New(sink).Deliver(context.Background(), sampleDelivery())
	if err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	if ref != "handoff-7" {
		t.Errorf("ref = %q, want handoff-7", ref)
	}
	if !sink.called {
		t.Fatal("sink.Submit not called")
	}
	if !strings.Contains(sink.got.Title, "payments") {
		t.Errorf("sink got wrong message: %+v", sink.got)
	}
	if sink.got.RunID != "run-1" || sink.got.BaselineCommit != "a1b2c3" {
		t.Errorf("sink missing identifiers: %+v", sink.got)
	}
}

func TestDelivererSinkErrorPropagates(t *testing.T) {
	sink := &fakeSink{err: errors.New("handoff down")}
	ref, err := New(sink).Deliver(context.Background(), sampleDelivery())
	if err == nil {
		t.Error("sink error must propagate")
	}
	if ref != "" {
		t.Errorf("ref = %q, want empty on error", ref)
	}
}
