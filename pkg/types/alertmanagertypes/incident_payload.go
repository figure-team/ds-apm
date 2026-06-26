package alertmanagertypes

import (
	"net/url"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/client_golang/prometheus"
)

const RedactedIncidentValue = "[redacted]"

// incidentRedactionsTotal counts every incident field value that gets redacted
// before leaving SigNoz (PII/secret masking). It is a package-level counter so
// the leaf sanitizer can observe it regardless of the calling channel; expose
// it on the scrape endpoint with RegisterRedactionMetrics.
var incidentRedactionsTotal = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "signoz_alertmanager_incident_redactions_total",
	Help: "Number of incident field values redacted (PII/secret masking) before delivery to a notification channel.",
})

var registerRedactionMetricsOnce sync.Once

// RegisterRedactionMetrics registers the redaction counter with the given
// registerer so it is exposed on the Prometheus scrape endpoint. The counter is
// a process-global total, so registration happens exactly once even though the
// alertmanager server is constructed per organization. It is safe to pass a nil
// registerer (used by hermetic unit tests, which read the counter directly via
// testutil).
func RegisterRedactionMetrics(r prometheus.Registerer) {
	if r == nil {
		return
	}
	registerRedactionMetricsOnce.Do(func() {
		r.MustRegister(incidentRedactionsTotal)
	})
}

type IncidentField struct {
	Key   string
	Title string
	Value string
	Short bool
}

var incidentJWTLikePattern = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{10,}\b`)

var (
	incidentValueEmailPattern        = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
	incidentValueKoreanMobilePattern = regexp.MustCompile(`(?:\+?82[-\s]?)?0?1[016789][-)\s]?\d{3,4}[-\s]?\d{4}`)
	incidentValueLongSecretPattern   = regexp.MustCompile(`\b[A-Za-z0-9_\-]{32,}\b`)
)

var incidentSensitiveURLKeys = map[string]struct{}{
	"access_token":  {},
	"api_key":       {},
	"apikey":        {},
	"auth":          {},
	"authorization": {},
	"bearer":        {},
	"client_secret": {},
	"password":      {},
	"secret":        {},
	"token":         {},
}

func BuildSafeIncidentInfo(labels, annotations template.KV) IncidentInfo {
	return SanitizeIncidentInfo(BuildIncidentInfo(labels, annotations))
}

func SanitizeIncidentInfo(info IncidentInfo) IncidentInfo {
	return IncidentInfo{
		ProjectID:             SanitizeIncidentValue(info.ProjectID),
		Environment:           SanitizeIncidentValue(info.Environment),
		ServiceName:           SanitizeIncidentValue(info.ServiceName),
		OwnerTeam:             SanitizeIncidentValue(info.OwnerTeam),
		Severity:              SanitizeIncidentValue(info.Severity),
		ImpactSummary:         SanitizeIncidentValue(info.ImpactSummary),
		NextAction:            SanitizeIncidentValue(info.NextAction),
		VendorRequest:         SanitizeIncidentValue(info.VendorRequest),
		CustomerUpdate:        SanitizeIncidentValue(info.CustomerUpdate),
		NotificationBody:      SanitizeIncidentValue(info.NotificationBody),
		SopID:                 SanitizeIncidentValue(info.SopID),
		SopURL:                SanitizeIncidentValue(info.SopURL),
		SopSource:             SanitizeIncidentValue(info.SopSource),
		SopTitle:              SanitizeIncidentValue(info.SopTitle),
		SopVersion:            SanitizeIncidentValue(info.SopVersion),
		SopBindingID:          SanitizeIncidentValue(info.SopBindingID),
		AIStrategyID:          SanitizeIncidentValue(info.AIStrategyID),
		AIStrategyStatus:      SanitizeIncidentValue(info.AIStrategyStatus),
		AIHeadline:            SanitizeIncidentValue(info.AIHeadline),
		AIFirstActions:        SanitizeIncidentValue(info.AIFirstActions),
		AIConfidence:          SanitizeIncidentValue(info.AIConfidence),
		AILimitations:         SanitizeIncidentValue(info.AILimitations),
		AIEvidenceRefs:        SanitizeIncidentValue(info.AIEvidenceRefs),
		RemediationSummary:    SanitizeIncidentValue(info.RemediationSummary),
		RemediationApproveURL: SanitizeIncidentApproveURL(info.RemediationApproveURL),
	}
}

// SanitizeIncidentApproveURL sanitizes the internally-generated remediation
// approval deep link. Unlike SanitizeIncidentValue it does NOT apply the
// long-token / PII redactors, which would mangle the legitimate UUID and rule
// identifiers in the query string into [redacted-secret] and break the link. It
// still strips sensitive query keys (defense in depth) and drops the value when
// it is not an absolute http(s) URL, since a relative URL cannot render as a
// clickable link anyway.
func SanitizeIncidentApproveURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sanitized, ok := sanitizeIncidentURL(value)
	if !ok {
		return ""
	}
	return sanitized
}

func SanitizeIncidentValue(value string) string {
	sanitized, redacted := sanitizeIncidentValue(value)
	if redacted {
		incidentRedactionsTotal.Inc()
	}
	return sanitized
}

// sanitizeIncidentValue performs the masking and reports whether any redaction
// was applied, so callers can drive the redaction metric. Its string output is
// identical to the historical SanitizeIncidentValue behavior; the bool is the
// only addition (the exported signature is unchanged — see SCOPE seam).
func sanitizeIncidentValue(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}

	redacted := false
	if sanitizedURL, ok := sanitizeIncidentURL(value); ok {
		if sanitizedURL != value {
			redacted = true
		}
		value = sanitizedURL
	}
	if incidentValueLooksSecret(value) {
		return RedactedIncidentValue, true
	}

	before := value
	value = redactIncidentPII(value)
	value = redactIncidentLongSecret(value)
	if value != before {
		redacted = true
	}

	return value, redacted
}

// SanitizeTemplateData returns a defensive copy of the Alertmanager template
// data with every label and annotation value passed through the incident
// sanitizer, so the outbound notification payload carries no raw PII/secret.
// The input is never mutated: callers (e.g. webhook URL templating) keep using
// the raw data. Each redacted value advances the redaction metric.
func SanitizeTemplateData(data *template.Data) *template.Data {
	if data == nil {
		return nil
	}

	out := *data
	out.GroupLabels = sanitizeIncidentKV(data.GroupLabels)
	out.CommonLabels = sanitizeIncidentKV(data.CommonLabels)
	out.CommonAnnotations = sanitizeIncidentKV(data.CommonAnnotations)

	if data.Alerts != nil {
		alerts := make(template.Alerts, len(data.Alerts))
		for i, alert := range data.Alerts {
			alert.Labels = sanitizeIncidentKV(alert.Labels)
			alert.Annotations = sanitizeIncidentKV(alert.Annotations)
			if alert.GeneratorURL != "" {
				alert.GeneratorURL = SanitizeIncidentValue(alert.GeneratorURL)
			}
			alerts[i] = alert
		}
		out.Alerts = alerts
	}

	return &out
}

func sanitizeIncidentKV(kv template.KV) template.KV {
	if kv == nil {
		return nil
	}
	out := make(template.KV, len(kv))
	for key, value := range kv {
		out[key] = SanitizeIncidentValue(value)
	}
	return out
}

func redactIncidentPII(value string) string {
	value = incidentValueEmailPattern.ReplaceAllString(value, "[redacted-email]")
	value = incidentValueKoreanMobilePattern.ReplaceAllString(value, "[redacted-phone]")
	return value
}

func redactIncidentLongSecret(value string) string {
	return incidentValueLongSecretPattern.ReplaceAllString(value, "[redacted-secret]")
}

func IncidentInfoFields(info IncidentInfo) []IncidentField {
	info = SanitizeIncidentInfo(info)
	fields := []IncidentField{
		{Key: "project_id", Title: "Project", Value: info.ProjectID, Short: true},
		{Key: "environment", Title: "Environment", Value: info.Environment, Short: true},
		{Key: "service_name", Title: "Service", Value: info.ServiceName, Short: true},
		{Key: "owner_team", Title: "Owner team", Value: info.OwnerTeam, Short: true},
		{Key: "severity", Title: "Severity", Value: info.Severity, Short: true},
		{Key: "impact_summary", Title: "Impact", Value: info.ImpactSummary},
		{Key: "next_action", Title: "Next action", Value: info.NextAction},
		{Key: "vendor_request", Title: "Vendor request", Value: info.VendorRequest},
		{Key: "customer_update", Title: "Customer update", Value: info.CustomerUpdate},
		{Key: "sop_id", Title: "SOP ID", Value: info.SopID, Short: true},
		{Key: "sop_title", Title: "SOP title", Value: info.SopTitle},
		{Key: "sop_version", Title: "SOP version", Value: info.SopVersion, Short: true},
		{Key: "sop_source", Title: "SOP source", Value: info.SopSource, Short: true},
		{Key: "sop_url", Title: "SOP URL", Value: info.SopURL},
		{Key: "sop_binding_id", Title: "SOP binding", Value: info.SopBindingID},
		{Key: "ai_strategy_id", Title: "AI strategy ID", Value: info.AIStrategyID},
		{Key: "ai_strategy_status", Title: "AI status", Value: info.AIStrategyStatus, Short: true},
		{Key: "ai_confidence", Title: "AI confidence", Value: info.AIConfidence, Short: true},
		{Key: "ai_headline", Title: "AI headline", Value: info.AIHeadline},
		{Key: "ai_first_actions", Title: "AI first actions", Value: info.AIFirstActions},
		{Key: "ai_limitations", Title: "AI limitations", Value: info.AILimitations},
		{Key: "ai_evidence_refs", Title: "AI evidence refs", Value: info.AIEvidenceRefs},
		{Key: "remediation_summary", Title: "Auto-remediation", Value: info.RemediationSummary},
		{Key: "remediation_approve_url", Title: "Approve", Value: info.RemediationApproveURL},
	}

	return nonEmptyIncidentFields(fields)
}

// IncidentInfoFieldsCompact returns the trimmed field set used by SOP-bound
// human notifications (slack/email/teams), whose message text is already the AI
// situation summary. It keeps only routing context plus a link to the SOP and
// drops the verbose AI/comms/debug fields (headline duplicates the title,
// customer update is already appended to the body, status/confidence/limitations
// /strategyId/source/binding are internal detail). Machine payloads keep the
// full set via IncidentInfoFields / IncidentInfoDetails.
func IncidentInfoFieldsCompact(info IncidentInfo) []IncidentField {
	info = SanitizeIncidentInfo(info)
	fields := []IncidentField{
		{Key: "project_id", Title: "Project", Value: info.ProjectID, Short: true},
		{Key: "environment", Title: "Environment", Value: info.Environment, Short: true},
		{Key: "service_name", Title: "Service", Value: info.ServiceName, Short: true},
		{Key: "owner_team", Title: "Owner team", Value: info.OwnerTeam, Short: true},
		{Key: "severity", Title: "Severity", Value: info.Severity, Short: true},
		{Key: "sop_title", Title: "SOP title", Value: info.SopTitle},
		{Key: "sop_url", Title: "SOP URL", Value: info.SopURL},
		{Key: "remediation_summary", Title: "Auto-remediation", Value: info.RemediationSummary},
		{Key: "remediation_approve_url", Title: "Approve", Value: info.RemediationApproveURL},
	}

	return nonEmptyIncidentFields(fields)
}

// nonEmptyIncidentFields drops fields whose value is blank, preserving order.
func nonEmptyIncidentFields(fields []IncidentField) []IncidentField {
	result := make([]IncidentField, 0, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(field.Value) == "" {
			continue
		}
		result = append(result, field)
	}

	return result
}

func IncidentInfoDetails(info IncidentInfo) map[string]string {
	fields := IncidentInfoFields(info)
	if len(fields) == 0 {
		return nil
	}

	details := make(map[string]string, len(fields))
	for _, field := range fields {
		details[field.Key] = field.Value
	}

	return details
}

func sanitizeIncidentURL(value string) (string, bool) {
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", false
	}

	parsed.User = nil
	query := parsed.Query()
	for key := range query {
		if incidentURLKeyLooksSensitive(key) {
			query.Del(key)
		}
	}
	parsed.RawQuery = query.Encode()

	return parsed.String(), true
}

func incidentURLKeyLooksSensitive(key string) bool {
	key = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return '_'
	}, strings.TrimSpace(key))
	_, ok := incidentSensitiveURLKeys[key]

	return ok
}

func incidentValueLooksSecret(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false
	}

	if strings.Contains(normalized, "bearer ") ||
		strings.Contains(normalized, "-----begin ") ||
		incidentJWTLikePattern.MatchString(value) {
		return true
	}
	for _, marker := range []string{
		"token=",
		"access_token",
		"client_secret",
		"api_key",
		"apikey",
		"password=",
		"secret=",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}

	return false
}
