---
id: F1
title: SOP Grounding & Store
status: planned
commits: [72944ecac, c7f4fd330]
source_paths:
  - pkg/ruler/sopstore/
  - pkg/ruler/sopstore/sqlsopstore/
  - pkg/ruler/signozruler/sop_document_file_store.go
  - pkg/types/ruletypes/sop_document.go
  - pkg/types/ruletypes/sop_preview.go
implements_uc: [UC-001, UC-003]
covered_by_wbs: [WBS-1.1]
updated: 2026-06-02
---

# F1 — SOP Grounding & Store

> **상태**: 착수 예정 (착수보고 기준)
> SigNoz alert 라벨을 사전 등록된 SOP 문서에 매칭(grounding)하고, SOP를 SQL 백엔드에 영속화하는 모듈.

## 책임 (Responsibility)

incident 발생 시 "어떤 SOP를 적용할지"를 결정한다. Grounding은 alert label `signoz_pilot_sop_id`를 1차 키로 사용하는 explicit-label binding이다 (vector retrieval은 v0.1 미도입). Store는 `SOPStore` 인터페이스로 추상화되며 PostgreSQL + bun ORM 기반 `sqlsopstore`가 구현한다. 모든 read/write는 `orgID`로 partition된다.

## 인터페이스 요지

```go
// pkg/types/ruletypes/sop_store.go
type SOPStore interface {
    Upsert(ctx context.Context, orgID string, doc SOPDocument) error
    GetLatest(ctx context.Context, orgID, sopID string) (SOPDocument, error)
    List(ctx context.Context, orgID string) ([]SOPDocument, error)
    // Delete / UpsertRunbook / DeleteRunbook — 상세는 sop_store.go 참조
}
var ErrSOPDocumentNotFound = errors.New("sop document not found")

// Grounding 진입점
func PreviewSOPDocumentBinding(docs []SOPDocument, req SOPBindingPreviewRequest) (SOPBindingPreviewResponse, error)
```

`GetLatest`는 version DESC 문자열 정렬 — caller는 `v01`, `v02`처럼 zero-pad 필수 (`v10 < v2`). 상세 구조체는 `pkg/types/ruletypes/sop_document.go` 참조.

## 핵심 동작

입력: alert labels (`project_id`, `environment`, `signoz_pilot_sop_id`) + orgID.

처리: tenant scope 검사 → SOP lookup → `approvalStatus` 확인 → binding 결과 결정.

출력: `SOPBindingPreviewResponse.Status` — `bound | missing | disabled | forbidden`.

SOP 상태 전이: `draft → approved → deprecated → (delete)`. `disabled`는 모든 상태에서 진입 가능. `UpsertRunbook`은 read-modify-write를 단일 transaction으로 수행한다.

## 예외·복구

| 경로 | 처리 |
|---|---|
| `sop_id` label 누락 | `status=missing`, `resolution=no_match` |
| Tenant scope mismatch | `status=forbidden` (존재 여부 누설 없음) |
| Cross-tenant `Get` | `ErrSOPDocumentNotFound` (타 tenant 존재 확인 불가) |
| `approvalStatus=disabled` | `status=disabled` |
| BodyMarkdown > 256 KiB | validation error |

## Acceptance Criteria

```gherkin
Feature: SOP grounding by explicit label
  Background:
    Given a SOPStore containing document "SOP-PAY-5xx" version "v01" approved for project "p-prod"

  Scenario: Bound to explicit label
    Given an alert with labels project_id="p-prod" environment="production" signoz_pilot_sop_id="SOP-PAY-5xx"
    When PreviewSOPDocumentBinding runs
    Then the response status is "bound" and resolution is "explicit_label"

  Scenario: Cross-tenant lookup is opaque
    Given an alert with project_id="p-other"
    When PreviewSOPDocumentBinding runs
    Then the response status is "forbidden"
```

## Traceability
- Implements UC: UC-001 (단계 4), UC-003 (전제)
- Covered by WBS: WBS-1.1
- Source: `pkg/ruler/sopstore/sqlsopstore/sop.go`, `pkg/types/ruletypes/sop_document.go`
- Commits: `72944ecac`, `c7f4fd330`
