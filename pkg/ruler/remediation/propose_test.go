package remediation

import (
	"context"
	"database/sql"
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

func (f *fakeStore) ListByOrg(_ context.Context, _ string, _ remediationstore.ListFilter) ([]ruletypes.RemediationExecution, error) {
	return nil, nil
}

func (f *fakeStore) ListActiveByFingerprint(_ context.Context, _, _ string) ([]ruletypes.RemediationExecution, error) {
	return nil, nil
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
	p := NewProposer(fs, nil, "https://apm.example.com", fixedNow)
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
	p := NewProposer(fs, nil, "https://x", fixedNow)
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
	p := NewProposer(fs, nil, "https://x", fixedNow)
	cfg := ruletypes.RemediationConfig{ExecutionEnabled: true}.WithDefaults()
	doc := ruletypes.SOPDocument{SOPID: "SOP-1", Version: "v1",
		Runbooks: []ruletypes.Runbook{{ID: "d", Status: ruletypes.RunbookStatusDraft, ExecutableScript: "x"}}}
	if _, ok := p.Propose(context.Background(), "org-1", "inc-1", "fp-1", nil, doc, cfg); ok {
		t.Fatal("no approved runbook must not propose")
	}
}

func TestPropose_CreateError_FailOpen(t *testing.T) {
	fs := &fakeStore{createErr: context.DeadlineExceeded}
	p := NewProposer(fs, nil, "https://x", fixedNow)
	cfg := ruletypes.RemediationConfig{ExecutionEnabled: true}.WithDefaults()
	if _, ok := p.Propose(context.Background(), "org-1", "inc-1", "fp-1", nil, docWithApprovedRunbook(), cfg); ok {
		t.Fatal("store error must yield no proposal (fail-open)")
	}
}

type fakeTargetStore struct {
	t     ruletypes.RemediationTarget
	found bool
}

func (f fakeTargetStore) Create(context.Context, string, ruletypes.RemediationTarget) error { return nil }
func (f fakeTargetStore) Update(context.Context, string, ruletypes.RemediationTarget) error { return nil }
func (f fakeTargetStore) Delete(context.Context, string, string) error                      { return nil }
func (f fakeTargetStore) Get(context.Context, string, string) (ruletypes.RemediationTarget, error) {
	return f.t, nil
}
func (f fakeTargetStore) List(context.Context, string) ([]ruletypes.RemediationTarget, error) {
	return []ruletypes.RemediationTarget{f.t}, nil
}
func (f fakeTargetStore) Resolve(context.Context, string, map[string]string) (ruletypes.RemediationTarget, error) {
	if !f.found {
		return ruletypes.RemediationTarget{}, sql.ErrNoRows
	}
	return f.t, nil
}

func TestPropose_FreezesTargetSnapshotWhenResolved(t *testing.T) {
	tgt := ruletypes.RemediationTarget{
		ID: "3f2504e0-4f89-41d3-9a0c-0305e82c3301", Host: "10.0.0.5", Port: 22,
		User: "deploy", HostKeyFingerprint: "SHA256:abc", Name: "prod-web-01",
	}
	fs := &fakeStore{}
	p := NewProposer(fs, fakeTargetStore{t: tgt, found: true}, "https://x", time.Now)
	cfg := ruletypes.RemediationConfig{ExecutionEnabled: true, ProposalTTLSeconds: 1800}
	_, ok := p.Propose(context.Background(), "org-1", "inc-1", "fp-1",
		map[string]string{"service.name": "payment"}, docWithApprovedRunbook(), cfg)
	if !ok {
		t.Fatal("expected proposal")
	}
	if len(fs.created) != 1 {
		t.Fatalf("want 1 created, got %d", len(fs.created))
	}
	got := fs.created[0]
	if got.TargetID != tgt.ID || got.TargetHost != "10.0.0.5" || got.TargetHostKeyFP != "SHA256:abc" {
		t.Fatalf("target snapshot not frozen: %+v", got)
	}
}

func TestPropose_LocalWhenNoTargetResolved(t *testing.T) {
	fs := &fakeStore{}
	p := NewProposer(fs, fakeTargetStore{found: false}, "https://x", time.Now)
	cfg := ruletypes.RemediationConfig{ExecutionEnabled: true, ProposalTTLSeconds: 1800}
	_, ok := p.Propose(context.Background(), "org-1", "inc-1", "fp-1",
		map[string]string{"service.name": "unknown"}, docWithApprovedRunbook(), cfg)
	if !ok || len(fs.created) != 1 || fs.created[0].TargetID != "" {
		t.Fatalf("expected local proposal (empty TargetID), created=%+v", fs.created)
	}
}
