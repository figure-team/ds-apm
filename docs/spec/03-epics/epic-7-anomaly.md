---
id: EPIC-7
title: 메트릭이 평소 패턴을 벗어나면 이상 알람을 받는다
type: epic
covers_feature: CF-7
maps_wbs: WBS-1.6
realizes_uj: []
stories: [7.1]
status: implemented
updated: 2026-06-12
---

# Epic 7: 메트릭이 평소 패턴을 벗어나면 이상 알람을 받는다

> **목표(Goal)**: 고정 임계치로는 잡기 어려운 메트릭 거동을 통계 기준선(이동평균 ± k·σ) 대비 이상 점수(z-score)로 평가해, 평소 패턴에서 벗어나면 이상 알람을 발생시킨다. 산출된 이상 알람은 일반 알람과 동일 경로로 흐르며, 연계 SOP가 없는 이상 알람은 CF-11(AI 코드베이스 RCA)의 트리거 조건이 된다.
> **커버 기능**: CF-7 · **관련 WBS**: WBS-1.6 · **여정**: UJ-1·UJ-5의 **트리거 소스**(implements 아님)
> **상태**: implemented (`anomaly_rule.go`). v1=통계(z-score), 학습형/계절성은 후속.

## 스토리 (별도 파일)

| 스토리 | 제목 | FR | 상태 |
|---|---|---|---|
| [Story 7.1](../04-stories/7.1.story.md) | 메트릭 이상 탐지 (z-score 기준선) | FR-CF7.1 | done |

> CF-7은 단일 구현 FR(FR-CF7.1)을 가진다. 학습형/계절성 기준선(FR-CF7.2)은 roadmap(PRD §7.3)으로, 본 에픽에 스토리 미포함.

## Traceability
- 커버 CF: [CF-7](../01-prd/features/CF-7-anomaly.md) · WBS: [WBS-1.6](../05-wbs/index.md) · 하류 소비: [CF-11](../01-prd/features/CF-11-code-rca.md) · 추적: [`../_shared/traceability.md`](../_shared/traceability.md)
