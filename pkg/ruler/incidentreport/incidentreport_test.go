package incidentreport

import (
	"context"
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

type fakeStrategyReader struct {
	rec ruletypes.AIStrategyHistoryRecord
	ok  bool
	err error
}

func (f fakeStrategyReader) GetLatest(_ context.Context, _ string, _ ruletypes.AIStrategyHistoryLookupRequest) (ruletypes.AIStrategyHistoryRecord, bool, error) {
	return f.rec, f.ok, f.err
}

type fakeRCAReader struct {
	finding CodeRCAFinding
	ok      bool
	err     error
}

func (f fakeRCAReader) LatestFinding(_ context.Context, _, _ string) (CodeRCAFinding, bool, error) {
	return f.finding, f.ok, f.err
}

func sampleStrategyRecord() ruletypes.AIStrategyHistoryRecord {
	return ruletypes.AIStrategyHistoryRecord{
		IncidentID:  "INC-1",
		Status:      "ready",
		Confidence:  "medium",
		GeneratedAt: "2026-06-15T05:00:00Z",
		Strategy: ruletypes.AIStrategy{
			Headline:            "결제 API 5xx 급증",
			Hypotheses:          []ruletypes.AIHypothesis{{Rank: 1, Text: "PG timeout"}},
			FirstActions:        []ruletypes.AIFirstAction{{Text: "PG 로그 확인", RequiresHumanApproval: true}},
			CustomerUpdateDraft: "[안내] 결제 지연 확인 중",
		},
	}
}

func TestBuild_AggregatesCF2AndCF11(t *testing.T) {
	g := NewGenerator(
		fakeStrategyReader{rec: sampleStrategyRecord(), ok: true},
		fakeRCAReader{ok: true, finding: CodeRCAFinding{
			RunID:       "run-1",
			RootCause:   "charge.go:26 divide by zero",
			ProposedFix: "len==0 guard 추가",
			CreatedAt:   1781499600,
			FinishedAt:  1781499660,
		}},
	)
	rep, err := g.Build(context.Background(), Params{
		OrgID: "org", IncidentID: "INC-1", Service: "payment-api", Severity: "critical", Now: "2026-06-15T06:00:00Z",
	})
	require.NoError(t, err)
	require.Equal(t, "결제 API 5xx 급증", rep.Title)
	require.Equal(t, "charge.go:26 divide by zero", rep.RootCause)
	require.Equal(t, "len==0 guard 추가", rep.ProposedFix)
	require.Equal(t, []string{"PG timeout"}, rep.Hypotheses)
	require.Equal(t, []string{"PG 로그 확인"}, rep.ProposedActions)
	require.Equal(t, "[안내] 결제 지연 확인 중", rep.CustomerNotice)
	require.Len(t, rep.Sources, 2, "both CF-2 and CF-11 contributed")
	require.GreaterOrEqual(t, len(rep.Timeline), 3, "CF-2 generated + CF-11 start/finish")
	// timeline sorted ascending
	require.True(t, rep.Timeline[0].At <= rep.Timeline[1].At)

	md := rep.RenderMarkdown()
	for _, section := range []string{"장애 개요", "발생 경과", "원인 분석", "조치 내역", "재발 방지", "고객 영향", "divide by zero"} {
		require.Contains(t, md, section)
	}
}

func TestBuild_MissingCF11_DegradesGracefully(t *testing.T) {
	g := NewGenerator(fakeStrategyReader{rec: sampleStrategyRecord(), ok: true}, fakeRCAReader{ok: false})
	rep, err := g.Build(context.Background(), Params{OrgID: "org", IncidentID: "INC-1", Service: "payment-api"})
	require.NoError(t, err)
	require.Empty(t, rep.RootCause, "no CF-11 finding → no root cause")
	require.Equal(t, "결제 API 5xx 급증", rep.Title, "CF-2 still present")
	require.Contains(t, rep.RenderMarkdown(), "확인 중", "root cause section degrades to 확인 중")
}

func TestBuild_MissingCF2_RecordsGap(t *testing.T) {
	g := NewGenerator(fakeStrategyReader{ok: false}, nil)
	rep, err := g.Build(context.Background(), Params{OrgID: "org", IncidentID: "INC-1"})
	require.NoError(t, err)
	require.Empty(t, rep.Title)
	joined := strings.Join(rep.Gaps, " | ")
	require.Contains(t, joined, "CF-2 AI 전략 이력")
	require.Contains(t, joined, "조치 실행 이력 미연동")
}

func TestRender_CustomTemplateOverridesLayout(t *testing.T) {
	rep := IncidentReport{Title: "T", Service: "svc", Hypotheses: []string{"h1", "h2"}}
	out, err := rep.Render("제목={{.Title}} | 서비스={{.Service}} | 가설수={{len .Hypotheses}}")
	require.NoError(t, err)
	require.Equal(t, "제목=T | 서비스=svc | 가설수=2", out)
}

func TestRender_MalformedTemplateErrors(t *testing.T) {
	_, err := IncidentReport{}.Render("{{.Nope")
	require.Error(t, err)
}

func TestRender_BlankUsesDefault(t *testing.T) {
	rep := IncidentReport{Title: "결제 장애"}
	out, err := rep.Render("")
	require.NoError(t, err)
	require.Contains(t, out, "# 장애보고서")
	require.Contains(t, out, "결제 장애")
}

func TestBuild_NilCodeRCAReader_OK(t *testing.T) {
	g := NewGenerator(fakeStrategyReader{rec: sampleStrategyRecord(), ok: true}, nil)
	rep, err := g.Build(context.Background(), Params{OrgID: "org", IncidentID: "INC-1", Service: "payment-api"})
	require.NoError(t, err)
	require.Empty(t, rep.RootCause)
	require.NotEmpty(t, rep.Title)
}
