---
id: EPIC-11
title: SOP 없는 이상 장애의 코드 근거 근본원인을 AI가 분석한다
type: epic
covers_feature: CF-11
maps_wbs: WBS-1.7
realizes_uj: [UJ-5]
stories: [11.1, 11.2, 11.3, 11.4, 11.5, 11.6]
status: implemented-mvp
updated: 2026-06-12
---

# Epic 11: SOP 없는 이상 장애의 코드 근거 근본원인을 AI가 분석한다

> **목표(Goal)**: 매칭 SOP가 없고(unbound) 이상(anomaly)이 감지된 알람에 대해, CLI 코딩 에이전트(claude/codex)를 read-only로 구동해 해당 서비스 코드베이스를 탐색하고 **근본원인 + 수정 제안**(검토용, HITL)을 운영자에게 전달한다. **비용·볼륨 제어가 #1 설계 동인** — 폭주가 토큰/프로세스 폭발로 이어지지 않도록 원자적 DB admission·lease·dedup·예산·동시성 캡으로 강제하고, 분석 실패는 알람 전달에 영향을 주지 않는다(fail-open).
> **커버 기능**: CF-11 · **관련 WBS**: WBS-1.7 · **사용자 여정**: UJ-5 · **트리거 소스**: CF-7(이상 탐지)
> **설계 원천**: [설계서](../../superpowers/specs/2026-06-11-cf11-code-rca-design.md) (Codex 4라운드 APPROVE)
> **상태**: implemented-mvp — 코어(M1~M3) TDD 완료, HTTP·FE·디스패치 훅 트리거·서버 배선은 통합 seam(설계 §11).

## 스토리 (별도 파일)

| 스토리 | 제목 | FR | 상태 |
|---|---|---|---|
| [Story 11.1](../04-stories/11.1.story.md) | 트리거 게이트 + AI 코드 RCA 보고서 | FR-CF11.1 | done (코어 · 디스패치 seam) |
| [Story 11.2](../04-stories/11.2.story.md) | 저장소·서비스 매핑·기능 토글 설정 | FR-CF11.2 | done (store·secretbox · HTTP/FE seam) |
| [Story 11.3](../04-stories/11.3.story.md) | 폭주 비용·볼륨 제어 (admission·lease) | FR-CF11.3 | done |
| [Story 11.4](../04-stories/11.4.story.md) | HITL 수정 제안·read-only 샌드박스 | FR-CF11.4 | done |
| [Story 11.5](../04-stories/11.5.story.md) | 분석 기준 커밋 pin·echo | FR-CF11.5 | done |
| [Story 11.6](../04-stories/11.6.story.md) | 분석 실패에도 알람 무영향 (fail-open) | FR-CF11.6 | done (코어 · e2e는 seam) |

> 인수 기준 원천은 [PRD CF-11](../01-prd/features/CF-11-code-rca.md). 스토리 번호 11.1~11.6은 FR-CF11.1~11.6과 1:1(결번 없음 — FR-CF7.1=이상 탐지는 CF-7/Epic 7로 분리).

## 권장 구현 순서 (의존 — 설계 §13 마일스톤)

```
M1 비용 제어 코어 ─▶ 11.3 (admission·lease·dedup·budget·동시성) [게이팅: flood-sim 통과 필수]
                       │
M2 순수 분석 코어 ─────┼─▶ 11.5 (source-state·resolver·baseline) + 11.4 일부(parser)
                       │
M3 어댑터·오케스트레이션 ─▶ 11.4 (clirunner read-only) ─▶ 11.1 (engine 배선) ─▶ 11.6 (fail-open·감사)
                       │
M4 표면 (후행) ────────▶ 11.2 (설정 HTTP/FE) + 디스패치 seam
```

- **11.3(비용 제어)이 게이팅** — flood-sim T1 통과 전 CLI 구동 경로 불가(설계 §13 M1). 11.2의 store는 M1, HTTP/FE는 M4(seam).

## Traceability
- 커버 CF: [CF-11](../01-prd/features/CF-11-code-rca.md) · WBS: [WBS-1.7](../05-wbs/index.md) · UJ-5 · 트리거 소스: [CF-7](../01-prd/features/CF-7-anomaly.md) · 추적: [`../_shared/traceability.md`](../_shared/traceability.md)
