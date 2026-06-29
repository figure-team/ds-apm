package remediation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/remediationstore"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

type fakeStore struct {
	created   []ruletypes.RemediationExecution
	createErr error
}

func (f *fakeStore) Create(_ context.Context, e ruletypes.RemediationExecution) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, e)
	return nil
}

func (f *fakeStore) Get(_ context.Context, _, _ string) (ruletypes.RemediationExecution, error) {
	return ruletypes.RemediationExecution{}, nil
}

func (f *fakeStore) ListByIncident(_ context.Context, _, _ string) ([]ruletypes.RemediationExecution, error) {
	return nil, nil
}

func (f *fakeStore) ListByStatus(_ context.Context, _, _ string) ([]ruletypes.RemediationExecution, error) {
	return nil, nil
}

func (f *fakeStore) TransitionToExecuting(_ context.Context, _, _, _, _ string, _ int64) (bool, error) {
	return false, nil
}

func (f *fakeStore) Transition(_ context.Context, _, _, _ string, _ remediationstore.TransitionPatch) error {
	return nil
}

func (f *fakeStore) CountActiveByOrg(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (f *fakeStore) GetConfig(_ context.Context, _ string) (ruletypes.RemediationConfig, error) {
	return ruletypes.RemediationConfig{}, nil
}

func (f *fakeStore) UpsertConfig(_ context.Context, _ string, _ ruletypes.RemediationConfig) error {
	return nil
}

func fixedNow() time.Time { return time.Date(2026, 6, 24, 0, 0, 0, 0, time.UTC) }

func docWithApprovedRunbook() ruletypes.SOPDocument {
	return ruletypes.SOPDocument{
		SOPID: "SOP-1", Version: "v1",
		Runbooks: []ruletypes.Runbook{
			{ID: "rb-draft", Title: "draft one", Status: ruletypes.RunbookStatusDraft, ExecutableScript: "echo no"},
			{ID: "rb-ok", Title: "Restart payment", Status: ruletypes.RunbookStatusApproved, ExecutableScript: "#!/bin/bash\nkubectl rollout restart deploy/payment\n", Confidence: 0.8},
		},
	}
}

func TestPropose_CreatesExecutionAndAnnotations(t *testing.T) {
	fs := &fakeStore{}
	p := NewProposer(fs, "https://apm.example.com", fixedNow)
	cfg := ruletypes.RemediationConfig{ExecutionEnabled: true, ProposalTTLSeconds: 1800}.WithDefaults()

	labels := map[string]string{"ruleId": "rule-123"}
	ann, ok := p.Propose(context.Background(), "org-1", "inc-1", "fp-1", labels, docWithApprovedRunbook(), cfg)
	if !ok {
		t.Fatal("expected proposal")
	}
	if len(fs.created) != 1 {
		t.Fatalf("want 1 created, got %d", len(fs.created))
	}
	e := fs.created[0]
	if e.RunbookID != "rb-ok" || e.Status != ruletypes.RemediationStatusProposed {
		t.Fatalf("wrong execution: %+v", e)
	}
	if e.ScriptSnapshot != "#!/bin/bash\nkubectl rollout restart deploy/payment\n" {
		t.Fatalf("snapshot must copy approved script: %q", e.ScriptSnapshot)
	}
	if e.ExpiresAt != "2026-06-24T00:30:00Z" {
		t.Fatalf("TTL wrong: %q", e.ExpiresAt)
	}
	if ann[alertmanagertypes.IncidentAnnotationRemediationID] != e.ID {
		t.Fatalf("annotation id mismatch")
	}
	approveURL := ann[alertmanagertypes.IncidentAnnotationRemediationApproveURL]
	if approveURL == "" {
		t.Fatalf("approve url missing")
	}
	if !strings.HasPrefix(approveURL, "https://apm.example.com/remediation/approve/") {
		t.Fatalf("approve url must target the standalone approval page, got %q", approveURL)
	}
	if !strings.HasSuffix(approveURL, "/remediation/approve/"+e.ID) {
		t.Fatalf("approve url must end with the remediation id path, got %q", approveURL)
	}
}

func TestPropose_FlagOff_NoOp(t *testing.T) {
	fs := &fakeStore{}
	p := NewProposer(fs, "https://x", fixedNow)
	cfg := ruletypes.RemediationConfig{ExecutionEnabled: false}.WithDefaults()
	if _, ok := p.Propose(context.Background(), "org-1", "inc-1", "fp-1", nil, docWithApprovedRunbook(), cfg); ok {
		t.Fatal("flag off must not propose")
	}
	if len(fs.created) != 0 {
		t.Fatal("nothing should be created")
	}
}

func TestPropose_NoApprovedRunbook_NoOp(t *testing.T) {
	fs := &fakeStore{}
	p := NewProposer(fs, "https://x", fixedNow)
	cfg := ruletypes.RemediationConfig{ExecutionEnabled: true}.WithDefaults()
	doc := ruletypes.SOPDocument{SOPID: "SOP-1", Version: "v1",
		Runbooks: []ruletypes.Runbook{{ID: "d", Status: ruletypes.RunbookStatusDraft, ExecutableScript: "x"}}}
	if _, ok := p.Propose(context.Background(), "org-1", "inc-1", "fp-1", nil, doc, cfg); ok {
		t.Fatal("no approved runbook must not propose")
	}
}

func TestPropose_CreateError_FailOpen(t *testing.T) {
	fs := &fakeStore{createErr: context.DeadlineExceeded}
	p := NewProposer(fs, "https://x", fixedNow)
	cfg := ruletypes.RemediationConfig{ExecutionEnabled: true}.WithDefaults()
	if _, ok := p.Propose(context.Background(), "org-1", "inc-1", "fp-1", nil, docWithApprovedRunbook(), cfg); ok {
		t.Fatal("store error must yield no proposal (fail-open)")
	}
}
