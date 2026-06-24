// Copyright (c) 2026 SigNoz, Inc.
// Copyright 2019 Prometheus Team
// SPDX-License-Identifier: Apache-2.0

package msteamsv2

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	commoncfg "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/promslog"
	"github.com/stretchr/testify/require"

	test "github.com/SigNoz/signoz/pkg/alertmanager/alertmanagernotify/alertmanagernotifytest"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/prometheus/alertmanager/config"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/types"
)

// This is a test URL that has been modified to not be valid.
var testWebhookURL, _ = url.Parse("https://example.westeurope.logic.azure.com:443/workflows/xxx/triggers/manual/paths/invoke?api-version=2016-06-01&sp=%2Ftriggers%2Fmanual%2Frun&sv=1.0&sig=xxx")

func TestMSTeamsV2Retry(t *testing.T) {
	notifier, err := New(
		&config.MSTeamsV2Config{
			WebhookURL: &config.SecretURL{URL: testWebhookURL},
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		`{{ template "msteamsv2.default.titleLink" . }}`,
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	for statusCode, expected := range test.RetryTests(test.DefaultRetryCodes()) {
		actual, _ := notifier.retrier.Check(statusCode, nil)
		require.Equal(t, expected, actual, "retry - error on status %d", statusCode)
	}
}

func TestNotifier_Notify_WithReason(t *testing.T) {
	tests := []struct {
		name            string
		statusCode      int
		responseContent string
		expectedReason  notify.Reason
		noError         bool
	}{
		{
			name:            "with a 2xx status code and response 1",
			statusCode:      http.StatusOK,
			responseContent: "1",
			noError:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier, err := New(
				&config.MSTeamsV2Config{
					WebhookURL: &config.SecretURL{URL: testWebhookURL},
					HTTPConfig: &commoncfg.HTTPClientConfig{},
				},
				test.CreateTmpl(t),
				`{{ template "msteamsv2.default.titleLink" . }}`,
				promslog.NewNopLogger(),
			)
			require.NoError(t, err)

			notifier.postJSONFunc = func(ctx context.Context, client *http.Client, url string, body io.Reader) (*http.Response, error) {
				resp := httptest.NewRecorder()
				_, err := resp.WriteString(tt.responseContent)
				require.NoError(t, err)
				resp.WriteHeader(tt.statusCode)
				return resp.Result(), nil
			}
			ctx := context.Background()
			ctx = notify.WithGroupKey(ctx, "1")

			alert1 := &types.Alert{
				Alert: model.Alert{
					StartsAt: time.Now(),
					EndsAt:   time.Now().Add(time.Hour),
				},
			}
			_, err = notifier.Notify(ctx, alert1)
			if tt.noError {
				require.NoError(t, err)
			} else {
				var reasonError *notify.ErrorWithReason
				require.ErrorAs(t, err, &reasonError)
				require.Equal(t, tt.expectedReason, reasonError.Reason)
			}
		})
	}
}

func TestMSTeamsV2Templating(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dec := json.NewDecoder(r.Body)
		out := make(map[string]any)
		err := dec.Decode(&out)
		if err != nil {
			panic(err)
		}
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)

	for _, tc := range []struct {
		title     string
		cfg       *config.MSTeamsV2Config
		titleLink string

		retry  bool
		errMsg string
	}{
		{
			title: "full-blown message",
			cfg: &config.MSTeamsV2Config{
				Title: `{{ template "msteams.default.title" . }}`,
				Text:  `{{ template "msteams.default.text" . }}`,
			},
			titleLink: `{{ template "msteamsv2.default.titleLink" . }}`,
			retry:     false,
		},
		{
			title: "title with templating errors",
			cfg: &config.MSTeamsV2Config{
				Title: "{{ ",
			},
			titleLink: `{{ template "msteamsv2.default.titleLink" . }}`,
			errMsg:    "template: :1: unclosed action",
		},
		{
			title: "message with title link templating errors",
			cfg: &config.MSTeamsV2Config{
				Title: `{{ template "msteams.default.title" . }}`,
				Text:  `{{ template "msteams.default.text" . }}`,
			},
			titleLink: `{{ `,
			errMsg:    "template: :1: unclosed action",
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			tc.cfg.WebhookURL = &config.SecretURL{URL: u}
			tc.cfg.HTTPConfig = &commoncfg.HTTPClientConfig{}
			pd, err := New(tc.cfg, test.CreateTmpl(t), tc.titleLink, promslog.NewNopLogger())
			require.NoError(t, err)

			ctx := context.Background()
			ctx = notify.WithGroupKey(ctx, "1")

			ok, err := pd.Notify(ctx, []*types.Alert{
				{
					Alert: model.Alert{
						Labels: model.LabelSet{
							"lbl1": "val1",
						},
						StartsAt: time.Now(),
						EndsAt:   time.Now().Add(time.Hour),
					},
				},
			}...)
			if tc.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			}
			require.Equal(t, tc.retry, ok)
		})
	}
}

// Non-SOP alerts render a minimal Korean incident block (severity, service,
// time), not the full label/annotation dump. SOP/AI fields and secrets present
// on the alert must neither be rendered nor leak into the card.
func TestMSTeamsV2NonSOPRendersMinimalKorean(t *testing.T) {
	var payload teamsMessage
	notifier, err := New(
		&config.MSTeamsV2Config{
			WebhookURL: &config.SecretURL{URL: testWebhookURL},
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		`{{ template "msteamsv2.default.titleLink" . }}`,
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)
	notifier.postJSONFunc = func(ctx context.Context, client *http.Client, url string, body io.Reader) (*http.Response, error) {
		require.NoError(t, json.NewDecoder(body).Decode(&payload))
		resp := httptest.NewRecorder()
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.WriteString("1")
		return resp.Result(), nil
	}

	ctx := notify.WithGroupKey(context.Background(), "test-group-key")

	_, err = notifier.Notify(ctx, alertWithDSAPMIncidentFields())

	require.NoError(t, err)
	facts := teamsFactValues(payload)
	require.Equal(t, "CRITICAL", facts["심각도"])
	require.Equal(t, "checkout-api", facts["서비스"])
	require.Contains(t, facts, "발생시간")
	// The verbose incident / label / annotation fields are no longer rendered.
	require.NotContains(t, facts, "SOP ID")
	require.NotContains(t, facts, "AI status")
	require.NotContains(t, facts, "AI headline")
	require.NotContains(t, facts, "SOP URL")

	encodedPayload, err := json.Marshal(payload)
	require.NoError(t, err)
	require.NotContains(t, string(encodedPayload), "token=hidden")
	require.NotContains(t, string(encodedPayload), "bearer abcdefghijklmnopqrstuvwxyz")
}

func teamsFactValues(message teamsMessage) map[string]string {
	values := map[string]string{}
	for _, attachment := range message.Attachments {
		for _, body := range attachment.Content.Body {
			for _, fact := range body.Facts {
				values[fact.Title] = fact.Value
			}
		}
	}

	return values
}

func alertWithDSAPMIncidentFields() *types.Alert {
	return &types.Alert{
		Alert: model.Alert{
			Labels: model.LabelSet{
				"alertname": "CheckoutLatencyHigh",
				model.LabelName(alertmanagertypes.IncidentLabelProjectID):   "customer-a",
				model.LabelName(alertmanagertypes.IncidentLabelEnvironment): "prod",
				model.LabelName(alertmanagertypes.IncidentLabelServiceName): "checkout-api",
				model.LabelName(alertmanagertypes.IncidentLabelOwnerTeam):   "sm-payments",
				model.LabelName(alertmanagertypes.IncidentLabelSeverity):    "critical",
				model.LabelName(alertmanagertypes.IncidentLabelSopID):       "SOP-PAY-001",
			},
			Annotations: model.LabelSet{
				model.LabelName(alertmanagertypes.IncidentAnnotationSopURL):           "https://runbooks.example.com/payment-latency?token=hidden&view=public",
				model.LabelName(alertmanagertypes.IncidentAnnotationSopTitle):         "Payment API 5xx response",
				model.LabelName(alertmanagertypes.IncidentAnnotationAIStrategyID):     "AIS-20260513-0005",
				model.LabelName(alertmanagertypes.IncidentAnnotationAIStrategyStatus): "quota_exhausted",
				model.LabelName(alertmanagertypes.IncidentAnnotationAIHeadline):       "bearer abcdefghijklmnopqrstuvwxyz",
				model.LabelName(alertmanagertypes.IncidentAnnotationAILimitations):    "AI strategy quota is exhausted for this period.",
			},
			StartsAt: time.Now(),
			EndsAt:   time.Now().Add(time.Hour),
		},
	}
}

func TestMSTeamsV2RedactedURL(t *testing.T) {
	ctx, u, fn := test.GetContextWithCancelingURL()
	defer fn()

	secret := "secret"
	notifier, err := New(
		&config.MSTeamsV2Config{
			WebhookURL: &config.SecretURL{URL: u},
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		`{{ template "msteamsv2.default.titleLink" . }}`,
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	test.AssertNotifyLeaksNoSecret(ctx, t, notifier, secret)
}

// TestMSTeamsUsesAIContentWhenBound verifies that when a single alert carries
// SOP-bound AI annotations (notification_body present), the card:
//  1. uses the sop_title as the first TextBlock Text (title override)
//  2. includes a TextBlock containing the notification_body
//  3. includes a customer-notice section (CollapsibleNoticeLabel + notice text)
//  4. does NOT include a Labels or Annotations FactSet
func TestMSTeamsUsesAIContentWhenBound(t *testing.T) {
	var capturedPayload teamsMessage

	notifier, err := New(
		&config.MSTeamsV2Config{
			Title:      `{{ template "msteams.default.title" . }}`,
			WebhookURL: &config.SecretURL{URL: testWebhookURL},
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		`{{ template "msteamsv2.default.titleLink" . }}`,
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	notifier.postJSONFunc = func(ctx context.Context, client *http.Client, url string, body io.Reader) (*http.Response, error) {
		require.NoError(t, json.NewDecoder(body).Decode(&capturedPayload))
		resp := httptest.NewRecorder()
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.WriteString("1")
		return resp.Result(), nil
	}

	ctx := notify.WithGroupKey(context.Background(), "ai-bound-test")

	alert := &types.Alert{
		Alert: model.Alert{
			Labels: model.LabelSet{
				"alertname": "ShippingLatencyHigh",
				model.LabelName(alertmanagertypes.IncidentLabelSopID): "SOP-SHIP-001",
			},
			Annotations: model.LabelSet{
				model.LabelName(alertmanagertypes.IncidentAnnotationSopTitle):        "Shipping 5xx 대응",
				model.LabelName(alertmanagertypes.IncidentAnnotationNotificationBody): "## 현황\n결제 서비스 지연 감지.",
				model.LabelName(alertmanagertypes.IncidentAnnotationCustomerUpdate):   "[안내] 점검 중입니다.",
			},
			StartsAt: time.Now(),
			EndsAt:   time.Now().Add(time.Hour),
		},
	}

	_, err = notifier.Notify(ctx, alert)
	require.NoError(t, err)

	bodies := capturedPayload.Attachments[0].Content.Body

	// 1. First TextBlock (title) must equal the sop_title
	require.NotEmpty(t, bodies, "card body must not be empty")
	require.Equal(t, "TextBlock", bodies[0].Type)
	require.Equal(t, "Shipping 5xx 대응", bodies[0].Text, "title TextBlock must use sop_title")

	// 2. Some TextBlock must contain the notification_body text
	bodyTexts := make([]string, 0, len(bodies))
	for _, b := range bodies {
		bodyTexts = append(bodyTexts, b.Text)
	}
	require.Contains(t, bodyTexts, "## 현황\n결제 서비스 지연 감지.", "body must contain notification_body TextBlock")

	// 3. Customer notice label and text must appear as separate TextBlocks
	require.Contains(t, bodyTexts, alertmanagertypes.CollapsibleNoticeLabel, "body must contain CollapsibleNoticeLabel TextBlock")
	require.Contains(t, bodyTexts, "[안내] 점검 중입니다.", "body must contain customer notice TextBlock")

	// 4. No FactSet bodies — AI path must NOT emit Labels/Annotations FactSets
	for _, b := range bodies {
		require.NotEqual(t, "FactSet", b.Type, "AI-bound path must not emit any FactSet blocks")
	}
}

// TestMSTeamsUsesFactsWhenUnbound verifies that when no notification_body annotation
// is present the existing per-alert Labels/Annotations FactSet path is preserved intact.
func TestMSTeamsUsesFactsWhenUnbound(t *testing.T) {
	var capturedPayload teamsMessage

	notifier, err := New(
		&config.MSTeamsV2Config{
			WebhookURL: &config.SecretURL{URL: testWebhookURL},
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		`{{ template "msteamsv2.default.titleLink" . }}`,
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	notifier.postJSONFunc = func(ctx context.Context, client *http.Client, url string, body io.Reader) (*http.Response, error) {
		require.NoError(t, json.NewDecoder(body).Decode(&capturedPayload))
		resp := httptest.NewRecorder()
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.WriteString("1")
		return resp.Result(), nil
	}

	ctx := notify.WithGroupKey(context.Background(), "unbound-test")

	alert := &types.Alert{
		Alert: model.Alert{
			Labels: model.LabelSet{
				"alertname": "MemoryHigh",
				"env":       "prod",
			},
			Annotations: model.LabelSet{
				"summary": "Memory usage above 90%",
			},
			StartsAt: time.Now(),
			EndsAt:   time.Now().Add(time.Hour),
		},
	}

	_, err = notifier.Notify(ctx, alert)
	require.NoError(t, err)

	bodies := capturedPayload.Attachments[0].Content.Body

	// Must contain at least one FactSet (Labels or Annotations)
	hasFactSet := false
	for _, b := range bodies {
		if b.Type == "FactSet" {
			hasFactSet = true
			break
		}
	}
	require.True(t, hasFactSet, "unbound path must emit Labels/Annotations FactSet blocks")

	// Must NOT contain CollapsibleNoticeLabel (AI path must not activate)
	bodyTexts := make([]string, 0, len(bodies))
	for _, b := range bodies {
		bodyTexts = append(bodyTexts, b.Text)
	}
	require.NotContains(t, bodyTexts, alertmanagertypes.CollapsibleNoticeLabel, "unbound path must not emit customer notice label")
}

func TestMSTeamsV2ReadingURLFromFile(t *testing.T) {
	ctx, u, fn := test.GetContextWithCancelingURL()
	defer fn()

	f, err := os.CreateTemp("", "webhook_url")
	require.NoError(t, err, "creating temp file failed")
	_, err = f.WriteString(u.String() + "\n")
	require.NoError(t, err, "writing to temp file failed")

	notifier, err := New(
		&config.MSTeamsV2Config{
			WebhookURLFile: f.Name(),
			HTTPConfig:     &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		`{{ template "msteamsv2.default.titleLink" . }}`,
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	test.AssertNotifyLeaksNoSecret(ctx, t, notifier, u.String())
}
