package ruletypes

import (
	"strings"
	"testing"
)

func validRemediationExecution() RemediationExecution {
	return RemediationExecution{
		ID:               "11111111-1111-1111-1111-111111111111",
		OrgID:            "org-1",
		IncidentID:       "inc-1",
		AlertFingerprint: "fp-1",
		SOPID:            "SOP-1",
		SOPVersion:       "2026-06-01.1",
		RunbookID:        "22222222-2222-2222-2222-222222222222",
		ScriptSnapshot:   "#!/bin/bash\necho hi\n",
		Status:           RemediationStatusProposed,
		ProposedAt:       "2026-06-24T00:00:00Z",
		ExpiresAt:        "2026-06-24T00:30:00Z",
	}
}

func TestValidateRemediationExecution_OK(t *testing.T) {
	if err := ValidateRemediationExecution(validRemediationExecution()); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestValidateRemediationExecution_RejectsBadID(t *testing.T) {
	e := validRemediationExecution()
	e.ID = "not-a-uuid"
	if err := ValidateRemediationExecution(e); err == nil || !strings.Contains(err.Error(), "id") {
		t.Fatalf("want id error, got %v", err)
	}
}

func TestValidateRemediationExecution_RejectsBadStatus(t *testing.T) {
	e := validRemediationExecution()
	e.Status = "bogus"
	if err := ValidateRemediationExecution(e); err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatalf("want status error, got %v", err)
	}
}

func TestValidateRemediationExecution_RejectsNULInScript(t *testing.T) {
	e := validRemediationExecution()
	e.ScriptSnapshot = "echo hi\x00; rm -rf /"
	if err := ValidateRemediationExecution(e); err == nil || !strings.Contains(err.Error(), "scriptSnapshot") {
		t.Fatalf("want scriptSnapshot error, got %v", err)
	}
}

func TestRemediationStatusTransition(t *testing.T) {
	// v1 live path: proposed→executing is the single atomic step (no approved
	// intermediate). proposed→approved and approved→executing are retired.
	ok := [][2]string{
		{RemediationStatusProposed, RemediationStatusExecuting}, // v1 direct approve path
		{RemediationStatusProposed, RemediationStatusRejected},
		{RemediationStatusProposed, RemediationStatusExpired},
		{RemediationStatusExecuting, RemediationStatusSucceeded},
		{RemediationStatusExecuting, RemediationStatusFailed},
		{RemediationStatusSucceeded, RemediationStatusVerified},
		{RemediationStatusSucceeded, RemediationStatusUnresolved},
	}
	for _, p := range ok {
		if err := ValidateRemediationStatusTransition(p[0], p[1]); err != nil {
			t.Errorf("%s→%s: want nil, got %v", p[0], p[1], err)
		}
	}
	bad := [][2]string{
		{RemediationStatusProposed, RemediationStatusApproved},  // retired two-step path
		{RemediationStatusApproved, RemediationStatusExecuting}, // retired two-step path
		{RemediationStatusApproved, RemediationStatusApproved},  // no-op
		{RemediationStatusRejected, RemediationStatusApproved},  // terminal
		{RemediationStatusVerified, RemediationStatusExecuting}, // terminal
		{"bogus", RemediationStatusApproved},
	}
	for _, p := range bad {
		if err := ValidateRemediationStatusTransition(p[0], p[1]); err == nil {
			t.Errorf("%s→%s: want error, got nil", p[0], p[1])
		}
	}
}

func TestValidateRemediationExecution_AllowsKnownSource(t *testing.T) {
	e := validRemediationExecution()
	e.Source = RemediationSourceLLMGenerated
	e.RunbookID = "" // llm-generated has no backing runbook (design §6.1)
	e.SelectionRationale = "적합한 Runbook이 없어 직접 제안한 스크립트"
	if err := ValidateRemediationExecution(e); err != nil {
		t.Fatalf("llm-generated source should validate: %v", err)
	}
}

func TestValidateRemediationExecution_RejectsUnknownSource(t *testing.T) {
	e := validRemediationExecution()
	e.Source = "wat"
	if err := ValidateRemediationExecution(e); err == nil {
		t.Fatalf("expected unknown source to be rejected")
	}
}

func TestValidateRemediationExecution_EmptySourceAllowed(t *testing.T) {
	e := validRemediationExecution()
	e.Source = ""
	if err := ValidateRemediationExecution(e); err != nil {
		t.Fatalf("empty source must stay valid for legacy rows: %v", err)
	}
}

// TestValidateRemediationExecution_LLMSource_EmptyRunbookID verifies that
// llm-generated executions with no backing runbook pass validation (design §6.1).
func TestValidateRemediationExecution_LLMSource_EmptyRunbookID(t *testing.T) {
	e := validRemediationExecution()
	e.Source = RemediationSourceLLMGenerated
	e.RunbookID = ""
	if err := ValidateRemediationExecution(e); err != nil {
		t.Fatalf("llm-generated + empty RunbookID must validate, got: %v", err)
	}
}

// TestValidateRemediationExecution_LLMSource_NonEmptyRunbookID verifies that
// llm-generated executions MUST NOT carry a RunbookID (would be inconsistent).
func TestValidateRemediationExecution_LLMSource_NonEmptyRunbookID(t *testing.T) {
	e := validRemediationExecution()
	e.Source = RemediationSourceLLMGenerated
	e.RunbookID = "22222222-2222-2222-2222-222222222222"
	if err := ValidateRemediationExecution(e); err == nil {
		t.Fatalf("llm-generated + non-empty RunbookID must error")
	}
}

// TestValidateRemediationExecution_RunbookSource_EmptyRunbookID verifies that
// the existing behavior is preserved: runbook-source (or empty source) still
// requires a non-empty RunbookID.
func TestValidateRemediationExecution_RunbookSource_EmptyRunbookID(t *testing.T) {
	e := validRemediationExecution()
	e.Source = RemediationSourceRunbook
	e.RunbookID = ""
	if err := ValidateRemediationExecution(e); err == nil {
		t.Fatalf("runbook source + empty RunbookID must error")
	}
}

func TestValidateRemediationExecution_EmptySource_EmptyRunbookID(t *testing.T) {
	e := validRemediationExecution()
	e.Source = ""
	e.RunbookID = ""
	if err := ValidateRemediationExecution(e); err == nil {
		t.Fatalf("empty source + empty RunbookID must error (legacy row behavior)")
	}
}
func TestValidateRemediationExecution_RemoteRequiresFrozenParams(t *testing.T) {
	e := validRemediationExecution() // 기존 헬퍼 (없으면 인접 테스트의 생성 코드 재사용)
	e.TargetID = "3f2504e0-4f89-41d3-9a0c-0305e82c3301"
	e.TargetHost = ""       // 프리즈 무결성 위반
	e.TargetHostKeyFP = ""
	if err := ValidateRemediationExecution(e); err == nil {
		t.Fatal("remote execution must require frozen TargetHost + TargetHostKeyFP")
	}
}

func TestValidateRemediationExecution_LocalIgnoresTargetFields(t *testing.T) {
	e := validRemediationExecution()
	e.TargetID = "" // 로컬 = 스냅샷 필드 불필요
	if err := ValidateRemediationExecution(e); err != nil {
		t.Fatalf("local execution must not require target fields: %v", err)
	}
}

func TestRemediationIsTerminal(t *testing.T) {
	for _, s := range []string{RemediationStatusVerified, RemediationStatusUnresolved, RemediationStatusFailed, RemediationStatusRejected, RemediationStatusExpired} {
		if !(RemediationExecution{Status: s}).IsTerminal() {
			t.Errorf("%s should be terminal", s)
		}
	}
	for _, s := range []string{RemediationStatusProposed, RemediationStatusApproved, RemediationStatusExecuting, RemediationStatusSucceeded} {
		if (RemediationExecution{Status: s}).IsTerminal() {
			t.Errorf("%s should NOT be terminal", s)
		}
	}
}
