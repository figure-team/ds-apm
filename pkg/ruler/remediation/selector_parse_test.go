package remediation

import (
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

func idset(xs ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		m[x] = struct{}{}
	}
	return m
}

func TestParseSelectionResponse_StripsCodeFenceAndParses(t *testing.T) {
	raw := "```json\n{\"outcome\":\"selected\",\"chosenRunbookId\":\"rb-1\",\"confidence\":\"high\",\"rationale\":\"맞음\"}\n```"
	d, err := ParseSelectionResponse(raw, idset("rb-1"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Outcome != ruletypes.RunbookSelectionOutcomeSelected || d.ChosenRunbookID != "rb-1" {
		t.Fatalf("got %+v", d)
	}
}

func TestParseSelectionResponse_DanglingDemoted(t *testing.T) {
	raw := `{"outcome":"selected","chosenRunbookId":"ghost","confidence":"high","rationale":"x"}`
	d, err := ParseSelectionResponse(raw, idset("rb-1"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Outcome != ruletypes.RunbookSelectionOutcomeNone {
		t.Fatalf("expected none, got %q", d.Outcome)
	}
}

func TestParseSelectionResponse_Garbage(t *testing.T) {
	if _, err := ParseSelectionResponse("not json at all", idset("rb-1")); err == nil {
		t.Fatalf("expected error on non-json")
	}
}
