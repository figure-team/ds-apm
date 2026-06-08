---
id: EPIC-5
title: 실패 알림이 유실 없이 재발송된다
type: epic
covers_feature: CF-5
maps_wbs: WBS-1.5
realizes_uj: [UJ-2]
stories: [5.1, 5.2, 5.3]
status: implemented-mvp
updated: 2026-06-08
---

# Epic 5: 실패 알림이 유실 없이 재발송된다

> **목표(Goal)**: 운영자가 채널 전송 최종 실패 알림이 유실되지 않고, 재발송 시 중복으로 두 번 가지 않음을 보장받는다.
> **커버 기능**: CF-5 · **관련 WBS**: WBS-1.5 · **사용자 여정**: UJ-2 · 상태: mvp(HMAC open)

## 스토리 (별도 파일)

| 스토리 | 제목 | FR | 상태 |
|---|---|---|---|
| [Story 5.1](../04-stories/5.1.story.md) | 최종 실패 알림 무유실 보존 | FR-CF5.1 | done |
| [Story 5.2](../04-stories/5.2.story.md) | 멱등 재발송 (중복 방지) | FR-CF5.2 | done |
| [Story 5.3](../04-stories/5.3.story.md) | 재발송 위변조 방지 (HMAC) | FR-CF5.3 | **planned (open)** |

> 인수 기준 원천: [PRD CF-5](../01-prd/features/CF-5-reliable-delivery.md).

## Traceability
- 커버 CF: [CF-5](../01-prd/features/CF-5-reliable-delivery.md) · WBS: [WBS-1.5](../05-wbs/index.md) · UJ-2 · 추적: [`../_shared/traceability.md`](../_shared/traceability.md)
