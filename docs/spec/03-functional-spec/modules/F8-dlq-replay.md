---
id: F8
title: DLQ + Idempotent Replay
status: planned
commits: [ade174bb8, 91b9ff5db]
source_paths:
  - pkg/alertmanager/alertmanagerserver/dispatcher.go
  - pkg/alertmanager/alertmanagerserver/dispatcher_dlq_test.go
  - pkg/alertmanager/alertmanagernotify/dlq/
  - pkg/types/ruletypes/pilot_audit_sink_jsonl.go
implements_uc: [UC-002]
covered_by_wbs: [WBS-1.5]
updated: 2026-06-02
open_items:
  - HMAC 정책 follow-up (replay payload 서명/검증 정책 미정 — NF-5.3.1)
---

# F8 — DLQ + Idempotent Replay

> **상태**: 착수 예정 (HMAC 정책 결정 필요)
> Terminal notify-stage 실패를 JSONL DLQ에 best-effort 영속화하고, replay 시 ledger로 중복 dispatch를 방지한다.

## 책임 (Responsibility)

두 컴포넌트로 구성된다. `JSONLDeadLetterSink`는 terminal notify failure를 JSONL로 append하고 50 MiB 임계치마다 rotate한다. `ReplayLedger`는 append-only `EventID` set으로, process restart 후에도 `MarkIfNew`의 idempotency를 보장한다. Dispatcher hot path에서 DLQ write는 best-effort — `Write` 실패 시 `WarnContext`만 남기고 알람은 계속 흐른다. **HMAC 정책 미결**: replay payload 무결성 보장 정책 미정 (NF-5.3.1).

## 인터페이스 요지

```go
// pkg/alertmanager/alertmanagernotify/dlq/dlq.go
type Sink interface { Write(e *Entry) error }
func NewJSONLDeadLetterSink(path string, rotateBytes int64) (*JSONLDeadLetterSink, error)

// pkg/alertmanager/alertmanagernotify/dlq/ledger.go
func NewReplayLedger(path string) (*ReplayLedger, error)
func (l *ReplayLedger) MarkIfNew(eventID string) bool  // true = 신규, false = 중복
```

DLQ entry: `{event_id, channel, payload (base64 alerts), failed_at, reason}`. Ledger: 1줄 = `EventID` 문자열. Scanner buffer 1 MiB per line. 상세는 `pkg/alertmanager/alertmanagernotify/dlq/` 참조.

## 핵심 동작

흐름: `notify.Stage.Exec` terminal error → `recordTerminalFailure` → `dlqSink.Write` → DLQ JSONL append.

Replay: operator 트리거 → ledger `MarkIfNew` 검사 → `true`면 notify retry → 성공(Delivered) 또는 실패 시 새 DLQ entry. context.Canceled는 DLQ에 쓰지 않는다.

현재 `EventID = alert.fingerprint` 단일 값. 동일 alert가 여러 채널로 갈 때 한 채널이 idempotent하면 다른 채널도 skip된다 — `(fingerprint, channel)` 튜플 확장이 follow-up.

## 예외·복구

| 경로 | 처리 |
|---|---|
| `dlqSink == nil` | DLQ 비활성. terminal failure는 log만. |
| `json.Marshal(alerts)` 실패 | empty payload로 entry 생성 + WarnContext |
| `dlqSink.Write` 실패 | WarnContext — dispatcher 계속 |
| Ledger write 실패 | `MarkIfNew=false` 반환 — 재전송 skip (안전 default) |
| Empty `EventID` | `MarkIfNew=false`, 저장 안 함 |
| ctx Canceled | DebugContext. DLQ write 없음. |

빈 파일은 절대 rotate하지 않는다. Sink + Ledger 모두 `sync.Mutex` 보호.

## Acceptance Criteria

```gherkin
Feature: Dead-letter queue and idempotent replay
  Background:
    Given a Dispatcher with dlqSink at "/tmp/dlq.jsonl"
    And a ReplayLedger at "/tmp/dlq.ledger"

  Scenario: Terminal failure produces one DLQ entry
    Given notify.Stage.Exec returns a non-canceled error for receiver "ops-slack"
    When recordTerminalFailure runs
    Then the DLQ file contains a JSON line with channel "ops-slack"
    And event_id equals the alert fingerprint

  Scenario: Replay is skipped for previously processed event
    Given a ReplayLedger containing event_id "abc123"
    When MarkIfNew("abc123") is invoked
    Then it returns false and no extra line is appended
```

## Traceability
- Implements UC: UC-002
- Covered by WBS: WBS-1.5
- Source: `pkg/alertmanager/alertmanagernotify/dlq/{dlq.go,ledger.go}`, `pkg/alertmanager/alertmanagerserver/dispatcher.go`
- Commits: `ade174bb8`, `91b9ff5db`
- Open: HMAC 정책 (NF-5.3.1), idempotency 키 `(fingerprint, channel)` 확장, replay CLI/UI
