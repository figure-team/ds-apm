---
id: WBS-1.5
title: DLQ 재처리 서비스 (DLQ Replay Service)
parent: WBS-1
status: planned
covers_features: [F8]
source_paths:
  - pkg/alertmanager/alertmanagerserver/dispatcher.go
  - pkg/alertmanager/alertmanagerserver/dispatcher_dlq_test.go
acceptance: pending
estimated_effort: 3w
schedule:
  start: 2026-08-03
  end: 2026-08-21
  duration: 3w
commits: [ade174bb8, 91b9ff5db]
updated: 2026-06-02
open_items:
  - HMAC 정책 결정 (NF-5.3.1)
---

# WBS-1.5 — DLQ 재처리 서비스 (DLQ Replay Service)

> **상태**: 착수 예정 (HMAC 정책 결정 필요)
> **일정**: 2026-08-03 ~ 2026-08-21 (3주, WBS-1.3 완료 후)

## Deliverable
JSONL 기반 dead-letter store with rotation (`pkg/alertmanager/alertmanagernotify/dlq`), idempotent replay ledger, dispatcher와의 와이어업 (`dispatcher.go`). 채널 4xx/5xx/429 실패 시 incident envelope을 DLQ로 enqueue하고, 재시도 시 ledger를 통해 중복 dispatch를 차단해야 한다.

## Acceptance Criteria
- [ ] F8.8 acceptance Gherkin pass — 채널 5xx/429 시 DLQ enqueue 후 재시도 정책 적용 (UC-002)
- [ ] DLQ rotation 정책에 따른 파일 분할이 동작해야 한다
- [ ] replay 시 idempotency ledger가 동일 envelope의 중복 dispatch를 차단해야 한다
- [ ] DLQ enqueue/replay 이벤트는 WBS-1.0 audit sink에 기록되어야 한다
- [ ] **HMAC 정책 결정 후** envelope 서명/검증 acceptance 추가 (NF-5.3.1)

## Work Package 일정 (일 단위)

> 영업일(주5일) 기준, 공휴일 미반영. 의존성 순서: 인터페이스·타입 → 구현 → 통합·검증.

| WP ID | 작업명 | 선행 | 시작일 | 종료일 | 기간(영업일) |
|---|---|---|---|---|---|
| 1.5.1 | JSONL DLQ Sink | 1.3.6 | 2026-08-03 | 2026-08-05 | 3 |
| 1.5.2 | Idempotent Replay Ledger | 1.5.1 | 2026-08-06 | 2026-08-10 | 3 |
| 1.5.3 | Dispatcher 통합 | 1.5.2 | 2026-08-11 | 2026-08-13 | 3 |
| 1.5.4 | Replay API 엔드포인트 | 1.5.3 | 2026-08-14 | 2026-08-17 | 2 |
| 1.5.5 | Replay 상태 머신 | 1.5.4 | 2026-08-18 | 2026-08-19 | 2 |
| 1.5.6 | HMAC 정책 (scaffolding only) | 1.5.5 | 2026-08-20 | 2026-08-21 | 2 |

## Work Packages (Lv3)

### WBS-1.5.1 — JSONL DLQ Sink 구현 (JSONL Dead-Letter Sink)

- **Deliverable**: `JSONLDeadLetterSink` — atomic append + 50 MiB rotation, `Close()` 포함
- **Acceptance**: DLQ 파일에 Entry가 newline-delimited JSON으로 기록된다; 50 MiB 초과 시 타임스탬프 sibling으로 rotate되고 빈 파일은 rotate하지 않는다; `dlqSink.Write` 실패 시 dispatcher hot path가 중단되지 않는다
- **Source**: `pkg/alertmanager/alertmanagernotify/dlq/dlq.go`
- **일정**: 2026-08-03 ~ 2026-08-05 (3영업일, 선행: 1.3.6)
- **Effort**: TBD

### WBS-1.5.2 — Idempotent Replay Ledger (중복 방지 Ledger)

- **Deliverable**: `ReplayLedger` — append-only EventID set, open 시 in-memory set 재구성, `MarkIfNew` / `Has` / `Close`
- **Acceptance**: `MarkIfNew`가 동일 EventID 두 번째 호출에서 `false`를 반환한다; process restart 후 ledger 재구성 시 기존 seen set이 유지된다; 1 MiB scanner buffer로 pathological entry에서 silently truncate되지 않는다
- **Source**: `pkg/alertmanager/alertmanagernotify/dlq/ledger.go`
- **일정**: 2026-08-06 ~ 2026-08-10 (3영업일, 선행: 1.5.1)
- **Effort**: TBD

### WBS-1.5.3 — Dispatcher 통합 (Dispatcher DLQ Integration)

- **Deliverable**: `dispatcher.go` — channel 4xx/5xx/5xx-persistent terminal failure 시 DLQ enqueue, ctx.Canceled 제외
- **Acceptance**: `notify.Stage.Exec`가 non-canceled error 반환 시 DLQ entry가 생성된다; `context.Canceled`는 DLQ write를 호출하지 않는다; DLQ enqueue 이벤트가 WBS-1.0 audit sink에 기록된다
- **Source**: `pkg/alertmanager/alertmanagerserver/dispatcher.go`
- **일정**: 2026-08-11 ~ 2026-08-13 (3영업일, 선행: 1.5.2)
- **Effort**: TBD

### WBS-1.5.4 — Replay API 엔드포인트 (Manual Replay Trigger API)

- **Deliverable**: 운영자용 manual replay 트리거 HTTP endpoint — DLQ entry 목록 조회 + replay 시작
- **Acceptance**: endpoint 호출 시 ledger를 통해 중복 dispatch가 차단된다; replay 이벤트가 audit sink에 기록된다; 인증 없이 endpoint에 접근할 수 없다
- **Source**: `pkg/alertmanager/alertmanagerserver/` (신규 handler)
- **일정**: 2026-08-14 ~ 2026-08-17 (2영업일, 선행: 1.5.3)
- **Effort**: TBD

### WBS-1.5.5 — Replay 상태 머신 (Replay State Machine + Audit)

- **Deliverable**: pending → replayed / failed 상태 전이 관리, audit sink 통합
- **Acceptance**: F8.4 상태 다이어그램의 모든 전이(`ReplayPending → Skipped`, `ReplayPending → Redelivered`, `Redelivered → TerminalFail`)가 테스트로 검증된다; 상태 전이마다 audit entry가 기록된다
- **Source**: `pkg/alertmanager/alertmanagernotify/dlq/`
- **일정**: 2026-08-18 ~ 2026-08-19 (2영업일, 선행: 1.5.4)
- **Effort**: TBD

### WBS-1.5.6 — HMAC 정책 (Replay Payload Signing — NF-5.3.1)

> **Policy decision pending / scaffolding only** — 구현 전 HMAC 정책 합의 필요 (open item).

- **Deliverable**: replay payload 서명·검증을 위한 HMAC-SHA256 정책 합의 및 scaffolding (`pkg/alertmanager/alertmanagernotify/dlq/hmac.go`)
- **Acceptance**: HMAC-SHA256 + nonce + timestamp window 정책이 팀 내 합의된다; 합의된 정책 기준으로 `Entry` 서명/검증 인터페이스 scaffolding이 존재한다; NF-5.3.1 acceptance 항목이 F8.7 Gherkin에 추가된다
- **Source**: `pkg/alertmanager/alertmanagernotify/dlq/` (신규 파일)
- **일정**: 2026-08-20 ~ 2026-08-21 (2영업일, 선행: 1.5.5)
- **Effort**: TBD

## Owner
TBD (TBC)

## Estimated Effort
TBD

## Dependencies
- WBS-1.0 공통 기반 모듈 (audit sink 재사용, JSONL 패턴 공유)
- WBS-1.3 알림 디스패처 (실패 분기 수신)

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
