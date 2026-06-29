package remediation

import (
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

func TestRenderSelectionPrompt_ListsRunbooksInOrderAndJSONOnly(t *testing.T) {
	in := SelectionInput{
		IncidentID:       "inc-1",
		AlertFingerprint: "fp-1",
		Labels:           map[string]string{"service_name": "payment", "severity": "critical"},
		SOP: ruletypes.SOPDocument{
			SOPID: "SOP-PAY", Version: "v3", Title: "결제 장애", BodyMarkdown: "## 1단계\n로그 확인",
		},
		Runbooks: []ruletypes.Runbook{
			{ID: "rb-1", Title: "재시작", Description: "롤링 재시작", ExecutableScript: "kubectl rollout restart deploy/payment", Confidence: 0.7},
			{ID: "rb-2", Title: "캐시 비우기", Description: "redis flush", ExecutableScript: "redis-cli flushall", Confidence: 0.4},
		},
	}
	system, user := RenderSelectionPrompt(in)

	if !strings.Contains(system, "JSON") {
		t.Fatalf("system prompt must demand single JSON output")
	}
	// Runbook 등록 순 보존: rb-1 이 rb-2 보다 먼저.
	i1, i2 := strings.Index(user, "rb-1"), strings.Index(user, "rb-2")
	if i1 < 0 || i2 < 0 || i1 > i2 {
		t.Fatalf("runbooks must appear in registration order; got i1=%d i2=%d", i1, i2)
	}
	// 라벨은 정렬되어야 함: service_name 이 severity 보다 먼저(알파벳).
	ls, lv := strings.Index(user, "service_name"), strings.Index(user, "severity")
	if ls < 0 || lv < 0 || ls > lv {
		t.Fatalf("labels must be alphabetically sorted")
	}
	if !strings.Contains(user, "SOP-PAY") {
		t.Fatalf("user prompt must include SOP id")
	}
}

func TestRenderSelectionPrompt_TruncatesLongScript(t *testing.T) {
	long := strings.Repeat("x", selectionScriptPreviewMax+500)
	in := SelectionInput{
		IncidentID: "inc-1",
		SOP:        ruletypes.SOPDocument{SOPID: "S", Version: "v1", Title: "t", BodyMarkdown: "b"},
		Runbooks:   []ruletypes.Runbook{{ID: "rb-1", Title: "t", ExecutableScript: long}},
	}
	_, user := RenderSelectionPrompt(in)
	if strings.Count(user, "x") > selectionScriptPreviewMax+10 {
		t.Fatalf("script should be truncated for the prompt")
	}
}
