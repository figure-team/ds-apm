---
id: EPIC-2
title: SOP 기반 AI 대응 가이드를 받는다
type: epic
covers_feature: CF-2
maps_wbs: WBS-1.2
realizes_uj: [UJ-1, UJ-3]
stories: [2.1, 2.2, 2.3, 2.4, 2.5, 2.6]
status: implemented
updated: 2026-06-08
---

# Epic 2: SOP 기반 AI 대응 가이드를 받는다

> **목표(Goal)**: 운영자가 인시던트 원인·대응 가이드를 직접 로그 대조 없이 AI로부터 받되, AI가 틀리거나 실패해도(HITL·fail-open) 안전하다.
> **커버 기능**: CF-2 · **관련 WBS**: WBS-1.2 · **사용자 여정**: UJ-1·3

## 스토리 (별도 파일)

| 스토리 | 제목 | FR | 상태 |
|---|---|---|---|
| [Story 2.1](../04-stories/2.1.story.md) | SOP 기반 AI 대응 가이드 생성 | FR-CF2.1 | done |
| [Story 2.2](../04-stories/2.2.story.md) | 전문가 없이 1차 대응 (안전 fallback) | FR-CF2.2 | done |
| [Story 2.3](../04-stories/2.3.story.md) | 사람 승인 강제 (HITL) | FR-CF2.3 | done |
| [Story 2.4](../04-stories/2.4.story.md) | AI 실패에도 알람 전달 (fail-open) | FR-CF2.4 | done |
| [Story 2.5](../04-stories/2.5.story.md) | 사용량 제어 — 초과해도 전달 유지 | FR-CF2.5 | done |
| [Story 2.6](../04-stories/2.6.story.md) | 과거 대응 이력 참조 | FR-CF2.6 | done |

> 인수 기준 원천: [PRD CF-2](../01-prd/features/CF-2-ai-assist.md).

## Traceability
- 커버 CF: [CF-2](../01-prd/features/CF-2-ai-assist.md) · WBS: [WBS-1.2](../05-wbs/index.md) · UJ-1·3 · 추적: [`../_shared/traceability.md`](../_shared/traceability.md)
