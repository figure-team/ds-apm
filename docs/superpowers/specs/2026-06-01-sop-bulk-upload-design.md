# SOP 일괄 업로드 기능 설계

**Date:** 2026-06-01  
**Status:** Approved  
**Scope:** SOP Documents 페이지에 Excel 파일 업로드를 통한 다건 SOP 일괄 등록 기능

---

## 배경

현재 SOP Documents 페이지는 건당 수동 폼 입력 방식으로, 다수의 SOP를 등록할 때 반복 작업이 많다. Excel 파일 업로드로 여러 건을 한 번에 등록할 수 있는 기능이 필요하다.

---

## 결정 사항

| 항목 | 결정 |
|---|---|
| 파일 포맷 | Excel (.xlsx) 우선 지원 |
| 업로드 흐름 | 미리보기 확인 후 일괄 등록 |
| 오류 처리 | 유효한 행만 등록, 오류 행은 건너뜀 |
| 템플릿 컬럼 | 전체 12개 필드 |
| UI 컨테이너 | 모달(업로드) → 드로어(미리보기/결과) |
| API 방식 | 새 배치 엔드포인트로 단일 호출 |

---

## 전체 데이터 흐름

```
[헤더 우측 "파일 업로드" 버튼]
        ↓ 클릭
[Upload Modal]
  - .xlsx 드래그&드롭 / 파일 선택
  - SheetJS(xlsx)로 브라우저 파싱
  - 행별 클라이언트 유효성 검사
  - 파싱 완료 → 모달 닫힘
        ↓ 자동 전환
[Preview Drawer (우측 슬라이드)]
  - 12컬럼 미리보기 테이블
  - 유효 행: 정상 / 오류 행: 빨간 배경 + 오류 메시지 (등록 제외)
  - 하단: "유효 N건 / 오류 M건" 요약
  - "N건 일괄 등록" 버튼 (유효 행만)
        ↓ 버튼 클릭
[POST /api/v2/ds/sop/documents/batch]
  - 서버사이드 ValidateSOPDocument() 적용
  - 유효한 것만 저장, 실패는 건너뜀
  - 건별 결과 반환
        ↓ 응답
[Drawer 결과 표시]
  - 성공/실패 건수 + 실패 사유
  - 닫기 → 문서 목록 자동 갱신
```

---

## 프론트엔드

### 신규 파일

| 파일 | 역할 |
|---|---|
| `frontend/src/pages/SOPDocuments/SopBulkUploadModal.tsx` | 파일 드롭존 + SheetJS 파싱 + 클라이언트 검증 |
| `frontend/src/pages/SOPDocuments/SopBulkPreviewDrawer.tsx` | 미리보기 테이블 + 일괄 등록 버튼 + 결과 표시 |
| `frontend/src/pages/SOPDocuments/parseSopExcel.ts` | SheetJS 파싱 → `SopDocumentFormState[]` 변환 유틸 |

### 기존 파일 변경

| 파일 | 변경 내용 |
|---|---|
| `frontend/src/pages/SOPDocuments/SOPDocuments.tsx` | 헤더에 "파일 업로드" / "템플릿 다운로드" 버튼 추가, 모달/드로어 상태 관리 |
| `frontend/src/api/v2/rules/sopDocuments.ts` | `createSopDocumentBatch()` 함수 + 배치 관련 타입 추가 |

### Excel 컬럼 매핑

| Excel 헤더 | SOP 필드 | 필수 | 기본값 |
|---|---|---|---|
| `sop_id` | `sopId` | ✓ | — |
| `title` | `title` | ✓ | — |
| `version` | `version` | ✓ | — |
| `owner_team` | `ownerTeam` | ✓ | — |
| `approval_status` | `approvalStatus` | — | `approved` |
| `source_id` | `source.sourceId` | — | `src-managed-markdown-default` |
| `project_ids` | `tenantScope.projectIds` | ✓ | — |
| `environments` | `tenantScope.environments` | ✓ | — |
| `display_url` | `displayUrl` | — | — |
| `tags` | `tags` | — | — |
| `service_account_profile` | `securityContext.serviceAccountProfile` | — | `managed-markdown-local` |
| `body_markdown` | `bodyMarkdown` | ✓ | — |

콤마 구분 필드(`project_ids`, `environments`, `tags`)는 기존 `parseTags()` 재사용.

### UX 디테일

- 헤더 우측에 "파일 업로드" 버튼과 "템플릿 다운로드" 버튼 나란히 배치
- 템플릿 다운로드: 12개 헤더 + 예시 행 1개가 포함된 빈 `.xlsx` 파일
- 유효 행 0건이면 "N건 일괄 등록" 버튼 비활성화
- 드로어 닫기 시 `loadDocuments()` 호출로 목록 자동 갱신

---

## 백엔드

### 새 엔드포인트

```
POST /api/v2/ds/sop/documents/batch
```

### Request

```json
{
  "contractVersion": "ds.sop_document_list.v1",
  "documents": [ ...SOPDocument[] ]
}
```

### Response

```json
{
  "contractVersion": "ds.sop_batch_result.v1",
  "total": 5,
  "succeeded": 4,
  "failed": 1,
  "results": [
    { "sopId": "SOP-PAY-001", "version": "2026-05-12.1", "status": "ok" },
    { "sopId": "SOP-CART-002", "version": "2026-05-12.1", "status": "error", "error": "bodyMarkdown: payload does not look like markdown" }
  ]
}
```

### 신규 Go 타입 (`pkg/types/ruletypes/sop_document.go`)

```go
type SOPDocumentBatchRequest struct {
    ContractVersion string        `json:"contractVersion"`
    Documents       []SOPDocument `json:"documents"`
}

type SOPDocumentBatchResponse struct {
    ContractVersion string                    `json:"contractVersion"`
    Total           int                       `json:"total"`
    Succeeded       int                       `json:"succeeded"`
    Failed          int                       `json:"failed"`
    Results         []SOPDocumentBatchResult  `json:"results"`
}

type SOPDocumentBatchResult struct {
    SOPID   string `json:"sopId"`
    Version string `json:"version"`
    Status  string `json:"status"` // "ok" | "error"
    Error   string `json:"error,omitempty"`
}
```

### 신규/변경 백엔드 파일

| 파일 | 역할 |
|---|---|
| `pkg/ruler/signozruler/sop_batch_handler.go` | `CreateSOPDocumentBatch` 핸들러 메서드 구현 |
| `pkg/apiserver/signozapiserver/ruler.go` | `POST /api/v2/ds/sop/documents/batch` 라우트 추가 (`addRulerRoutes` 내부) |

### 처리 로직

- 각 document에 기존 `ValidateSOPDocument()` 적용
- 유효한 것만 `sopstore`에 저장
- 실패한 것은 건너뛰고 오류 메시지를 결과에 포함
- 부분 성공이어도 **HTTP 200** 반환 (결과 배열로 성공/실패 구분)

---

## 오류 처리

| 단계 | 오류 유형 | 표시 방식 |
|---|---|---|
| 모달 (파싱) | .xlsx 아닌 파일 | 모달 내 인라인 에러 |
| 모달 (파싱) | 필수 헤더 컬럼 누락 | 모달 내 인라인 에러 |
| 드로어 (미리보기) | 필수 필드 누락 | 해당 행 빨간 배경 + 오류 셀 강조 |
| 드로어 (미리보기) | `approval_status` 허용 외 값 | 해당 행 오류 표시 |
| 드로어 (등록 후) | 서버사이드 검증 실패 | 결과 테이블 실패 행 + 사유 |
| 드로어 (등록 후) | 네트워크 오류 | 기존 `getErrorMessage()` 패턴 재사용 |

---

## 의존성

- `xlsx` (SheetJS) — **신규 설치 필요** (`npm install xlsx`). 브라우저에서 `.xlsx` 파싱 및 템플릿 파일 온더플라이 생성에 사용
- Ant Design `Modal`, `Drawer`, `Upload`, `Table` — 기존 UI 라이브러리 재사용 (추가 설치 불필요)
