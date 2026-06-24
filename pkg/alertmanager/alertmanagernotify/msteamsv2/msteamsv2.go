// Copyright (c) 2026 SigNoz, Inc.
// Copyright 2019 Prometheus Team
// SPDX-License-Identifier: Apache-2.0

package msteamsv2

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
	"github.com/prometheus/common/model"

	"github.com/prometheus/alertmanager/config"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/types"
)

const (
	colorRed   = "Attention"
	colorGreen = "Good"
	colorGrey  = "Warning"
)

const (
	Integration = "msteamsv2"
)

type Notifier struct {
	conf         *config.MSTeamsV2Config
	titleLink    string
	tmpl         *template.Template
	logger       *slog.Logger
	client       *http.Client
	retrier      *notify.Retrier
	webhookURL   *config.SecretURL
	postJSONFunc func(ctx context.Context, client *http.Client, url string, body io.Reader) (*http.Response, error)
}

// https://learn.microsoft.com/en-us/connectors/teams/?tabs=text1#adaptivecarditemschema
type Content struct {
	Schema  string   `json:"$schema"`
	Type    string   `json:"type"`
	Version string   `json:"version"`
	Body    []Body   `json:"body"`
	Msteams Msteams  `json:"msteams,omitempty"`
	Actions []Action `json:"actions"`
}

type Body struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	Weight string `json:"weight,omitempty"`
	Size   string `json:"size,omitempty"`
	Wrap   bool   `json:"wrap,omitempty"`
	Style  string `json:"style,omitempty"`
	Color  string `json:"color,omitempty"`
	Facts  []Fact `json:"facts,omitempty"`
}

type Action struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

type Fact struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

type Msteams struct {
	Width string `json:"width"`
}

type Attachment struct {
	ContentType string  `json:"contentType"`
	ContentURL  *string `json:"contentUrl"` // Use a pointer to handle null values
	Content     Content `json:"content"`
}

type teamsMessage struct {
	Type        string       `json:"type"`
	Attachments []Attachment `json:"attachments"`
}

// New returns a new notifier that uses the Microsoft Teams Power Platform connector.
func New(c *config.MSTeamsV2Config, t *template.Template, titleLink string, l *slog.Logger, httpOpts ...commoncfg.HTTPClientOption) (*Notifier, error) {
	client, err := notify.NewClientWithTracing(*c.HTTPConfig, Integration, httpOpts...)
	if err != nil {
		return nil, err
	}

	n := &Notifier{
		conf:         c,
		titleLink:    titleLink,
		tmpl:         t,
		logger:       l,
		client:       client,
		retrier:      &notify.Retrier{},
		webhookURL:   c.WebhookURL,
		postJSONFunc: notify.PostJSON,
	}

	return n, nil
}

func (n *Notifier) Notify(ctx context.Context, as ...*types.Alert) (bool, error) {
	key, err := notify.ExtractGroupKey(ctx)
	if err != nil {
		return false, err
	}

	n.logger.DebugContext(ctx, "extracted group key", slog.String("key", string(key)))

	data := notify.GetTemplateData(ctx, n.tmpl, as, n.logger)
	tmpl := notify.TmplText(n.tmpl, data, &err)
	if err != nil {
		return false, err
	}

	title := tmpl(n.conf.Title)
	if err != nil {
		return false, err
	}

	// Override title with AI-generated SOP title when available.
	// We peek at CommonAnnotations here; full aiOK check happens below after card init.
	if sopTitle := strings.TrimSpace(data.CommonAnnotations[alertmanagertypes.IncidentAnnotationSopTitle]); sopTitle != "" {
		if strings.TrimSpace(data.CommonAnnotations[alertmanagertypes.IncidentAnnotationNotificationBody]) != "" {
			title = sopTitle
		}
	} else if aiHeadline := strings.TrimSpace(data.CommonAnnotations[alertmanagertypes.IncidentAnnotationAIHeadline]); aiHeadline != "" {
		if strings.TrimSpace(data.CommonAnnotations[alertmanagertypes.IncidentAnnotationNotificationBody]) != "" {
			title = aiHeadline
		}
	}

	titleLink := tmpl(n.titleLink)
	if err != nil {
		return false, err
	}

	alerts := types.Alerts(as...)
	color := colorGrey
	switch alerts.Status() {
	case model.AlertFiring:
		color = colorRed
	case model.AlertResolved:
		color = colorGreen
	}

	var url string
	if n.conf.WebhookURL != nil {
		url = n.conf.WebhookURL.String()
	} else {
		content, err := os.ReadFile(n.conf.WebhookURLFile)
		if err != nil {
			return false, errors.WrapInternalf(err, errors.CodeInternal, "read webhook_url_file")
		}
		url = strings.TrimSpace(string(content))
	}

	// A message as referenced in https://learn.microsoft.com/en-us/connectors/teams/?tabs=text1%2Cdotnet#request-body-schema
	t := teamsMessage{
		Type: "message",
		Attachments: []Attachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				ContentURL:  nil,
				Content: Content{
					Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
					Type:    "AdaptiveCard",
					Version: "1.2",
					Body: []Body{
						{
							Type:   "TextBlock",
							Text:   title,
							Weight: "Bolder",
							Size:   "Medium",
							Wrap:   true,
							Style:  "heading",
							Color:  color,
						},
					},
					Actions: []Action{
						{
							Type:  "Action.OpenUrl",
							Title: "View Alert",
							URL:   titleLink,
						},
					},
					Msteams: Msteams{
						Width: "full",
					},
				},
			},
		},
	}

	// If CommonAnnotations carry an AI-generated notification body, use it
	// to replace the per-alert Labels/Annotations FactSet blocks (fail-open).
	notif, aiOK := alertmanagertypes.ResolveSOPBoundNotification(data.CommonAnnotations)
	if aiOK {
		// AI main body block
		t.Attachments[0].Content.Body = append(t.Attachments[0].Content.Body, Body{
			Type: "TextBlock",
			Text: notif.Body,
			Wrap: true,
		})
		if notif.CustomerNotice != "" {
			t.Attachments[0].Content.Body = append(t.Attachments[0].Content.Body, collapsibleNoticeBlocks(notif.CustomerNotice)...)
		}
	} else {
		// Non-SOP alerts render a minimal Korean incident block (severity,
		// service, time, error, action) instead of dumping every label and
		// annotation, keeping the card readable for on-call operators.
		for _, alert := range as {
			t.Attachments[0].Content.Body = append(t.Attachments[0].Content.Body, koreanIncidentBody(alert)...)
		}
	}

	var payload bytes.Buffer
	if err = json.NewEncoder(&payload).Encode(t); err != nil {
		return false, err
	}

	resp, err := n.postJSONFunc(ctx, n.client, url, &payload) //nolint:bodyclose
	if err != nil {
		return true, notify.RedactURL(err)
	}
	defer notify.Drain(resp) //drain is used to close the body of the response hence the nolint directive

	// https://learn.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/connectors-using?tabs=cURL#rate-limiting-for-connectors
	shouldRetry, err := n.retrier.Check(resp.StatusCode, resp.Body)
	if err != nil {
		return shouldRetry, notify.NewErrorWithReason(notify.GetFailureReasonFromStatusCode(resp.StatusCode), err)
	}
	return shouldRetry, err
}

// koreanIncidentBody renders the practitioner-facing minimal incident block for
// a non-SOP alert: severity, service, and time as a FactSet, followed by the
// error description and (when present) the recommended action. It deliberately
// omits the raw label/annotation dump so the Teams card stays readable.
func koreanIncidentBody(alert *types.Alert) []Body {
	severity := strings.ToUpper(strings.TrimSpace(string(alert.Labels[alertmanagertypes.IncidentLabelSeverity])))
	if severity == "" {
		severity = "-"
	}
	service := strings.TrimSpace(string(alert.Labels[alertmanagertypes.IncidentLabelServiceName]))
	if service == "" {
		service = "-"
	}

	bodies := []Body{{
		Type: "FactSet",
		Facts: []Fact{
			{Title: "심각도", Value: severity},
			{Title: "서비스", Value: service},
			{Title: "발생시간", Value: alertmanagertypes.FormatKST(alert.StartsAt)},
		},
	}}

	if desc := alertmanagertypes.SanitizeIncidentValue(string(alert.Annotations["description"])); desc != "" {
		bodies = append(bodies,
			Body{Type: "TextBlock", Text: "📋 오류 내용", Weight: "Bolder", Wrap: true},
			Body{Type: "TextBlock", Text: desc, Wrap: true},
		)
	}
	if action := alertmanagertypes.SanitizeIncidentValue(string(alert.Annotations[alertmanagertypes.IncidentAnnotationNextAction])); action != "" {
		bodies = append(bodies,
			Body{Type: "TextBlock", Text: "✅ 조치 사항", Weight: "Bolder", Wrap: true},
			Body{Type: "TextBlock", Text: action, Wrap: true},
		)
	}

	return bodies
}

// collapsibleNoticeBlocks returns TextBlocks representing a labeled customer-notice section.
// The Body/Action structs do not support ToggleVisibility (no ID/IsVisible/TargetElements fields),
// so we use two TextBlocks: a bold label and the notice text (acceptance bar: "appears as own section").
func collapsibleNoticeBlocks(notice string) []Body {
	return []Body{
		{Type: "TextBlock", Text: alertmanagertypes.CollapsibleNoticeLabel, Weight: "Bolder", Wrap: true},
		{Type: "TextBlock", Text: notice, Wrap: true},
	}
}
