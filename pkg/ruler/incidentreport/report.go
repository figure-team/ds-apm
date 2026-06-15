// Package incidentreport composes a Korean-SI-style 장애보고서 (incident report)
// by aggregating artifacts already produced for an incident — the CF-2 AI
// response strategy and (when available) the CF-11 code-RCA finding — into one
// formatted document. It is an aggregation artifact: it creates little new
// content, instead pulling together response, root cause, and timeline.
//
// The report LAYOUT is a managed template, not hardcoded: Render executes a Go
// text/template over the aggregated IncidentReport data, so each org/customer
// can supply its own 양식. DefaultReportTemplate is used when none is set.
//
// MVP scope (specs/2026-06-15-incident-report-generation-design.md):
//   - sources: CF-2 strategy + CF-11 finding (optional)
//   - output: Markdown via template (PDF/HWP conversion is a later step)
//   - known gaps surfaced in the report: action-EXECUTION timeline is not
//     persisted (actions are AI proposals, marked for manual confirmation);
//     CF-11 run is correlated to the incident by service+time, not a hard key.
package incidentreport

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// ContractVersion identifies the report schema for audit/versioning.
const ContractVersion = "ds.incident_report.v1"

// TimelineEntry is one point on the incident timeline (RFC3339 At + 설명).
type TimelineEntry struct {
	At    string `json:"at"`
	Event string `json:"event"`
}

// IncidentReport is the composed report DATA. The generator fills it from the
// source artifacts; Render turns it into a document via a (managed) template.
type IncidentReport struct {
	ContractVersion  string `json:"contractVersion"`
	IncidentID       string `json:"incidentId"`
	AlertFingerprint string `json:"alertFingerprint,omitempty"`
	Service          string `json:"service,omitempty"`
	Severity         string `json:"severity,omitempty"`
	GeneratedAt      string `json:"generatedAt"` // report build time (RFC3339)

	Title  string `json:"title"`            // CF-2 headline
	Status string `json:"status,omitempty"` // incident/strategy status

	Confidence string `json:"confidence,omitempty"`

	RootCause  string   `json:"rootCause,omitempty"`  // CF-11 (직접/근본)
	Hypotheses []string `json:"hypotheses,omitempty"` // CF-2 가설

	ProposedActions []string `json:"proposedActions,omitempty"` // CF-2 firstActions (제안)
	ProposedFix     string   `json:"proposedFix,omitempty"`     // CF-11 재발방지 제안

	CustomerNotice string `json:"customerNotice,omitempty"` // CF-2 customerUpdateDraft

	Timeline []TimelineEntry `json:"timeline,omitempty"`

	Sources []string `json:"sources,omitempty"` // 기여한 산출물(CF-2/CF-11)
	Gaps    []string `json:"gaps,omitempty"`    // 결손/한계 명시
}

// reportFuncs are the helpers a report template may use.
var reportFuncs = template.FuncMap{
	// ph renders a placeholder when the value is blank, so empty sections stay
	// visible for the reviewer to fill in.
	"ph": func(s string) string {
		if strings.TrimSpace(s) == "" {
			return "확인 중"
		}
		return s
	},
	"inc":  func(i int) int { return i + 1 },
	"join": func(sep string, items []string) string { return strings.Join(items, sep) },
}

// DefaultReportTemplate is the built-in Korean-SI layout. Orgs may override it
// with their own template (same data, different 양식).
const DefaultReportTemplate = `# 장애보고서

> 본 보고서는 AI가 사고 대응(CF-2)·코드 근본원인(CF-11) 산출물을 집약한 **초안**입니다. 발송 전 담당자 검토·보완이 필요합니다.

## 1. 장애 개요
- 제목: {{ph .Title}}
- 인시던트 ID: {{ph .IncidentID}}
- 대상 서비스: {{ph .Service}}
- 심각도: {{ph .Severity}}
- 상태: {{ph .Status}}
- 신뢰도: {{ph .Confidence}}

## 2. 발생 경과 (타임라인)
{{- if .Timeline}}
{{- range .Timeline}}
- {{ph .At}} — {{.Event}}
{{- end}}
{{- else}}
- (확인 중) 발생/인지/복구 시각은 알람 상태이력 연동 시 자동 채워집니다.
{{- end}}

## 3. 원인 분석
{{- if .RootCause}}
### 근본 원인 (코드 RCA)
{{.RootCause}}
{{- end}}
{{- if .Hypotheses}}

### 가설 (1차 분석)
{{- range $i, $h := .Hypotheses}}
{{inc $i}}. {{$h}}
{{- end}}
{{- end}}
{{- if and (not .RootCause) (not .Hypotheses)}}
- 확인 중
{{- end}}

## 4. 조치 내역
> ⚠️ 아래는 AI가 제안한 조치이며, 실제 수행 여부·시각은 담당자가 수기 보완해야 합니다(조치 실행 이력 미연동).
{{- if .ProposedActions}}
{{- range .ProposedActions}}
- [ ] {{.}}
{{- end}}
{{- else}}
- 확인 중
{{- end}}

## 5. 재발 방지 대책
{{- if .ProposedFix}}
{{.ProposedFix}}
{{- else}}
- 확인 중
{{- end}}

## 6. 고객 영향 및 공지
{{- if .CustomerNotice}}
{{.CustomerNotice}}
{{- else}}
- 해당 없음 / 확인 중
{{- end}}

## 7. 메타
- 생성 시각: {{ph .GeneratedAt}}
- 보고서 계약버전: {{ph .ContractVersion}}
{{- if .Sources}}
- 집약 출처: {{join ", " .Sources}}
{{- end}}
{{- if .Gaps}}
- 한계/결손:
{{- range .Gaps}}
  - {{.}}
{{- end}}
{{- end}}
`

// Render executes tmplText (a Go text/template) over the report. When tmplText
// is blank, DefaultReportTemplate is used. A malformed template returns an error
// so a bad org template is surfaced rather than producing a broken document.
func (r IncidentReport) Render(tmplText string) (string, error) {
	if strings.TrimSpace(tmplText) == "" {
		tmplText = DefaultReportTemplate
	}
	tmpl, err := template.New("incident_report").Funcs(reportFuncs).Parse(tmplText)
	if err != nil {
		return "", fmt.Errorf("incidentreport: parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, r); err != nil {
		return "", fmt.Errorf("incidentreport: execute template: %w", err)
	}
	return buf.String(), nil
}

// RenderMarkdown renders with the default template. Convenience for callers that
// do not manage a custom template; the default is known-good so errors are not
// expected, but any are returned as the document text for visibility.
func (r IncidentReport) RenderMarkdown() string {
	out, err := r.Render("")
	if err != nil {
		return "보고서 렌더 실패: " + err.Error()
	}
	return out
}
