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
estimated_effort: TBD
commits: [026863650, 8a55208ef, 3fa604e03]
updated: 2026-05-29
---

# WBS-1.0 — 공통 기반 모듈 (Foundation Core)

> **상태**: 착수 예정 (착수보고 기준)

## Deliverable
Pilot 계약 스키마 (`pilot_contract`), 관리형 markdown 페이로드 (`pilot_managed_markdown`), 테넌트 격리 정책 (`tenant_policy`), 감사 sink 추상화 및 JSONL 구현 (`pilot_audit_sink`, `pilot_audit_sink_jsonl`), community 진입점 와이어업. AIOpsAgent의 하위 컴포넌트 모두가 공유하는 기반 타입·정책·감사 통로를 제공해야 한다.

## Acceptance Criteria
- [ ] F0.7 acceptance Gherkin pass — pilot contract 직렬화·검증 및 managed markdown 라운드트립
- [ ] F4.7 acceptance Gherkin pass — tenant scope 위반 시 SOP/strategy 접근 거부
- [ ] F5.7 acceptance Gherkin pass — SOP / draft / dispatch 이벤트가 audit sink로 기록되어야 한다
- [ ] community 바이너리 부팅 시 pilot contract와 audit sink가 의존성 그래프에 등록됨

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
