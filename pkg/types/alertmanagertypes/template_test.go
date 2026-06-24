package alertmanagertypes

import (
	"net/url"
	"testing"
	"time"

	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromGlobs(t *testing.T) {
	template, err := FromGlobs([]string{})
	require.NoError(t, err)
	template.ExternalURL = &url.URL{Scheme: "http", Host: "localhost:8080", Path: ""}

	testCases := []struct {
		name     string
		alerts   []*types.Alert
		expected string
	}{
		{
			name: "SingleAlertWithValidRuleId",
			alerts: []*types.Alert{
				{
					Alert: model.Alert{
						Labels: model.LabelSet{
							"ruleId": "01961575-461c-7668-875f-05d374062bfc",
						},
					},
					UpdatedAt: time.Now(),
					Timeout:   false,
				},
			},
			expected: "http://localhost:8080/alerts/edit?ruleId=01961575-461c-7668-875f-05d374062bfc",
		},
		{
			name: "SingleAlertWithValidRuleUUIDv4",
			alerts: []*types.Alert{
				{
					Alert: model.Alert{
						Labels: model.LabelSet{
							"ruleId": "2d8edca5-4f24-4266-afd1-28cefadcfa88",
						},
					},
					UpdatedAt: time.Now(),
					Timeout:   false,
				},
			},
			expected: "http://localhost:8080/alerts/edit?ruleId=2d8edca5-4f24-4266-afd1-28cefadcfa88",
		},
		{
			name: "MultipleAlertsWithMismatchingRuleId",
			alerts: []*types.Alert{
				{
					Alert: model.Alert{
						Labels: model.LabelSet{
							"ruleId": "01961575-461c-7668-875f-05d374062bfc",
						},
					},
					UpdatedAt: time.Now(),
					Timeout:   false,
				},
				{
					Alert: model.Alert{
						Labels: model.LabelSet{
							"ruleId": "0196156c-990e-7ec5-b28f-8a3cfbb9c865",
						},
					},
					UpdatedAt: time.Now(),
					Timeout:   false,
				},
			},
			expected: "http://localhost:8080/alerts",
		},
		{
			name: "MultipleAlertsWithMatchingRuleId",
			alerts: []*types.Alert{
				{
					Alert: model.Alert{
						Labels: model.LabelSet{
							"ruleId": "01961575-461c-7668-875f-05d374062bfc",
						},
					},
					UpdatedAt: time.Now(),
					Timeout:   false,
				},
				{
					Alert: model.Alert{
						Labels: model.LabelSet{
							"ruleId": "01961575-461c-7668-875f-05d374062bfc",
						},
					},
					UpdatedAt: time.Now(),
					Timeout:   false,
				},
			},
			expected: "http://localhost:8080/alerts/edit?ruleId=01961575-461c-7668-875f-05d374062bfc",
		},
		{
			name: "MultipleAlertsWithNoRuleId",
			alerts: []*types.Alert{
				{
					Alert: model.Alert{
						Labels: model.LabelSet{
							"label1": "1",
						},
					},
					UpdatedAt: time.Now(),
					Timeout:   false,
				},
				{
					Alert: model.Alert{
						Labels: model.LabelSet{
							"label2": "2",
						},
					},
					UpdatedAt: time.Now(),
					Timeout:   false,
				},
			},
			expected: "http://localhost:8080/alerts",
		},
		{
			name: "TestAlertWithNoRuleId",
			alerts: []*types.Alert{
				{
					Alert: model.Alert{
						Labels: model.LabelSet{
							"testalert": "true",
						},
					},
					UpdatedAt: time.Now(),
					Timeout:   false,
				},
			},
			expected: "http://localhost:8080/alerts",
		},
		{
			name: "TestAlertWithRuleId",
			alerts: []*types.Alert{
				{
					Alert: model.Alert{
						Labels: model.LabelSet{
							"testalert": "true",
							"ruleId":    "01961575-461c-7668-875f-05d374062bfc",
						},
					},
					UpdatedAt: time.Now(),
					Timeout:   false,
				},
			},
			expected: "http://localhost:8080/alerts/edit?ruleId=01961575-461c-7668-875f-05d374062bfc&isTestAlert=true",
		},
		{
			name: "TestAlertWithRuleIdWithSpacesAndSymbol",
			alerts: []*types.Alert{
				{
					Alert: model.Alert{
						Labels: model.LabelSet{
							"testalert": "true",
							"ruleId":    "Prom + Alert & Rule",
						},
					},
					UpdatedAt: time.Now(),
					Timeout:   false,
				},
			},
			expected: "http://localhost:8080/alerts/edit?ruleId=Prom+%2B+Alert+%26+Rule&isTestAlert=true",
		},
		{
			name: "AlertWithBlankRuleId",
			alerts: []*types.Alert{
				{
					Alert: model.Alert{
						Labels: model.LabelSet{
							"ruleId": "",
						},
					},
					UpdatedAt: time.Now(),
					Timeout:   false,
				},
			},
			expected: "http://localhost:8080/alerts",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := template.Data("__receiver", model.LabelSet{}, tc.alerts...)

			url, err := template.ExecuteTextString(`{{ template "__alertmanagerURL" . }}`, data)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, url)

			url, err = template.ExecuteHTMLString(`{{ template "__alertmanagerURL" . }}`, data)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, url)
		})
	}
}

func TestToKST(t *testing.T) {
	fn := AdditionalFuncMap()["toKST"].(func(time.Time) string)

	// 06:05 UTC + 9h = 15:05 KST.
	require.Equal(t, "2026-06-24 15:05 KST", fn(time.Date(2026, 6, 24, 6, 5, 0, 0, time.UTC)))
	// Crossing midnight: 18:30 UTC + 9h = 03:30 KST next day.
	require.Equal(t, "2026-06-25 03:30 KST", fn(time.Date(2026, 6, 24, 18, 30, 0, 0, time.UTC)))
	// Zero time renders empty so an absent timestamp does not print a bogus date.
	require.Equal(t, "", fn(time.Time{}))
}

// slackDefaultText mirrors SlackInitialConfig.text in
// frontend/src/container/CreateAlertChannels/defaults.ts. Kept in sync so this
// test fails if the rendered shape of the practitioner-facing default drifts
// from the Go template engine's capabilities (toKST, index on "service.name").
const slackDefaultText = `{{ range .Alerts }}심각도: {{ if .Labels.severity }}{{ .Labels.severity | toUpper }}{{ else }}-{{ end }}
서비스: {{ if index .Labels "service.name" }}{{ index .Labels "service.name" }}{{ else }}-{{ end }}
발생시간: {{ .StartsAt | toKST }}

📋 오류 내용
{{ .Annotations.description }}{{ if .Annotations.next_action }}

✅ 조치 사항
{{ .Annotations.next_action }}{{ end }}
{{ end }}`

func TestSlackDefaultTemplateRendersKoreanIncident(t *testing.T) {
	tmpl, err := FromGlobs([]string{})
	require.NoError(t, err)
	tmpl.ExternalURL = &url.URL{Scheme: "http", Host: "localhost:8080"}

	startsAt := time.Date(2026, 6, 24, 6, 5, 0, 0, time.UTC)

	t.Run("WithNextAction", func(t *testing.T) {
		alerts := []*types.Alert{{
			Alert: model.Alert{
				Labels: model.LabelSet{
					"alertname":    "PaymentLatencyHigh",
					"severity":     "critical",
					"service.name": "payment-api",
				},
				Annotations: model.LabelSet{
					"description": "p99 지연이 2s 임계치를 초과했습니다.",
					"next_action": "런북 확인 후 결제 인스턴스 롤백",
				},
				StartsAt: startsAt,
			},
		}}
		data := tmpl.Data("__receiver", model.LabelSet{}, alerts...)

		out, err := tmpl.ExecuteTextString(slackDefaultText, data)
		require.NoError(t, err)

		require.Contains(t, out, "심각도: CRITICAL")
		require.Contains(t, out, "서비스: payment-api")
		require.Contains(t, out, "발생시간: 2026-06-24 15:05 KST")
		require.Contains(t, out, "p99 지연이 2s 임계치를 초과했습니다.")
		require.Contains(t, out, "✅ 조치 사항")
		require.Contains(t, out, "런북 확인 후 결제 인스턴스 롤백")
	})

	t.Run("WithoutNextActionOrService", func(t *testing.T) {
		alerts := []*types.Alert{{
			Alert: model.Alert{
				Labels: model.LabelSet{
					"alertname": "DiskUsageHigh",
				},
				Annotations: model.LabelSet{
					"description": "디스크 사용량이 임계치를 초과했습니다.",
				},
				StartsAt: startsAt,
			},
		}}
		data := tmpl.Data("__receiver", model.LabelSet{}, alerts...)

		out, err := tmpl.ExecuteTextString(slackDefaultText, data)
		require.NoError(t, err)

		require.Contains(t, out, "심각도: -")
		require.Contains(t, out, "서비스: -")
		require.Contains(t, out, "디스크 사용량이 임계치를 초과했습니다.")
		// next_action absent → the optional action block is omitted entirely.
		require.NotContains(t, out, "✅ 조치 사항")
	})
}
