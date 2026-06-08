---
id: CF-4
title: 민감정보 비노출 (PII Safety)
status: implemented
jtbd: [JTBD-4]
maps_modules: [F7]
source_paths:
  - pkg/types/alertmanagertypes/incident_payload.go
implements_uj: [UJ-1]
covered_by_wbs: [WBS-1.4]
fr_ids: [FR-CF4.1]
updated: 2026-06-08
caveats: "AIOpsAgent ingress 단일 지점 redaction — instrumentation·OTel Collector 단계 미적용(README, §9.2)"
---

# CF-4 — 민감정보 비노출 (PII Safety)

> **고객 가치 (JTBD-4)**: 보안담당자는 외부 알림 채널로 나가는 메시지에 이메일·전화번호·토큰·비밀번호 같은 민감정보가 노출되지 않음을 보장받는다. (전략 §6 데이터/안전 리스크 대응)
> **상태**: implemented. 단 AIOpsAgent ingress 단일 지점 — 가장 이른 단계(instrumentation·OTel Collector)는 미적용(§9.2).

## CF-4.1 개요 (사용자 관점)

알림에는 알람 라벨·SOP 본문·AI 초안이 섞이는데, 여기에 운영자 이메일·고객 전화번호·API 토큰이 끼어들 수 있다. CF-4는 **채널 전송 직전에 민감 패턴을 마스킹**해, 보안담당자가 별도 검수 없이도 외부 비노출을 신뢰할 수 있게 한다. 마스킹은 값만 가리고 알람 식별 라벨은 보존한다(추적성 유지).

## CF-4.2 기능 요구 (FR)

### FR-CF4.1 — 보안담당자는 외부 메시지에 민감정보가 노출되지 않음을 보장받는다
- **무엇을**: 외부로 나가는 incident 필드에서 이메일·국내 전화번호·긴 시크릿은 부분 마스킹, 토큰·비밀번호·JWT는 값 전체를 가린다. URL의 민감 쿼리 키는 제거하되 URL·비민감 파라미터는 보존한다.
- **Acceptance**:
  ```gherkin
  Given 알림 필드에 "Contact ops@example.com" / "긴급 010-1234-5678" / "Authorization: Bearer abcdefghij" / "https://signoz.example.com/api?token=xyz&svc=payment" 가 들어올 때
  When 마스킹이 적용되면
  Then 이메일은 "[redacted-email]", 전화는 "[redacted-phone]"로 부분 치환되고
   And Bearer 토큰이 든 값은 전체가 "[redacted]"로 가려지며
   And URL에서 "token="은 제거되되 "svc=payment"는 보존된다
  ```
- **구현 근거**: `incident_payload.go: SanitizeIncidentValue` 순서 — ① URL 파싱(http/https만, `User` info drop, sensitive query key 제거) ② secret-looking(`bearer `, JWT, marker) 전체 `[redacted]` ③ email regex ④ 국내 모바일 regex ⑤ 32+자 long secret regex. 컴파일 1회 global. URL sensitive keys: `access_token|api_key|apikey|auth|authorization|bearer|client_secret|password|secret|token`. · WBS-1.4

## CF-4.3 비기능 요건 (feature-specific)
- **NF-CF4.1** 마스킹은 채널 adapter 호출 **이전**에 완료(sanitized 값만 외부로). → NF-5.2.2
- **NF-CF4.2** Email/phone/long secret은 **부분 치환**(나머지 보존), secret-looking 검출은 **전체 drop**.
- **NF-CF4.3** URL `User` info는 항상 제거(`https://user:pw@host` → `https://host`).
- **NF-CF4.4** Regex는 컴파일 1회 + global(allocation 0).

## CF-4.4 예외·복구

| 상황 | 처리 |
|---|---|
| 입력 값 empty | 그대로 반환 |
| URL 파싱 실패 / non-http(s) | URL 처리 skip, 일반 regex만 |
| secret-looking 매칭 | 값 전체 `[redacted]`(early return) |
| regex 매칭 0건 | 값 그대로 |

## CF-4.5 Open / Non-goal
- **OTel Collector 단계 미적용** — 현재 AIOpsAgent ingress 단일 지점. 가장 이른 단계(instrumentation·Collector `transform`/`redaction`/`filter`)는 로드맵.
- **Redaction rate metric + threshold 초과 meta-alert** — 미구현.
- **카테고리 확장**(credit card, IP truncation, user_id hashing) — 미구현. hash reversibility 검토 필요.

## CF-4.6 Traceability
- JTBD: 4(안전) · User Journey: UJ-1 · NFR: NF-5.2.2
- User Journey: UJ-1(단계 3) · WBS: WBS-1.4
- 구 모듈: F7(PII Redaction)
- Commits: 
- → 상위: [`../index.md`](../index.md) §7.1 · 전략: [`source-strategy-brief.md`](../../_foundation/source-strategy-brief.md) §6(데이터 품질·안전)
