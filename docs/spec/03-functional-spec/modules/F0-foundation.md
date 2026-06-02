---
id: F0
title: 공통 기반 모듈 (Foundation Core) / Pilot Scaffolding
status: planned
commits: [026863650]
source_paths:
  - cmd/community/
  - pkg/types/ruletypes/pilot_contract.go
  - pkg/types/ruletypes/pilot_managed_markdown.go
implements_uc: [UC-001]
covered_by_wbs: [WBS-1.0]
updated: 2026-06-02
---

# F0 — 공통 기반 모듈 (Foundation Core) / Pilot Scaffolding

> **상태**: 착수 예정 (착수보고 기준)
> AIOpsAgent 네이티브 MVP의 토대 — pilot contract, managed markdown, community 진입점 통합.

## 책임 (Responsibility)

F1~F8 전 모듈이 의존하는 contract 식별자·enum을 한 곳에서 export한다. v0.1에서 지원하는 유일한 SOP source인 managed markdown의 구조를 정의하며, `cmd/community/main.go` 부팅 시 pilot audit JSONL sink를 등록한다 (F5 연결).

## 인터페이스 요지

```go
// 진입점 — cmd/community/main.go
ruletypes.RegisterPilotAuditEventSink(jsonlSink)  // boot 시 등록, 실패 시 Nop으로 진행

// 5종 contract validator — pkg/types/ruletypes/pilot_contract.go
func ValidatePilotSOPSourceCatalog(resp PilotSOPSourceCatalogResponse) error
func ValidatePilotAuditEvent(event PilotAuditEvent) error
// 나머지 3종(Health/ServiceAccount/Configuration)은 동일 패턴 — 상세는 pilot_contract.go 참조
```

Contract version 상수 5개(`ds-apm.sop-source-catalog.v1` 등)는 frozen string — 절대 자동 변경 안 됨. 상세는 `pkg/types/ruletypes/pilot_contract.go` 참조.

## 핵심 동작

부팅 → JSONL sink 등록 시도 → 실패 시 NopSink 사용 (서버 진행 무중단). 이후 F1~F8은 `DispatchPilotAuditEvent`로 이벤트를 기록한다.

Contract validator는 위반 사항을 `errors.Join`으로 모두 한번에 보고한다. `hasPilotSecretLikeValue()`가 `token=`, `api_key`, JWT 패턴을 contract 응답에서 차단한다.

Body markdown 상한: 256 KiB (`PilotSOPFetchBodyMarkdownMaxBytes`).

## 예외·복구

| 경로 | 처리 |
|---|---|
| JSONL sink 초기화 실패 | `zap.Warn` 후 `NopPilotAuditEventSink`로 진행. 부팅 절대 중단 안 함. |
| Contract validation 실패 | `errors.Join`으로 전체 위반 반환. 호출자가 fail-closed/open 선택. |
| Secret-like 값 검출 | validation error 반환. caller가 reject. |

## Acceptance Criteria

```gherkin
Feature: Foundation contract validation

  Scenario: Credential leakage is rejected
    Given a PilotSOPSource with secretRefVisible set to true
    When ValidatePilotSOPSourceCatalog is called
    Then the validator returns an error mentioning "secretRefVisible"

  Scenario: Audit JSONL sink failure does not abort boot
    Given the audit log directory is read-only
    When the community main() initializes the JSONL sink
    Then NopPilotAuditEventSink remains active and the server proceeds
```

## Traceability
- Implements UC: UC-001
- Covered by WBS: WBS-1.0
- Source: `cmd/community/`, `pkg/types/ruletypes/pilot_contract.go`, `pkg/types/ruletypes/pilot_managed_markdown.go`
- Commits: `026863650`
