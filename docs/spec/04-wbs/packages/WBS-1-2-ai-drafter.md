---
id: WBS-1.2
title: AI Drafter
parent: WBS-1
status: implemented
covers_features: [F2, F3]
source_paths:
  - pkg/ruler/aigenerator/
  - pkg/ruler/aigenerator/llmaigenerator/
  - pkg/ruler/aigenerator/storeaware.go
  - pkg/ruler/aigenerator/dispatchhook/
  - pkg/ruler/runbookdrafter/
  - pkg/ruler/runbookdrafter/llmrunbookdrafter/
  - pkg/types/ruletypes/ai_strategy.go
  - pkg/types/ruletypes/ai_strategy_history.go
acceptance: pending
estimated_effort: completed
commits: [a6757136e, cb29d2a59]
updated: 2026-05-29
---

# WBS-1.2 — AI Drafter

> **상태**: 구현 완료

## Deliverable
LLM 기반 runbook drafter (`llmrunbookdrafter`), AI generator 추상화 (`aigenerator` + `llmaigenerator` + `mockaigenerator`), store-aware generator wrapper, dispatch hook 통합, quota 제어 (fail-open) 및 AI strategy/strategy history 영속 타입. SOP grounding 결과를 입력으로 받아 채널 dispatch 직전까지 사용할 runbook 초안을 생성해야 한다.

## Acceptance Criteria
- [ ] F2.7 acceptance Gherkin pass — SOP-grounded incident에 대해 runbook draft가 생성되어야 한다
- [ ] F2.7 — 동일 alert의 strategy history가 최신 N건 유지되어야 한다
- [ ] F3.7 acceptance Gherkin pass — LLM auth/quota 실패 시 SOP fallback으로 fail-open 동작 (UC-003)
- [ ] dispatch hook이 draft 생성 결과를 WBS-1.3 dispatcher에 안전하게 전달해야 한다
- [ ] AI 호출/실패 이벤트는 WBS-1.0의 audit sink에 기록되어야 한다

## Owner
TBD (TBC)

## Estimated Effort
완료 (커밋 `a6757136e`, `cb29d2a59`)

## Dependencies
- WBS-1.0 Foundation (audit sink, tenant policy)
- WBS-1.1 SOP Engine (grounding 입력)

## Verification
- `pkg/ruler/aigenerator/llmaigenerator/llm_test.go`
- `pkg/ruler/aigenerator/llmaigenerator/prompt_test.go`
- `pkg/ruler/aigenerator/mockaigenerator/mock_test.go`
- `pkg/ruler/aigenerator/dispatchhook/hook_test.go`
- `pkg/ruler/runbookdrafter/mockrunbookdrafter/mock_test.go`
- `pkg/types/ruletypes/ai_strategy_test.go`
- `pkg/types/ruletypes/ai_strategy_history_test.go`

## Covers Features
- F2 AI Runbook Drafting
- F3 AI Quota Controls

## Source Paths
- `pkg/ruler/aigenerator/`
- `pkg/ruler/aigenerator/llmaigenerator/`
- `pkg/ruler/aigenerator/storeaware.go`
- `pkg/ruler/aigenerator/dispatchhook/`
- `pkg/ruler/runbookdrafter/`
- `pkg/ruler/runbookdrafter/llmrunbookdrafter/`
- `pkg/types/ruletypes/ai_strategy.go`
- `pkg/types/ruletypes/ai_strategy_history.go`

## Open Items
- LLM 실제 provider별 통합 테스트 (현재 mock 위주, llmaigenerator는 단위 수준)
