---
id: WBS-1.3
title: 알림 디스패처 (Notification Dispatcher)
parent: WBS-1
status: planned
covers_features: [F6]
source_paths:
  - pkg/alertmanager/alertmanagernotify/slack/
  - pkg/alertmanager/alertmanagernotify/msteamsv2/
  - pkg/alertmanager/alertmanagernotify/pagerduty/
  - pkg/alertmanager/alertmanagernotify/webhook/
  - pkg/alertmanager/alertmanagernotify/email/
  - pkg/alertmanager/alertmanagerserver/dispatcher.go
  - pkg/alertmanager/alertmanagertemplate/
  - pkg/types/ruletypes/notification_template_preview.go
acceptance: pending
estimated_effort: 3w
schedule:
  start: 2026-07-13
  end: 2026-07-31
  duration: 3w
commits: [5c036c806]
updated: 2026-06-02
---

# WBS-1.3 — 알림 디스패처 (Notification Dispatcher)

> **상태**: 착수 예정 (착수보고 기준)
> **일정**: 2026-07-13 ~ 2026-07-31 (3주, WBS-1.1·1.2 완료 후)

## Deliverable
채널 독립 dispatcher (`alertmanagerserver/dispatcher.go`), 5개 채널 adapter (Slack / MS Teams v2 / PagerDuty / Webhook / Email), 알람·SOP·AI draft를 통합한 템플릿 시스템 (`alertmanagertemplate`), 운영자용 템플릿 미리보기 (`notification_template_preview`). SOP/AI 컨텍스트가 포함된 incident 메시지를 채널별 페이로드로 정규화·전송해야 한다.

## Acceptance Criteria
- [ ] F6.7 acceptance Gherkin pass — SOP/AI draft 본문이 5개 채널에 채널별 포맷으로 전달되어야 한다
- [ ] 각 채널 mock provider에 대해 페이로드 스키마가 유효해야 한다
- [ ] 템플릿 미리보기 API는 실제 dispatch와 동일한 본문을 반환해야 한다
- [ ] 채널 4xx/5xx 실패 시 WBS-1.5 DLQ로 분기되어야 한다 (UC-002)
- [ ] dispatch 이벤트는 WBS-1.0 audit sink로 기록되어야 한다

## Work Packages (Lv3)

### WBS-1.3.1 — Dispatcher wrapping (`dispatch.Dispatcher`)

- **Deliverable**: `alertmanagerserver/dispatcher.go` — `dlqSink` + `aiHook` 필드를 갖는 `Dispatcher` struct 및 `NewDispatcher` 생성자. `aggrGroup.run()` flush 경로에 `applyAIHook` 진입점 삽입. DLQ terminal-failure wire 포함.
- **Acceptance**: `NewDispatcher`에 `aiHook == nil` 전달 시 hook skip; terminal error 발생 시 `dlqSink.Write` 호출됨; dispatcher goroutine이 context cancel에 정상 종료됨 (F6.7 Gherkin 3개 scenario pass)
- **Source**: `pkg/alertmanager/alertmanagerserver/dispatcher.go`
- **Effort**: TBD

### WBS-1.3.2 — AI context propagation 로직

- **Deliverable**: `dispatchhook.Hook.Apply()` 호출 + annotations 머지 로직. 입력 annotations 불변(새 map 반환). `knownIncidentTemplateFields` 22종 정의 및 `MissingIncidentTemplateVariables` 유효성 검사.
- **Acceptance**: `Apply()` 반환 map에 `ai_headline` 포함; 원본 annotations 불변 확인; unknown variable `$incident.foo_bar` → `MissingIncidentTemplateVariables` 결과에 `incident_foo_bar` 포함 (F6.7 Gherkin scenario 4 pass)
- **Source**: `pkg/alertmanager/alertmanagerserver/dispatcher.go`, `pkg/types/ruletypes/notification_template_preview.go`
- **Effort**: TBD

### WBS-1.3.3 — Slack + MS Teams v2 adapter

- **Deliverable**: Slack Block Kit 페이로드 빌더 (`severity` → `[SEV-x]` prefix, `sop_url` → actions button, `ai_headline`/`ai_first_actions` → section mrkdwn). MS Teams v2 Adaptive Card 빌더 (`Action.OpenUrl`만 사용, `Action.Submit` 제외).
- **Acceptance**: 두 채널 모두 mock provider에 대해 페이로드 스키마 유효; `ai_headline` 본문 포함; MS Teams `Action.Submit` 미사용 확인
- **Source**: `pkg/alertmanager/alertmanagernotify/slack/`, `pkg/alertmanager/alertmanagernotify/msteamsv2/`
- **Effort**: TBD

### WBS-1.3.4 — PagerDuty adapter

- **Deliverable**: PagerDuty Events API v2 페이로드 빌더. `severity` → `payload.severity`, `service_name` → `payload.component`, `ai_headline` → `payload.summary`, `sop_url`/`ai_first_actions`/`ai_confidence` → `payload.custom_details`.
- **Acceptance**: mock provider에 페이로드 스키마 유효; severity 매핑 4단계(`critical`/`error`/`warning`/`info`) 정확; `custom_details.runbook_url` 필드 포함
- **Source**: `pkg/alertmanager/alertmanagernotify/pagerduty/`
- **Effort**: TBD

### WBS-1.3.5 — Webhook + Email adapter

- **Deliverable**: 일반 JSON Webhook 빌더 (`severity`, `service`, `sop_url`, `ai_headline`, `ai_first_actions` array 포함). SMTP Email 빌더 (subject prefix `[SEV-x]`, body에 `ai_headline` + bullet list `ai_first_actions`, inline `sop_url`).
- **Acceptance**: 두 채널 모두 mock provider에 페이로드/MIME 스키마 유효; Email subject에 severity prefix 포함; Webhook `ai_first_actions` 필드가 array 타입
- **Source**: `pkg/alertmanager/alertmanagernotify/webhook/`, `pkg/alertmanager/alertmanagernotify/email/`
- **Effort**: TBD

### WBS-1.3.6 — 5채널 통합 라우팅·전송 검증 (Integration & Fault Tolerance)

- **Deliverable**: 5채널 fan-out 통합 테스트 스위트. 단일 alert에서 5채널 동시 dispatch, partial failure(1개 이상 채널 4xx/5xx) 시 성공 채널 결과 보존 + 실패 채널 DLQ 분기 검증. `recordTerminalFailure` → `dlqSink.Write` 경로 E2E 커버 포함.
- **Acceptance**: 5채널 전체 성공 시나리오 pass; 채널 1개 terminal error 시 나머지 4채널 전달 완료 + DLQ entry 1건 생성 확인; context cancel 시 DLQ 기록 없이 graceful 종료 확인 (F6.7 Gherkin 전 scenario pass)
- **Source**: `pkg/alertmanager/alertmanagerserver/dispatcher.go`, `pkg/alertmanager/alertmanagernotify/{slack,msteamsv2,pagerduty,webhook,email}/`
- **Effort**: TBD

## Owner
TBD (TBC)

## Estimated Effort
TBD

## Dependencies
- WBS-1.0 공통 기반 모듈 (audit sink)
- WBS-1.1 SOP 그라운딩 서비스 (grounding 컨텍스트)
- WBS-1.2 AI 초안 매니저 (draft 본문)

## Verification
- `pkg/alertmanager/alertmanagernotify/slack/slack_test.go`
- `pkg/alertmanager/alertmanagernotify/msteamsv2/msteamsv2_test.go`
- `pkg/alertmanager/alertmanagernotify/pagerduty/pagerduty_test.go`
- `pkg/alertmanager/alertmanagernotify/webhook/webhook_test.go`
- `pkg/alertmanager/alertmanagernotify/email/email_test.go`
- `pkg/alertmanager/alertmanagertemplate/alertmanagertemplate_test.go`
- `pkg/types/ruletypes/notification_template_preview_test.go`
- `pkg/types/alertmanagertypes/template_test.go`

## Covers Features
- F6 Notification Dispatch

## Source Paths
- `pkg/alertmanager/alertmanagernotify/{slack,msteamsv2,pagerduty,webhook,email}/`
- `pkg/alertmanager/alertmanagerserver/dispatcher.go`
- `pkg/alertmanager/alertmanagertemplate/`
- `pkg/types/ruletypes/notification_template_preview.go`
