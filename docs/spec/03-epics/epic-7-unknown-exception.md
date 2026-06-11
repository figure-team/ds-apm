---
id: EPIC-7
title: 미등록 예외도 인지·분석·자산화된다
type: epic
covers_feature: CF-7
maps_wbs: WBS-1.6
realizes_uj: [UJ-5]
stories: [7.2, 7.3, 7.4, 7.5, 7.6, 7.7]
status: planned
updated: 2026-06-11
---

# Epic 7: 미등록 예외도 인지·분석·자산화된다

> **목표(Goal)**: Alert Rule에 등록되지 않은 예외가 발생해도 운영자가 이를 자동 인지하고, 코드베이스(git/svn) 근거가 포함된 원인 분석 보고서와 초기 대응 가이드를 받으며, 검토된 결과를 신규 SOP/Runbook으로 자산화한다. 어떤 분기에서도 발생 알림은 누락되지 않는다(silent drop 0, HITL 불변).
> **커버 기능**: CF-7 (1차) · **관련 WBS**: WBS-1.6 · **사용자 여정**: UJ-5
> **설계 원천**: [설계서](../../superpowers/specs/2026-06-11-unknown-exception-response-design.md)

## 스토리 (별도 파일)

| 스토리 | 제목 | FR | 상태 |
|---|---|---|---|
| [Story 7.2](../04-stories/7.2.story.md) | 미등록 신규 예외 자동 인지·1차 알림 | FR-CF7.2 | planned |
| [Story 7.3](../04-stories/7.3.story.md) | 코드베이스 근거 원인 분석 보고서 | FR-CF7.3 | planned |
| [Story 7.4](../04-stories/7.4.story.md) | 저장소 연결·코드 전송 통제 | FR-CF7.4 | planned |
| [Story 7.5](../04-stories/7.5.story.md) | 보고서의 SOP/Runbook 자산화 (HITL) | FR-CF7.5 | planned |
| [Story 7.6](../04-stories/7.6.story.md) | 반복 알림 억제·무시 처리 | FR-CF7.6 | planned |
| [Story 7.7](../04-stories/7.7.story.md) | 분석 실패에도 발생 알림 보장 (fail-open) | FR-CF7.7 | planned |

> 스토리 번호 7.1은 의도적 결번 — FR-CF7.1(메트릭 기준선 학습)은 CF-7 후속 범위로 본 에픽에 미포함(PRD §7.3). 인수 기준 원천은 [PRD CF-7](../01-prd/features/CF-7-unknown-exception.md).

## 권장 구현 순서 (의존)

```
7.4 (VCS 미러·설정)  ─┐
7.2 (스캐너·1차 알림) ─┼─▶ 7.3 (분석 워커·보고서) ─▶ 7.5 (SOP 등록)
7.6 (억제·dismiss)    ─┘         7.7 (fail-open 분기 — 7.2·7.3 횡단)
```

- 7.2와 7.4는 병렬 가능. 7.3은 둘에 의존. 7.7은 7.2·7.3의 실패 분기 검증으로 마지막 통합 단계에서 확인.

## Traceability
- 커버 CF: [CF-7](../01-prd/features/CF-7-unknown-exception.md) · WBS: [WBS-1.6](../05-wbs/index.md) · UJ-5 · 추적: [`../_shared/traceability.md`](../_shared/traceability.md)
