package ruletypes

import (
	"strings"
	"testing"
)

func ids(xs ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		m[x] = struct{}{}
	}
	return m
}

func TestValidateRunbookSelectionDecision_Selected_OK(t *testing.T) {
	d := RunbookSelectionDecision{
		ContractVersion: RunbookSelectionContractVersion,
		Outcome:         RunbookSelectionOutcomeSelected,
		ChosenRunbookID: "rb-1",
		Confidence:      AIConfidenceHigh,
		Rationale:       "결제 타임아웃 패턴이 이 Runbook과 일치",
	}
	got, err := ValidateRunbookSelectionDecision(d, ids("rb-1", "rb-2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Outcome != RunbookSelectionOutcomeSelected || got.ChosenRunbookID != "rb-1" {
		t.Fatalf("expected selected rb-1, got %+v", got)
	}
	if !got.IsActionable() {
		t.Fatalf("selected with valid id should be actionable")
	}
}

func TestValidateRunbookSelectionDecision_DanglingID_DemotedToNone(t *testing.T) {
	d := RunbookSelectionDecision{
		ContractVersion: RunbookSelectionContractVersion,
		Outcome:         RunbookSelectionOutcomeSelected,
		ChosenRunbookID: "rb-ghost",
		Confidence:      AIConfidenceHigh,
		Rationale:       "근거",
	}
	got, err := ValidateRunbookSelectionDecision(d, ids("rb-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Outcome != RunbookSelectionOutcomeNone {
		t.Fatalf("dangling id must demote to none, got %q", got.Outcome)
	}
	if got.IsActionable() {
		t.Fatalf("none must not be actionable")
	}
	if got.Rationale != "" {
		t.Fatalf("demotion must clear Rationale, got %q", got.Rationale)
	}
}

func TestValidateRunbookSelectionDecision_Fallback_RejectsOversizedScript(t *testing.T) {
	d := RunbookSelectionDecision{
		ContractVersion: RunbookSelectionContractVersion,
		Outcome:         RunbookSelectionOutcomeFallback,
		Confidence:      AIConfidenceMedium,
		Rationale:       "적합한 Runbook 없음",
		FallbackScript:  strings.Repeat("x", RunbookMaxScriptLen+1),
		FallbackSummary: "재시작",
	}
	if _, err := ValidateRunbookSelectionDecision(d, ids("rb-1")); err == nil {
		t.Fatalf("expected oversized script to be rejected")
	}
}

func TestValidateRunbookSelectionDecision_Fallback_RejectsBadScript(t *testing.T) {
	d := RunbookSelectionDecision{
		ContractVersion: RunbookSelectionContractVersion,
		Outcome:         RunbookSelectionOutcomeFallback,
		Confidence:      AIConfidenceMedium,
		Rationale:       "적합한 Runbook 없음",
		FallbackScript:  "echo hi\x00rm",
		FallbackSummary: "재시작",
	}
	if _, err := ValidateRunbookSelectionDecision(d, ids("rb-1")); err == nil {
		t.Fatalf("expected NUL-byte script to be rejected")
	}
}

func TestValidateRunbookSelectionDecision_Fallback_OK(t *testing.T) {
	d := RunbookSelectionDecision{
		ContractVersion: RunbookSelectionContractVersion,
		Outcome:         RunbookSelectionOutcomeFallback,
		Confidence:      AIConfidenceMedium,
		Rationale:       "적합한 Runbook 없음, 안전한 재시작 제안",
		FallbackScript:  "#!/bin/bash\nset -e\nkubectl rollout restart deploy/payment\n",
		FallbackSummary: "payment 디플로이 롤링 재시작",
	}
	got, err := ValidateRunbookSelectionDecision(d, ids("rb-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.IsActionable() {
		t.Fatalf("valid fallback should be actionable")
	}
}

func TestValidateRunbookSelectionDecision_BadOutcome(t *testing.T) {
	d := RunbookSelectionDecision{
		ContractVersion: RunbookSelectionContractVersion,
		Outcome:         "bogus",
		Confidence:      AIConfidenceLow,
	}
	if _, err := ValidateRunbookSelectionDecision(d, ids()); err == nil {
		t.Fatalf("expected invalid outcome to error")
	}
}
