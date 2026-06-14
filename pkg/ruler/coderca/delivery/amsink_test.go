package delivery

import (
	"context"
	"errors"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
)

// fakeAlertPutter captures calls to PutAlerts for assertion.
type fakeAlertPutter struct {
	orgID  string
	alerts alertmanagertypes.PostableAlerts
	err    error
}

func (f *fakeAlertPutter) PutAlerts(_ context.Context, orgID string, alerts alertmanagertypes.PostableAlerts) error {
	f.orgID = orgID
	f.alerts = alerts
	return f.err
}

func TestAlertmanagerSinkSubmitsMetaAlert(t *testing.T) {
	putter := &fakeAlertPutter{}
	sink := NewAlertmanagerSink(putter)

	msg := HandoffMessage{
		OrgID:          "org-1",
		Service:        "pay",
		RunID:          "r1",
		BaselineCommit: "abc",
		Title:          "T",
		Body:           "B",
	}

	ref, err := sink.Submit(context.Background(), msg)
	if err != nil {
		t.Fatalf("Submit: unexpected error: %v", err)
	}
	if ref != "r1" {
		t.Errorf("ref = %q, want %q", ref, "r1")
	}
	if putter.orgID != "org-1" {
		t.Errorf("orgID = %q, want %q", putter.orgID, "org-1")
	}
	if len(putter.alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(putter.alerts))
	}

	a := putter.alerts[0]

	// Assert labels (LabelSet is map[string]string)
	wantLabels := map[string]string{
		"alertname":    "CodeRCASuggestion",
		"service.name": "pay",
		"severity":     "info",
		"coderca":      "true",
	}
	for k, v := range wantLabels {
		if got := a.Alert.Labels[k]; got != v {
			t.Errorf("label[%q] = %q, want %q", k, got, v)
		}
	}

	// Assert annotations (LabelSet is map[string]string)
	wantAnnotations := map[string]string{
		"summary":                 "T",
		"description":             "B",
		"coderca.run_id":          "r1",
		"coderca.baseline_commit": "abc",
	}
	for k, v := range wantAnnotations {
		if got := a.Annotations[k]; got != v {
			t.Errorf("annotation[%q] = %q, want %q", k, got, v)
		}
	}
}

func TestAlertmanagerSinkPropagatesPutterError(t *testing.T) {
	putter := &fakeAlertPutter{err: errors.New("am down")}
	sink := NewAlertmanagerSink(putter)

	msg := HandoffMessage{OrgID: "org-1", Service: "pay", RunID: "r1"}
	ref, err := sink.Submit(context.Background(), msg)
	if err == nil {
		t.Error("expected error from putter, got nil")
	}
	if ref != "" {
		t.Errorf("ref = %q, want empty on error", ref)
	}
}
