---
id: CF-7
title: 메트릭 이상 탐지 — 통계 기준선 이탈 경고 (Anomaly Detection)
status: implemented
jtbd: [JTBD-1]
maps_modules: [F9]
source_paths:
  - pkg/query-service/rules/anomaly_rule.go
implements_uj: []
covered_by_wbs: [WBS-1.6]
fr_ids: [FR-CF7.1]
updated: 2026-06-12
caveats: "v1=통계 기반(z-score, 이동평균±k·σ) 구현. 학습형/계절성 모델은 후속(FR-CF7.2, planned). CF-7은 알람을 *생성*하는 탐지 기능 — 여정의 트리거 소스이지 implements 아님(implements_uj 빈 값)."
open_items:
  - 학습형/계절성 기준선 모델 (FR-CF7.2 — planned, 전략 2단계 '이상 탐지 모델 데이터 학습')
  - CF-11 트리거용 anomaly provenance 노출(ruleId→rule-type) — CF-11 설계 §10 seam과 정합
---

# CF-7 — 메트릭 이상 탐지 (통계 기준선 이탈)

> **고객 가치 (JTBD-1)**: 운영자는 고정 임계치로는 잡기 어려운 *평소와 다른* 메트릭 거동(완만한 드리프트·간헐 스파이크)을 사전 경고로 받는다 — 임계치를 매번 손으로 튜닝하지 않고도 이상을 인지한다.
> **상태**: implemented (`anomaly_rule.go`). v1은 단일 평가 윈도우 위의 통계(z-score). 학습형/계절성 모델은 후속.

## CF-7.1 개요 (사용자 관점)

기존 `ThresholdRule`은 원시값을 고정 목표치와 비교한다. 정상 범위가 시간대·부하에 따라 변하는 지표에서는 임계치가 너무 빡빡하거나(오탐) 너무 느슨하다(미탐). CF-7의 `AnomalyRule`은 각 시계열을 **이상 점수(z-score)** 로 변환한다 — 직전 데이터포인트들로 형성한 **이동평균 ± k·σ 기준선**에 대한 최신 데이터포인트의 편차. 이 점수를 기존 임계 파이프라인에 통과시키되, 목표치는 **σ 단위의 밴드 폭 k** 가 된다.

즉 운영자는 "값이 X를 넘으면"이 아니라 "값이 평소 패턴에서 k·σ 이상 벗어나면" 경고를 받는다. 산출된 이상 알람은 다른 알람과 동일하게 디스패치 경로로 흐르며(UJ-1), 특히 **연계 SOP가 없는 이상 알람은 CF-11(AI 코드베이스 RCA)의 트리거 조건** 이 된다.

**범위 명확화**: CF-7은 *탐지/알람 생성* 기능이다. SOP 연계·AI 가이드·핸드오프(CF-1·2·3)는 생성된 알람을 소비하는 하류이며, CF-7은 그 여정의 **트리거 소스**다(implements 아님).

## CF-7.2 기능 요구 (FR)

### FR-CF7.1 — 운영자는 메트릭이 통계 기준선을 벗어나면 이상 알람을 받는다
- **무엇을**: 시스템은 각 시계열을 이동평균 ± k·σ 기준선 대비 z-score로 변환하고, 그 점수가 밴드 폭(k·σ)을 넘으면 이상 알람을 발생시킨다. 기준선은 평가 윈도우 직전 데이터포인트들로 형성되고 최신 데이터포인트가 그에 대해 평가된다.
- **Acceptance**:
  ```gherkin
  Given Anomaly 타입 rule이 시계열에 대해 기준선(이동평균±k·σ)을 형성하고
    And 최신 데이터포인트가 기준선에서 밴드 폭(k·σ)을 넘어 이탈했을 때
  When rule이 평가되면
  Then 해당 시계열은 이상 점수가 임계를 초과해 이상 알람(샘플)이 생성된다

  Given 최신 데이터포인트가 기준선 밴드 내에 있을 때
  When rule이 평가되면
  Then 이상 알람은 생성되지 않는다 (매칭 샘플 비어 있음)
  ```
- **구현 근거**: [`pkg/query-service/rules/anomaly_rule.go`](../../../../pkg/query-service/rules/anomaly_rule.go) — `AnomalyRule`(BaseRule 임베드), `scoreSeries`(원시 시계열 → 부호 있는 z-score), `evalSeries`(점수를 이상 임계 target=k·σ로 평가, 최신 포인트가 밴드 내면 빈 결과), `ruletypes.Baseline`. rule 타입 식별은 `RuleTypeAnomaly`([`pkg/types/ruletypes/rule_type.go`](../../../../pkg/types/ruletypes/rule_type.go)). · WBS-1.6

## CF-7.3 비기능 요건 (feature-specific)

- **NF-CF7.1** v1은 **단일 평가 윈도우 위의 단순 통계** — 결정적·설명 가능(z-score). 학습형/계절성 모델은 후속(FR-CF7.2).
- **NF-CF7.2** 이상 점수는 **기존 임계 파이프라인 재사용** — 별도 평가 경로 신설 없이 target=k·σ로 통합(`ThresholdRule`과 동일 하류).
- **NF-CF7.3** 생성된 이상 알람은 일반 알람과 동일하게 PII·DLQ·감사·핸드오프 경로를 따른다(CF-3·4·5·6 재사용).

## CF-7.4 Open / Non-goal

- **학습형/계절성 기준선 (FR-CF7.2 — planned)**: 시간대·요일 등 계절성을 학습하는 기준선 모델. 전략 2단계 "AI 이상 탐지 모델 데이터 학습(기준선 설정)"에 대응. 현재 v1(통계)로 충분한 1차 가치 제공, 학습형은 데이터 축적 후.
- **CF-11 연계**: 연계 SOP 없는 이상 알람은 CF-11 트리거의 anomaly 조건. CF-11은 현재 명시적 anomaly 라벨/주석 기반(fail-closed)이며, `ruleId`→rule-type(`RuleTypeAnomaly`) provenance 노출은 CF-11 설계 §10 seam과 정합해 후행.
- **메트릭 수집·저장**: OTel/ClickHouse upstream 전제 — Non-goal(PRD §9.1).

## CF-7.5 Traceability

- JTBD: JTBD-1(상시 모니터링 자동화) · User Journey: UJ-1·UJ-5의 **알람 트리거 소스**(implements 아님 — traceability §1 (트리거))
- Covered by WBS: WBS-1.6 · Epic: [Epic 7](../../03-epics/epic-7-anomaly.md) · Stories: 7.1
- 모듈: F9 (metric anomaly detection — implemented, 신규 테이블 없음 — rule 엔진 확장)
- → 상위: [`../index.md`](../index.md) §6·§7.1 · 하류 소비: [CF-11 AI 코드베이스 RCA](CF-11-code-rca.md) · 전략: [`source-strategy-brief.md`](../../_foundation/source-strategy-brief.md) 2단계
