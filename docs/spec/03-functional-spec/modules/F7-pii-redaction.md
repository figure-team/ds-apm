---
id: F7
title: PII Redaction
status: implemented-mvp
commits: [3e9dfa557]
source_paths:
  - pkg/types/alertmanagertypes/incident_payload.go
implements_uc: [UC-001]
covered_by_wbs: [WBS-1.4]
updated: 2026-05-29
caveats: "README 명시: PII 처리 production-ready 아님 — DS-APM ingress 단일 지점 redaction, OTel Collector 단계 미적용"
---

# F7 — PII Redaction

> **상태**: 구현 완료 (early MVP)
> Incident payload 내 email / KR phone / long unmarked secret / sensitive URL query를 redaction.
> **README 경고**: production-ready 아님. 본 모듈은 DS-APM ingress의 단일 지점 redaction이며, OpenTelemetry Collector 단계의 가장 이른 redaction (권장 위치)은 아직 미적용.

## F7.1 개요

본 모듈은 incident payload (Slack / MSTeams / PagerDuty / webhook / email로 흘러나가는 사용자-가시 필드)에서 4종 sensitive pattern을 redaction한다.

| 카테고리 | 전략 | 결과 토큰 |
|---|---|---|
| Email 주소 | regex replace | `[redacted-email]` |
| 한국 모바일 번호 (`010xxxx`, `+82-10-xxx` 등) | regex replace | `[redacted-phone]` |
| Long unmarked secret (32+ chars alnum/`_`/`-`) | regex replace | `[redacted-secret]` |
| Bearer token / `client_secret`/`api_key`/`password` marker / JWT-like | 전체 값 drop | `[redacted]` |
| URL query 안의 sensitive key (`access_token`, `api_key`, `bearer`, …) | parameter delete (key 보존, value 제거) | sanitized URL |

OpenTelemetry security 가이드(§9 in research-skills-c-domain.md)의 4-processor 정책(`Attributes / Filter / Redaction / Transform`)에 비추어 보면, 현재 구현은 **DS-APM ingress 단계의 `Transform` (regex) + 부분적 `Attributes` (URL key drop)에 해당**. 가장 이른 단계인 instrumentation / OTel Collector 적용은 아직 없음.

## F7.2 인터페이스

```go
// pkg/types/alertmanagertypes/incident_payload.go
const RedactedIncidentValue = "[redacted]"

func BuildSafeIncidentInfo(labels, annotations template.KV) IncidentInfo
func SanitizeIncidentInfo(info IncidentInfo) IncidentInfo
func SanitizeIncidentValue(value string) string

func IncidentInfoFields(info IncidentInfo) []IncidentField     // sanitize 후 non-empty만
func IncidentInfoDetails(info IncidentInfo) map[string]string  // sanitize 후 key→value map
```

`SanitizeIncidentValue` 처리 순서:
1. URL 파싱 시도 — `http`/`https`만 허용, `User` info drop, sensitive query key drop
2. 값 전체가 secret-looking (`bearer `, `-----begin `, JWT, marker 포함) → `[redacted]` 전체 치환
3. Email regex replace
4. KR phone regex replace
5. Long secret regex replace

## F7.3 데이터 모델

```go
type IncidentField struct {
    Key   string
    Title string
    Value string  // 항상 sanitized
    Short bool
}

// Sanitize 대상 필드 (22개, IncidentInfo struct):
//   ProjectID, Environment, ServiceName, OwnerTeam, Severity,
//   ImpactSummary, NextAction, VendorRequest, CustomerUpdate,
//   SopID, SopURL, SopSource, SopTitle, SopVersion, SopBindingID,
//   AIStrategyID, AIStrategyStatus, AIHeadline, AIFirstActions,
//   AIConfidence, AILimitations, AIEvidenceRefs
```

Regex 패턴 (compiled once):

```go
incidentValueEmailPattern        = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
incidentValueKoreanMobilePattern = regexp.MustCompile(`(?:\+?82[-\s]?)?0?1[016789][-)\s]?\d{3,4}[-\s]?\d{4}`)
incidentValueLongSecretPattern   = regexp.MustCompile(`\b[A-Za-z0-9_\-]{32,}\b`)
incidentJWTLikePattern           = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{10,}\b`)
```

URL sensitive keys: `access_token | api_key | apikey | auth | authorization | bearer | client_secret | password | secret | token`.

## F7.4 파이프라인 위치

- **현재 (v0.1)** — DS-APM ingress 단일 지점. `BuildSafeIncidentInfo` / `SanitizeIncidentInfo`가 channel template 직전에 호출됨.
- **권장 (research §9.1)** — `instrumentation > OTel Collector > DS-APM ingress` 3단 방어. 가장 이른 단계가 최선.
- **OTel Collector processor 매핑 (미구현, 권장)**:

| Category | OTel Processor | DS-APM 대응 |
|---|---|---|
| Email / Phone / Long secret regex | `transform` | `SanitizeIncidentValue` (현재) |
| Sensitive URL query | `attributes` (delete) | `sanitizeIncidentURL` (현재) |
| Bearer / JWT 전체 drop | `redaction` (allowlist) | secret-looking check (현재) |
| 잠재적 sensitive span 차단 | `filter` | 미구현 |

## F7.5 예외 및 복구

| 경로 | 처리 |
|---|---|
| 입력 값 empty | 그대로 반환 |
| URL 파싱 실패 또는 non-http(s) scheme | URL 처리 skip, 일반 regex만 적용 |
| Secret-looking 패턴 매칭 | 값 전체를 `[redacted]`로 교체 (early return) |
| Regex 매칭 0건 | 값 그대로 반환 |
| `IncidentInfoFields` 호출 시 모든 필드 empty | empty slice (`[]IncidentField{}`) |

## F7.6 비기능 요건 (NFR)

- **NF-F7.1** Redaction 적용은 channel adapter 호출 **이전**에 완료되어야 한다 (sanitized value만 외부로 나감).
- **NF-F7.2** Regex 패턴은 컴파일 1회 + global var (allocation 0).
- **NF-F7.3** Email / phone / long secret redaction은 **부분 치환** (값의 나머지 보존). Secret-looking 검출은 **전체 drop**.
- **NF-F7.4** URL의 `User` info는 항상 제거되어야 한다 (e.g., `https://user:pw@host` → `https://host`).
- **NF-F7.5** Redaction rate metric (per-category count)을 노출해서 threshold 초과 시 meta-alert 트리거 가능해야 한다 — 현재 미구현, follow-up.

## F7.7 Acceptance Criteria (Gherkin)

```gherkin
Feature: Incident payload PII redaction
  Scenario: Email is replaced with placeholder
    Given a field value "Contact ops@example.com for details"
    When SanitizeIncidentValue is called
    Then the result equals "Contact [redacted-email] for details"

  Scenario: Korean mobile number is redacted
    Given a field value "긴급 010-1234-5678"
    When SanitizeIncidentValue is called
    Then the result equals "긴급 [redacted-phone]"

  Scenario: Bearer token replaces the entire value
    Given a field value "Authorization: Bearer abcdefghij"
    When SanitizeIncidentValue is called
    Then the result equals "[redacted]"

  Scenario: Sensitive URL query is dropped but URL is preserved
    Given a field value "https://signoz.example.com/api?token=xyz&svc=payment"
    When SanitizeIncidentValue is called
    Then the result does not contain "token="
    And the result contains "svc=payment"

  Scenario: Long alnum secret is redacted
    Given a field value "key=AKIAIOSFODNN7EXAMPLEKEYABCDEFGHIJKL"
    When SanitizeIncidentValue is called
    Then the result contains "[redacted-secret]"
```

## F7.8 Traceability
- Implements UC: UC-001 (단계 3)
- Covered by WBS: WBS-1.4
- Source: `pkg/types/alertmanagertypes/incident_payload.go`
- Commits: `3e9dfa557`

## F7.9 Open Items
- production-readiness 강화 (README 경고):
  - OTel Collector 단계 (`transform`/`redaction`/`filter` processor) 도입
  - Redaction rate metric 노출 + threshold 초과 시 meta-alert
  - 카테고리 확장 (credit card, IP truncation, generic user_id hashing 등 — research-skills-c-domain.md §9.2)
  - Hash 전략의 reversibility 검토 (입력 공간 작으면 hash도 안전하지 않음 — OTel 가이드)
