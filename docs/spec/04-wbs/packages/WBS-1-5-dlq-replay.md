---
id: WBS-1.5
title: DLQ + Replay
parent: WBS-1
status: implemented-hmac-pending
covers_features: [F8]
source_paths:
  - pkg/alertmanager/alertmanagerserver/dispatcher.go
  - pkg/alertmanager/alertmanagerserver/dispatcher_dlq_test.go
acceptance: pending
estimated_effort: completed
commits: [ade174bb8, 91b9ff5db]
updated: 2026-05-29
open_items:
  - HMAC 정책 follow-up (NF-5.3.1)
---

# WBS-1.5 — DLQ + Replay

> **상태**: 구현 완료 (HMAC 정책 미해결)

## Deliverable
JSONL 기반 dead-letter store with rotation (`pkg/alertmanager/alertmanagernotify/dlq`), idempotent replay ledger, dispatcher와의 와이어업 (`dispatcher.go`). 채널 4xx/5xx/429 실패 시 incident envelope을 DLQ로 enqueue하고, 재시도 시 ledger를 통해 중복 dispatch를 차단해야 한다.

## Acceptance Criteria
- [ ] F8.8 acceptance Gherkin pass — 채널 5xx/429 시 DLQ enqueue 후 재시도 정책 적용 (UC-002)
- [ ] DLQ rotation 정책에 따른 파일 분할이 동작해야 한다
- [ ] replay 시 idempotency ledger가 동일 envelope의 중복 dispatch를 차단해야 한다
- [ ] DLQ enqueue/replay 이벤트는 WBS-1.0 audit sink에 기록되어야 한다
- [ ] **HMAC 정책 결정 후** envelope 서명/검증 acceptance 추가 (NF-5.3.1)

## Owner
TBD (TBC)

## Estimated Effort
완료 (커밋 `ade174bb8`, `91b9ff5db`)

## Dependencies
- WBS-1.0 Foundation (audit sink 재사용, JSONL 패턴 공유)
- WBS-1.3 Notification Dispatcher (실패 분기 수신)

## Verification
- `pkg/alertmanager/alertmanagerserver/dispatcher_dlq_test.go`
- `pkg/alertmanager/alertmanagernotify/dlq/dlq_test.go`
- `pkg/alertmanager/alertmanagernotify/dlq/ledger_test.go`
- `pkg/types/ruletypes/pilot_audit_sink_jsonl_test.go` (JSONL rotation 패턴 재사용)

## Covers Features
- F8 DLQ + Replay

## Source Paths
- `pkg/alertmanager/alertmanagerserver/dispatcher.go`
- `pkg/alertmanager/alertmanagerserver/dispatcher_dlq_test.go`
- `pkg/alertmanager/alertmanagernotify/dlq/` (DLQ store + ledger 구현)

## Open Items
- **HMAC 정책 follow-up** — replay 시 envelope 서명/검증 정책 미정 (NF-5.3.1)
- replay 트리거 운영 UI/CLI 노출 범위 미정 — 현재 dispatcher 내부 자동 재시도만 지원
