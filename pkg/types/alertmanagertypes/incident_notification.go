package alertmanagertypes

import (
	"strings"

	"github.com/prometheus/alertmanager/template"
)

// SOPBoundNotification은 SOP 바인딩 알림의 AI 생성 제목/본문/고객공지를 담는다.
type SOPBoundNotification struct {
	Title          string // sop_title → 없으면 ai_headline → 없으면 ""
	Body           string // notification_body (메인 본문)
	CustomerNotice string // customer_update (자세히보기 섹션, 비어 있을 수 있음)
}

// ResolveSOPBoundNotification은 annotation에 AI 메인 본문(notification_body)이
// 있으면 제목/본문/고객공지를 구성해 (notif, true)를, 없으면 (_, false)를 반환한다.
// 순수 함수 — I/O·로깅·에러 없음. 모든 notifier가 공유한다.
func ResolveSOPBoundNotification(annotations template.KV) (SOPBoundNotification, bool) {
	body := strings.TrimSpace(annotations[IncidentAnnotationNotificationBody])
	if body == "" {
		return SOPBoundNotification{}, false
	}
	title := strings.TrimSpace(annotations[IncidentAnnotationSopTitle])
	if title == "" {
		title = strings.TrimSpace(annotations[IncidentAnnotationAIHeadline])
	}
	return SOPBoundNotification{
		Title:          title,
		Body:           body,
		CustomerNotice: strings.TrimSpace(annotations[IncidentAnnotationCustomerUpdate]),
	}, true
}

// CollapsibleNoticeLabel은 collapsible 미지원 채널에서 고객 공지 앞에 붙이는 라벨.
const CollapsibleNoticeLabel = "▼ 고객 공지 (자세히보기)"
