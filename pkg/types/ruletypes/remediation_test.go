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
