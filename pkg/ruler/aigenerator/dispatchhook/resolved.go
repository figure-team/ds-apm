package dispatchhook

import (
	"fmt"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// kstZone mirrors alertmanagertypes.FormatKST's fixed +9 offset so the short
// clock form ("15:04") lands in the same timezone as the full stamp.
var kstZone = time.FixedZone("KST", 9*60*60)

// resolvedExecutionRank orders executions for the resolved notice: the most
// conclusive outcome wins. Statuses that never ran a script (proposed,
// rejected, expired) are excluded entirely — they are not "조치".
var resolvedExecutionRank = map[string]int{
	ruletypes.RemediationStatusVerified:   5,
	ruletypes.RemediationStatusSucceeded:  4,
	ruletypes.RemediationStatusUnresolved: 3,
	ruletypes.RemediationStatusFailed:     2,
	ruletypes.RemediationStatusExecuting:  1,
}

// pickResolvedExecution selects the execution the resolved notice reports:
// highest outcome rank first, then most recent ExecutedAt (RFC3339 strings
// compare chronologically). Returns nil when nothing actually ran.
//
// notBefore guards against stale attribution: the store keys executions by
// alert fingerprint (the hook's incidentID), which is IDENTICAL across every
// recurrence of the same failure — without the cutoff, a runbook approved for
// last week's occurrence would be reported as this incident's 조치. Executions
// stamped before the current incident's StartsAt are therefore excluded, as
// are entries whose timestamps don't parse (unknown age must not be
// attributed). A zero notBefore disables the cutoff.
func pickResolvedExecution(execs []ruletypes.RemediationExecution, notBefore time.Time) *ruletypes.RemediationExecution {
	var best *ruletypes.RemediationExecution
	bestRank := 0
	for i := range execs {
		rank, ran := resolvedExecutionRank[execs[i].Status]
		if !ran {
			continue
		}
		if !notBefore.IsZero() {
			ts := execs[i].ExecutedAt
			if strings.TrimSpace(ts) == "" {
				ts = execs[i].ProposedAt
			}
			at, err := time.Parse(time.RFC3339, strings.TrimSpace(ts))
			if err != nil || at.Before(notBefore) {
				continue
			}
		}
		if best == nil || rank > bestRank ||
			(rank == bestRank && execs[i].ExecutedAt > best.ExecutedAt) {
			best = &execs[i]
			bestRank = rank
		}
	}
	return best
}

// remediationOutcomeKR renders the execution's terminal state as the operator
// should read it. Only facts the system recorded — no inference about what
// actually cleared the alert.
func remediationOutcomeKR(exec ruletypes.RemediationExecution) string {
	switch exec.Status {
	case ruletypes.RemediationStatusVerified:
		return "성공, 지표 정상화 검증 완료"
	case ruletypes.RemediationStatusSucceeded:
		return "성공(지표 검증 대기)"
	case ruletypes.RemediationStatusUnresolved:
		return "실행 완료, 자동 검증 기준 지표 미회복"
	case ruletypes.RemediationStatusFailed:
		if exec.ExitCode != nil {
			return fmt.Sprintf("실행 실패(exit %d)", *exec.ExitCode)
		}
		return "실행 실패"
	case ruletypes.RemediationStatusExecuting:
		return "실행 중"
	}
	return exec.Status
}

// formatDurationKR renders an incident duration for the resolved headline.
func formatDurationKR(d time.Duration) string {
	if d < time.Minute {
		return "1분 미만"
	}
	mins := int(d.Round(time.Minute) / time.Minute)
	if mins < 60 {
		return fmt.Sprintf("%d분", mins)
	}
	if mins%60 == 0 {
		return fmt.Sprintf("%d시간", mins/60)
	}
	return fmt.Sprintf("%d시간 %d분", mins/60, mins%60)
}

// kstClock renders an RFC3339 timestamp as "15:04" in KST; empty on parse
// failure so a malformed stored value degrades to no timestamp, not garbage.
func kstClock(rfc3339 string) string {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(rfc3339))
	if err != nil {
		return ""
	}
	return t.In(kstZone).Format("15:04")
}

// buildResolvedBody assembles the deterministic resolved-notification body:
// which alert fired, how long it lasted, and — when the system actually ran
// something — which runbook with what recorded outcome. exec == nil states
// plainly that no automated action is on record.
func buildResolvedBody(alertname, severity string, startsAt, endsAt time.Time, exec *ruletypes.RemediationExecution, runbookTitle string) string {
	var b strings.Builder

	subject := strings.TrimSpace(alertname)
	if subject == "" {
		subject = "알림"
	}
	if sev := strings.TrimSpace(severity); sev != "" {
		subject += "(" + sev + ")"
	}

	if !startsAt.IsZero() && !endsAt.IsZero() && endsAt.After(startsAt) {
		fmt.Fprintf(&b, "%s가 %s 발생 후 %s 만에 해소되었습니다.",
			subject, alertmanagertypes.FormatKST(startsAt), formatDurationKR(endsAt.Sub(startsAt)))
	} else {
		fmt.Fprintf(&b, "%s가 해소되었습니다.", subject)
	}

	b.WriteString("\n\n*조치:* ")
	if exec == nil {
		b.WriteString("자동대응 실행 이력이 없습니다. 수동 조치로 해소되었습니다(상세 미기록).")
		return b.String()
	}

	title := strings.TrimSpace(runbookTitle)
	if title == "" {
		title = "런북"
	}
	fmt.Fprintf(&b, "자동대응 \"%s\" 실행", title)

	var meta []string
	if clock := kstClock(exec.ExecutedAt); clock != "" {
		meta = append(meta, clock)
	}
	if by := strings.TrimSpace(exec.ApprovedBy); by != "" {
		meta = append(meta, by+" 승인")
	}
	if len(meta) > 0 {
		fmt.Fprintf(&b, "(%s)", strings.Join(meta, ", "))
	}
	fmt.Fprintf(&b, " → %s.", remediationOutcomeKR(*exec))
	return b.String()
}

// buildResolvedNotice renders the customer-facing all-clear for the resolved
// notification's collapsible section.
func buildResolvedNotice(sopTitle string) string {
	if t := strings.TrimSpace(sopTitle); t != "" {
		return fmt.Sprintf("[안내] '%s' 관련 증상이 정상화되었습니다.", t)
	}
	return "[안내] 서비스 이상 증상이 정상화되었습니다."
}
