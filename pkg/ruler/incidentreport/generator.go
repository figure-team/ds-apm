package incidentreport

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// AIStrategyReader reads the latest CF-2 response strategy for an incident.
// Satisfied by ruletypes.AIStrategyHistoryStore.
type AIStrategyReader interface {
	GetLatest(ctx context.Context, orgID string, lookup ruletypes.AIStrategyHistoryLookupRequest) (ruletypes.AIStrategyHistoryRecord, bool, error)
}

// CodeRCAFinding is the slice of a CF-11 code-RCA run the report needs. It is a
// local DTO so the report package does not depend on the coderca run store; an
// adapter at the wiring seam maps a run record into this shape.
type CodeRCAFinding struct {
	RunID          string
	RootCause      string
	ProposedFix    string
	Confidence     string
	Limitations    string
	BaselineCommit string
	CreatedAt      int64 // unix seconds, 0 = unknown
	FinishedAt     int64 // unix seconds, 0 = not finished
}

// CodeRCAReader returns the most recent usable code-RCA finding for a service.
// MVP correlation is by service (+time), not a hard incident key — see Gaps.
type CodeRCAReader interface {
	LatestFinding(ctx context.Context, orgID, service string) (CodeRCAFinding, bool, error)
}

// Params drives one report build.
type Params struct {
	OrgID            string
	IncidentID       string
	AlertFingerprint string
	Service          string
	Severity         string
	// Now is the report build timestamp (RFC3339), injected by the caller so
	// Build stays deterministic and testable.
	Now string
}

// Generator composes an IncidentReport from the available artifacts. codeRCA is
// optional: when nil (or it returns no finding) the report omits the code-RCA
// root cause and notes the gap.
type Generator struct {
	strategies AIStrategyReader
	codeRCA    CodeRCAReader
}

// NewGenerator builds a Generator. strategies is required; codeRCA may be nil.
func NewGenerator(strategies AIStrategyReader, codeRCA CodeRCAReader) *Generator {
	return &Generator{strategies: strategies, codeRCA: codeRCA}
}

// Build aggregates the incident's artifacts into a render-ready report. Missing
// sources degrade gracefully (the section shows "확인 중" and a gap is recorded)
// rather than failing the whole report.
func (g *Generator) Build(ctx context.Context, p Params) (IncidentReport, error) {
	report := IncidentReport{
		ContractVersion:  ContractVersion,
		IncidentID:       strings.TrimSpace(p.IncidentID),
		AlertFingerprint: strings.TrimSpace(p.AlertFingerprint),
		Service:          strings.TrimSpace(p.Service),
		Severity:         strings.TrimSpace(p.Severity),
		GeneratedAt:      strings.TrimSpace(p.Now),
	}

	// CF-2: response strategy (headline, hypotheses, actions, customer notice).
	if g.strategies != nil {
		rec, ok, err := g.strategies.GetLatest(ctx, p.OrgID, ruletypes.AIStrategyHistoryLookupRequest{
			IncidentID:       p.IncidentID,
			AlertFingerprint: p.AlertFingerprint,
		})
		if err != nil {
			return IncidentReport{}, fmt.Errorf("incidentreport: read AI strategy: %w", err)
		}
		if ok {
			s := rec.Strategy
			report.Title = s.Headline
			report.Status = firstNonEmpty(rec.Status, s.Status)
			report.Confidence = firstNonEmpty(rec.Confidence, s.Confidence)
			report.Hypotheses = hypothesisTexts(s.Hypotheses)
			report.ProposedActions = firstActionTexts(s.FirstActions)
			report.CustomerNotice = s.CustomerUpdateDraft
			if at := strings.TrimSpace(rec.GeneratedAt); at != "" {
				report.Timeline = append(report.Timeline, TimelineEntry{At: at, Event: "AI 1차 분석 초안 생성 (CF-2)"})
			}
			report.Sources = append(report.Sources, "CF-2 AI 대응 전략")
		} else {
			report.Gaps = append(report.Gaps, "CF-2 AI 전략 이력을 찾지 못함 — 개요·고객공지·가설이 비어 있음")
		}
	}

	// CF-11: code-RCA root cause + proposed fix (optional).
	if g.codeRCA != nil && report.Service != "" {
		f, ok, err := g.codeRCA.LatestFinding(ctx, p.OrgID, report.Service)
		if err != nil {
			return IncidentReport{}, fmt.Errorf("incidentreport: read code RCA: %w", err)
		}
		if ok && strings.TrimSpace(f.RootCause) != "" {
			report.RootCause = f.RootCause
			report.ProposedFix = f.ProposedFix
			if report.Confidence == "" {
				report.Confidence = f.Confidence
			}
			if f.CreatedAt > 0 {
				report.Timeline = append(report.Timeline, TimelineEntry{At: unixToRFC3339(f.CreatedAt), Event: "코드 RCA 분석 시작 (CF-11)"})
			}
			if f.FinishedAt > 0 {
				report.Timeline = append(report.Timeline, TimelineEntry{At: unixToRFC3339(f.FinishedAt), Event: "코드 RCA 근본원인 도출 (CF-11)"})
			}
			report.Sources = append(report.Sources, "CF-11 코드 RCA (run "+f.RunID+")")
			report.Gaps = append(report.Gaps, "CF-11 결과는 service+시간 기준 근사 매칭 — incident 직접 연결 키 부재")
		}
	}

	// Standing limitations of the v1 aggregation.
	report.Gaps = append(report.Gaps, "조치 실행 이력 미연동 — '조치 내역'은 AI 제안 기반이며 실제 수행/시각은 수기 보완 필요")
	report.Timeline = sortTimeline(report.Timeline)

	return report, nil
}

func hypothesisTexts(hs []ruletypes.AIHypothesis) []string {
	out := make([]string, 0, len(hs))
	for _, h := range hs {
		if t := strings.TrimSpace(h.Text); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func firstActionTexts(as []ruletypes.AIFirstAction) []string {
	out := make([]string, 0, len(as))
	for _, a := range as {
		if t := strings.TrimSpace(a.Text); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func unixToRFC3339(sec int64) string {
	return time.Unix(sec, 0).UTC().Format(time.RFC3339)
}

// sortTimeline orders entries by their RFC3339 At ascending (string compare is
// chronological for RFC3339). Entries with empty At sink to the end.
func sortTimeline(entries []TimelineEntry) []TimelineEntry {
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0; j-- {
			if lessAt(entries[j].At, entries[j-1].At) {
				entries[j], entries[j-1] = entries[j-1], entries[j]
			} else {
				break
			}
		}
	}
	return entries
}

func lessAt(a, b string) bool {
	if a == "" {
		return false
	}
	if b == "" {
		return true
	}
	return a < b
}
