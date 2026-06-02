---
id: F2
title: AI Runbook Drafting (with history)
status: planned
commits: [cb29d2a59]
source_paths:
  - pkg/ruler/aigenerator/
  - pkg/ruler/aigenerator/llmaigenerator/
  - pkg/ruler/runbookdrafter/
  - pkg/ruler/runbookdrafter/llmrunbookdrafter/
  - pkg/types/ruletypes/ai_strategy.go
  - pkg/types/ruletypes/ai_strategy_history.go
implements_uc: [UC-001, UC-003]
covered_by_wbs: [WBS-1.2]
updated: 2026-06-02
---

# F2 — AI Runbook Drafting (with history)

> **상태**: 착수 예정 (착수보고 기준)
> bound된 SOP를 컨텍스트로 LLM이 incident 대응 strategy + runbook draft를 생성하고, 결과를 `ai_strategy_history`에 기록한다.

## 책임 (Responsibility)

bound된 SOP와 alert context를 받아 `AIStrategy`(가설·첫 조치·customer update draft·vendor request draft)를 생성한다. 운영자 검수 후 `AIStrategyHistoryStore`에 best-effort upsert하여 동일 incident 재발 시 lookup 가능하게 한다. **모든 `FirstAction.RequiresHumanApproval`은 반드시 `true`** — 자동 실행 주장 패턴은 validator가 차단한다.

## 인터페이스 요지

```go
// pkg/types/ruletypes
type AIStrategyGenerator interface {
    Generate(ctx context.Context, req AIStrategyRequest) (AIStrategy, error)
}
type AIStrategyHistoryStore interface {
    Upsert(ctx context.Context, orgID string, record AIStrategyHistoryRecord) error
    Lookup(ctx context.Context, orgID string, req AIStrategyHistoryLookupRequest) (AIStrategyHistoryRecord, bool, error)
}

// pkg/ruler/aigenerator/aigenerator.go
func New(cfg Config) (ruletypes.AIStrategyGenerator, error)
```

Provider: `cfg.Provider ∈ {"", "local", "mock", "llm"}`. `llm`일 때 `{claude, codex} × {api, cli}` 4조합. 상세 구조체는 `pkg/types/ruletypes/ai_strategy*.go` 참조.

## 핵심 동작

입력: alert labels/annotations + bound SOPDocument + EvidenceRefs.

처리: quota/license/timeout control 검사 → LLM 호출 → strategy 생성 → validator 통과 → history upsert.

출력: `AIStrategy.Status` — `ready | sop_missing | evidence_unavailable | quota_exhausted | timeout | blocked_by_policy | unavailable`.

`StrategyID`가 비어 있으면 `sha256(incidentID || fingerprint || sopID || sopVersion)`의 앞 16 hex로 결정론적 생성. `audit.redactionApplied=true`여야 strategy 출력이 사용된다.

## 예외·복구

| 경로 | 처리 |
|---|---|
| LLM 5xx / timeout | `Status=timeout`, Limitations 메시지 포함 |
| Quota 초과 | `Status=quota_exhausted` (F3 fail-open 참조) |
| SOP 미바인딩 | `Status=sop_missing`, "연결된 SOP 문서가 없어" headline |
| 자동 실행 주장 패턴 검출 | validator 거부 |
| History upsert 실패 | `WarnContext`만, dispatch 계속 |

## Acceptance Criteria

```gherkin
Feature: AI runbook drafting with SOP grounding
  Background:
    Given AIStrategyGenerator configured with provider "local"
    And SOPDocument "SOP-PAY-5xx" bound to the request

  Scenario: Ready strategy enforces human approval
    Given the request carries at least one evidence ref
    When Generate is called
    Then strategy.Status is "ready"
    And every first action has RequiresHumanApproval=true
    And audit.redactionApplied is true

  Scenario: Automatic-operation claim is rejected
    Given a hypothesis text containing "자동 재시작"
    When ValidateAIStrategy is called
    Then validation fails with "must not claim automatic operational execution"
```

## Traceability
- Implements UC: UC-001 (단계 5), UC-003 (degraded path)
- Covered by WBS: WBS-1.2
- Source: `pkg/ruler/aigenerator/`, `pkg/ruler/runbookdrafter/llmrunbookdrafter/`, `pkg/types/ruletypes/ai_strategy*.go`
- Commits: `cb29d2a59`
