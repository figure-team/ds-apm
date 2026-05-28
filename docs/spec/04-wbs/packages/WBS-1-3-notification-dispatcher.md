---
id: WBS-1.3
title: Notification Dispatcher
parent: WBS-1
status: implemented
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
estimated_effort: completed
commits: [5c036c806]
updated: 2026-05-29
---

# WBS-1.3 — Notification Dispatcher

> **상태**: 구현 완료

## Deliverable
채널 독립 dispatcher (`alertmanagerserver/dispatcher.go`), 5개 채널 adapter (Slack / MS Teams v2 / PagerDuty / Webhook / Email), 알람·SOP·AI draft를 통합한 템플릿 시스템 (`alertmanagertemplate`), 운영자용 템플릿 미리보기 (`notification_template_preview`). SOP/AI 컨텍스트가 포함된 incident 메시지를 채널별 페이로드로 정규화·전송해야 한다.

## Acceptance Criteria
- [ ] F6.7 acceptance Gherkin pass — SOP/AI draft 본문이 5개 채널에 채널별 포맷으로 전달되어야 한다
- [ ] 각 채널 mock provider에 대해 페이로드 스키마가 유효해야 한다
- [ ] 템플릿 미리보기 API는 실제 dispatch와 동일한 본문을 반환해야 한다
- [ ] 채널 4xx/5xx 실패 시 WBS-1.5 DLQ로 분기되어야 한다 (UC-002)
- [ ] dispatch 이벤트는 WBS-1.0 audit sink로 기록되어야 한다

## Owner
TBD (TBC)

## Estimated Effort
완료 (커밋 `5c036c806`)

## Dependencies
- WBS-1.0 Foundation (audit sink)
- WBS-1.1 SOP Engine (grounding 컨텍스트)
- WBS-1.2 AI Drafter (draft 본문)

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
