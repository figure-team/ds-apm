---
id: WBS-1.4
title: PII Redactor
parent: WBS-1
status: implemented-mvp
covers_features: [F7]
source_paths:
  - pkg/types/alertmanagertypes/incident_payload.go
acceptance: pending
estimated_effort: completed
commits: [3e9dfa557]
updated: 2026-05-29
caveats: "README 명시: PII 처리 production-ready 아님"
---

# WBS-1.4 — PII Redactor

> **상태**: 구현 완료 (MVP)
> **경고**: README 기준 production-ready 아님 — 강화 follow-up 필요

## Deliverable
Incident payload 직렬화 단계의 PII redaction 유틸리티. email 주소, 한국 휴대전화 번호(KR phone), unmarked long secret (예: 토큰/키 패턴) 3 카테고리에 대해 마스킹 처리해야 한다. 채널 dispatch 직전 단계에서 invocation 되어 모든 5개 채널 페이로드에 일관 적용되어야 한다.

## Acceptance Criteria
- [ ] F7.7 acceptance Gherkin pass — email / KR phone / long secret 패턴이 마스킹되어야 한다
- [ ] redaction 후에도 incident 식별자(alertname, fingerprint 등)는 유지되어야 한다
- [ ] redaction 미적용 페이로드는 dispatch 경로로 넘어가지 않아야 한다 (정책상 hard gate)

## Owner
TBD (TBC)

## Estimated Effort
완료 (커밋 `3e9dfa557`)

## Dependencies
- WBS-1.0 Foundation (incident payload 타입)

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
