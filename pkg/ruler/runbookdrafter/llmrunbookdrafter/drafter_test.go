package llmrunbookdrafter

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// fakeProvider implements the llmaigenerator.Provider interface.
// Records the system + user prompts it received.
type fakeProvider struct {
	response  string
	err       error
	gotSystem string
	gotUser   string
}

func (f *fakeProvider) Complete(_ context.Context, system, user string) (string, error) {
	f.gotSystem = system
	f.gotUser = user
	if f.err != nil {
		return "", f.err
	}
	return f.response, nil
}

func validDraftRequest() ruletypes.RunbookDraftRequest {
	return ruletypes.RunbookDraftRequest{
		SOP: ruletypes.SOPDocument{
			SOPID: "SOP-PAY-001",
			Title: "Payment latency",
		},
		ErrorExamples: []string{"timeout: dial tcp redis: i/o timeout"},
		Source:        "manual-paste",
	}
}

func TestDrafter_ParsesValidJSON(t *testing.T) {
	resp := `{
		"title":"Restart worker",
		"description":"## Steps\n1. Restart\n",
		"executableScript":"#!/bin/bash\nkubectl rollout restart deploy/payment\n",
		"confidence":0.85,
		"rationale":"connection pool stuck"
	}`
	p := &fakeProvider{response: resp}
	d := New(p, "claude-sonnet-4-6")
	rb, err := d.Draft(context.Background(), validDraftRequest())
	if err != nil {
		t.Fatalf("Draft: %v", err)
	}
	if rb.Title != "Restart worker" {
		t.Fatalf("Title: %q", rb.Title)
	}
	if !strings.HasPrefix(rb.ExecutableScript, "#!/bin/bash") {
		t.Fatalf("Script: %q", rb.ExecutableScript)
	}
	if rb.Confidence != 0.85 {
		t.Fatalf("Confidence: %v", rb.Confidence)
	}
	if rb.Status != ruletypes.RunbookStatusDraft {
		t.Fatalf("Status: %s (must default to draft)", rb.Status)
	}
	if rb.AIDraftedBy != "claude-sonnet-4-6" {
		t.Fatalf("AIDraftedBy: %s", rb.AIDraftedBy)
	}
	if len(rb.SourceErrorExamples) != 1 || rb.SourceErrorExamples[0] != "timeout: dial tcp redis: i/o timeout" {
		t.Fatalf("SourceErrorExamples lost: %v", rb.SourceErrorExamples)
	}
	if rb.UpdatedBy != "ai" {
		t.Fatalf("UpdatedBy: %s", rb.UpdatedBy)
	}
	if rb.ID == "" {
		t.Fatalf("ID must be assigned")
	}
}

func TestDrafter_RejectsMalformedJSON(t *testing.T) {
	p := &fakeProvider{response: "not json {"}
	d := New(p, "claude-sonnet-4-6")
	_, err := d.Draft(context.Background(), validDraftRequest())
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error; got %v", err)
	}
}

func TestDrafter_RejectsMissingTitle(t *testing.T) {
	resp := `{"description":"x","executableScript":"#!/bin/bash\necho hi"}`
	p := &fakeProvider{response: resp}
	d := New(p, "claude-sonnet-4-6")
	_, err := d.Draft(context.Background(), validDraftRequest())
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("expected title error; got %v", err)
	}
}

func TestDrafter_ClampsConfidence(t *testing.T) {
	cases := []struct {
		raw  string
		want float64
	}{
		{`"confidence":1.7,`, 1.0},
		{`"confidence":-0.5,`, 0.0},
	}
	for _, c := range cases {
		resp := `{"title":"t","description":"d","executableScript":"#!/bin/bash\nhi",` + c.raw + `"rationale":"r"}`
		p := &fakeProvider{response: resp}
		d := New(p, "model")
		rb, err := d.Draft(context.Background(), validDraftRequest())
		if err != nil {
			t.Fatalf("Draft: %v", err)
		}
		if rb.Confidence != c.want {
			t.Fatalf("raw=%s confidence=%v want=%v", c.raw, rb.Confidence, c.want)
		}
	}
}

func TestDrafter_DefaultConfidenceWhenMissing(t *testing.T) {
	resp := `{"title":"t","description":"d","executableScript":"#!/bin/bash\nhi"}`
	p := &fakeProvider{response: resp}
	d := New(p, "model")
	rb, err := d.Draft(context.Background(), validDraftRequest())
	if err != nil {
		t.Fatalf("Draft: %v", err)
	}
	if rb.Confidence != 0.5 {
		t.Fatalf("default confidence: %v (want 0.5)", rb.Confidence)
	}
}

func TestDrafter_PropagatesProviderError(t *testing.T) {
	p := &fakeProvider{err: errors.New("auth failed")}
	d := New(p, "model")
	_, err := d.Draft(context.Background(), validDraftRequest())
	if err == nil || !strings.Contains(err.Error(), "auth failed") {
		t.Fatalf("expected wrapped provider error; got %v", err)
	}
}

func TestRenderPrompt_IncludesAllErrorExamples(t *testing.T) {
	req := validDraftRequest()
	req.ErrorExamples = []string{"err A", "err B", "err C"}
	_, user := renderPrompt(req)
	for _, ex := range req.ErrorExamples {
		if !strings.Contains(user, ex) {
			t.Fatalf("user prompt missing %q: %s", ex, user)
		}
	}
}
