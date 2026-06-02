---
id: WBS-1.4
title: PII 마스킹 필터 (PII Masking Filter)
parent: WBS-1
status: planned
covers_features: [F7]
source_paths:
  - pkg/types/alertmanagertypes/incident_payload.go
acceptance: pending
estimated_effort: 2w
schedule:
  start: 2026-07-13
  end: 2026-07-24
  duration: 2w
commits: [3e9dfa557]
updated: 2026-06-02
caveats: "README 명시: PII 처리 production-ready 아님"
---

# WBS-1.4 — PII 마스킹 필터 (PII Masking Filter)

> **상태**: 착수 예정 (착수보고 기준)
> **일정**: 2026-07-13 ~ 2026-07-24 (2주, WBS-1.3과 병렬)
> **경고**: README 기준 production-ready 아님 — 강화 follow-up 필요

## Deliverable
Incident payload 직렬화 단계의 PII redaction 유틸리티. email 주소, 한국 휴대전화 번호(KR phone), unmarked long secret (예: 토큰/키 패턴) 3 카테고리에 대해 마스킹 처리해야 한다. 채널 dispatch 직전 단계에서 invocation 되어 모든 5개 채널 페이로드에 일관 적용되어야 한다.

## Acceptance Criteria
- [ ] F7.7 acceptance Gherkin pass — email / KR phone / long secret 패턴이 마스킹되어야 한다
- [ ] redaction 후에도 incident 식별자(alertname, fingerprint 등)는 유지되어야 한다
- [ ] redaction 미적용 페이로드는 dispatch 경로로 넘어가지 않아야 한다 (정책상 hard gate)

## Work Package 일정 (일 단위)

> 영업일(주5일) 기준, 공휴일 미반영. 의존성 순서: 인터페이스·타입 → 구현 → 통합·검증.

| WP ID | 작업명 | 선행 | 시작일 | 종료일 | 기간(영업일) |
|---|---|---|---|---|---|
| 1.4.1 | Redaction rule engine | 1.0.6 | 2026-07-13 | 2026-07-14 | 2 |
| 1.4.2 | Incident payload redaction 적용 | 1.4.1 | 2026-07-15 | 2026-07-16 | 2 |
| 1.4.3 | Audit sink 연동 | 1.4.2 | 2026-07-17 | 2026-07-20 | 2 |
| 1.4.4 | Tenant별 룰 확장 훅 | 1.4.3 | 2026-07-21 | 2026-07-22 | 2 |
| 1.4.5 | OTel Collector 단 이동 검토 | 1.4.4 | 2026-07-23 | 2026-07-24 | 2 |

## Work Packages (Lv3)

### WBS-1.4.1 — Redaction rule engine (Rule Engine)

- **Deliverable**: email / KR phone / long secret / JWT-like / URL sensitive key 5종 정규식 패턴 및 마스킹 포맷 구현체
- **Acceptance**:
  - `SanitizeIncidentValue` 단위 테스트: F7.7 Gherkin 5 시나리오 전부 통과
  - 패턴은 패키지 초기화 시 1회 컴파일(allocation 0) — NF-F7.2 충족
- **Source**: `pkg/types/alertmanagertypes/incident_payload.go`
- **일정**: 2026-07-13 ~ 2026-07-14 (2영업일, 선행: 1.0.6)
- **Effort**: TBD

### WBS-1.4.2 — Incident payload redaction 적용 (`incident_payload`)

- **Deliverable**: `BuildSafeIncidentInfo` / `SanitizeIncidentInfo` 호출이 channel adapter 직전에 hard gate로 삽입된 상태 — 모든 5개 채널(Slack / MSTeams / PagerDuty / webhook / email) 동일 경로 통과
- **Acceptance**:
  - redaction 미적용 페이로드가 dispatch 경로로 넘어가지 않음 (hard gate)
  - incident 식별자(`alertname`, `fingerprint` 등) 마스킹 후에도 보존됨
  - 통합 테스트: channel adapter stub 기준 sanitized value만 수신
- **Source**: `pkg/types/alertmanagertypes/incident_payload.go`
- **일정**: 2026-07-15 ~ 2026-07-16 (2영업일, 선행: 1.4.1)
- **Effort**: TBD

### WBS-1.4.3 — Audit sink 연동 (Audit Sink Integration)

- **Deliverable**: 마스킹 전 원본 카테고리 정보와 마스킹 후 결과를 audit 기록으로 남기는 sink — F5(Audit Trail)와 cross-cut
- **Acceptance**:
  - redacted_count(category별) 카운터가 audit 로그에 기록됨 — NF-F7.5 기초 충족
  - audit 기록에 원본 값이 포함되지 않음 (카테고리·필드명·카운트만)
- **Source**: `pkg/types/alertmanagertypes/incident_payload.go` (카운터 훅 추가 위치 TBD)
- **일정**: 2026-07-17 ~ 2026-07-20 (2영업일, 선행: 1.4.2)
- **Effort**: TBD

### WBS-1.4.4 — Tenant별 룰 확장 훅 (Tenant-Scoped Rule Extension Hook)

- **Deliverable**: 향후 tenant별 추가 패턴(카드번호·주민번호·커스텀 regex 등)을 주입할 수 있는 인터페이스 골격 — 현재는 글로벌 룰만 활성화, OTel Collector 단계 이동 검토 시 기반으로 활용
- **Acceptance**:
  - `RedactionRuleSet` 인터페이스(또는 등가 구조체) 정의 존재
  - 글로벌 기본 룰셋이 기존 동작과 동일하게 통과
  - tenant override 진입점에 TODO/OpenItem 주석 포함(구현 stub 허용)
- **Source**: `pkg/types/alertmanagertypes/` (신규 파일 또는 기존 확장)
- **일정**: 2026-07-21 ~ 2026-07-22 (2영업일, 선행: 1.4.3)
- **Effort**: TBD

### WBS-1.4.5 — OTel Collector 단 이동 검토 (Collector-Stage Migration Scaffold)

- **Deliverable**: Collector processor 단 redaction 이동 가능성을 평가하는 설계 메모 및 stub — 착수 후 결정 사항, 본 work package는 scaffolding only / decision pending
- **Acceptance**:
  - Collector processor 매핑 분석 문서(ADR 또는 설계 메모) 존재 — `transform` / `redaction` / `filter` 프로세서 대응표 포함
  - AIOpsAgent ingress 단 유지 vs Collector 이동 트레이드오프 정리
  - 구현 착수 여부는 WBS-1.4.5 검토 완료 후 결정 (현 단계 코드 변경 없음)
- **Source**: 설계 메모 (`docs/spec/` 또는 ADR 디렉터리), `pkg/types/alertmanagertypes/` (착수 시 참조)
- **일정**: 2026-07-23 ~ 2026-07-24 (2영업일, 선행: 1.4.4)
- **Effort**: TBD

## Owner
TBD (TBC)

## Estimated Effort
TBD

## Dependencies
- WBS-1.0 공통 기반 모듈 (incident payload 타입)

## Verification
- `pkg/types/alertmanagertypes/incident_payload_test.go`

## Covers Features
- F7 PII Redaction

## Source Paths
- `pkg/types/alertmanagertypes/incident_payload.go`

## Open Items
- production-readiness 강화 — 카테고리 추가 (주민등록번호·여권번호·카드번호 등), false positive/negative 측정
- redaction 단계를 OTel Collector processor 단으로 이동시켜 source-of-truth에 가깝게 처리할지 검토
- redaction metric 노출 (redacted_count by category) 추가 검토
