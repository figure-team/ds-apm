---
id: WBS-1.0
title: 공통 기반 모듈 (Foundation Core)
parent: WBS-1
status: planned
covers_features: [F0, F4, F5]
source_paths:
  - cmd/community/
  - pkg/types/ruletypes/pilot_contract.go
  - pkg/types/ruletypes/pilot_managed_markdown.go
  - pkg/types/ruletypes/tenant_policy.go
  - pkg/types/ruletypes/pilot_audit_sink.go
  - pkg/types/ruletypes/pilot_audit_sink_jsonl.go
acceptance: pending
estimated_effort: 3w
schedule:
  start: 2026-05-25
  end: 2026-06-12
  duration: 3w
commits: [026863650, 8a55208ef, 3fa604e03]
updated: 2026-06-02
---

# WBS-1.0 — 공통 기반 모듈 (Foundation Core)

> **상태**: 착수 예정 (착수보고 기준)
> **일정**: 2026-05-25 ~ 2026-06-12 (3주)

## Deliverable
Pilot 계약 스키마 (`pilot_contract`), 관리형 markdown 페이로드 (`pilot_managed_markdown`), 테넌트 격리 정책 (`tenant_policy`), 감사 sink 추상화 및 JSONL 구현 (`pilot_audit_sink`, `pilot_audit_sink_jsonl`), community 진입점 와이어업. AIOpsAgent의 하위 컴포넌트 모두가 공유하는 기반 타입·정책·감사 통로를 제공해야 한다.

## Acceptance Criteria
- [ ] F0.7 acceptance Gherkin pass — pilot contract 직렬화·검증 및 managed markdown 라운드트립
- [ ] F4.7 acceptance Gherkin pass — tenant scope 위반 시 SOP/strategy 접근 거부
- [ ] F5.7 acceptance Gherkin pass — SOP / draft / dispatch 이벤트가 audit sink로 기록되어야 한다
- [ ] community 바이너리 부팅 시 pilot contract와 audit sink가 의존성 그래프에 등록됨

## Work Package 일정 (일 단위)

> 영업일(주5일) 기준, 공휴일 미반영. 의존성 순서: 인터페이스·타입 → 구현 → 통합·검증.

| WP ID | 작업명 | 선행 | 시작일 | 종료일 | 기간(영업일) |
|---|---|---|---|---|---|
| 1.0.1 | Pilot 계약 스키마 | — | 2026-05-25 | 2026-05-27 | 3 |
| 1.0.2 | 관리형 Markdown 페이로드 | 1.0.1 | 2026-05-28 | 2026-06-01 | 3 |
| 1.0.3 | 테넌트 격리 정책 | 1.0.2 | 2026-06-02 | 2026-06-04 | 3 |
| 1.0.4 | 감사 Sink 추상화 | 1.0.3 | 2026-06-05 | 2026-06-08 | 2 |
| 1.0.5 | JSONL 감사 Sink 구현 | 1.0.4 | 2026-06-09 | 2026-06-10 | 2 |
| 1.0.6 | community 진입점 와이어업 | 1.0.5 | 2026-06-11 | 2026-06-12 | 2 |

## Work Packages (Lv3)

### WBS-1.0.1 — Pilot 계약 스키마 (pilot_contract)

- **Deliverable**: `PilotSOPSource`, `PilotManagedMarkdownDocument`, `PilotConfiguration` 등 5종 contract struct + 버전 상수 + 5종 validator 함수 (`ValidatePilotSOPSourceCatalog`, `ValidatePilotSOPSourceHealth`, `ValidatePilotAuditEvent`, `ValidatePilotServiceAccountProfile`, `ValidatePilotConfiguration`). credential-leak 차단 로직 (`hasPilotSecretLikeValue`) 포함.
- **Acceptance**:
  - `PilotSOPSource.SecretRefVisible = true`이면 `ValidatePilotSOPSourceCatalog`가 "secretRefVisible" 언급 error 반환 (F0.7 Scenario 1)
  - Contract version 상수는 절대 자동 변경되지 않음 (NF-F0.1)
  - validator가 `errors.Join`으로 모든 위반을 한꺼번에 보고
- **Source**: `pkg/types/ruletypes/pilot_contract.go`
- **일정**: 2026-05-25 ~ 2026-05-27 (3영업일, 선행: —)
- **Effort**: TBD

### WBS-1.0.2 — 관리형 Markdown 페이로드 (pilot_managed_markdown)

- **Deliverable**: `PilotManagedMarkdownDocument` struct + body 상한 상수 (`PilotSOPFetchBodyMarkdownMaxBytes = 256 KiB`) + 직렬화·검증 로직. v0.1에서 유일하게 구현된 SOP source kind(`managed_markdown`)의 페이로드 정의.
- **Acceptance**:
  - `PilotManagedMarkdownDocument` 라운드트립 직렬화가 손실 없이 통과 (F0.7 Background)
  - body markdown이 256 KiB 초과 시 validation error 반환 (NF-F0.4)
  - `managed_markdown` 외 kind는 미구현 상태로 문서화됨 (F0.3 enum 주석)
- **Source**: `pkg/types/ruletypes/pilot_managed_markdown.go`
- **일정**: 2026-05-28 ~ 2026-06-01 (3영업일, 선행: 1.0.1)
- **Effort**: TBD

### WBS-1.0.3 — 테넌트 격리 정책 (tenant_policy)

- **Deliverable**: `PilotAuditTenant` + `PilotTenantScope` struct, `PilotTenantFromLabels` / `PilotTenantIsComplete` / `PilotTenantScopeAllows` 함수, scope normalization (`normalizePilotTenantScope`). label→tenant 추출부터 scope 매칭까지 전 계층.
- **Acceptance**:
  - `project_id="p-prod", environment="production"` 정확 매칭 → `true` (F4.7 Scenario 1)
  - `Environments=["*"]` 와일드카드 시 임의 environment 허용 (F4.7 Scenario 2)
  - `project_id` label 누락 시 `PilotTenantIsComplete()=false` → 접근 거부 (F4.7 Scenario 3)
- **Source**: `pkg/types/ruletypes/tenant_policy.go`
- **일정**: 2026-06-02 ~ 2026-06-04 (3영업일, 선행: 1.0.2)
- **Effort**: TBD

### WBS-1.0.4 — 감사 Sink 추상화 (pilot_audit_sink)

- **Deliverable**: `PilotAuditEventSink` interface + `NopPilotAuditEventSink` default 구현 + global registry (`RegisterPilotAuditEventSink`, `CurrentPilotAuditEventSink`, `DispatchPilotAuditEvent`). sink 미등록 시 Nop fallback 보장.
- **Acceptance**:
  - `RegisterPilotAuditEventSink(nil)` 호출 시 Nop으로 reset
  - `DispatchPilotAuditEvent`는 sink error를 반환하되, 호출자는 best-effort로 무시 가능 (F5.5 convention)
  - SOP access/fetch path에서 audit event 누락률 0% (NF-F5.5, in-process 범위)
- **Source**: `pkg/types/ruletypes/pilot_audit_sink.go`
- **일정**: 2026-06-05 ~ 2026-06-08 (2영업일, 선행: 1.0.3)
- **Effort**: TBD

### WBS-1.0.5 — JSONL 감사 Sink 구현 (pilot_audit_sink_jsonl)

- **Deliverable**: `PilotAuditEventJSONLSink` struct + `NewPilotAuditEventJSONLSink` 생성자 + append/rotate 로직. 50 MiB 임계치마다 timestamped sibling 파일로 rotate. thread-safe (`sync.Mutex`).
- **Acceptance**:
  - 유효 이벤트 → 파일에 JSON object + newline 1행 추가 (F5.7 Scenario 1)
  - 무효 이벤트(empty EventID 등) → error 없이 silent drop + warn log (F5.7 Scenario 2)
  - 활성 파일이 임계치 초과 시 rename → fresh 파일에 신규 이벤트 기록 (F5.7 Scenario 3)
- **Source**: `pkg/types/ruletypes/pilot_audit_sink_jsonl.go`
- **일정**: 2026-06-09 ~ 2026-06-10 (2영업일, 선행: 1.0.4)
- **Effort**: TBD

### WBS-1.0.6 — community 진입점 와이어업 (cmd/community)

- **Deliverable**: `cmd/community/main.go` — 부팅 시 JSONL audit sink 초기화·등록, `registerServer` + `cmd.RegisterGenerate` + `cmd.Execute` 연결. sink 초기화 실패 시 fail-open (warn log 후 Nop sink 유지, 부팅 계속).
- **Acceptance**:
  - audit log 디렉터리가 read-only여도 warn log 후 `NopPilotAuditEventSink`로 서버 진행 (F0.7 Scenario 2)
  - 부팅 성공 시 pilot contract와 audit sink가 의존성 그래프에 등록됨 (WBS-1.0 Acceptance Criteria 마지막 항목)
  - `main()` 내 sink 등록이 `registerServer` 호출 전에 완료됨 (순서 보장)
- **Source**: `cmd/community/`
- **일정**: 2026-06-11 ~ 2026-06-12 (2영업일, 선행: 1.0.5)
- **Effort**: TBD

## Owner
TBD (TBC)

## Estimated Effort
TBD

## Dependencies
없음 (AIOpsAgent 모듈 그룹 루트, SigNoz upstream 진입점에만 의존)

## Verification
- `pkg/types/ruletypes/pilot_contract_test.go`
- `pkg/types/ruletypes/pilot_managed_markdown_test.go`
- `pkg/types/ruletypes/pilot_audit_sink_jsonl_test.go`
- tenant policy 단위 테스트는 현재 부재 — F4 follow-up으로 추적

## Covers Features
- F0 Foundation
- F4 Multi-tenant Scope
- F5 Audit

## Source Paths
- `cmd/community/`
- `pkg/types/ruletypes/pilot_contract.go`
- `pkg/types/ruletypes/pilot_managed_markdown.go`
- `pkg/types/ruletypes/tenant_policy.go`
- `pkg/types/ruletypes/pilot_audit_sink.go`
- `pkg/types/ruletypes/pilot_audit_sink_jsonl.go`

## Open Items
- `tenant_policy.go` 전용 단위 테스트 보강 (현재 통합 경로에서만 간접 검증)
- audit sink의 ClickHouse/원격 sink 구현은 P-단계 후속 (현재 file/JSONL만)
