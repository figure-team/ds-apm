package dispatchhook

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

type fakeExecLister struct {
	execs []ruletypes.RemediationExecution
	err   error
	calls int
}

func (f *fakeExecLister) ListByIncident(_ context.Context, _, _ string) ([]ruletypes.RemediationExecution, error) {
	f.calls++
	return f.execs, f.err
}

func TestFormatDurationKR(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "1분 미만"},
		{time.Minute, "1분"},
		{23 * time.Minute, "23분"},
		{time.Hour, "1시간"},
		{83 * time.Minute, "1시간 23분"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, formatDurationKR(tc.d), "d=%s", tc.d)
	}
}

func TestPickResolvedExecution(t *testing.T) {
	verified := ruletypes.RemediationExecution{ID: "e-verified", Status: ruletypes.RemediationStatusVerified, ExecutedAt: "2026-07-10T05:10:00Z"}
	failed := ruletypes.RemediationExecution{ID: "e-failed", Status: ruletypes.RemediationStatusFailed, ExecutedAt: "2026-07-10T05:20:00Z"}
	proposed := ruletypes.RemediationExecution{ID: "e-proposed", Status: ruletypes.RemediationStatusProposed}
	laterFailed := ruletypes.RemediationExecution{ID: "e-failed-2", Status: ruletypes.RemediationStatusFailed, ExecutedAt: "2026-07-10T05:30:00Z"}

	// Never-ran statuses are ignored entirely.
	require.Nil(t, pickResolvedExecution([]ruletypes.RemediationExecution{proposed}, time.Time{}))

	// Outcome rank beats recency: verified wins over a later failed run.
	got := pickResolvedExecution([]ruletypes.RemediationExecution{failed, verified, proposed}, time.Time{})
	require.NotNil(t, got)
	require.Equal(t, "e-verified", got.ID)

	// Same rank falls back to the most recent execution.
	got = pickResolvedExecution([]ruletypes.RemediationExecution{failed, laterFailed}, time.Time{})
	require.NotNil(t, got)
	require.Equal(t, "e-failed-2", got.ID)
}

// Regression: executions are keyed by alert fingerprint, which repeats across
// occurrences — a verified run approved for LAST WEEK's occurrence must not be
// reported as the current incident's 조치 (observed live 2026-07-10: an
// 8-day-old verified execution surfaced as "승인 → 성공" on a fresh incident
// nobody approved).
func TestPickResolvedExecution_StartsAtCutsStaleOccurrences(t *testing.T) {
	staleVerified := ruletypes.RemediationExecution{
		ID: "e-stale", Status: ruletypes.RemediationStatusVerified,
		ProposedAt: "2026-07-02T05:56:21Z", ExecutedAt: "2026-07-02T05:57:38Z",
	}
	freshFailed := ruletypes.RemediationExecution{
		ID: "e-fresh", Status: ruletypes.RemediationStatusFailed,
		ProposedAt: "2026-07-10T05:30:00Z", ExecutedAt: "2026-07-10T05:31:00Z",
	}
	noTimestamps := ruletypes.RemediationExecution{
		ID: "e-no-ts", Status: ruletypes.RemediationStatusVerified,
	}
	startsAt := time.Date(2026, 7, 10, 5, 29, 7, 0, time.UTC)

	// Only the stale record exists → nothing attributable to this incident.
	require.Nil(t, pickResolvedExecution([]ruletypes.RemediationExecution{staleVerified}, startsAt))

	// A lower-ranked but in-window run beats a higher-ranked stale one.
	got := pickResolvedExecution([]ruletypes.RemediationExecution{staleVerified, freshFailed}, startsAt)
	require.NotNil(t, got)
	require.Equal(t, "e-fresh", got.ID)

	// Unparseable timestamps are excluded once a cutoff is in force.
	require.Nil(t, pickResolvedExecution([]ruletypes.RemediationExecution{noTimestamps}, startsAt))
}

func TestBuildResolvedBody(t *testing.T) {
	startsAt := time.Date(2026, 7, 10, 5, 2, 0, 0, time.UTC) // 14:02 KST
	endsAt := startsAt.Add(23 * time.Minute)

	t.Run("no execution on record", func(t *testing.T) {
		body := buildResolvedBody("AdServiceGetAdsFailure", "critical", startsAt, endsAt, nil, "")
		require.Equal(t,
			"AdServiceGetAdsFailure(critical)가 2026-07-10 14:02 KST 발생 후 23분 만에 해소되었습니다.\n\n"+
				"*조치:* 자동대응 실행 이력이 없습니다. 수동 조치로 해소되었습니다(상세 미기록).",
			body)
	})

	t.Run("verified execution with approver", func(t *testing.T) {
		exec := &ruletypes.RemediationExecution{
			Status:     ruletypes.RemediationStatusVerified,
			ExecutedAt: "2026-07-10T05:10:00Z", // 14:10 KST
			ApprovedBy: "jinhyeok",
		}
		body := buildResolvedBody("AdServiceGetAdsFailure", "critical", startsAt, endsAt, exec, "광고 캐시 재기동")
		require.Equal(t,
			"AdServiceGetAdsFailure(critical)가 2026-07-10 14:02 KST 발생 후 23분 만에 해소되었습니다.\n\n"+
				"*조치:* 자동대응 \"광고 캐시 재기동\" 실행(14:10, jinhyeok 승인) → 성공, 지표 정상화 검증 완료.",
			body)
	})

	t.Run("failed execution surfaces exit code", func(t *testing.T) {
		code := 7
		exec := &ruletypes.RemediationExecution{
			Status:   ruletypes.RemediationStatusFailed,
			ExitCode: &code,
		}
		body := buildResolvedBody("AdServiceGetAdsFailure", "", startsAt, endsAt, exec, "")
		require.Contains(t, body, "자동대응 \"런북\" 실행 → 실행 실패(exit 7).")
		require.NotContains(t, body, "(critical)")
	})

	t.Run("zero times degrade to plain headline", func(t *testing.T) {
		body := buildResolvedBody("X", "warning", time.Time{}, time.Time{}, nil, "")
		require.Contains(t, body, "X(warning)가 해소되었습니다.")
	})
}

func TestBuildResolvedNotice(t *testing.T) {
	require.Equal(t, "[안내] 'Ad 서비스 응답 지연/오류 대응' 관련 증상이 정상화되었습니다.",
		buildResolvedNotice("Ad 서비스 응답 지연/오류 대응"))
	require.Equal(t, "[안내] 서비스 이상 증상이 정상화되었습니다.", buildResolvedNotice("  "))
}

func TestApplyResolved_BoundWithExecution(t *testing.T) {
	const orgID = "customer-a"
	gen := &stubGen{}
	hook, sops, _, seed := seedHookFixture(t, orgID, gen)

	// Attach a known runbook to the seed SOP so the resolved notice can name it.
	doc := seed.SOPDocument
	doc.Runbooks = append(doc.Runbooks, ruletypes.Runbook{
		ID:               "rb-resolved-test",
		Title:            "광고 캐시 재기동",
		ExecutableScript: "systemctl restart ad-cache",
		Status:           ruletypes.RunbookStatusApproved,
	})
	require.NoError(t, sops.Upsert(context.Background(), orgID, doc))

	lister := &fakeExecLister{execs: []ruletypes.RemediationExecution{{
		ID:         "exec-1",
		RunbookID:  "rb-resolved-test",
		Status:     ruletypes.RemediationStatusVerified,
		ExecutedAt: "2026-07-10T05:10:00Z",
		ApprovedBy: "jinhyeok",
	}}}
	hook.SetExecutionLister(lister)

	startsAt := time.Date(2026, 7, 10, 5, 2, 0, 0, time.UTC)
	endsAt := startsAt.Add(23 * time.Minute)

	merged := hook.ApplyResolved(context.Background(), orgID,
		seed.Alert.IncidentID, seed.Alert.Fingerprint,
		seed.Alert.Labels, seed.Alert.Annotations, startsAt, endsAt)

	require.Equal(t, 1, lister.calls)
	require.Zero(t, gen.calls, "resolved path must never invoke the LLM generator")

	body := merged[alertmanagertypes.IncidentAnnotationNotificationBody]
	require.Contains(t, body, "해소되었습니다")
	require.Contains(t, body, "자동대응 \"광고 캐시 재기동\" 실행(14:10, jinhyeok 승인)")
	require.Contains(t, body, "성공, 지표 정상화 검증 완료")

	require.Contains(t, merged[alertmanagertypes.IncidentAnnotationCustomerUpdate], "정상화되었습니다")
	require.NotEmpty(t, merged[alertmanagertypes.IncidentAnnotationSopTitle])

	// The remediation approval CTA must not ride a resolved notification.
	require.Empty(t, merged[alertmanagertypes.IncidentAnnotationRemediationScriptSummary])
	require.Empty(t, merged[alertmanagertypes.IncidentAnnotationRemediationApproveURL])
}

func TestApplyResolved_NoListerReportsNoActionOnRecord(t *testing.T) {
	const orgID = "customer-a"
	gen := &stubGen{}
	hook, _, _, seed := seedHookFixture(t, orgID, gen)

	merged := hook.ApplyResolved(context.Background(), orgID,
		seed.Alert.IncidentID, seed.Alert.Fingerprint,
		seed.Alert.Labels, seed.Alert.Annotations,
		time.Now().Add(-10*time.Minute), time.Now())

	require.Zero(t, gen.calls)
	require.Contains(t, merged[alertmanagertypes.IncidentAnnotationNotificationBody],
		"자동대응 실행 이력이 없습니다")
}

func TestApplyResolved_UnboundLeavesAnnotationsUntouched(t *testing.T) {
	const orgID = "customer-a"
	gen := &stubGen{}
	hook, _, _, _ := seedHookFixture(t, orgID, gen)

	in := map[string]string{"summary": "no SOP matches these labels"}
	merged := hook.ApplyResolved(context.Background(), orgID,
		"inc-x", "fp-x",
		map[string]string{"alertname": "TotallyUnknownAlert"}, in,
		time.Now().Add(-time.Minute), time.Now())

	require.Zero(t, gen.calls)
	require.Equal(t, in, merged)
	require.Empty(t, merged[alertmanagertypes.IncidentAnnotationNotificationBody])
}
