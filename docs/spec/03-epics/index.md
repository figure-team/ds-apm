---
id: EPICS-INDEX
title: DS-APM 에픽 & 스토리 (BMAD)
type: epics-index
status: implemented
updated: 2026-06-08
---

# DS-APM 에픽 & 스토리

> **BMAD 방식 에픽/스토리** — 사용자 가치 단위로 묶은 작업 정의. PRD(CF·FR)를 **서술형 스토리**(역할·요구·목적) + 인수 기준(Given/When/Then) 형태로 구현 단위화한다.
> **에픽 ≠ WBS**: 에픽/스토리는 **애자일 작업 정의**, [WBS](../05-wbs/index.md)는 **PMI 컴포넌트·일정 분해**(별도 산출물). 둘 다 [PRD](../01-prd/index.md)에서 파생되며 서로 매핑된다(아래 표).

## 에픽 목록

| Epic | 제목 (사용자 가치) | 커버 CF | 스토리 수 | 사용자 여정 | 관련 WBS | 파일 |
|---|---|---|:---:|---|---|---|
| **Epic 1** | 알람에 맞는 SOP를 자동으로 받는다 | CF-1 | 5 | UJ-1 | WBS-1.1 | [epic-1-sop-grounding.md](epic-1-sop-grounding.md) |
| **Epic 2** | SOP 기반 AI 대응 가이드를 받는다 | CF-2 | 6 | UJ-1·3 | WBS-1.2 | [epic-2-ai-assist.md](epic-2-ai-assist.md) |
| **Epic 3** | 평소 채널로 SOP·AI 알림을 받는다 | CF-3 | 3 | UJ-1·2 | WBS-1.3 | [epic-3-handoff.md](epic-3-handoff.md) |
| **Epic 4** | 민감정보 노출 없이 전달받는다 | CF-4 | 1 | UJ-1 | WBS-1.4 | [epic-4-pii-safety.md](epic-4-pii-safety.md) |
| **Epic 5** | 실패 알림이 유실 없이 재발송된다 | CF-5 | 3 | UJ-2 | WBS-1.5 | [epic-5-reliable-delivery.md](epic-5-reliable-delivery.md) |
| **Epic 6** | 정책·감사 기반이 갖춰진다 | CF-6 | 3 | UJ-1·2·3 | WBS-1.0 | [epic-6-foundation-audit.md](epic-6-foundation-audit.md) |

> 스토리 21건 = PRD FR 21건과 1:1. 인수 기준 상세(Given/When/Then)는 각 에픽 + [PRD CF feature](../01-prd/index.md) §7.

## 에픽 ↔ WBS ↔ CF 매핑

| Epic (애자일 작업 정의) | WBS (PMI 컴포넌트·일정) | CF (기능) |
|---|---|---|
| Epic 1~6 | WBS-1.0~1.5 | CF-1~6 |

> 에픽은 *무엇을 사용자 가치로 전달하나*, WBS는 *어떤 컴포넌트를 언제 만드나*. 같은 작업의 두 관점.

→ PRD: [`../01-prd/index.md`](../01-prd/index.md) · WBS: [`../05-wbs/index.md`](../05-wbs/index.md) · 추적: [`../_shared/traceability.md`](../_shared/traceability.md)
