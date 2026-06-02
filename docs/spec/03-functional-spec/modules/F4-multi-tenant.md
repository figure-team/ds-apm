---
id: F4
title: Multi-tenant Scope
status: planned
commits: [3fa604e03]
source_paths:
  - pkg/types/ruletypes/tenant_policy.go
implements_uc: [UC-001]
covered_by_wbs: [WBS-1.0]
updated: 2026-06-02
caveats: "README 명시: 멀티테넌트 격리 production-ready 아님 — label-based filter일 뿐 row-level security 아님"
---

# F4 — Multi-tenant Scope

> **상태**: 착수 예정 (착수보고 기준)
> SOP / AI strategy 접근을 `project_id × environment` tenant scope로 필터링.
> **README 경고**: production-ready 아님. label-based filter 수준이며 row-level security가 아니다.

## 책임 (Responsibility)

SOP grounding과 AI strategy generation 단계에서 alert label의 tenant identity(`project_id`, `environment`)가 SOP `TenantScope`와 일치하는지 검사하는 stateless predicate를 제공한다. 불일치 시 `forbidden / denied / blocked_by_policy` 중 호출자가 적절한 상태로 매핑한다. cross-tenant lookup은 `ErrSOPDocumentNotFound`로 통일하여 존재 여부를 누설하지 않는다.

## 인터페이스 요지

```go
// pkg/types/ruletypes/tenant_policy.go
func PilotTenantFromLabels(labels map[string]string) PilotAuditTenant
func PilotTenantScopeAllows(scope PilotTenantScope, tenant PilotAuditTenant) bool

const SOPTenantPolicyDeniedWarning = "sop document is outside requested tenant scope"
```

매칭 규칙: `ProjectIDs`와 `Environments` 각각에서 정확 일치 또는 `"*"` 와일드카드. 정규식·glob 미지원. 상세는 `pkg/types/ruletypes/tenant_policy.go` 참조.

## 핵심 동작

입력: alert labels → `PilotAuditTenant{ProjectID, Environment}` 추출.

처리: `PilotTenantIsComplete` 검사 → false면 전면 deny. `PilotTenantScopeAllows`로 SOP scope와 대조.

출력: `bool` — `false`이면 호출자가 `forbidden / denied / blocked_by_policy`로 매핑.

Scope normalization: trim + 빈 값 제거 + 중복 제거. `ProjectIDs` 또는 `Environments` 중 하나라도 빈 리스트면 validator가 거부.

**Production 격차** (README 명시): DB row-level security 없음, `project_id` spoofing 방어 없음. production 등급에는 JWT-derived tenant claim 검증 필요.

## 예외·복구

| 경로 | 처리 |
|---|---|
| `project_id` 또는 `environment` label 누락 | `PilotTenantIsComplete=false` → deny |
| Scope에 빈 리스트 | validator가 거부 |
| Cross-tenant `SOPStore.Get` | `ErrSOPDocumentNotFound` (존재 누설 없음) |

## Acceptance Criteria

```gherkin
Feature: Multi-tenant scope enforcement
  Background:
    Given a SOPDocument with TenantScope { ProjectIDs: ["p-prod"], Environments: ["production"] }

  Scenario: Exact match is allowed
    Given an alert with project_id="p-prod" environment="production"
    When PilotTenantScopeAllows is called
    Then the result is true

  Scenario: Missing project_id label denies access
    Given an alert with environment="production" and no project_id label
    When PilotTenantScopeAllows is called
    Then the result is false
```

## Traceability
- Implements UC: UC-001 (전제)
- Covered by WBS: WBS-1.0
- Source: `pkg/types/ruletypes/tenant_policy.go`
- Commits: `3fa604e03`
- Open: DB row-level security, JWT-derived tenant claim, glob/regex scope — follow-up milestone
