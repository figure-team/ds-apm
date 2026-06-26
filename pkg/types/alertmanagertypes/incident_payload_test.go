package alertmanagertypes

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestBuildSafeIncidentInfoRedactsSecretLikeValues(t *testing.T) {
	info := BuildSafeIncidentInfo(
		template.KV{
			IncidentLabelProjectID:   "customer-a",
			IncidentLabelEnvironment: "prod",
			IncidentLabelServiceName: "checkout-api",
			IncidentLabelSopID:       "SOP-PAY-001",
		},
		template.KV{
			IncidentAnnotationSopURL:           "https://runbooks.example.com/sop?token=hidden&view=public",
			IncidentAnnotationAIHeadline:       "bearer abcdefghijklmnopqrstuvwxyz",
			IncidentAnnotationAIFirstActions:   "Inspect PG timeout logs.",
			IncidentAnnotationAIStrategyStatus: "ready",
		},
	)

	require.Equal(t, "https://runbooks.example.com/sop?view=public", info.SopURL)
	require.Equal(t, RedactedIncidentValue, info.AIHeadline)
	require.Equal(t, "Inspect PG timeout logs.", info.AIFirstActions)

	fields := IncidentInfoFields(info)
	require.Contains(t, fields, IncidentField{
		Key:   "ai_headline",
		Title: "AI headline",
		Value: RedactedIncidentValue,
	})
	require.NotContains(t, IncidentInfoDetails(info)["sop_url"], "token=hidden")
}

func TestSanitizeIncidentValueRedactsEmail(t *testing.T) {
	got := SanitizeIncidentValue("contact 김철수 chulsoo@example.co.kr immediately")
	if strings.Contains(got, "chulsoo@example.co.kr") {
		t.Fatalf("email not redacted: %q", got)
	}
}

func TestSanitizeIncidentValueRedactsKoreanMobile(t *testing.T) {
	got := SanitizeIncidentValue("notify 010-1234-5678 by 5pm")
	if strings.Contains(got, "010-1234-5678") {
		t.Fatalf("KR mobile not redacted: %q", got)
	}
}

func TestSanitizeIncidentValueRedactsLongUnmarkedSecret(t *testing.T) {
	raw := "abcdefghijklmnopqrstuvwxyz0123456789" // 36 chars, no marker
	got := SanitizeIncidentValue("token leaked: " + raw)
	if strings.Contains(got, raw) {
		t.Fatalf("long secret not redacted: %q", got)
	}
}

func TestSanitizeIncidentValuePreservesShortInnocuousText(t *testing.T) {
	got := SanitizeIncidentValue("short status: degraded for 3 minutes")
	if got != "short status: degraded for 3 minutes" {
		t.Fatalf("innocuous text mutated: %q", got)
	}
}

// TestSanitize_AllPatterns is the DoD acceptance for SCOPE row 1: every
// enumerated sensitive pattern (email, phone, opaque token, JWT, URL with a
// sensitive query key) must be stripped of its raw secret, while identifying,
// non-sensitive labels survive verbatim.
func TestSanitize_AllPatterns(t *testing.T) {
	redacted := []struct {
		name   string
		input  string
		leaked string // raw substring that MUST NOT survive sanitization
	}{
		{"email", "reach chulsoo@example.co.kr now", "chulsoo@example.co.kr"},
		{"korean_mobile", "call 010-1234-5678 asap", "010-1234-5678"},
		{"opaque_token", "token=ghp_aBcDeF0123456789abcdef0123456789abcd", "ghp_aBcDeF0123456789abcdef0123456789abcd"},
		{"jwt", "auth eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"url_sensitive_key", "https://rb.example.com/sop?access_token=s3cr3tvalue&view=public", "s3cr3tvalue"},
	}
	for _, tc := range redacted {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeIncidentValue(tc.input)
			require.NotContains(t, got, tc.leaked, "raw sensitive token leaked through sanitization: %q", got)
		})
	}

	preserved := []string{"checkout-api", "prod", "critical", "SOP-PAY-001", "customer-a"}
	for _, keep := range preserved {
		t.Run("preserve/"+keep, func(t *testing.T) {
			require.Equal(t, keep, SanitizeIncidentValue(keep), "identifying label must be preserved verbatim")
		})
	}
}

// TestRedactionMetric is the DoD acceptance for SCOPE row 2: each value that
// gets redacted advances the redaction counter by exactly one, and innocuous
// values leave it untouched ("마스킹 N건 발생 → redaction metric += N").
func TestRedactionMetric(t *testing.T) {
	before := testutil.ToFloat64(incidentRedactionsTotal)

	redactedValues := []string{
		"reach chulsoo@example.co.kr now",                  // email
		"call 010-1234-5678 asap",                          // korean mobile
		"token=ghp_aBcDeF0123456789abcdef0123456789abcdef", // secret marker
	}
	for _, v := range redactedValues {
		SanitizeIncidentValue(v)
	}

	require.Equal(t, float64(len(redactedValues)), testutil.ToFloat64(incidentRedactionsTotal)-before,
		"expected one redaction increment per redacted value")

	// Innocuous values must not be counted as redactions.
	mid := testutil.ToFloat64(incidentRedactionsTotal)
	SanitizeIncidentValue("degraded for 3 minutes")
	SanitizeIncidentValue("checkout-api")
	require.Equal(t, mid, testutil.ToFloat64(incidentRedactionsTotal),
		"innocuous values must not move the redaction counter")
}

// TestSanitizeTemplateData drives the masking of the full outbound webhook
// payload (template.Data) — not just the derived incident block. It is the Go
// underpinning for SCOPE row 3: the external payload must carry zero raw PII,
// while routing/identifying metadata survives and the input is left untouched.
func TestSanitizeTemplateData(t *testing.T) {
	in := &template.Data{
		Receiver:          "webhook",
		Status:            "firing",
		GroupLabels:       template.KV{"alertname": "PaymentFailures", "service.name": "checkout-api"},
		CommonLabels:      template.KV{"severity": "critical", "owner_email": "oncall@example.com"},
		CommonAnnotations: template.KV{"impact_summary": "call 010-1234-5678", "next_action": "investigate"},
		Alerts: template.Alerts{
			{
				Status:       "firing",
				Labels:       template.KV{"customer": "reach chulsoo@example.co.kr"},
				Annotations:  template.KV{"customer_update": "token=ghp_aBcDeF0123456789abcdef0123456789abcd"},
				GeneratorURL: "https://signoz.example.com/alert?access_token=s3cr3tvalue",
			},
		},
	}

	out := SanitizeTemplateData(in)

	// No raw PII survives anywhere in the serialized outbound payload.
	blob, err := json.Marshal(out)
	require.NoError(t, err)
	for _, leaked := range []string{
		"oncall@example.com",
		"010-1234-5678",
		"chulsoo@example.co.kr",
		"ghp_aBcDeF0123456789abcdef0123456789abcd",
		"s3cr3tvalue",
	} {
		require.NotContains(t, string(blob), leaked, "raw PII leaked in outbound payload")
	}

	// Identifying / routing metadata is preserved verbatim.
	require.Equal(t, "checkout-api", out.GroupLabels["service.name"])
	require.Equal(t, "PaymentFailures", out.GroupLabels["alertname"])
	require.Equal(t, "critical", out.CommonLabels["severity"])
	require.Equal(t, "investigate", out.CommonAnnotations["next_action"])

	// The input must not be mutated: webhook.go still templates the URL from the
	// raw data, so sanitization has to be a defensive copy.
	require.Equal(t, "oncall@example.com", in.CommonLabels["owner_email"])
	require.Equal(t, "call 010-1234-5678", in.CommonAnnotations["impact_summary"])
	require.Equal(t, "reach chulsoo@example.co.kr", in.Alerts[0].Labels["customer"])
}

// TestSanitizeIncidentInfoCopiesNotificationBody is the regression guard for I1:
// SanitizeIncidentInfo must copy NotificationBody through to the sanitized
// struct, just as it does for the adjacent CustomerUpdate field.
func TestSanitizeIncidentInfoCopiesNotificationBody(t *testing.T) {
	in := IncidentInfo{
		CustomerUpdate:   "Payment latency is under investigation.",
		NotificationBody: "결제 서비스 지연 발생. SOP 기준으로 대응 중입니다.",
	}
	out := SanitizeIncidentInfo(in)
	require.Equal(t, in.CustomerUpdate, out.CustomerUpdate, "CustomerUpdate must survive sanitization")
	require.Equal(t, in.NotificationBody, out.NotificationBody, "NotificationBody must survive sanitization")

	// An absent NotificationBody must remain empty (not become RedactedIncidentValue).
	outEmpty := SanitizeIncidentInfo(IncidentInfo{CustomerUpdate: "some update"})
	require.Empty(t, outEmpty.NotificationBody, "absent NotificationBody must not be fabricated")
}

func TestIncidentInfoFieldsOmitEmptyValues(t *testing.T) {
	fields := IncidentInfoFields(IncidentInfo{
		SopID:            "SOP-PAY-001",
		AIStrategyStatus: "quota_exhausted",
	})

	require.Equal(t, []IncidentField{
		{Key: "sop_id", Title: "SOP ID", Value: "SOP-PAY-001", Short: true},
		{Key: "ai_strategy_status", Title: "AI status", Value: "quota_exhausted", Short: true},
	}, fields)
}

func TestIncidentInfoFieldsCompactKeepsOnlyRoutingAndSOPLink(t *testing.T) {
	// SOP-bound notifications carry the AI body as their text; the field block is
	// trimmed to routing context + a link to the SOP. Verbose AI/comms/debug
	// fields are dropped even when populated.
	info := IncidentInfo{
		ProjectID:        "customer-a",
		Environment:      "prod",
		ServiceName:      "frontend",
		OwnerTeam:        "frontend",
		Severity:         "critical",
		ImpactSummary:    "loading failures up",
		NextAction:       "check CDN",
		VendorRequest:    "ask vendor",
		CustomerUpdate:   "under investigation",
		SopID:            "SOP-FE-001",
		SopTitle:         "Frontend 5xx 대응",
		SopVersion:       "2026-06-17.1",
		SopSource:        "src-managed-markdown-default",
		SopURL:           "https://kb.example/sop/SOP-FE-001",
		SopBindingID:     "explicit_label",
		AIStrategyID:     "llm-abc123",
		AIStrategyStatus: "low_confidence",
		AIConfidence:     "low",
		AIHeadline:       "frontend 5xx 알람",
		AILimitations:    "no evidence",
	}

	require.Equal(t, []IncidentField{
		{Key: "project_id", Title: "Project", Value: "customer-a", Short: true},
		{Key: "environment", Title: "Environment", Value: "prod", Short: true},
		{Key: "service_name", Title: "Service", Value: "frontend", Short: true},
		{Key: "owner_team", Title: "Owner team", Value: "frontend", Short: true},
		{Key: "severity", Title: "Severity", Value: "critical", Short: true},
		{Key: "sop_title", Title: "SOP title", Value: "Frontend 5xx 대응"},
		{Key: "sop_url", Title: "SOP URL", Value: "https://kb.example/sop/SOP-FE-001"},
	}, IncidentInfoFieldsCompact(info))
}

func TestIncidentInfoFieldsCompactIncludesRemediationWhenPresent(t *testing.T) {
	// Human-gated auto-remediation surfaces a summary + an approval deep link in
	// the compact (SOP-bound human) field set used by Slack/Email.
	info := IncidentInfo{
		ServiceName:           "payment",
		RemediationSummary:    "Restart payment (승인 시 웹 UI에서 실행)",
		RemediationApproveURL: "https://apm.example.com/alerts/overview?remediation=rem-1&ruleId=r1",
	}
	fields := IncidentInfoFieldsCompact(info)

	byKey := map[string]string{}
	for _, f := range fields {
		byKey[f.Key] = f.Value
	}
	require.Equal(t, "Restart payment (승인 시 웹 UI에서 실행)", byKey["remediation_summary"])
	require.Equal(t,
		"https://apm.example.com/alerts/overview?remediation=rem-1&ruleId=r1",
		byKey["remediation_approve_url"])
}

func TestSanitizeIncidentApproveURL(t *testing.T) {
	// A real approval link carries a UUID remediation id; the generic secret
	// redactor would mangle that 36-char token, so the URL-aware sanitizer must
	// keep it intact.
	uuidURL := "https://apm.example.com/alerts/overview?remediation=550e8400-e29b-41d4-a716-446655440000&ruleId=rule-123"
	require.Equal(t, uuidURL, SanitizeIncidentApproveURL(uuidURL))
	require.NotContains(t, SanitizeIncidentApproveURL(uuidURL), "[redacted")

	// Generic sanitizer (for comparison) mangles the same URL — the reason a
	// dedicated approve-URL sanitizer exists.
	require.NotEqual(t, uuidURL, SanitizeIncidentValue(uuidURL))
	require.Contains(t, SanitizeIncidentValue(uuidURL), "[redacted")

	// Sensitive query keys are still stripped (defense in depth).
	require.Equal(t,
		"https://apm.example.com/alerts/overview?remediation=rem-1",
		SanitizeIncidentApproveURL("https://apm.example.com/alerts/overview?remediation=rem-1&token=secret"))

	// A relative URL cannot render as a clickable link — drop it.
	require.Equal(t, "", SanitizeIncidentApproveURL("/alerts/overview?remediation=rem-1"))
	require.Equal(t, "", SanitizeIncidentApproveURL(""))
}

func TestBuildIncidentInfoMapsRemediationAnnotations(t *testing.T) {
	info := BuildIncidentInfo(nil, template.KV{
		IncidentAnnotationRemediationScriptSummary: "Restart payment (승인 시 웹 UI에서 실행)",
		IncidentAnnotationRemediationApproveURL:    "https://apm.example.com/alerts/overview?remediation=rem-1",
	})
	require.Equal(t, "Restart payment (승인 시 웹 UI에서 실행)", info.RemediationSummary)
	require.Equal(t, "https://apm.example.com/alerts/overview?remediation=rem-1", info.RemediationApproveURL)
	require.False(t, info.IsZero(), "remediation-only info must not be zero")
}

func TestIncidentInfoFieldsCompactOmitsEmptyValues(t *testing.T) {
	fields := IncidentInfoFieldsCompact(IncidentInfo{
		ProjectID: "customer-a",
		Severity:  "critical",
	})

	require.Equal(t, []IncidentField{
		{Key: "project_id", Title: "Project", Value: "customer-a", Short: true},
		{Key: "severity", Title: "Severity", Value: "critical", Short: true},
	}, fields)
}
