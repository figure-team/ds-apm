package delivery

import (
	"context"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/prometheus/alertmanager/api/v2/models"
)

// AlertPutter is the narrow slice of alertmanager.Alertmanager the sink needs.
type AlertPutter interface {
	PutAlerts(ctx context.Context, orgID string, alerts alertmanagertypes.PostableAlerts) error
}

// AlertmanagerSink delivers a code-RCA handoff as a meta-alert through the
// normal CF-3 dispatch path (channels, templates, PII filter all reused).
type AlertmanagerSink struct {
	am AlertPutter
}

// NewAlertmanagerSink builds the sink over the running alertmanager.
func NewAlertmanagerSink(am AlertPutter) *AlertmanagerSink {
	return &AlertmanagerSink{am: am}
}

// Submit publishes the handoff as a PostableAlert; ref = run id.
func (s *AlertmanagerSink) Submit(ctx context.Context, msg HandoffMessage) (string, error) {
	alert := &alertmanagertypes.PostableAlert{
		Annotations: models.LabelSet{
			"summary":                 msg.Title,
			"description":             msg.Body,
			"coderca.run_id":          msg.RunID,
			"coderca.baseline_commit": msg.BaselineCommit,
		},
		Alert: models.Alert{
			Labels: models.LabelSet{
				"alertname":    "CodeRCASuggestion",
				"service.name": msg.Service,
				"severity":     "info",
				"coderca":      "true",
			},
		},
	}
	if err := s.am.PutAlerts(ctx, msg.OrgID, alertmanagertypes.PostableAlerts{alert}); err != nil {
		return "", err
	}
	return msg.RunID, nil
}

var _ HandoffSink = (*AlertmanagerSink)(nil)
