// Copyright (c) 2026 SigNoz, Inc.
// Copyright 2019 Prometheus Team
// SPDX-License-Identifier: Apache-2.0

package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	commoncfg "github.com/prometheus/common/config"

	"github.com/prometheus/alertmanager/config"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/types"
)

const (
	Integration = "slack"
)

// https://api.slack.com/reference/messaging/attachments#legacy_fields - 1024, no units given, assuming runes or characters.
const maxTitleLenRunes = 1024

// Notifier implements a Notifier for Slack notifications.
type Notifier struct {
	conf    *config.SlackConfig
	tmpl    *template.Template
	logger  *slog.Logger
	client  *http.Client
	retrier *notify.Retrier

	postJSONFunc func(ctx context.Context, client *http.Client, url string, body io.Reader) (*http.Response, error)
}

// New returns a new Slack notification handler.
func New(c *config.SlackConfig, t *template.Template, l *slog.Logger, httpOpts ...commoncfg.HTTPClientOption) (*Notifier, error) {
	client, err := notify.NewClientWithTracing(*c.HTTPConfig, Integration, httpOpts...)
	if err != nil {
		return nil, err
	}

	return &Notifier{
		conf:         c,
		tmpl:         t,
		logger:       l,
		client:       client,
		retrier:      &notify.Retrier{},
		postJSONFunc: notify.PostJSON,
	}, nil
}

// request is the request for sending a slack notification.
type request struct {
	Channel     string       `json:"channel,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	IconURL     string       `json:"icon_url,omitempty"`
	LinkNames   bool         `json:"link_names,omitempty"`
	Text        string       `json:"text,omitempty"`
	Attachments []attachment `json:"attachments"`
	// ThreadTS threads a follow-up message under a parent. Set on the customer
	// notice reply so Slack hides it behind "N replies" while the SOP body in the
	// parent message stays fully visible (not subject to that message's
	// length-based "show more" truncation). Only honored by the web API
	// (chat.postMessage); incoming webhooks ignore it and post a standalone reply.
	ThreadTS string `json:"thread_ts,omitempty"`
}

// attachment is used to display a richly-formatted message block.
type attachment struct {
	Title      string               `json:"title,omitempty"`
	TitleLink  string               `json:"title_link,omitempty"`
	Pretext    string               `json:"pretext,omitempty"`
	Text       string               `json:"text"`
	Fallback   string               `json:"fallback"`
	CallbackID string               `json:"callback_id"`
	Fields     []config.SlackField  `json:"fields,omitempty"`
	Actions    []config.SlackAction `json:"actions,omitempty"`
	ImageURL   string               `json:"image_url,omitempty"`
	ThumbURL   string               `json:"thumb_url,omitempty"`
	Footer     string               `json:"footer"`
	Color      string               `json:"color,omitempty"`
	MrkdwnIn   []string             `json:"mrkdwn_in,omitempty"`
}

// Notify implements the Notifier interface.
func (n *Notifier) Notify(ctx context.Context, as ...*types.Alert) (bool, error) {

	key, err := notify.ExtractGroupKey(ctx)
	if err != nil {
		return false, err
	}
	logger := n.logger.With(slog.Any("group_key", key))
	logger.DebugContext(ctx, "extracted group key")

	var (
		data     = notify.GetTemplateData(ctx, n.tmpl, as, logger)
		tmplText = notify.TmplText(n.tmpl, data, &err)
	)
	var markdownIn []string

	if len(n.conf.MrkdwnIn) == 0 {
		markdownIn = []string{"fallback", "pretext", "text"}
	} else {
		markdownIn = n.conf.MrkdwnIn
	}

	title, truncated := notify.TruncateInRunes(tmplText(n.conf.Title), maxTitleLenRunes)
	if truncated {
		logger.WarnContext(ctx, "Truncated title", slog.Int("max_runes", maxTitleLenRunes))
	}
	text := tmplText(n.conf.Text)

	// DS-APM: SOP 바운드 알림이면 채널 템플릿 대신 AI 제목·본문·고객공지로 대체.
	notif, sopBound := alertmanagertypes.ResolveSOPBoundNotification(data.Status, data.CommonAnnotations)
	if sopBound {
		if notif.Title != "" {
			title, _ = notify.TruncateInRunes(notif.Title, maxTitleLenRunes)
		}
		// SOP body, with the human-gated auto-remediation suggestion (요약 + 승인
		// 링크) appended at the bottom so operators see it inline with the SOP they
		// must act on — not as a separate reply/field. The customer notice still
		// moves to a second attachment (below) so Slack's "show more" collapses it
		// rather than truncating the SOP body.
		text = notif.Body
		if remediation := buildSlackRemediationText(data.CommonAnnotations); remediation != "" {
			text = strings.TrimRight(text, "\n") + "\n\n" + remediation
		}
	}

	att := &attachment{
		Title:      title,
		TitleLink:  tmplText(n.conf.TitleLink),
		Pretext:    tmplText(n.conf.Pretext),
		Text:       text,
		Fallback:   tmplText(n.conf.Fallback),
		CallbackID: tmplText(n.conf.CallbackID),
		ImageURL:   tmplText(n.conf.ImageURL),
		ThumbURL:   tmplText(n.conf.ThumbURL),
		Footer:     tmplText(n.conf.Footer),
		Color:      tmplText(n.conf.Color),
		MrkdwnIn:   markdownIn,
	}

	numFields := len(n.conf.Fields)
	if numFields > 0 {
		fields := make([]config.SlackField, numFields)
		for index, field := range n.conf.Fields {
			// Check if short was defined for the field otherwise fallback to the global setting
			var short bool
			if field.Short != nil {
				short = *field.Short
			} else {
				short = n.conf.ShortFields
			}

			// Rebuild the field by executing any templates and setting the new value for short
			fields[index] = config.SlackField{
				Title: tmplText(field.Title),
				Value: tmplText(field.Value),
				Short: &short,
			}
		}
		att.Fields = fields
	}
	var secondaryAtt *attachment
	if sopBound && notif.CustomerNotice != "" {
		// Secondary attachment: customer notice only. The SOP metadata fields
		// (project/severity/SOP link 등) are intentionally omitted, and the
		// auto-remediation now lives at the bottom of the SOP body. Kept out of the
		// primary attachment so Slack's "show more" collapses the notice instead of
		// truncating the SOP body operators must read first.
		secondaryAtt = &attachment{
			Fallback: title,
			MrkdwnIn: markdownIn,
			Text:     alertmanagertypes.CollapsibleNoticeLabel + "\n" + notif.CustomerNotice,
		}
	}
	// Non-SOP alerts intentionally append no incident fields: the channel text
	// template already carries the practitioner-facing summary (severity,
	// service, time, error, action), so the verbose English IncidentInfoFields
	// block is omitted to keep the message minimal.

	numActions := len(n.conf.Actions)
	if numActions > 0 {
		actions := make([]config.SlackAction, numActions)
		for index, action := range n.conf.Actions {
			slackAction := config.SlackAction{
				Type:  tmplText(action.Type),
				Text:  tmplText(action.Text),
				URL:   tmplText(action.URL),
				Style: tmplText(action.Style),
				Name:  tmplText(action.Name),
				Value: tmplText(action.Value),
			}

			if action.ConfirmField != nil {
				slackAction.ConfirmField = &config.SlackConfirmationField{
					Title:       tmplText(action.ConfirmField.Title),
					Text:        tmplText(action.ConfirmField.Text),
					OkText:      tmplText(action.ConfirmField.OkText),
					DismissText: tmplText(action.ConfirmField.DismissText),
				}
			}

			actions[index] = slackAction
		}
		att.Actions = actions
	}

	// Primary message: title + SOP body (with the auto-remediation suggestion
	// appended at the bottom). For SOP-bound alerts the customer notice is
	// deliberately left off here and posted as a threaded reply below, so this
	// message stays short enough that Slack's length-based "show more" truncation
	// does not fold away the SOP body operators must read first.
	req := &request{
		Channel:     tmplText(n.conf.Channel),
		Username:    tmplText(n.conf.Username),
		IconEmoji:   tmplText(n.conf.IconEmoji),
		IconURL:     tmplText(n.conf.IconURL),
		LinkNames:   n.conf.LinkNames,
		Text:        tmplText(n.conf.MessageText),
		Attachments: []attachment{*att},
	}
	if err != nil {
		return false, err
	}

	u, err := n.resolveURL()
	if err != nil {
		return false, err
	}

	if n.conf.Timeout > 0 {
		postCtx, cancel := context.WithTimeoutCause(ctx, n.conf.Timeout, errors.NewInternalf(errors.CodeTimeout, "configured slack timeout reached (%s)", n.conf.Timeout))
		defer cancel()
		ctx = postCtx
	}

	ts, retry, err := n.post(ctx, u, req)
	if err != nil {
		return retry, err
	}

	// TEMP DEBUG (remove after verifying threading): record the threading decision.
	logger.InfoContext(ctx, "DSAPM slack threading decision",
		slog.Bool("sopBound", sopBound),
		slog.Bool("hasSecondary", secondaryAtt != nil),
		slog.Int("bodyLen", len(text)),
		slog.String("parentTS", ts))

	// Threaded follow-up: customer notice only. Best-effort and fail-open — the
	// operator-critical body is already delivered, so a reply failure is logged
	// rather than failing (and re-sending) the whole alert.
	// With the web API we thread under the parent message (ts); an incoming
	// webhook returns no ts, so this lands as a standalone follow-up — either way
	// it sits outside the parent body's truncation window.
	if secondaryAtt != nil {
		reply := &request{
			Channel:     req.Channel,
			Username:    req.Username,
			IconEmoji:   req.IconEmoji,
			IconURL:     req.IconURL,
			LinkNames:   req.LinkNames,
			Attachments: []attachment{*secondaryAtt},
			ThreadTS:    ts,
		}
		if replyTS, _, replyErr := n.post(ctx, u, reply); replyErr != nil {
			logger.WarnContext(ctx, "failed to post SOP customer notice reply", slog.Any("err", replyErr))
		} else {
			// TEMP DEBUG (remove after verifying threading).
			logger.InfoContext(ctx, "DSAPM slack reply posted", slog.String("replyTS", replyTS), slog.String("threadedUnder", ts))
		}
	}

	return retry, nil
}

// buildSlackRemediationText renders the human-gated auto-remediation suggestion
// (one-line summary + approval deep link) as Slack mrkdwn, to be appended at the
// bottom of the SOP body. It returns "" when no remediation is proposed. The full
// script is never included — only the summary and a link to the approval card.
func buildSlackRemediationText(annotations template.KV) string {
	summary := alertmanagertypes.SanitizeIncidentValue(strings.TrimSpace(annotations[alertmanagertypes.IncidentAnnotationRemediationScriptSummary]))
	if summary == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString("🔧 *자동 대응*\n")
	b.WriteString(summary)
	// Render the approval link as Slack mrkdwn (<url|label>) so it shows as a blue
	// "승인" hyperlink rather than a raw URL. The URL is our own generated deep
	// link, so it goes through the URL-aware sanitizer (which keeps the UUID/rule
	// ids intact) — Slack only renders absolute http(s) URLs as links.
	if approveURL := alertmanagertypes.SanitizeIncidentApproveURL(annotations[alertmanagertypes.IncidentAnnotationRemediationApproveURL]); approveURL != "" {
		b.WriteString("\n<")
		b.WriteString(approveURL)
		b.WriteString("|승인>")
	}

	return b.String()
}

// resolveURL returns the Slack endpoint from either the inline APIURL or the
// APIURLFile.
func (n *Notifier) resolveURL() (string, error) {
	if n.conf.APIURL != nil {
		return n.conf.APIURL.String(), nil
	}
	content, err := os.ReadFile(n.conf.APIURLFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// post sends one Slack request. On success it returns the parent message
// timestamp (ts) when Slack's web API supplies one (empty for incoming
// webhooks); the bool is the alertmanager retry signal.
func (n *Notifier) post(ctx context.Context, u string, req *request) (string, bool, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return "", false, err
	}

	resp, err := n.postJSONFunc(ctx, n.client, u, &buf) //nolint:bodyclose
	if err != nil {
		if ctx.Err() != nil {
			err = errors.NewInternalf(errors.CodeInternal, "failed to post JSON to slack: %v", context.Cause(ctx))
		}
		return "", true, notify.RedactURL(err)
	}
	defer notify.Drain(resp)

	// Use a retrier to generate an error message for non-200 responses and
	// classify them as retriable or not.
	retry, err := n.retrier.Check(resp.StatusCode, resp.Body)
	if err != nil {
		err = errors.NewInternalf(errors.CodeInternal, "channel %q: %v", req.Channel, err)
		return "", retry, notify.NewErrorWithReason(notify.GetFailureReasonFromStatusCode(resp.StatusCode), err)
	}

	// Slack web API might return errors with a 200 response code.
	// https://docs.slack.dev/tools/node-slack-sdk/web-api/#handle-errors
	ts, retry, err := checkResponseError(resp)
	if err != nil {
		err = errors.NewInternalf(errors.CodeInternal, "channel %q: %v", req.Channel, err)
		return "", retry, notify.NewErrorWithReason(notify.ClientErrorReason, err)
	}

	return ts, retry, nil
}

// checkResponseError parses out the error message from a Slack API response and
// returns the parent message timestamp (ts) when present (web API only).
func checkResponseError(resp *http.Response) (string, bool, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", true, errors.WrapInternalf(err, errors.CodeInternal, "could not read response body")
	}

	if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		return checkJSONResponseError(body)
	}
	return checkTextResponseError(body)
}

// checkTextResponseError classifies plaintext responses from Slack.
// A plaintext (non-JSON) response is successful if it's a string "ok".
// This is typically a response for an Incoming Webhook
// (https://api.slack.com/messaging/webhooks#handling_errors). Webhooks carry no
// message timestamp, so the returned ts is always empty.
func checkTextResponseError(body []byte) (string, bool, error) {
	if !bytes.Equal(body, []byte("ok")) {
		return "", false, errors.NewInternalf(errors.CodeInternal, "received an error response from Slack: %s", string(body))
	}
	return "", false, nil
}

// checkJSONResponseError classifies JSON responses from Slack and extracts the
// posted message timestamp (ts), used to thread the follow-up reply.
func checkJSONResponseError(body []byte) (string, bool, error) {
	// response is for parsing out errors and the message ts from the JSON response.
	type response struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		TS    string `json:"ts"`
	}

	var data response
	if err := json.Unmarshal(body, &data); err != nil {
		return "", true, errors.NewInternalf(errors.CodeInternal, "could not unmarshal JSON response %q: %v", string(body), err)
	}
	if !data.OK {
		return "", false, errors.NewInternalf(errors.CodeInternal, "error response from Slack: %s", data.Error)
	}
	return data.TS, false, nil
}
