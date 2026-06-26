// Copyright (c) 2026 SigNoz, Inc.
// Copyright 2019 Prometheus Team
// SPDX-License-Identifier: Apache-2.0

package slack

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	commoncfg "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/promslog"
	"github.com/stretchr/testify/require"

	"github.com/prometheus/alertmanager/config"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/notify/test"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/types"
)

func TestSlackRetry(t *testing.T) {
	notifier, err := New(
		&config.SlackConfig{
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	for statusCode, expected := range test.RetryTests(test.DefaultRetryCodes()) {
		actual, _ := notifier.retrier.Check(statusCode, nil)
		require.Equal(t, expected, actual, "error on status %d", statusCode)
	}
}

func TestSlackRedactedURL(t *testing.T) {
	ctx, u, fn := test.GetContextWithCancelingURL()
	defer fn()

	notifier, err := New(
		&config.SlackConfig{
			APIURL:     &config.SecretURL{URL: u},
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	test.AssertNotifyLeaksNoSecret(ctx, t, notifier, u.String())
}

func TestGettingSlackURLFromFile(t *testing.T) {
	ctx, u, fn := test.GetContextWithCancelingURL()
	defer fn()

	f, err := os.CreateTemp(t.TempDir(), "slack_test")
	require.NoError(t, err, "creating temp file failed")
	_, err = f.WriteString(u.String())
	require.NoError(t, err, "writing to temp file failed")

	notifier, err := New(
		&config.SlackConfig{
			APIURLFile: f.Name(),
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	test.AssertNotifyLeaksNoSecret(ctx, t, notifier, u.String())
}

func TestTrimmingSlackURLFromFile(t *testing.T) {
	ctx, u, fn := test.GetContextWithCancelingURL()
	defer fn()

	f, err := os.CreateTemp(t.TempDir(), "slack_test_newline")
	require.NoError(t, err, "creating temp file failed")
	_, err = f.WriteString(u.String() + "\n\n")
	require.NoError(t, err, "writing to temp file failed")

	notifier, err := New(
		&config.SlackConfig{
			APIURLFile: f.Name(),
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	test.AssertNotifyLeaksNoSecret(ctx, t, notifier, u.String())
}

func TestNotifier_Notify_WithReason(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedReason notify.Reason
		expectedErr    string
		expectedRetry  bool
		noError        bool
	}{
		{
			name:           "with a 4xx status code",
			statusCode:     http.StatusUnauthorized,
			expectedReason: notify.ClientErrorReason,
			expectedRetry:  false,
			expectedErr:    "unexpected status code 401",
		},
		{
			name:           "with a 5xx status code",
			statusCode:     http.StatusInternalServerError,
			expectedReason: notify.ServerErrorReason,
			expectedRetry:  true,
			expectedErr:    "unexpected status code 500",
		},
		{
			name:           "with a 3xx status code",
			statusCode:     http.StatusTemporaryRedirect,
			expectedReason: notify.DefaultReason,
			expectedRetry:  false,
			expectedErr:    "unexpected status code 307",
		},
		{
			name:           "with a 1xx status code",
			statusCode:     http.StatusSwitchingProtocols,
			expectedReason: notify.DefaultReason,
			expectedRetry:  false,
			expectedErr:    "unexpected status code 101",
		},
		{
			name:           "2xx response with invalid JSON",
			statusCode:     http.StatusOK,
			responseBody:   `{"not valid json"}`,
			expectedReason: notify.ClientErrorReason,
			expectedRetry:  true,
			expectedErr:    "could not unmarshal",
		},
		{
			name:           "2xx response with a JSON error",
			statusCode:     http.StatusOK,
			responseBody:   `{"ok":false,"error":"error_message"}`,
			expectedReason: notify.ClientErrorReason,
			expectedRetry:  false,
			expectedErr:    "error response from Slack: error_message",
		},
		{
			name:           "2xx response with a plaintext error",
			statusCode:     http.StatusOK,
			responseBody:   "no_channel",
			expectedReason: notify.ClientErrorReason,
			expectedRetry:  false,
			expectedErr:    "error response from Slack: no_channel",
		},
		{
			name:         "successful JSON response",
			statusCode:   http.StatusOK,
			responseBody: `{"ok":true}`,
			noError:      true,
		},
		{
			name:         "successful plaintext response",
			statusCode:   http.StatusOK,
			responseBody: "ok",
			noError:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiurl, _ := url.Parse("https://slack.com/post.Message")
			notifier, err := New(
				&config.SlackConfig{
					NotifierConfig: config.NotifierConfig{},
					HTTPConfig:     &commoncfg.HTTPClientConfig{},
					APIURL:         &config.SecretURL{URL: apiurl},
					Channel:        "channelname",
				},
				test.CreateTmpl(t),
				promslog.NewNopLogger(),
			)
			require.NoError(t, err)

			notifier.postJSONFunc = func(ctx context.Context, client *http.Client, url string, body io.Reader) (*http.Response, error) {
				resp := httptest.NewRecorder()
				if strings.HasPrefix(tt.responseBody, "{") {
					resp.Header().Add("Content-Type", "application/json; charset=utf-8")
				}
				resp.WriteHeader(tt.statusCode)
				_, _ = resp.WriteString(tt.responseBody)
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
			retry, err := notifier.Notify(ctx, alert1)
			require.Equal(t, tt.expectedRetry, retry)
			if tt.noError {
				require.NoError(t, err)
			} else {
				var reasonError *notify.ErrorWithReason
				require.ErrorAs(t, err, &reasonError)
				require.Equal(t, tt.expectedReason, reasonError.Reason)
				require.Contains(t, err.Error(), tt.expectedErr)
				require.Contains(t, err.Error(), "channelname")
			}
		})
	}
}

func TestSlackTimeout(t *testing.T) {
	tests := map[string]struct {
		latency time.Duration
		timeout time.Duration
		wantErr bool
	}{
		"success": {latency: 100 * time.Millisecond, timeout: 120 * time.Millisecond, wantErr: false},
		"error":   {latency: 100 * time.Millisecond, timeout: 80 * time.Millisecond, wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			u, _ := url.Parse("https://slack.com/post.Message")
			notifier, err := New(
				&config.SlackConfig{
					NotifierConfig: config.NotifierConfig{},
					HTTPConfig:     &commoncfg.HTTPClientConfig{},
					APIURL:         &config.SecretURL{URL: u},
					Channel:        "channelname",
					Timeout:        tt.timeout,
				},
				test.CreateTmpl(t),
				promslog.NewNopLogger(),
			)
			require.NoError(t, err)
			notifier.postJSONFunc = func(ctx context.Context, client *http.Client, url string, body io.Reader) (*http.Response, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(tt.latency):
					resp := httptest.NewRecorder()
					resp.Header().Set("Content-Type", "application/json; charset=utf-8")
					resp.WriteHeader(http.StatusOK)
					_, _ = resp.WriteString(`{"ok":true}`)

					return resp.Result(), nil
				}
			}
			ctx := context.Background()
			ctx = notify.WithGroupKey(ctx, "1")

			alert := &types.Alert{
				Alert: model.Alert{
					StartsAt: time.Now(),
					EndsAt:   time.Now().Add(time.Hour),
				},
			}
			_, err = notifier.Notify(ctx, alert)
			require.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestSlackMessageField(t *testing.T) {
	// 1. Setup a fake Slack server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}

		// 2. VERIFY: Top-level text exists
		if body["text"] != "My Top Level Message" {
			t.Errorf("Expected top-level 'text' to be 'My Top Level Message', got %v", body["text"])
		}

		// 3. VERIFY: Old attachments still exist
		attachments, ok := body["attachments"].([]any)
		if !ok || len(attachments) == 0 {
			t.Errorf("Expected attachments to exist")
		} else {
			first := attachments[0].(map[string]any)
			if first["title"] != "Old Attachment Title" {
				t.Errorf("Expected attachment title 'Old Attachment Title', got %v", first["title"])
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	// 4. Configure Notifier with BOTH new and old fields
	u, _ := url.Parse(server.URL)
	conf := &config.SlackConfig{
		APIURL:      &config.SecretURL{URL: u},
		MessageText: "My Top Level Message", // Your NEW field
		Title:       "Old Attachment Title", // An OLD field
		Channel:     "#test-channel",
		HTTPConfig:  &commoncfg.HTTPClientConfig{},
	}

	tmpl, err := template.FromGlobs([]string{})
	if err != nil {
		t.Fatal(err)
	}
	tmpl.ExternalURL = u

	logger := slog.New(slog.DiscardHandler)
	notifier, err := New(conf, tmpl, logger)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ctx = notify.WithGroupKey(ctx, "test-group-key")

	if _, err := notifier.Notify(ctx); err != nil {
		t.Fatal("Notify failed:", err)
	}
}

// Non-SOP alerts render the minimal practitioner message (the channel text
// template), not the verbose English incident-field block. Even though the
// alert annotations carry SOP/AI fields and secrets, none of them must be
// rendered or leak into the payload.
func TestSlackNonSOPRendersMinimalWithoutLeakingSecrets(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	notifier, err := New(
		&config.SlackConfig{
			APIURL:     &config.SecretURL{URL: u},
			Channel:    "#test-channel",
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	ctx := notify.WithGroupKey(context.Background(), "test-group-key")
	alert := alertWithDSAPMIncidentFields()

	_, err = notifier.Notify(ctx, alert)

	require.NoError(t, err)
	attachments, ok := body["attachments"].([]any)
	require.True(t, ok)
	// Only the primary attachment; no secondary block and no verbose incident fields.
	require.Len(t, attachments, 1)
	if fields, ok := attachments[0].(map[string]any)["fields"].([]any); ok {
		fieldValues := slackFieldValues(fields)
		require.NotContains(t, fieldValues, "SOP ID")
		require.NotContains(t, fieldValues, "AI status")
		require.NotContains(t, fieldValues, "AI headline")
		require.NotContains(t, fieldValues, "SOP URL")
	}
	encodedBody, err := json.Marshal(body)
	require.NoError(t, err)
	require.NotContains(t, string(encodedBody), "token=hidden")
	require.NotContains(t, string(encodedBody), "bearer abcdefghijklmnopqrstuvwxyz")
}

func slackFieldValues(fields []any) map[string]string {
	values := make(map[string]string, len(fields))
	for _, field := range fields {
		fieldMap, ok := field.(map[string]any)
		if !ok {
			continue
		}
		title, _ := fieldMap["title"].(string)
		value, _ := fieldMap["value"].(string)
		values[title] = value
	}

	return values
}

func TestSlackUsesAIContentWhenBound(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
		bodies = append(bodies, b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true, "ts": "1700000000.000100"}`))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	notifier, err := New(
		&config.SlackConfig{
			APIURL:     &config.SecretURL{URL: u},
			Channel:    "#test-channel",
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	ctx := notify.WithGroupKey(context.Background(), "test-group-key")
	alert := &types.Alert{
		Alert: model.Alert{
			Labels: model.LabelSet{
				"alertname": "ShippingHighError",
			},
			Annotations: model.LabelSet{
				model.LabelName(alertmanagertypes.IncidentAnnotationNotificationBody): "## 현황\n서비스 5xx 급증",
				model.LabelName(alertmanagertypes.IncidentAnnotationSopTitle):         "Shipping 5xx 대응",
				model.LabelName(alertmanagertypes.IncidentAnnotationCustomerUpdate):   "[안내] 점검 중",
			},
			StartsAt: time.Now(),
			EndsAt:   time.Now().Add(time.Hour),
		},
	}

	_, err = notifier.Notify(ctx, alert)
	require.NoError(t, err)

	// SOP-bound alert posts the SOP body, then a threaded reply for the notice.
	require.Len(t, bodies, 2, "SOP-bound alert posts the body and a threaded reply")

	// Parent message: SOP body only — the customer notice is NOT here, so Slack's
	// length-based truncation cannot fold away the body.
	parentAtts := bodies[0]["attachments"].([]any)
	require.Len(t, parentAtts, 1, "parent message carries only the body attachment")
	att := parentAtts[0].(map[string]any)
	require.Equal(t, "Shipping 5xx 대응", att["title"], "attachment title should be AI sop_title")
	text, _ := att["text"].(string)
	require.Contains(t, text, "## 현황", "primary attachment text is the AI body")
	require.NotContains(t, text, alertmanagertypes.CollapsibleNoticeLabel, "customer notice must not sit in the body message")
	require.NotContains(t, text, "[안내] 점검 중", "customer notice must not sit in the body message")
	require.Empty(t, bodies[0]["thread_ts"], "parent message is not itself threaded")

	// Threaded reply: customer notice, threaded under the parent message ts.
	require.Equal(t, "1700000000.000100", bodies[1]["thread_ts"], "reply threads under the parent message ts")
	replyAtts := bodies[1]["attachments"].([]any)
	require.Len(t, replyAtts, 1)
	secText, _ := replyAtts[0].(map[string]any)["text"].(string)
	require.Contains(t, secText, alertmanagertypes.CollapsibleNoticeLabel, "reply carries the collapsible label")
	require.Contains(t, secText, "[안내] 점검 중", "reply carries the customer notice")
}

func slackFieldTitles(att map[string]any) []string {
	var titles []string
	if fs, ok := att["fields"].([]any); ok {
		for _, f := range fs {
			if m, ok := f.(map[string]any); ok {
				if t, ok := m["title"].(string); ok {
					titles = append(titles, t)
				}
			}
		}
	}
	return titles
}

func TestSlackAppendsRemediationToBodyAndDropsMetadataFields(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
		bodies = append(bodies, b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true, "ts": "1700000000.000100"}`))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	notifier, err := New(
		&config.SlackConfig{
			APIURL:     &config.SecretURL{URL: u},
			Channel:    "#test-channel",
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	// Realistic absolute URL with a UUID remediation id: the URL-aware sanitizer
	// must keep it intact (the generic secret redactor would mangle the 36-char
	// UUID into [redacted-secret] and break the link).
	const approveURL = "https://apm.example.com/alerts/overview?remediation=550e8400-e29b-41d4-a716-446655440000&ruleId=rule-123"
	ctx := notify.WithGroupKey(context.Background(), "test-group-key")
	alert := &types.Alert{
		Alert: model.Alert{
			Labels: model.LabelSet{
				"alertname": "ShippingHighError",
				model.LabelName(alertmanagertypes.IncidentLabelProjectID): "customer-a",
				model.LabelName(alertmanagertypes.IncidentLabelSeverity):  "critical",
			},
			Annotations: model.LabelSet{
				model.LabelName(alertmanagertypes.IncidentAnnotationNotificationBody):         "## 현황\n서비스 5xx 급증",
				model.LabelName(alertmanagertypes.IncidentAnnotationSopTitle):                 "Shipping 5xx 대응",
				model.LabelName(alertmanagertypes.IncidentAnnotationSopURL):                   "https://kb.example/sop/SOP-SHIP-001",
				model.LabelName(alertmanagertypes.IncidentAnnotationSopVersion):               "2026-06-17.1",
				model.LabelName(alertmanagertypes.IncidentAnnotationAIStrategyStatus):         "low_confidence",
				model.LabelName(alertmanagertypes.IncidentAnnotationCustomerUpdate):           "[안내] 점검 중",
				model.LabelName(alertmanagertypes.IncidentAnnotationRemediationScriptSummary): "결제 서비스 재시작",
				model.LabelName(alertmanagertypes.IncidentAnnotationRemediationApproveURL):    approveURL,
			},
			StartsAt: time.Now(),
			EndsAt:   time.Now().Add(time.Hour),
		},
	}

	_, err = notifier.Notify(ctx, alert)
	require.NoError(t, err)

	require.Len(t, bodies, 2, "SOP-bound alert posts the body and a threaded reply")

	// Parent (body) message carries no incident fields, but the auto-remediation
	// suggestion (summary + approval link) is appended at the bottom of the body.
	parentAtts := bodies[0]["attachments"].([]any)
	require.Len(t, parentAtts, 1)
	parentAtt := parentAtts[0].(map[string]any)
	require.Empty(t, slackFieldTitles(parentAtt), "body message must not carry incident fields")
	bodyText, _ := parentAtt["text"].(string)
	require.Contains(t, bodyText, "## 현황", "SOP body stays in the parent message")
	require.Contains(t, bodyText, "자동 대응", "auto-remediation label appended to the body")
	require.Contains(t, bodyText, "결제 서비스 재시작", "remediation summary appended to the body")
	require.Contains(t, bodyText, "<"+approveURL+"|승인>", "approval link rendered as a blue '승인' mrkdwn hyperlink")
	require.NotContains(t, bodyText, "[redacted-secret]", "the trusted approve URL must not be mangled by the secret redactor")

	// Threaded reply carries only the customer notice — no SOP metadata fields.
	replyAtts := bodies[1]["attachments"].([]any)
	require.Len(t, replyAtts, 1)
	replyAtt := replyAtts[0].(map[string]any)
	require.Empty(t, slackFieldTitles(replyAtt), "reply must not carry any SOP metadata fields")
	replyText, _ := replyAtt["text"].(string)
	require.Contains(t, replyText, alertmanagertypes.CollapsibleNoticeLabel)
	require.Contains(t, replyText, "[안내] 점검 중", "reply carries the customer notice")
	require.NotContains(t, replyText, "SOP URL")
	require.NotContains(t, replyText, "결제 서비스 재시작", "remediation is in the body, not the reply")
}

// TestSlackThreadReplyFallsBackForWebhook verifies that an incoming webhook
// (plaintext "ok" response, no message ts) still receives the notice/metadata as
// a second, standalone post — keeping it out of the body's truncation window
// even though true threading is unavailable.
func TestSlackThreadReplyFallsBackForWebhook(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
		bodies = append(bodies, b)
		// Incoming-webhook style: plaintext "ok", no JSON, no ts.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	notifier, err := New(
		&config.SlackConfig{
			APIURL:     &config.SecretURL{URL: u},
			Channel:    "#test-channel",
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	ctx := notify.WithGroupKey(context.Background(), "test-group-key")
	alert := &types.Alert{
		Alert: model.Alert{
			Labels: model.LabelSet{"alertname": "ShippingHighError"},
			Annotations: model.LabelSet{
				model.LabelName(alertmanagertypes.IncidentAnnotationNotificationBody): "## 현황\n서비스 5xx 급증",
				model.LabelName(alertmanagertypes.IncidentAnnotationSopTitle):         "Shipping 5xx 대응",
				model.LabelName(alertmanagertypes.IncidentAnnotationCustomerUpdate):   "[안내] 점검 중",
			},
			StartsAt: time.Now(),
			EndsAt:   time.Now().Add(time.Hour),
		},
	}

	_, err = notifier.Notify(ctx, alert)
	require.NoError(t, err)

	require.Len(t, bodies, 2, "webhook still gets body + standalone notice follow-up")
	require.Empty(t, bodies[1]["thread_ts"], "webhook reply cannot thread (no parent ts)")
	replyAtts := bodies[1]["attachments"].([]any)
	secText, _ := replyAtts[0].(map[string]any)["text"].(string)
	require.Contains(t, secText, alertmanagertypes.CollapsibleNoticeLabel)
	require.Contains(t, secText, "[안내] 점검 중")
}

func TestSlackUsesTemplateWhenUnbound(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	confTitle := "Alert: TestAlert"
	confText := "Something went wrong"
	notifier, err := New(
		&config.SlackConfig{
			APIURL:     &config.SecretURL{URL: u},
			Channel:    "#test-channel",
			Title:      confTitle,
			Text:       confText,
			HTTPConfig: &commoncfg.HTTPClientConfig{},
		},
		test.CreateTmpl(t),
		promslog.NewNopLogger(),
	)
	require.NoError(t, err)

	ctx := notify.WithGroupKey(context.Background(), "test-group-key")
	// Alert with NO notification_body annotation → should use conf template values
	alert := &types.Alert{
		Alert: model.Alert{
			Labels: model.LabelSet{
				"alertname": "TestAlert",
			},
			Annotations: model.LabelSet{
				// only non-AI annotations; no notification_body
				model.LabelName(alertmanagertypes.IncidentAnnotationSopTitle): "some sop title",
			},
			StartsAt: time.Now(),
			EndsAt:   time.Now().Add(time.Hour),
		},
	}

	_, err = notifier.Notify(ctx, alert)
	require.NoError(t, err)

	attachments, ok := capturedBody["attachments"].([]any)
	require.True(t, ok)
	require.Len(t, attachments, 1)

	att := attachments[0].(map[string]any)
	require.Equal(t, confTitle, att["title"], "title should use conf template when unbound")
	require.Equal(t, confText, att["text"], "text should use conf template when unbound")
	require.NotContains(t, att["text"], alertmanagertypes.CollapsibleNoticeLabel, "collapsible label must not appear when unbound")
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
