---
id: WBS-1.1
title: SOP 그라운딩 서비스 (SOP Grounding Service)
parent: WBS-1
status: planned
covers_features: [F1]
source_paths:
  - pkg/ruler/sopstore/
  - pkg/ruler/sopstore/sqlsopstore/
  - pkg/ruler/signozruler/sop_document_file_store.go
  - pkg/types/ruletypes/sop_document.go
  - pkg/types/ruletypes/sop_preview.go
acceptance: pending
estimated_effort: 3w
schedule:
  start: 2026-06-15
  end: 2026-07-03
  duration: 3w
commits: [72944ecac, c7f4fd330]
updated: 2026-06-02
---

# WBS-1.1 — SOP 그라운딩 서비스 (SOP Grounding Service)

> **상태**: 착수 예정 (착수보고 기준)
> **일정**: 2026-06-15 ~ 2026-07-03 (3주, WBS-1.0 완료 후)

## Deliverable
SOP store 인터페이스 (`sopstore.Store`), SQL 구현체 (`sqlsopstore`), 파일 영속화 구현체 (`sop_document_file_store`), SOP 문서 도메인 타입 (`sop_document`) 및 미리보기 타입 (`sop_preview`), runbook handler의 SOP 조회/등록 라우트. Alert와 업로드된 SOP를 결합(grounding)하는 검색 표면을 제공해야 한다.

## Acceptance Criteria
- [ ] F1.7 acceptance Gherkin pass — `alertname` / `runbook_url`로 SOP 조회 시 정확한 문서가 반환되어야 한다
- [ ] SOP 등록 → 파일 영속화 → 재기동 후 재로드까지 라운드트립이 보존되어야 한다
- [ ] 동일 alert에 다중 SOP 매핑 시 우선순위 규칙대로 grounding이 결정되어야 한다
- [ ] SOP 조회 이벤트는 WBS-1.0의 audit sink로 기록되어야 한다 (F5와 cross-cut)

## Work Packages (Lv3)

### WBS-1.1.1 — SOP Store 인터페이스 정의 (`SOPStore`)

- **Deliverable**: `SOPStore` interface (Upsert / Get / GetLatest / List / Delete / UpsertRunbook / DeleteRunbook) + `ErrSOPDocumentNotFound` sentinel 상수 — `pkg/types/ruletypes/sop_store.go`
- **Acceptance**: 인터페이스가 컴파일되고, `sopstoretest` contract suite 실행 시 모든 구현체가 동일한 시그니처를 충족해야 한다; cross-tenant `Get`은 반드시 `ErrSOPDocumentNotFound`를 반환해야 한다 (NF-F1.1)
- **Source**: `pkg/types/ruletypes/`
- **Effort**: TBD

### WBS-1.1.2 — SQL 스토어 구현체 (`sqlsopstore`)

- **Deliverable**: `sqlsopstore.NewSOPStore` — bun ORM 기반 PostgreSQL 구현체; `(org_id, sop_id, version)` 복합 키 upsert; migration 078 `ds_sop_documents` 스키마
- **Acceptance**: `orgID`로 partition — 다른 org의 row는 조회 불가; `UpsertRunbook`은 단일 transaction read-modify-write로 완료해야 한다 (NF-F1.2); `sqlsopstore/sop_test.go` 전 케이스 통과
- **Source**: `pkg/ruler/sopstore/sqlsopstore/`
- **Effort**: TBD

### WBS-1.1.3 — 파일 영속화 구현체 (`sop_document_file_store`)

- **Deliverable**: `sop_document_file_store.go` — SOP 문서를 디스크에 직렬화하고 재기동 시 재로드하는 파일 기반 store 구현체
- **Acceptance**: Upsert → 서버 재기동 → GetLatest 라운드트립에서 원본 문서가 손실 없이 복원되어야 한다; `runbook_handler_test.go` 영속화 케이스 통과
- **Source**: `pkg/ruler/signozruler/sop_document_file_store.go`
- **Effort**: TBD

### WBS-1.1.4 — SOP 도메인 타입 (`sop_document`, `sop_preview`)

- **Deliverable**: `SOPDocument` struct (ContractVersion / SOPID / Version / Checksum / Source / BodyMarkdown / DisplayURL / ApprovalStatus / TenantScope 등) + `SOPBindingPreviewRequest` / `SOPBindingPreviewResponse` 타입
- **Acceptance**: Checksum은 `sha256:<hex>` 포맷을 강제해야 한다 (NF-F1.3); `DisplayURL`은 http/https만 허용하고 sensitive query param 자동 제거 (`safeDisplayURL`) (NF-F1.4); BodyMarkdown > 256 KiB 시 validation error; `sop_preview_test.go` 전 케이스 통과
- **Source**: `pkg/types/ruletypes/sop_document.go`, `pkg/types/ruletypes/sop_preview.go`
- **Effort**: TBD

### WBS-1.1.5 — Grounding 로직 (`PreviewSOPDocumentBinding`)

- **Deliverable**: `PreviewSOPDocumentBinding(docs []SOPDocument, req SOPBindingPreviewRequest) (SOPBindingPreviewResponse, error)` — explicit-label binding, tenant scope 검사, disabled 처리 포함
- **Acceptance**: F1.7 Gherkin 3 시나리오 전부 통과 (bound / forbidden / disabled); `signoz_pilot_sop_id` 라벨 누락 시 `status=missing`; tenant scope mismatch 시 `status=forbidden`; cross-tenant 존재 여부 누설 금지
- **Source**: `pkg/types/ruletypes/sop_document.go`
- **Effort**: TBD

### WBS-1.1.6 — Runbook Handler SOP 라우트

- **Deliverable**: Runbook handler에 SOP 등록(Upsert) / 조회(Get·List) / 삭제(Delete) HTTP 엔드포인트 추가; 각 엔드포인트는 WBS-1.0 audit sink로 이벤트를 기록해야 한다 (F5 cross-cut)
- **Acceptance**: SOP 조회 이벤트가 audit sink에 기록되어야 한다; 등록·삭제 API가 200/404/403 응답을 올바르게 반환해야 한다; `runbook_handler_test.go` API 케이스 통과
- **Source**: `pkg/ruler/signozruler/` (runbook_handler)
- **Effort**: TBD

## Owner
TBD (TBC)

## Estimated Effort
TBD

## Dependencies
- WBS-1.0 공통 기반 모듈 (pilot contract, audit sink, tenant policy)

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
