---
id: WBS-1.2
title: AI 초안 매니저 (AI Drafter Manager)
parent: WBS-1
status: planned
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
estimated_effort: 4w
schedule:
  start: 2026-06-15
  end: 2026-07-10
  duration: 4w
commits: [a6757136e, cb29d2a59]
updated: 2026-06-02
---

# WBS-1.2 — AI 초안 매니저 (AI Drafter Manager)

> **상태**: 착수 예정 (착수보고 기준)
> **일정**: 2026-06-15 ~ 2026-07-10 (4주, WBS-1.1과 병렬)

## Deliverable
LLM 기반 runbook drafter (`llmrunbookdrafter`), AI generator 추상화 (`aigenerator` + `llmaigenerator` + `mockaigenerator`), store-aware generator wrapper, dispatch hook 통합, quota 제어 (fail-open) 및 AI strategy/strategy history 영속 타입. SOP grounding 결과를 입력으로 받아 채널 dispatch 직전까지 사용할 runbook 초안을 생성해야 한다.

## Acceptance Criteria
- [ ] F2.7 acceptance Gherkin pass — SOP-grounded incident에 대해 runbook draft가 생성되어야 한다
- [ ] F2.7 — 동일 alert의 strategy history가 최신 N건 유지되어야 한다
- [ ] F3.7 acceptance Gherkin pass — LLM auth/quota 실패 시 SOP fallback으로 fail-open 동작 (UC-003)
- [ ] dispatch hook이 draft 생성 결과를 WBS-1.3 dispatcher에 안전하게 전달해야 한다
- [ ] AI 호출/실패 이벤트는 WBS-1.0의 audit sink에 기록되어야 한다

## Work Package 일정 (일 단위)

> 영업일(주5일) 기준, 공휴일 미반영. 의존성 순서: 인터페이스·타입 → 구현 → 통합·검증.

| WP ID | 작업명 | 선행 | 시작일 | 종료일 | 기간(영업일) |
|---|---|---|---|---|---|
| 1.2.1 | AI Strategy 도메인 타입 | 1.0.6 | 2026-06-15 | 2026-06-18 | 4 |
| 1.2.2 | LLM Provider 어댑터 | 1.2.1 | 2026-06-19 | 2026-06-24 | 4 |
| 1.2.3 | Strategy 생성·persistence | 1.2.2 | 2026-06-25 | 2026-06-29 | 3 |
| 1.2.4 | Strategy History append | 1.2.3 | 2026-06-30 | 2026-07-02 | 3 |
| 1.2.5 | Quota Controller (fail-open) | 1.2.4 | 2026-07-03 | 2026-07-07 | 3 |
| 1.2.6 | Dispatch Hook Integration | 1.2.5 | 2026-07-08 | 2026-07-10 | 3 |

## Work Packages (Lv3)

### WBS-1.2.1 — AI Strategy 도메인 타입 (ai_strategy / ai_strategy_history)

- **Deliverable**: `AIStrategy`, `AIStrategyHistoryRecord`, `AIStrategyControls`, `AIStrategyAudit` 타입 정의 및 validator (자동 실행 주장 패턴 차단, history record 필드 일치 검증 포함)
- **Acceptance**:
  - `ValidateAIStrategy`가 `자동 재시작` 등 자동 실행 주장 패턴 검출 시 거부한다
  - `ValidateAIStrategyHistoryRecord`가 embedded Strategy 필드와 record 루트 필드 불일치 시 거부한다
  - `ContractVersion` 상수가 `ds.ai_strategy.v1` / `ds.ai_strategy_history.v1`로 고정된다
- **Source**: `pkg/types/ruletypes/ai_strategy.go`, `pkg/types/ruletypes/ai_strategy_history.go`
- **일정**: 2026-06-15 ~ 2026-06-18 (4영업일, 선행: 1.0.6)
- **Effort**: TBD

### WBS-1.2.2 — LLM Provider 어댑터 (llm_provider_adapter)

- **Deliverable**: `AIStrategyGenerator` 인터페이스 구현체 3종 — `local` (deterministic), `mock` (fixture 기반), `llm` (Claude/Codex × API/CLI 4조합) — 및 `aigenerator.New()` factory
- **Acceptance**:
  - `cfg.Provider ∈ {"", "local", "mock", "llm"}` 전 케이스 factory 분기가 동작한다
  - `llm` provider에서 LLMProvider × LLMTransport 4조합이 인스턴스화된다
  - `mock` provider 선택 시 `buildFromAIConfig`가 명시적 error를 반환해 `StoreAware`가 `envFallback`으로 전환한다
- **Source**: `pkg/ruler/aigenerator/`, `pkg/ruler/aigenerator/llmaigenerator/`
- **일정**: 2026-06-19 ~ 2026-06-24 (4영업일, 선행: 1.2.1)
- **Effort**: TBD

### WBS-1.2.3 — Strategy 생성·persistence 로직 (strategy_generation)

- **Deliverable**: `Generate()` hot path 구현 — SOP 바인딩 검증, evidence ref 확인, `deterministicAIStrategyID()` 생성, `audit.redactionApplied` 강제, 상태 전이 (`ready|sop_missing|evidence_unavailable|blocked_by_policy`) 처리
- **Acceptance**:
  - SOP bound + evidence 있을 때 `Status=ready`, 모든 `FirstAction.RequiresHumanApproval=true`
  - `SOPDocument` 비어 있을 때 `Status=sop_missing`, headline 문구 포함
  - `StrategyID` 미입력 시 `sha256(incidentID || fingerprint || sopID || sopVersion)` 앞 16 hex로 결정론적 생성
- **Source**: `pkg/ruler/aigenerator/`, `pkg/ruler/runbookdrafter/llmrunbookdrafter/`
- **일정**: 2026-06-25 ~ 2026-06-29 (3영업일, 선행: 1.2.2)
- **Effort**: TBD

### WBS-1.2.4 — Strategy History append 로직 (strategy_history_persistence)

- **Deliverable**: `AIStrategyHistoryStore.Upsert()` 구현 및 dispatch hook 내 history append 흐름 — 운영자 승인 후 best-effort upsert, 실패 시 WarnLog 후 dispatch 계속 진행
- **Acceptance**:
  - Upsert 성공 시 동일 `alertFingerprint`로 Lookup이 최신 1건을 반환한다
  - Upsert 실패 시 hook이 error를 반환하지 않고 `WarnContext`만 기록한다
  - History record의 `IncidentID`, `StrategyID`, `Status`, `Confidence`, `GeneratedAt`이 embedded Strategy와 정확히 일치한다
- **Source**: `pkg/ruler/aigenerator/storeaware.go`, `pkg/types/ruletypes/ai_strategy_history.go`
- **일정**: 2026-06-30 ~ 2026-07-02 (3영업일, 선행: 1.2.3)
- **Effort**: TBD

### WBS-1.2.5 — Quota Controller (fail-open) (quota_controller)

- **Deliverable**: 4종 control (`ProviderEnabled` / `LicenseAllowed` / `QuotaLimit` / `TimeoutBudget`) 평가 로직 및 fail-open degrade — 위반 시 `Status=unavailable|blocked_by_policy|quota_exhausted|timeout`으로 안전하게 전환, audit 필드에 quota 수치 기록
- **Acceptance**:
  - `QuotaUsed >= QuotaLimit` 시 `Status=quota_exhausted`, `Audit.QuotaRemaining=0` 기록
  - `LicenseAllowed=false` 시 `Status=blocked_by_policy`, `Confidence=low`
  - 4종 control 위반 시에도 dispatcher가 알람 전달을 계속 진행한다 (fail-open)
  - `Audit.QuotaLimit` / `QuotaUsed` / `QuotaRemaining` 필드가 fail-open 판정 후에도 감사 기록에 남는다
- **Source**: `pkg/ruler/aigenerator/storeaware.go`, `pkg/ruler/aigenerator/dispatchhook/`
- **일정**: 2026-07-03 ~ 2026-07-07 (3영업일, 선행: 1.2.4)
- **Effort**: TBD

### WBS-1.2.6 — Dispatch Hook Integration (dispatch_hook_integration)

- **Deliverable**: `dispatchhook.Hook.Apply()` 구현 — AI context를 dispatcher에 전파 (F6 cross-cut), `DefaultGenerateTimeout=1s` 강제, SOP lookup 실패·generator timeout·history upsert 실패 전 케이스에서 error 미반환 보장
- **Acceptance**:
  - generator가 1초 초과 시 입력 annotations 그대로 반환, `"ai dispatch hook: generate failed"` warn 로그 발생
  - `sopStore.List` 실패 시 hook이 error를 반환하지 않고 입력 annotations 그대로 반환한다
  - `StoreAware`의 `AIConfigStore.Get` 실패 시 `envFallback.Generate`가 처리를 이어받는다
- **Source**: `pkg/ruler/aigenerator/dispatchhook/`
- **일정**: 2026-07-08 ~ 2026-07-10 (3영업일, 선행: 1.2.5)
- **Effort**: TBD

## Owner
TBD (TBC)

## Estimated Effort
TBD

## Dependencies
- WBS-1.0 공통 기반 모듈 (audit sink, tenant policy)
- WBS-1.1 SOP 그라운딩 서비스 (grounding 입력)

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
