# SOP 일괄 업로드 구현 계획

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** SOP Documents 페이지 우측 상단에 Excel 업로드 버튼을 추가하고, 모달(파일 선택) → 드로어(미리보기/확인) → 배치 API(단일 호출) 흐름으로 다건 SOP를 일괄 등록한다.

**Architecture:** 프론트엔드에서 SheetJS로 .xlsx를 파싱해 행별 검증 후 미리보기 드로어에 표시, 유효 행만 새 배치 엔드포인트 `POST /api/v2/ds/sop/documents/batch`에 단일 요청으로 전송한다. 백엔드는 행별 `ValidateSOPDocument`를 적용하고 성공/실패 결과를 배열로 반환한다. 부분 성공도 HTTP 200을 반환한다.

**Tech Stack:** Go (gorilla/mux, existing binding/render helpers), TypeScript/React, Ant Design (Modal, Drawer, Upload, Table), SheetJS (`xlsx` npm package)

---

## 파일 구조

| 파일 | 신규/변경 | 역할 |
|---|---|---|
| `pkg/ruler/handler.go` | 변경 | Handler 인터페이스에 `CreateSOPDocumentBatch` 추가 |
| `pkg/types/ruletypes/sop_document.go` | 변경 | 배치 타입 3개 + 상수 추가 |
| `pkg/ruler/signozruler/sop_batch_handler.go` | 신규 | `CreateSOPDocumentBatch` 핸들러 구현 |
| `pkg/ruler/signozruler/sop_batch_handler_test.go` | 신규 | 배치 핸들러 테스트 |
| `pkg/apiserver/signozapiserver/ruler.go` | 변경 | 배치 라우트 등록 |
| `frontend/src/api/v2/rules/sopDocuments.ts` | 변경 | 배치 타입 + `createSopDocumentBatch()` 추가 |
| `frontend/src/pages/SOPDocuments/parseSopExcel.ts` | 신규 | SheetJS 파싱 → `SopDocument[]` 변환 유틸 |
| `frontend/src/pages/SOPDocuments/__tests__/parseSopExcel.test.ts` | 신규 | 파서 단위 테스트 |
| `frontend/src/pages/SOPDocuments/SopBulkUploadModal.tsx` | 신규 | 파일 드롭존 모달 |
| `frontend/src/pages/SOPDocuments/SopBulkPreviewDrawer.tsx` | 신규 | 미리보기/결과 드로어 |
| `frontend/src/pages/SOPDocuments/SOPDocuments.tsx` | 변경 | 버튼 + 상태 연결 |

---

## Task 1: Go 배치 타입 추가

**Files:**
- Modify: `pkg/types/ruletypes/sop_document.go`

- [ ] **Step 1: 상수와 타입 추가**

`pkg/types/ruletypes/sop_document.go` 파일 맨 아래에 다음을 추가한다:

```go
const (
	SOPBatchResultContractVersion = "ds.sop_batch_result.v1"
	SOPBatchResultStatusOk        = "ok"
	SOPBatchResultStatusError     = "error"
)

type SOPDocumentBatchRequest struct {
	ContractVersion string        `json:"contractVersion"`
	Documents       []SOPDocument `json:"documents"`
}

type SOPDocumentBatchResponse struct {
	ContractVersion string                   `json:"contractVersion"`
	Total           int                      `json:"total"`
	Succeeded       int                      `json:"succeeded"`
	Failed          int                      `json:"failed"`
	Results         []SOPDocumentBatchResult `json:"results"`
}

type SOPDocumentBatchResult struct {
	SOPID   string `json:"sopId"`
	Version string `json:"version"`
	Status  string `json:"status"` // "ok" | "error"
	Error   string `json:"error,omitempty"`
}
```

- [ ] **Step 2: 컴파일 확인**

```bash
cd pkg/types/ruletypes && go build ./...
```

Expected: 오류 없이 완료

- [ ] **Step 3: 커밋**

```bash
git add pkg/types/ruletypes/sop_document.go
git commit -m "feat(ruletypes): add SOPDocumentBatch request/response types"
```

---

## Task 2: Handler 인터페이스 업데이트

**Files:**
- Modify: `pkg/ruler/handler.go`

- [ ] **Step 1: 인터페이스에 메서드 추가**

`pkg/ruler/handler.go`의 `CreateSOPDocument` 줄 바로 아래에 추가:

```go
CreateSOPDocumentBatch(http.ResponseWriter, *http.Request)
```

결과:
```go
CreateSOPDocument(http.ResponseWriter, *http.Request)
CreateSOPDocumentBatch(http.ResponseWriter, *http.Request)
ListSOPDocuments(http.ResponseWriter, *http.Request)
```

- [ ] **Step 2: 컴파일 확인 (구현 전이므로 실패해야 함)**

```bash
go build ./pkg/ruler/...
```

Expected: `signozruler.handler does not implement ruler.Handler` 에러 (정상 — Task 3에서 구현)

- [ ] **Step 3: 커밋**

```bash
git add pkg/ruler/handler.go
git commit -m "feat(ruler): add CreateSOPDocumentBatch to Handler interface"
```

---

## Task 3: 배치 핸들러 구현 + 테스트

**Files:**
- Create: `pkg/ruler/signozruler/sop_batch_handler.go`
- Create: `pkg/ruler/signozruler/sop_batch_handler_test.go`

- [ ] **Step 1: 실패 테스트 작성**

`pkg/ruler/signozruler/sop_batch_handler_test.go` 파일 생성:

```go
package signozruler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

func TestCreateSOPDocumentBatch_HappyPath(t *testing.T) {
	h := newTestHandler()

	doc1 := validSOPDocumentRequest(t, "2026-06-01.1", ruletypes.SOPApprovalStatusApproved)
	doc2 := validSOPDocumentRequest(t, "2026-06-01.2", ruletypes.SOPApprovalStatusApproved)
	doc2.SOPID = "SOP-CART-001"
	doc2.Title = "Cart Redis timeout"

	body, err := json.Marshal(ruletypes.SOPDocumentBatchRequest{
		ContractVersion: ruletypes.SOPDocumentListContractVersion,
		Documents:       []ruletypes.SOPDocument{doc1, doc2},
	})
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents/batch", bytes.NewReader(body))
	req = withSOPTestClaims(req)
	h.CreateSOPDocumentBatch(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	var got struct {
		Data ruletypes.SOPDocumentBatchResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &got))
	require.Equal(t, ruletypes.SOPBatchResultContractVersion, got.Data.ContractVersion)
	require.Equal(t, 2, got.Data.Total)
	require.Equal(t, 2, got.Data.Succeeded)
	require.Equal(t, 0, got.Data.Failed)
	require.Len(t, got.Data.Results, 2)
	require.Equal(t, ruletypes.SOPBatchResultStatusOk, got.Data.Results[0].Status)
}

func TestCreateSOPDocumentBatch_PartialFailure(t *testing.T) {
	h := newTestHandler()

	validDoc := validSOPDocumentRequest(t, "2026-06-01.1", ruletypes.SOPApprovalStatusApproved)
	invalidDoc := validSOPDocumentRequest(t, "2026-06-01.2", ruletypes.SOPApprovalStatusApproved)
	invalidDoc.BodyMarkdown = "Rotate with access_token=hidden" // secret-like string → validation error

	body, err := json.Marshal(ruletypes.SOPDocumentBatchRequest{
		ContractVersion: ruletypes.SOPDocumentListContractVersion,
		Documents:       []ruletypes.SOPDocument{validDoc, invalidDoc},
	})
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents/batch", bytes.NewReader(body))
	req = withSOPTestClaims(req)
	h.CreateSOPDocumentBatch(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	var got struct {
		Data ruletypes.SOPDocumentBatchResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &got))
	require.Equal(t, 2, got.Data.Total)
	require.Equal(t, 1, got.Data.Succeeded)
	require.Equal(t, 1, got.Data.Failed)
	require.Equal(t, ruletypes.SOPBatchResultStatusOk, got.Data.Results[0].Status)
	require.Equal(t, ruletypes.SOPBatchResultStatusError, got.Data.Results[1].Status)
	require.NotEmpty(t, got.Data.Results[1].Error)
}

func TestCreateSOPDocumentBatch_RequiresClaims(t *testing.T) {
	body, err := json.Marshal(ruletypes.SOPDocumentBatchRequest{
		ContractVersion: ruletypes.SOPDocumentListContractVersion,
		Documents:       []ruletypes.SOPDocument{},
	})
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents/batch", bytes.NewReader(body))
	// No claims attached

	newTestHandler().CreateSOPDocumentBatch(rw, req)

	require.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestCreateSOPDocumentBatch_EmptyDocuments(t *testing.T) {
	h := newTestHandler()

	body, err := json.Marshal(ruletypes.SOPDocumentBatchRequest{
		ContractVersion: ruletypes.SOPDocumentListContractVersion,
		Documents:       []ruletypes.SOPDocument{},
	})
	require.NoError(t, err)

	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/sop/documents/batch", bytes.NewReader(body))
	req = withSOPTestClaims(req)
	h.CreateSOPDocumentBatch(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	var got struct {
		Data ruletypes.SOPDocumentBatchResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &got))
	require.Equal(t, 0, got.Data.Total)
	require.Equal(t, 0, got.Data.Succeeded)
	require.Equal(t, 0, got.Data.Failed)
}
```

- [ ] **Step 2: 테스트 실패 확인**

```bash
cd pkg/ruler/signozruler && go test -run TestCreateSOPDocumentBatch -v
```

Expected: `FAIL — undefined: handler.CreateSOPDocumentBatch`

- [ ] **Step 3: 핸들러 구현**

`pkg/ruler/signozruler/sop_batch_handler.go` 파일 생성:

```go
package signozruler

import (
	"net/http"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

func (handler *handler) CreateSOPDocumentBatch(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	var batchReq ruletypes.SOPDocumentBatchRequest
	if err := binding.JSON.BindBody(req.Body, &batchReq); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	results := make([]ruletypes.SOPDocumentBatchResult, 0, len(batchReq.Documents))
	succeeded := 0
	failed := 0

	for _, doc := range batchReq.Documents {
		result := ruletypes.SOPDocumentBatchResult{
			SOPID:   doc.SOPID,
			Version: doc.Version,
		}

		if err := ruletypes.ValidateSOPDocument(doc); err != nil {
			result.Status = ruletypes.SOPBatchResultStatusError
			result.Error = err.Error()
			failed++
			results = append(results, result)
			continue
		}

		if err := handler.sopStore.Upsert(req.Context(), orgID, doc); err != nil {
			result.Status = ruletypes.SOPBatchResultStatusError
			result.Error = "failed to persist document"
			failed++
			results = append(results, result)
			continue
		}

		result.Status = ruletypes.SOPBatchResultStatusOk
		succeeded++
		results = append(results, result)
	}

	render.Success(rw, http.StatusOK, ruletypes.SOPDocumentBatchResponse{
		ContractVersion: ruletypes.SOPBatchResultContractVersion,
		Total:           len(batchReq.Documents),
		Succeeded:       succeeded,
		Failed:          failed,
		Results:         results,
	})
}
```

- [ ] **Step 4: 테스트 통과 확인**

```bash
cd pkg/ruler/signozruler && go test -run TestCreateSOPDocumentBatch -v
```

Expected: 4개 테스트 모두 PASS

- [ ] **Step 5: 전체 패키지 테스트**

```bash
cd pkg/ruler/signozruler && go test ./... -v 2>&1 | tail -20
```

Expected: 전체 PASS

- [ ] **Step 6: 커밋**

```bash
git add pkg/ruler/signozruler/sop_batch_handler.go pkg/ruler/signozruler/sop_batch_handler_test.go
git commit -m "feat(signozruler): add CreateSOPDocumentBatch handler with partial-failure support"
```

---

## Task 4: 배치 라우트 등록

**Files:**
- Modify: `pkg/apiserver/signozapiserver/ruler.go`

- [ ] **Step 1: 라우트 추가**

`ruler.go`의 기존 `CreateSOPDocument` 라우트 블록(라인 193~207) 바로 뒤에 다음을 추가한다:

```go
if err := router.Handle("/api/v2/ds/sop/documents/batch", handler.New(provider.authZ.EditAccess(provider.rulerHandler.CreateSOPDocumentBatch), handler.OpenAPIDef{
    ID:                  "CreateSOPDocumentBatch",
    Tags:                []string{"rules"},
    Summary:             "Batch create SOP documents",
    Description:         "Registers multiple ds.sop_document.v1 documents in a single request; each document is validated independently — valid ones are stored, invalid ones are reported with error details",
    Request:             new(ruletypes.SOPDocumentBatchRequest),
    RequestContentType:  "application/json",
    Response:            new(ruletypes.SOPDocumentBatchResponse),
    ResponseContentType: "application/json",
    SuccessStatusCode:   http.StatusOK,
    ErrorStatusCodes:    []int{http.StatusBadRequest, http.StatusUnauthorized},
    SecuritySchemes:     newSecuritySchemes(types.RoleEditor),
})).Methods(http.MethodPost).GetError(); err != nil {
    return err
}
```

- [ ] **Step 2: 컴파일 확인**

```bash
go build ./pkg/apiserver/...
```

Expected: 오류 없이 완료

- [ ] **Step 3: 커밋**

```bash
git add pkg/apiserver/signozapiserver/ruler.go
git commit -m "feat(apiserver): register POST /api/v2/ds/sop/documents/batch route"
```

---

## Task 5: xlsx 설치 + 프론트엔드 배치 API 타입

**Files:**
- Modify: `frontend/src/api/v2/rules/sopDocuments.ts`

- [ ] **Step 1: xlsx 패키지 설치**

```bash
cd frontend && npm install xlsx
```

Expected: `package.json`에 `"xlsx": "^0.18.x"` 추가됨

- [ ] **Step 2: 배치 타입 + 함수 추가**

`frontend/src/api/v2/rules/sopDocuments.ts` 맨 아래에 추가:

```typescript
export const SOP_BATCH_RESULT_CONTRACT_VERSION = 'ds.sop_batch_result.v1';

export type SopDocumentBatchRequest = {
	contractVersion: string;
	documents: SopDocument[];
};

export type SopDocumentBatchResult = {
	sopId: string;
	version: string;
	status: 'ok' | 'error';
	error?: string;
};

export type SopDocumentBatchResponse = {
	contractVersion: string;
	total: number;
	succeeded: number;
	failed: number;
	results: SopDocumentBatchResult[];
};

export function createSopDocumentBatch(
	data: SopDocumentBatchRequest,
): Promise<ApiResponse<SopDocumentBatchResponse>> {
	return GeneratedAPIInstance<ApiResponse<SopDocumentBatchResponse>>({
		url: '/api/v2/ds/sop/documents/batch',
		method: 'POST',
		data,
	});
}
```

- [ ] **Step 3: TypeScript 컴파일 확인**

```bash
cd frontend && npx tsc --noEmit 2>&1 | grep sopDocuments
```

Expected: 출력 없음 (오류 없음)

- [ ] **Step 4: 커밋**

```bash
git add frontend/package.json frontend/package-lock.json frontend/src/api/v2/rules/sopDocuments.ts
git commit -m "feat(api): add xlsx dep and createSopDocumentBatch API function"
```

---

## Task 6: parseSopExcel 유틸 + 테스트

**Files:**
- Create: `frontend/src/pages/SOPDocuments/parseSopExcel.ts`
- Create: `frontend/src/pages/SOPDocuments/__tests__/parseSopExcel.test.ts`

- [ ] **Step 1: 실패 테스트 작성**

`frontend/src/pages/SOPDocuments/__tests__/parseSopExcel.test.ts` 생성:

```typescript
import { parseSopRows } from '../parseSopExcel';

const VALID_ROW = {
	sop_id: 'SOP-PAY-001',
	title: 'Payment API 5xx',
	version: '2026-06-01.1',
	owner_team: 'payments',
	approval_status: 'approved',
	source_id: 'src-managed-markdown-default',
	project_ids: 'customer-a',
	environments: 'prod',
	display_url: 'https://kb.example/sop/SOP-PAY-001',
	tags: 'payment-api,critical',
	service_account_profile: 'managed-markdown-local',
	body_markdown: '# Payment API 5xx\n\n1. Check logs',
};

describe('parseSopRows', () => {
	it('parses a valid row into a SopDocument', () => {
		const result = parseSopRows([VALID_ROW]);
		expect(result.validCount).toBe(1);
		expect(result.errorCount).toBe(0);
		expect(result.rows[0].valid).toBe(true);
		expect(result.rows[0].document?.sopId).toBe('SOP-PAY-001');
		expect(result.rows[0].document?.approvalStatus).toBe('approved');
		expect(result.rows[0].document?.tenantScope.projectIds).toEqual(['customer-a']);
		expect(result.rows[0].document?.tags).toEqual(['payment-api', 'critical']);
		expect(result.rows[0].document?.checksum).toMatch(/^sha256:/);
	});

	it('applies defaults for optional fields when empty', () => {
		const row = { ...VALID_ROW, approval_status: '', source_id: '', service_account_profile: '' };
		const result = parseSopRows([row]);
		expect(result.rows[0].valid).toBe(true);
		expect(result.rows[0].document?.approvalStatus).toBe('approved');
		expect(result.rows[0].document?.source.sourceId).toBe('src-managed-markdown-default');
		expect(result.rows[0].document?.securityContext.serviceAccountProfile).toBe('managed-markdown-local');
	});

	it('marks a row with missing required field as error', () => {
		const row = { ...VALID_ROW, sop_id: '' };
		const result = parseSopRows([row]);
		expect(result.validCount).toBe(0);
		expect(result.errorCount).toBe(1);
		expect(result.rows[0].valid).toBe(false);
		expect(result.rows[0].error).toContain('sop_id');
	});

	it('marks a row with invalid approval_status as error', () => {
		const row = { ...VALID_ROW, approval_status: 'invalid-status' };
		const result = parseSopRows([row]);
		expect(result.rows[0].valid).toBe(false);
		expect(result.rows[0].error).toContain('approval_status');
	});

	it('handles mixed valid and invalid rows', () => {
		const invalidRow = { ...VALID_ROW, sop_id: '', version: '' };
		const result = parseSopRows([VALID_ROW, invalidRow]);
		expect(result.validCount).toBe(1);
		expect(result.errorCount).toBe(1);
	});

	it('returns empty result for empty rows array', () => {
		const result = parseSopRows([]);
		expect(result.validCount).toBe(0);
		expect(result.errorCount).toBe(0);
		expect(result.rows).toHaveLength(0);
	});
});
```

- [ ] **Step 2: 테스트 실패 확인**

```bash
cd frontend && npx jest parseSopExcel --no-coverage 2>&1 | tail -10
```

Expected: `Cannot find module '../parseSopExcel'`

- [ ] **Step 3: parseSopExcel.ts 구현**

`frontend/src/pages/SOPDocuments/parseSopExcel.ts` 생성:

```typescript
import * as XLSX from 'xlsx';
import SHA256 from 'crypto-js/sha256';
import {
	SOP_DOCUMENT_CONTRACT_VERSION,
	type SopApprovalStatus,
	type SopDocument,
} from 'api/v2/rules/sopDocuments';

const REQUIRED_COLUMNS = [
	'sop_id',
	'title',
	'version',
	'owner_team',
	'project_ids',
	'environments',
	'body_markdown',
] as const;

const ALLOWED_APPROVAL_STATUSES: SopApprovalStatus[] = [
	'approved',
	'draft',
	'deprecated',
	'disabled',
];

export type ParsedSopRow = {
	rowIndex: number;
	valid: boolean;
	error?: string;
	document?: SopDocument;
	raw: Record<string, string>;
};

export type ParseSopExcelResult = {
	rows: ParsedSopRow[];
	validCount: number;
	errorCount: number;
};

function parseTags(value: string): string[] {
	if (!value) return [];
	return value
		.split(',')
		.map((t) => t.trim())
		.filter(Boolean);
}

function checksumForMarkdown(bodyMarkdown: string): string {
	return `sha256:${SHA256(bodyMarkdown).toString()}`;
}

export function parseSopRows(rows: Record<string, string>[]): ParseSopExcelResult {
	const parsed: ParsedSopRow[] = rows.map((row, idx) => {
		const missingFields = REQUIRED_COLUMNS.filter(
			(col) => !String(row[col] ?? '').trim(),
		);
		if (missingFields.length > 0) {
			return {
				rowIndex: idx + 2,
				valid: false,
				error: `필수 필드 누락: ${missingFields.join(', ')}`,
				raw: row,
			};
		}

		const approvalStatus = (
			String(row.approval_status ?? '').trim() || 'approved'
		) as SopApprovalStatus;
		if (!ALLOWED_APPROVAL_STATUSES.includes(approvalStatus)) {
			return {
				rowIndex: idx + 2,
				valid: false,
				error: `approval_status 허용 값: ${ALLOWED_APPROVAL_STATUSES.join(', ')}`,
				raw: row,
			};
		}

		const bodyMarkdown = String(row.body_markdown ?? '').trim();
		const document: SopDocument = {
			contractVersion: SOP_DOCUMENT_CONTRACT_VERSION,
			sopId: String(row.sop_id).trim(),
			title: String(row.title).trim(),
			version: String(row.version).trim(),
			checksum: checksumForMarkdown(bodyMarkdown),
			source: {
				type: 'managed_markdown',
				sourceId:
					String(row.source_id ?? '').trim() || 'src-managed-markdown-default',
			},
			bodyMarkdown,
			displayUrl: String(row.display_url ?? '').trim() || undefined,
			ownerTeam: String(row.owner_team).trim(),
			approvalStatus,
			tenantScope: {
				projectIds: parseTags(String(row.project_ids ?? '')),
				environments: parseTags(String(row.environments ?? '')),
			},
			tags: parseTags(String(row.tags ?? '')),
			updatedAt: new Date().toISOString(),
			securityContext: {
				serviceAccountProfile:
					String(row.service_account_profile ?? '').trim() ||
					'managed-markdown-local',
				secretRefVisible: false,
				browserCredentialsUsed: false,
				redactionApplied: true,
			},
		};

		return { rowIndex: idx + 2, valid: true, document, raw: row };
	});

	return {
		rows: parsed,
		validCount: parsed.filter((r) => r.valid).length,
		errorCount: parsed.filter((r) => !r.valid).length,
	};
}

export function parseSopExcel(file: File): Promise<ParseSopExcelResult> {
	return new Promise((resolve, reject) => {
		const reader = new FileReader();
		reader.onload = (e): void => {
			try {
				const data = e.target?.result;
				const workbook = XLSX.read(data, { type: 'binary' });
				const sheetName = workbook.SheetNames[0];
				const sheet = workbook.Sheets[sheetName];
				const rows = XLSX.utils.sheet_to_json<Record<string, string>>(sheet, {
					raw: false,
				});

				if (rows.length === 0) {
					resolve({ rows: [], validCount: 0, errorCount: 0 });
					return;
				}

				const missingColumns = REQUIRED_COLUMNS.filter(
					(col) => !(col in rows[0]),
				);
				if (missingColumns.length > 0) {
					reject(
						new Error(`필수 컬럼 누락: ${missingColumns.join(', ')}`),
					);
					return;
				}

				resolve(parseSopRows(rows));
			} catch (err) {
				reject(err instanceof Error ? err : new Error('파일 파싱 실패'));
			}
		};
		reader.onerror = (): void => reject(new Error('파일 읽기 실패'));
		reader.readAsBinaryString(file);
	});
}

export function downloadSopExcelTemplate(): void {
	const headers = [
		'sop_id',
		'title',
		'version',
		'owner_team',
		'approval_status',
		'source_id',
		'project_ids',
		'environments',
		'display_url',
		'tags',
		'service_account_profile',
		'body_markdown',
	];
	const example = [
		'SOP-PAY-001',
		'Payment API 5xx response',
		'2026-06-01.1',
		'payments',
		'approved',
		'src-managed-markdown-default',
		'customer-a',
		'prod',
		'https://kb.example/sop/SOP-PAY-001',
		'payment-api,critical',
		'managed-markdown-local',
		'# Payment API 5xx response\n\n1. Check payment dashboard\n2. Inspect PG timeout logs',
	];

	const wb = XLSX.utils.book_new();
	const ws = XLSX.utils.aoa_to_sheet([headers, example]);
	XLSX.utils.book_append_sheet(wb, ws, 'SOP Template');
	XLSX.writeFile(wb, 'sop-template.xlsx');
}
```

- [ ] **Step 4: 테스트 통과 확인**

```bash
cd frontend && npx jest parseSopExcel --no-coverage
```

Expected: 6개 테스트 모두 PASS

- [ ] **Step 5: 커밋**

```bash
git add frontend/src/pages/SOPDocuments/parseSopExcel.ts frontend/src/pages/SOPDocuments/__tests__/parseSopExcel.test.ts
git commit -m "feat(sop): add parseSopExcel util with row-level validation"
```

---

## Task 7: SopBulkUploadModal 컴포넌트

**Files:**
- Create: `frontend/src/pages/SOPDocuments/SopBulkUploadModal.tsx`

- [ ] **Step 1: 컴포넌트 생성**

`frontend/src/pages/SOPDocuments/SopBulkUploadModal.tsx` 생성:

```tsx
import { useState } from 'react';
import { Alert, Modal, Upload } from 'antd';
import { InboxOutlined } from '@ant-design/icons';
import type { UploadFile } from 'antd/es/upload/interface';
import { parseSopExcel, type ParseSopExcelResult } from './parseSopExcel';

type Props = {
	open: boolean;
	onClose: () => void;
	onParsed: (result: ParseSopExcelResult) => void;
};

function SopBulkUploadModal({ open, onClose, onParsed }: Props): JSX.Element {
	const [parsing, setParsing] = useState(false);
	const [parseError, setParseError] = useState('');

	const handleFile = async (file: File): Promise<false> => {
		if (!file.name.endsWith('.xlsx')) {
			setParseError('.xlsx 파일만 지원합니다.');
			return false;
		}
		setParsing(true);
		setParseError('');
		try {
			const result = await parseSopExcel(file);
			onParsed(result);
		} catch (err) {
			setParseError(err instanceof Error ? err.message : '파일 파싱 실패');
		} finally {
			setParsing(false);
		}
		return false; // prevent antd auto-upload
	};

	const handleClose = (): void => {
		setParseError('');
		onClose();
	};

	return (
		<Modal
			confirmLoading={parsing}
			footer={null}
			onCancel={handleClose}
			open={open}
			title="SOP 일괄 업로드"
			width={480}
		>
			{parseError && (
				<Alert
					message={parseError}
					showIcon
					style={{ marginBottom: 16 }}
					type="error"
				/>
			)}
			<Upload.Dragger
				accept=".xlsx"
				beforeUpload={handleFile}
				fileList={[] as UploadFile[]}
				multiple={false}
				showUploadList={false}
			>
				<p className="ant-upload-drag-icon">
					<InboxOutlined />
				</p>
				<p className="ant-upload-text">
					.xlsx 파일을 드래그하거나 클릭해서 선택
				</p>
				<p className="ant-upload-hint">
					헤더 행 포함 Excel 파일. 템플릿을 먼저 다운로드하세요.
				</p>
			</Upload.Dragger>
		</Modal>
	);
}

export default SopBulkUploadModal;
```

- [ ] **Step 2: TypeScript 컴파일 확인**

```bash
cd frontend && npx tsc --noEmit 2>&1 | grep SopBulkUploadModal
```

Expected: 출력 없음

- [ ] **Step 3: 커밋**

```bash
git add frontend/src/pages/SOPDocuments/SopBulkUploadModal.tsx
git commit -m "feat(sop): add SopBulkUploadModal file drop component"
```

---

## Task 8: SopBulkPreviewDrawer 컴포넌트

**Files:**
- Create: `frontend/src/pages/SOPDocuments/SopBulkPreviewDrawer.tsx`

- [ ] **Step 1: 컴포넌트 생성**

`frontend/src/pages/SOPDocuments/SopBulkPreviewDrawer.tsx` 생성:

```tsx
import { useCallback, useState } from 'react';
import { Alert, Button, Drawer, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
	createSopDocumentBatch,
	SOP_DOCUMENT_LIST_CONTRACT_VERSION,
	type SopDocumentBatchResult,
} from 'api/v2/rules/sopDocuments';
import type { ParseSopExcelResult, ParsedSopRow } from './parseSopExcel';

type Props = {
	open: boolean;
	parseResult: ParseSopExcelResult | null;
	onClose: () => void;
	onRegistered: () => void;
};

type RowWithResult = ParsedSopRow & { batchResult?: SopDocumentBatchResult };

function getErrorMessage(error: unknown): string {
	if (typeof error === 'object' && error !== null && 'response' in error) {
		const response = (
			error as { response?: { data?: { error?: string; message?: string } | string } }
		).response;
		if (typeof response?.data === 'string') return response.data;
		return response?.data?.error || response?.data?.message || '요청 실패';
	}
	return error instanceof Error ? error.message : '요청 실패';
}

function SopBulkPreviewDrawer({
	open,
	parseResult,
	onClose,
	onRegistered,
}: Props): JSX.Element {
	const [submitting, setSubmitting] = useState(false);
	const [submitError, setSubmitError] = useState('');
	const [rowResults, setRowResults] = useState<Map<number, SopDocumentBatchResult>>(
		new Map(),
	);

	const rows: RowWithResult[] = (parseResult?.rows ?? []).map((row) => ({
		...row,
		batchResult: rowResults.get(row.rowIndex),
	}));

	const validDocs = rows.filter((r) => r.valid).map((r) => r.document!);

	const handleRegister = useCallback(async (): Promise<void> => {
		setSubmitting(true);
		setSubmitError('');
		try {
			const response = await createSopDocumentBatch({
				contractVersion: SOP_DOCUMENT_LIST_CONTRACT_VERSION,
				documents: validDocs,
			});
			const resultMap = new Map<number, SopDocumentBatchResult>();
			const validRows = rows.filter((r) => r.valid);
			response.data.results.forEach((res, idx) => {
				if (validRows[idx]) {
					resultMap.set(validRows[idx].rowIndex, res);
				}
			});
			setRowResults(resultMap);
			onRegistered();
		} catch (err) {
			setSubmitError(getErrorMessage(err));
		} finally {
			setSubmitting(false);
		}
	}, [validDocs, rows, onRegistered]);

	const handleClose = (): void => {
		setSubmitError('');
		setRowResults(new Map());
		onClose();
	};

	const columns: ColumnsType<RowWithResult> = [
		{
			title: '행',
			dataIndex: 'rowIndex',
			key: 'rowIndex',
			width: 50,
		},
		{
			title: 'SOP ID',
			key: 'sopId',
			render: (_, record): string => record.document?.sopId ?? record.raw.sop_id ?? '',
		},
		{
			title: 'Title',
			key: 'title',
			render: (_, record): string => record.document?.title ?? record.raw.title ?? '',
		},
		{
			title: 'Version',
			key: 'version',
			width: 120,
			render: (_, record): string =>
				record.document?.version ?? record.raw.version ?? '',
		},
		{
			title: 'Owner',
			key: 'owner',
			width: 110,
			render: (_, record): string =>
				record.document?.ownerTeam ?? record.raw.owner_team ?? '',
		},
		{
			title: '상태',
			key: 'status',
			width: 120,
			render: (_, record): JSX.Element => {
				if (record.batchResult) {
					return record.batchResult.status === 'ok' ? (
						<Tag color="green">등록 완료</Tag>
					) : (
						<Tag color="red">서버 오류</Tag>
					);
				}
				return record.valid ? (
					<Tag color="blue">유효</Tag>
				) : (
					<Tag color="red">오류</Tag>
				);
			},
		},
		{
			title: '오류 내용',
			key: 'error',
			render: (_, record): string | undefined =>
				record.batchResult?.error ?? record.error,
		},
	];

	const registeredCount = [...rowResults.values()].filter(
		(r) => r.status === 'ok',
	).length;
	const serverErrorCount = [...rowResults.values()].filter(
		(r) => r.status === 'error',
	).length;

	return (
		<Drawer
			extra={
				rowResults.size === 0 ? (
					<Button
						disabled={validDocs.length === 0}
						loading={submitting}
						onClick={handleRegister}
						type="primary"
					>
						{validDocs.length}건 일괄 등록
					</Button>
				) : null
			}
			onClose={handleClose}
			open={open}
			size="large"
			title="SOP 일괄 업로드 미리보기"
			width={900}
		>
			<div style={{ marginBottom: 12 }}>
				{rowResults.size === 0 ? (
					<Typography.Text type="secondary">
						유효 {parseResult?.validCount ?? 0}건 / 오류{' '}
						{parseResult?.errorCount ?? 0}건
					</Typography.Text>
				) : (
					<Typography.Text>
						등록 완료 {registeredCount}건 / 서버 오류 {serverErrorCount}건
					</Typography.Text>
				)}
			</div>
			{submitError && (
				<Alert
					message={submitError}
					showIcon
					style={{ marginBottom: 12 }}
					type="error"
				/>
			)}
			<Table
				columns={columns}
				dataSource={rows}
				pagination={false}
				rowKey="rowIndex"
				rowClassName={(record): string =>
					!record.valid ||
					(record.batchResult && record.batchResult.status === 'error')
						? 'sop-preview-row--error'
						: ''
				}
				scroll={{ x: 800 }}
				size="small"
			/>
		</Drawer>
	);
}

export default SopBulkPreviewDrawer;
```

- [ ] **Step 2: TypeScript 컴파일 확인**

```bash
cd frontend && npx tsc --noEmit 2>&1 | grep SopBulkPreviewDrawer
```

Expected: 출력 없음

- [ ] **Step 3: 커밋**

```bash
git add frontend/src/pages/SOPDocuments/SopBulkPreviewDrawer.tsx
git commit -m "feat(sop): add SopBulkPreviewDrawer with per-row result display"
```

---

## Task 9: SOPDocuments.tsx 연결

**Files:**
- Modify: `frontend/src/pages/SOPDocuments/SOPDocuments.tsx`
- Modify: `frontend/src/pages/SOPDocuments/SOPDocuments.styles.scss`

- [ ] **Step 1: import 추가**

`SOPDocuments.tsx` 상단 import 섹션에 추가:

```tsx
import { DownloadOutlined, UploadOutlined } from '@ant-design/icons';
import { downloadSopExcelTemplate, type ParseSopExcelResult } from './parseSopExcel';
import SopBulkUploadModal from './SopBulkUploadModal';
import SopBulkPreviewDrawer from './SopBulkPreviewDrawer';
```

- [ ] **Step 2: 상태 추가**

`SOPDocuments` 함수 내부 기존 `useState` 블록 바로 뒤에 추가:

```tsx
const [uploadModalOpen, setUploadModalOpen] = useState(false);
const [previewDrawerOpen, setPreviewDrawerOpen] = useState(false);
const [parseResult, setParseResult] = useState<ParseSopExcelResult | null>(null);
```

- [ ] **Step 3: 파싱 완료 핸들러 추가**

`handleCreateDocument` 함수 뒤에 추가:

```tsx
const handleParsed = useCallback((result: ParseSopExcelResult): void => {
    setParseResult(result);
    setUploadModalOpen(false);
    setPreviewDrawerOpen(true);
}, []);

const handleRegistered = useCallback((): void => {
    void loadDocuments();
}, [loadDocuments]);
```

- [ ] **Step 4: 헤더에 버튼 추가**

`<header className="sop-documents-page__header">` 블록을 다음으로 교체:

```tsx
<header className="sop-documents-page__header">
    <div className="sop-documents-page__header-row">
        <div>
            <h1>DS-APM SOP documents</h1>
            <p>
                Register managed Markdown SOPs that SigNoz alert rules can bind with
                <code>sop_id</code> and feed into SOP-grounded AI response strategy.
            </p>
        </div>
        <div className="sop-documents-page__header-actions">
            <Button
                icon={<DownloadOutlined />}
                onClick={downloadSopExcelTemplate}
            >
                템플릿 다운로드
            </Button>
            <Button
                icon={<UploadOutlined />}
                onClick={(): void => setUploadModalOpen(true)}
                type="primary"
            >
                파일 업로드
            </Button>
        </div>
    </div>
</header>
```

- [ ] **Step 5: 모달/드로어 컴포넌트 추가**

`</div>` (최종 닫는 div) 바로 앞에 추가:

```tsx
<SopBulkUploadModal
    onClose={(): void => setUploadModalOpen(false)}
    onParsed={handleParsed}
    open={uploadModalOpen}
/>
<SopBulkPreviewDrawer
    onClose={(): void => setPreviewDrawerOpen(false)}
    onRegistered={handleRegistered}
    open={previewDrawerOpen}
    parseResult={parseResult}
/>
```

- [ ] **Step 6: SCSS에 헤더 레이아웃 추가**

`SOPDocuments.styles.scss`의 `&__header` 블록 아래에 추가:

```scss
&__header-row {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--spacing-8);
}

&__header-actions {
    display: flex;
    gap: var(--spacing-4);
    flex-shrink: 0;
}

.sop-preview-row--error td {
    background: rgba(255, 77, 79, 0.06) !important;
}
```

- [ ] **Step 7: TypeScript 컴파일 확인**

```bash
cd frontend && npx tsc --noEmit 2>&1 | grep SOPDocuments
```

Expected: 출력 없음

- [ ] **Step 8: 기존 SOPDocuments 테스트 통과 확인**

```bash
cd frontend && npx jest SOPDocuments --no-coverage
```

Expected: 기존 테스트 모두 PASS

- [ ] **Step 9: 커밋**

```bash
git add frontend/src/pages/SOPDocuments/SOPDocuments.tsx frontend/src/pages/SOPDocuments/SOPDocuments.styles.scss
git commit -m "feat(sop): wire bulk upload modal + preview drawer into SOPDocuments page"
```

---

## 완료 검증

- [ ] 백엔드 전체 테스트 통과: `go test ./pkg/ruler/... -v 2>&1 | tail -5`
- [ ] 프론트엔드 전체 테스트 통과: `cd frontend && npx jest --no-coverage 2>&1 | tail -10`
- [ ] TypeScript 오류 없음: `cd frontend && npx tsc --noEmit`
