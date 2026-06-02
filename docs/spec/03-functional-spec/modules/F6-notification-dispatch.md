---
id: F6
title: Notification Dispatch (5 채널)
status: planned
commits: [5c036c806]
source_paths:
  - pkg/alertmanager/alertmanagernotify/slack/
  - pkg/alertmanager/alertmanagernotify/msteamsv2/
  - pkg/alertmanager/alertmanagernotify/pagerduty/
  - pkg/alertmanager/alertmanagernotify/webhook/
  - pkg/alertmanager/alertmanagernotify/email/
  - pkg/alertmanager/alertmanagerserver/dispatcher.go
  - pkg/alertmanager/alertmanagertemplate/
  - pkg/types/ruletypes/notification_template_preview.go
implements_uc: [UC-001, UC-002]
covered_by_wbs: [WBS-1.3]
updated: 2026-06-02
---

# F6 — Notification Dispatch (5 채널)

> **상태**: 착수 예정 (착수보고 기준)
> SOP / AI strategy annotation을 5개 채널(Slack, MS Teams v2, PagerDuty, Webhook, Email)로 dispatch한다. dispatcher hot path에서 AI hook을 호출하고, 실패 시 DLQ로 분기 (F8).

## 책임 (Responsibility)

`dispatch.Dispatcher`를 wrapping하여 두 가지를 끼워넣는다: alert flush 시점에 `dispatchhook.Hook.Apply`를 호출(SOP grounding → AI strategy → annotations 머지)하고, terminal notify failure를 `dlq.Sink`에 best-effort write한다. 5채널 adapter는 SigNoz upstream path를 패치하며 `$incident.{key}` 22종 template variable을 지원한다. MS Teams v2 제약: `Action.Submit` 미지원, `Action.OpenUrl`만 사용 (research §7.2).

## 인터페이스 요지

```go
// pkg/alertmanager/alertmanagerserver/dispatcher.go
func NewDispatcher(ap provider.Alerts, r *dispatch.Route, s notify.Stage,
    mk types.GroupMarker, to func(time.Duration) time.Duration,
    lim Limits, l *slog.Logger, m *DispatcherMetrics,
    n nfmanager.NotificationManager, orgID string,
    dlqSink dlq.Sink, aiHook *dispatchhook.Hook) *Dispatcher

// pkg/types/ruletypes/notification_template_preview.go
func PreviewNotificationTemplate(ctx context.Context, req PreviewNotificationTemplateRequest) (*PreviewNotificationTemplateResponse, error)
func MissingIncidentTemplateVariables(template string) []string
```

Template variable `$incident.{key}` 22종 — Tenant(5) · Impact(4) · SOP(6) · AI(7). 상세는 `notification_template_preview.go` 참조.

## 핵심 동작

흐름: alert 수신 → route 매칭 → aggregation group → timer flush → `aiHook.Apply` → `notify.Stage.Exec` → 성공(Delivered) 또는 terminal failure(DLQ).

`aiHook == nil`이면 hook 단계를 skip한다 (AIOpsAgent 미설치 시 정상 동작). context.Canceled는 graceful shutdown으로 처리하며 DLQ에 쓰지 않는다. Maintenance ticker가 30초 주기로 empty aggregation group을 GC한다.

## 예외·복구

| 경로 | 처리 |
|---|---|
| `aiHook.Apply` 내부 실패 | annotations 그대로 — F3 참조 |
| `notify.Stage.Exec` terminal error | `recordTerminalFailure` → DLQ enqueue (F8) |
| DLQ marshal 실패 | empty payload로 entry 생성 + WarnContext |
| ctx Canceled | DebugContext. DLQ write 안 함. |
| Aggregation group 한도 초과 | metric increment + ErrorContext. 신규 group 차단. |

채널 dispatch 실패는 dispatcher 자체를 중단시키지 않는다.

## Acceptance Criteria

```gherkin
Feature: Notification dispatch with AI hook
  Background:
    Given a Dispatcher with non-nil aiHook and dlqSink
    And the notificationManager matches the alert to receiver "ops-slack"

  Scenario: Successful dispatch merges AI annotations
    Given the AI hook returns annotations containing "ai_headline"
    When the aggrGroup flushes
    Then notify.Stage.Exec receives an alert with ai_headline annotation

  Scenario: Terminal failure is persisted to DLQ
    Given notify.Stage.Exec returns a non-canceled error
    When the aggrGroup flushes
    Then dlqSink.Write is called with channel "ops-slack" and event_id equal to the alert fingerprint
```

## Traceability
- Implements UC: UC-001 (단계 8), UC-002 (실패 분기)
- Covered by WBS: WBS-1.3
- Source: `pkg/alertmanager/alertmanagernotify/{slack,msteamsv2,pagerduty,webhook,email}/`, `pkg/alertmanager/alertmanagerserver/dispatcher.go`, `pkg/types/ruletypes/notification_template_preview.go`
- Commits: `5c036c806`
