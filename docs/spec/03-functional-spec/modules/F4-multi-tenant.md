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

## F4.1 개요

본 모듈은 SOP grounding과 AI strategy generation 단계에서 alert label에 박힌 tenant identity (`project_id`, `environment`)가 SOP document의 `TenantScope`와 일치하는지 검사한다. 매칭 정책은 **단순 list 포함 + 와일드카드 `"*"`** 지원이며, 매칭 실패 시:

- SOP grounding → `SOPBindingStatusForbidden` 반환
- Managed markdown fetch → `PilotSOPFetchStatusDenied` + audit outcome `denied`
- AI strategy generation → `AIStrategyStatusBlockedByPolicy`

**production-readiness 격차** (README 명시):
- DB row-level security 없음 — `SOPStore`가 `orgID` partition만 강제, tenant scope는 application layer filter
- `project_id` label spoofing 방어 없음 — alertmanager가 신뢰 가능한 source인지에 의존
- `Environments` list의 와일드카드 `"*"`는 단순 string match. 정규식·glob 미지원

## F4.2 인터페이스

```go
// pkg/types/ruletypes/tenant_policy.go
func PilotTenantFromLabels(labels map[string]string) PilotAuditTenant
func PilotTenantIsComplete(tenant PilotAuditTenant) bool
func PilotTenantScopeAllows(scope PilotTenantScope, tenant PilotAuditTenant) bool

const (
    SOPTenantPolicyMissingLabelsWarning = "project_id and environment labels are required for SOP tenant policy"
    SOPTenantPolicyDeniedWarning        = "sop document is outside requested tenant scope"
)
```

## F4.3 데이터 모델

```go
type PilotAuditTenant struct {
    ProjectID   string  // label "project_id" — trim된 값
    Environment string  // label "environment"
}

type PilotTenantScope struct {
    ProjectIDs   []string  // "*" 허용
    Environments []string  // "*" 허용
}
```

매칭 규칙 (`PilotTenantScopeAllows`):

```go
return PilotTenantIsComplete(tenant) &&
    contains(scope.ProjectIDs,   tenant.ProjectID)  &&
    contains(scope.Environments, tenant.Environment)
// contains: 정확 일치 또는 candidate == "*"
```

Normalization (`normalizePilotTenantScope`): trim + 빈 값 제거 + 중복 제거. 둘 다 최소 1개 entry 필수 (validator).

## F4.4 상태 전이

해당 없음. 본 모듈은 stateless predicate.

## F4.5 예외 및 복구

| 경로 | 처리 |
|---|---|
| `project_id` 또는 `environment` label 누락 | `PilotTenantIsComplete()=false` → 모든 scope 검사 deny |
| Scope mismatch | `forbidden` / `denied` / `blocked_by_policy` 중 호출자가 매핑한 상태로 반환 |
| Scope에 `ProjectIDs=[]` 또는 `Environments=[]` | validator가 거부 (`"must include at least one"`) |
| `"*"` 와일드카드 | 모든 값과 일치로 처리 |

## F4.6 비기능 요건 (NFR)

- **NF-F4.1** Tenant scope 위반 시 호출자에게 SOP 존재 여부를 누설하지 않아야 한다 (cross-tenant `Get`은 `ErrSOPDocumentNotFound`로 통일 — F1.6 참조).
- **NF-F4.2** Tenant scope 매칭은 stateless하고 deterministic해야 한다 — 같은 입력은 같은 결과.
- **NF-F4.3** Production 등급 격리에는 (a) DB row-level security 또는 (b) JWT-derived tenant claim 검증이 필요하다. 현재 MVP는 둘 다 미구현.

## F4.7 Acceptance Criteria (Gherkin)

```gherkin
Feature: Multi-tenant scope enforcement
  Background:
    Given a SOPDocument with TenantScope { ProjectIDs: ["p-prod"], Environments: ["production"] }

  Scenario: Exact match is allowed
    Given an alert with labels project_id="p-prod", environment="production"
    When PilotTenantScopeAllows is called
    Then the result is true

  Scenario: Wildcard environment matches any
    Given a SOPDocument with Environments ["*"]
    And an alert with environment="staging"
    When PilotTenantScopeAllows is called
    Then the result is true

  Scenario: Missing project_id label denies access
    Given an alert with environment="production" and no project_id label
    When PilotTenantScopeAllows is called
    Then the result is false

  Scenario: Cross-tenant lookup does not reveal existence
    Given a SOP exists for tenant "p-prod"
    When tenant "p-other" issues SOPStore.Get
    Then ErrSOPDocumentNotFound is returned
```

## F4.8 Traceability
- Implements UC: UC-001 (전제)
- Covered by WBS: WBS-1.0
- Source: `pkg/types/ruletypes/tenant_policy.go`
- Commits: `3fa604e03`

## F4.9 Open Items
- README에 명시된 production-readiness 격차:
  - DB row-level security 도입 (현재 application-layer filter)
  - JWT-derived tenant claim 검증 (현재 alertmanager 신뢰)
  - `Environments` glob/regex 지원 (현재 plain string + `"*"`만)
