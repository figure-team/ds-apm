---
id: F3
title: AI Quota Controls (fail-open)
status: planned
commits: [a6757136e]
source_paths:
  - pkg/ruler/aigenerator/storeaware.go
  - pkg/ruler/aigenerator/dispatchhook/
implements_uc: [UC-003]
covered_by_wbs: [WBS-1.2]
updated: 2026-06-02
---

# F3 — AI Quota Controls (fail-open)

> **상태**: 착수 예정 (착수보고 기준)
> LLM provider disabled / license 불허 / quota 초과 / timeout 초과 시 fail-open 정책으로 SOP 원문 fallback. 알람 전달 자체는 절대 막지 않는다.

## 책임 (Responsibility)

"AI는 보조, 알람은 항상 전달" 원칙을 집행한다. AI 경로가 어떤 이유로 실패해도 dispatcher가 멈추지 않도록 두 가지를 보장한다: `StoreAware` generator가 per-org config 조회 실패 시 env fallback으로 전환하고, dispatch hook은 모든 내부 오류를 흡수하여 `error`를 절대 반환하지 않는다.

## 인터페이스 요지

```go
// pkg/ruler/aigenerator/storeaware.go
func NewStoreAware(store ruletypes.AIConfigStore, cipher *secretbox.Cipher,
    envFallback ruletypes.AIStrategyGenerator) *StoreAware
func (s *StoreAware) Generate(ctx context.Context, req ruletypes.AIStrategyRequest) (ruletypes.AIStrategy, error)
func (s *StoreAware) Invalidate(orgID string)  // config PUT 시 호출 필수

// pkg/ruler/aigenerator/dispatchhook/hook.go
const DefaultGenerateTimeout = time.Second
func (h *Hook) Apply(ctx context.Context, orgID, incidentID, alertFingerprint string,
    labels, annotations map[string]string) map[string]string  // error 반환 없음
```

4종 control(`ProviderEnabled`, `LicenseAllowed`, `QuotaLimit`, `TimeoutBudget`) 위반 시 `Status=quota_exhausted|blocked_by_policy|unavailable|timeout`으로 degrade (F2.5 참조). 상세는 `pkg/ruler/aigenerator/storeaware.go` 참조.

## 핵심 동작

입력: dispatcher가 alert flush 시점에 `Hook.Apply` 호출.

처리: `StoreAware`가 per-org `AIConfig` cache hit → miss 시 store 조회 → 실패 시 `envFallback`. Hook 내부 오류(SOP list 실패·timeout·history upsert 실패)는 전부 `WarnLog`만 남기고 입력 annotations 그대로 반환.

출력: merged annotations map (입력 변형 없음, 새 map 반환).

`StoreAware`는 per-org generator를 `RWMutex` 보호된 map으로 캐시한다.

## 예외·복구

| 경로 | 처리 |
|---|---|
| Per-org AIConfig 미설정 또는 cipher 실패 | `envFallback.Generate` |
| `sopStore.List` 실패 | 입력 annotations 그대로 반환 + WarnLog |
| `generator.Generate` 1초 초과 | 입력 annotations 그대로 반환 + WarnLog |
| `aiHistoryStore.Upsert` 실패 | WarnLog만, dispatch 계속 |

핵심 불변: **dispatcher hot path에서 어떤 실패도 알람 전달을 막지 않는다.**

## Acceptance Criteria

```gherkin
Feature: AI quota controls fail open

  Scenario: Generator timeout returns input annotations unchanged
    Given a dispatch hook whose generator sleeps longer than DefaultGenerateTimeout
    When Hook.Apply runs against an alert
    Then the returned annotations equal the input annotations
    And a warn-level log "ai dispatch hook: generate failed" is emitted

  Scenario: Store lookup failure falls back to env generator
    Given a StoreAware whose AIConfigStore.Get returns an error
    When Generate is called
    Then the envFallback generator handles the request
```

## Traceability
- Implements UC: UC-003
- Covered by WBS: WBS-1.2
- Source: `pkg/ruler/aigenerator/storeaware.go`, `pkg/ruler/aigenerator/dispatchhook/hook.go`
- Commits: `a6757136e`
