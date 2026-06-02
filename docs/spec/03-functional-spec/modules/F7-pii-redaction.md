---
id: F7
title: PII Redaction
status: planned
commits: [3e9dfa557]
source_paths:
  - pkg/types/alertmanagertypes/incident_payload.go
implements_uc: [UC-001]
covered_by_wbs: [WBS-1.4]
updated: 2026-06-02
caveats: "README 명시: PII 처리 production-ready 아님 — AIOpsAgent ingress 단일 지점 redaction, OTel Collector 단계 미적용"
---

# F7 — PII Redaction

> **상태**: 착수 예정 (착수보고 기준)
> Incident payload 내 email / KR phone / long unmarked secret / sensitive URL query를 redaction.
> **README 경고**: production-ready 아님. AIOpsAgent ingress 단일 지점이며, OTel Collector 단계의 가장 이른 redaction은 아직 미적용.

## 책임 (Responsibility)

channel adapter 호출 이전에 incident payload의 4종 sensitive pattern을 제거한다. `BuildSafeIncidentInfo` / `SanitizeIncidentInfo`가 channel template 직전에 호출되어 sanitized value만 외부로 나간다. AI Engine 도달 전 redaction이 완료되어야 한다 (UC-001 Minimal Guarantee).

## 인터페이스 요지

```go
// pkg/types/alertmanagertypes/incident_payload.go
func BuildSafeIncidentInfo(labels, annotations template.KV) IncidentInfo
func SanitizeIncidentValue(value string) string
func IncidentInfoFields(info IncidentInfo) []IncidentField  // sanitize 후 non-empty만
```

`SanitizeIncidentValue` 처리 순서: URL sensitive query drop → secret-looking 전체 drop → email regex → KR phone regex → long secret regex. Regex 4종은 compile 1회 global var. 상세 패턴은 `incident_payload.go` 참조.

## 핵심 동작

| 카테고리 | 전략 | 결과 토큰 |
|---|---|---|
| Email 주소 | regex replace | `[redacted-email]` |
| 한국 모바일 번호 | regex replace | `[redacted-phone]` |
| Long alnum secret (32+ chars) | regex replace | `[redacted-secret]` |
| Bearer / JWT / secret marker | 값 전체 drop | `[redacted]` |
| URL sensitive query key | parameter 삭제, URL 보존 | sanitized URL |

URL의 `User` info(`user:pw@host`)는 항상 제거된다. 대상 필드 22개(`IncidentInfo` struct 전체). Production 강화 권장: OTel Collector `transform / redaction / filter` processor 3단 방어 (현재 ingress 1단계만).

## 예외·복구

| 경로 | 처리 |
|---|---|
| 입력 값 empty | 그대로 반환 |
| URL 파싱 실패 / non-http(s) scheme | URL 처리 skip, regex만 적용 |
| 모든 필드 empty | `IncidentInfoFields` → empty slice |

Open: Redaction rate metric 노출 + meta-alert, OTel Collector 단계 도입 — follow-up.

## Acceptance Criteria

```gherkin
Feature: Incident payload PII redaction

  Scenario: Email is replaced with placeholder
    Given a field value "Contact ops@example.com for details"
    When SanitizeIncidentValue is called
    Then the result equals "Contact [redacted-email] for details"

  Scenario: Bearer token replaces the entire value
    Given a field value "Authorization: Bearer abcdefghij"
    When SanitizeIncidentValue is called
    Then the result equals "[redacted]"

  Scenario: Sensitive URL query is dropped but URL is preserved
    Given a field value "https://signoz.example.com/api?token=xyz&svc=payment"
    When SanitizeIncidentValue is called
    Then the result does not contain "token=" and contains "svc=payment"
```

## Traceability
- Implements UC: UC-001 (단계 3)
- Covered by WBS: WBS-1.4
- Source: `pkg/types/alertmanagertypes/incident_payload.go`
- Commits: `3e9dfa557`
