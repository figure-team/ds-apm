---
id: WBS-APPENDIX-PHASES
title: WBS 부록 — Phase 시간선 (P0~P5)
type: wbs-appendix
status: historical
updated: 2026-05-28
---

# 부록 A. Phase 시간선 (역사 추적용)

> 이 문서는 11개 DS-APM 커밋의 **시간 순 분할**이다.
> 메인 WBS(`index.md`)는 component-oriented이고, 이 부록은 어떤 순서로 만들어졌는지의 기록.
> 새 작업 분류 시에는 component WBS를 우선 사용할 것.

## 베이스라인
- 분기점 (SigNoz upstream): `feea9e9b3 refactor: remove light mode styles ... (#11080)`
- 종료점 (DS-APM HEAD): `91b9ff5db feat(ds-apm): wire dead-letter sink into alertmanager dispatcher`

## Phase 매핑

| Phase | 내용 | 커밋 (시간 순) | 커버 Component (WBS) |
|---|---|---|---|
| P0 | Foundation | `026863650` | WBS-1.0 |
| P1 | SOP Layer | `72944ecac` → `8a55208ef` → `3fa604e03` → `c7f4fd330` | WBS-1.0, WBS-1.1 |
| P2 | AI Layer | `a6757136e` → `cb29d2a59` | WBS-1.2 |
| P3 | Notification | `5c036c806` | WBS-1.3 |
| P4 | Safety (PII) | `3e9dfa557` | WBS-1.4 |
| P5 | Reliability (DLQ/Replay) | `ade174bb8` → `91b9ff5db` | WBS-1.5 |

## 사용처
- 회고/감리 — 무엇이 언제 만들어졌는지
- 변경 영향도 분석 — 어느 커밋이 어느 component를 건드렸는지

## 변경 시
- 새 커밋은 본 부록에 시간 순 append만 (rebase/squash 시 갱신)
- Component WBS는 `index.md`와 `packages/` 아래에서 관리
