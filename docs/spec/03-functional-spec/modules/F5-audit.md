---
id: F5
title: Audit (SOP access & dispatch)
status: planned
commits: [8a55208ef]
source_paths:
  - pkg/types/ruletypes/pilot_audit_sink.go
  - pkg/types/ruletypes/pilot_audit_sink_jsonl.go
implements_uc: [UC-001, UC-002, UC-003]
covered_by_wbs: [WBS-1.0]
updated: 2026-06-02
---

# F5 — Audit (SOP access & dispatch)

> **상태**: 착수 예정 (착수보고 기준)
> SOP 검색·preview·fetch, evidence 수집, AI summary 요청/결과를 JSONL로 영속 기록한다. 정책은 best-effort — sink 실패는 원 operation을 막지 않는다.

## 책임 (Responsibility)

`PilotAuditEventSink` 추상화와 두 구현(`NopPilotAuditEventSink` default, `PilotAuditEventJSONLSink` 운영용)을 제공한다. 운영 sink는 newline-delimited JSON을 로컬 파일에 append하고, 50 MiB 임계치마다 timestamped sibling으로 rotate한다. 이벤트 validation 실패는 dropped(caller block 없음) — 호출 컨벤션은 best-effort.

## 인터페이스 요지

```go
// pkg/types/ruletypes/pilot_audit_sink.go
type PilotAuditEventSink interface {
    Record(ctx context.Context, event PilotAuditEvent) error
}
func RegisterPilotAuditEventSink(sink PilotAuditEventSink)
func DispatchPilotAuditEvent(ctx context.Context, event PilotAuditEvent) error

// pkg/types/ruletypes/pilot_audit_sink_jsonl.go
func NewPilotAuditEventJSONLSink(path string, maxSizeBytes int64) (*PilotAuditEventJSONLSink, error)
const DefaultPilotAuditJSONLMaxSizeBytes int64 = 50 * 1024 * 1024
```

이벤트 타입 8종: `sop.search | sop.preview | sop.fetch | sop.health_check | evidence.collect_request | evidence.collect_result | ai.summary_request | ai.summary_result`. Outcome 5종: `allowed | denied | redacted | failed | deferred`. 상세 구조체는 `pilot_audit_sink.go` 참조.

## 핵심 동작

입력: `PilotAuditEvent` (contractVersion, eventType, outcome, actor, tenant, resource, securityContext 필수).

처리: `ValidatePilotAuditEvent` 통과 → `json.Marshal` → rotation 검사 → `mu.Lock` 후 append.

출력: JSONL 파일 1줄 (`json line + '\n'`). Rotation 시 `<path>.<RFC3339 timestamp>` sibling 생성.

Security invariants (validator 강제): `secretRefVisible=false`, `browserCredentialsUsed=false`, `serviceAccountProfile` non-empty. `sop.*` 이벤트는 `resource.sourceId` non-empty 필수.

## 예외·복구

| 경로 | 처리 |
|---|---|
| Validation 실패 | `zap.Warn` 후 `nil` 반환. caller 차단 없음. |
| Write 실패 (디스크 full 등) | error 전파. caller가 best-effort로 무시 가능. |
| Rotation `os.Rename` 실패 | error 전파. 파일 원본 유지. |
| Sink 미등록 | `NopPilotAuditEventSink` (no-op, 항상 `nil`). |

빈 파일은 절대 rotate하지 않는다.

## Acceptance Criteria

```gherkin
Feature: Pilot audit event sink

  Scenario: Valid event is appended as one JSONL line
    Given a valid PilotAuditEvent with EventType "sop.fetch" and Outcome "allowed"
    When Record is called
    Then the file contains a single JSON object followed by newline

  Scenario: Invalid event is silently dropped
    Given a PilotAuditEvent with empty EventID
    When Record is called
    Then no error is returned and the file size remains unchanged
```

## Traceability
- Implements UC: UC-001 (단계 10), UC-002, UC-003
- Covered by WBS: WBS-1.0
- Source: `pkg/types/ruletypes/pilot_audit_sink.go`, `pkg/types/ruletypes/pilot_audit_sink_jsonl.go`
- Commits: `8a55208ef`
