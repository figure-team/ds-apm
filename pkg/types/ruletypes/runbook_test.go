package ruletypes

import (
	"strings"
	"testing"
)

func validRunbook() Runbook {
	return Runbook{
		ID:               "01928374-5566-77ab-89cd-eeff00112233",
		Title:            "Restart payment worker",
		Description:      "## When\nLatency > 2s\n",
		ExecutableScript: "#!/bin/bash\nset -e\nkubectl rollout restart deploy/payment\n",
		Status:           RunbookStatusDraft,
		Confidence:       0.7,
		AIDraftedBy:      "claude-sonnet-4-6",
		CreatedAt:        "2026-05-22T00:00:00Z",
		UpdatedAt:        "2026-05-22T00:00:00Z",
		UpdatedBy:        "ai",
	}
}

func TestValidateRunbook_AcceptsValid(t *testing.T) {
	if err := ValidateRunbook(validRunbook()); err != nil {
		t.Fatalf("expected valid; got %v", err)
	}
}

func TestValidateRunbook_RejectsEmptyTitle(t *testing.T) {
	r := validRunbook()
	r.Title = "   "
	err := ValidateRunbook(r)
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("expected title error; got %v", err)
	}
}

func TestValidateRunbook_RejectsOverLongTitle(t *testing.T) {
	r := validRunbook()
	r.Title = strings.Repeat("x", RunbookMaxTitleLen+1)
	err := ValidateRunbook(r)
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("expected title length error; got %v", err)
	}
}

func TestValidateRunbook_RejectsOverLongDescription(t *testing.T) {
	r := validRunbook()
	r.Description = strings.Repeat("x", RunbookMaxDescriptionLen+1)
	err := ValidateRunbook(r)
	if err == nil || !strings.Contains(err.Error(), "description") {
		t.Fatalf("expected description length error; got %v", err)
	}
}

func TestValidateRunbook_RejectsOverLongScript(t *testing.T) {
	r := validRunbook()
	r.ExecutableScript = strings.Repeat("x", RunbookMaxScriptLen+1)
	err := ValidateRunbook(r)
	if err == nil || !strings.Contains(err.Error(), "executableScript") {
		t.Fatalf("expected script length error; got %v", err)
	}
}

func TestValidateRunbook_RejectsNULByteInScript(t *testing.T) {
	r := validRunbook()
	r.ExecutableScript = "echo hi\x00; rm -rf /"
	err := ValidateRunbook(r)
	if err == nil || !strings.Contains(err.Error(), "executableScript") {
		t.Fatalf("expected NUL byte error; got %v", err)
	}
}

func TestValidateRunbook_RejectsUnknownStatus(t *testing.T) {
	r := validRunbook()
	r.Status = "weird-status"
	err := ValidateRunbook(r)
	if err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatalf("expected status error; got %v", err)
	}
}

func TestValidateRunbook_RejectsConfidenceOutOfRange(t *testing.T) {
	r := validRunbook()
	r.Confidence = 1.5
	err := ValidateRunbook(r)
	if err == nil || !strings.Contains(err.Error(), "confidence") {
		t.Fatalf("expected confidence error; got %v", err)
	}

	r.Confidence = -0.1
	err = ValidateRunbook(r)
	if err == nil || !strings.Contains(err.Error(), "confidence") {
		t.Fatalf("expected confidence error (negative); got %v", err)
	}
}

func TestValidateRunbook_RejectsTooManySourceExamples(t *testing.T) {
	r := validRunbook()
	r.SourceErrorExamples = []string{"a", "b", "c", "d"}
	err := ValidateRunbook(r)
	if err == nil || !strings.Contains(err.Error(), "sourceErrorExamples") {
		t.Fatalf("expected sourceErrorExamples length error; got %v", err)
	}
}

func TestValidateRunbook_RejectsOverLongSourceExample(t *testing.T) {
	r := validRunbook()
	r.SourceErrorExamples = []string{strings.Repeat("x", RunbookMaxSourceExampleLen+1)}
	err := ValidateRunbook(r)
	if err == nil || !strings.Contains(err.Error(), "sourceErrorExamples[0]") {
		t.Fatalf("expected per-entry length error; got %v", err)
	}
}

func TestValidateRunbook_RejectsBadID(t *testing.T) {
	r := validRunbook()
	r.ID = "not-a-uuid"
	err := ValidateRunbook(r)
	if err == nil || !strings.Contains(err.Error(), "id") {
		t.Fatalf("expected id format error; got %v", err)
	}
}

func TestRunbookStatusTransitionAllowed(t *testing.T) {
	allowed := []struct{ from, to string }{
		{RunbookStatusDraft, RunbookStatusApproved},
		{RunbookStatusDraft, RunbookStatusDeprecated},
		{RunbookStatusApproved, RunbookStatusDeprecated},
		{RunbookStatusApproved, RunbookStatusDraft},
		{RunbookStatusDeprecated, RunbookStatusDraft},
	}
	for _, c := range allowed {
		if err := ValidateRunbookStatusTransition(c.from, c.to); err != nil {
			t.Fatalf("expected %s→%s allowed; got %v", c.from, c.to, err)
		}
	}
}

func TestRunbookStatusTransitionRejected(t *testing.T) {
	rejected := []struct{ from, to string }{
		{RunbookStatusDeprecated, RunbookStatusApproved}, // must transit through draft
		{RunbookStatusDraft, RunbookStatusDraft},          // same-status
		{RunbookStatusApproved, RunbookStatusApproved},
		{RunbookStatusDeprecated, RunbookStatusDeprecated},
	}
	for _, c := range rejected {
		if err := ValidateRunbookStatusTransition(c.from, c.to); err == nil {
			t.Fatalf("expected %s→%s rejected", c.from, c.to)
		}
	}
}

func TestRunbookStatusTransition_RejectsUnknownStatus(t *testing.T) {
	// from invalid
	if err := ValidateRunbookStatusTransition("bogus", RunbookStatusApproved); err == nil ||
		!strings.Contains(err.Error(), "from") {
		t.Fatalf("expected invalid-from error; got %v", err)
	}
	// to invalid
	if err := ValidateRunbookStatusTransition(RunbookStatusDraft, "bogus"); err == nil ||
		!strings.Contains(err.Error(), "to") {
		t.Fatalf("expected invalid-to error; got %v", err)
	}
}
