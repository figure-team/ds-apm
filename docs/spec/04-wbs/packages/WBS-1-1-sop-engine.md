---
id: WBS-1.1
title: SOP Engine
parent: WBS-1
status: implemented
covers_features: [F1]
source_paths:
  - pkg/ruler/sopstore/
  - pkg/ruler/sopstore/sqlsopstore/
  - pkg/ruler/signozruler/sop_document_file_store.go
  - pkg/types/ruletypes/sop_document.go
  - pkg/types/ruletypes/sop_preview.go
acceptance: pending
estimated_effort: completed
commits: [72944ecac, c7f4fd330]
updated: 2026-05-29
---

# WBS-1.1 — SOP Engine

> **상태**: 구현 완료

## Deliverable
SOP store 인터페이스 (`sopstore.Store`), SQL 구현체 (`sqlsopstore`), 파일 영속화 구현체 (`sop_document_file_store`), SOP 문서 도메인 타입 (`sop_document`) 및 미리보기 타입 (`sop_preview`), runbook handler의 SOP 조회/등록 라우트. Alert와 업로드된 SOP를 결합(grounding)하는 검색 표면을 제공해야 한다.

## Acceptance Criteria
- [ ] F1.7 acceptance Gherkin pass — `alertname` / `runbook_url`로 SOP 조회 시 정확한 문서가 반환되어야 한다
- [ ] SOP 등록 → 파일 영속화 → 재기동 후 재로드까지 라운드트립이 보존되어야 한다
- [ ] 동일 alert에 다중 SOP 매핑 시 우선순위 규칙대로 grounding이 결정되어야 한다
- [ ] SOP 조회 이벤트는 WBS-1.0의 audit sink로 기록되어야 한다 (F5와 cross-cut)

## Owner
TBD (TBC)

## Estimated Effort
완료 (커밋 `72944ecac`, `c7f4fd330`)

## Dependencies
- WBS-1.0 Foundation (pilot contract, audit sink, tenant policy)

## Verification
- `pkg/ruler/sopstore/sqlsopstore/sop_test.go`
- `pkg/ruler/sopstore/sopstoretest/sop.go` (store 인터페이스 contract suite)
- `pkg/ruler/signozruler/runbook_handler_test.go`
- `pkg/types/ruletypes/sop_preview_test.go`

## Covers Features
- F1 SOP Grounding & Store

## Source Paths
- `pkg/ruler/sopstore/`
- `pkg/ruler/sopstore/sqlsopstore/`
- `pkg/ruler/signozruler/sop_document_file_store.go`
- `pkg/types/ruletypes/sop_document.go`
- `pkg/types/ruletypes/sop_preview.go`
