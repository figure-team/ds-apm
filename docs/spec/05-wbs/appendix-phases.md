---
id: WBS-APPENDIX-PHASES
title: 부록 §A — 전략 로드맵 ↔ WBS·CF 연계
type: wbs-appendix
status: draft
updated: 2026-06-08
---

# 부록 §A — 전략 로드맵 ↔ WBS·CF 연계

> 사업 전략서([`../_foundation/source-strategy-brief.md`](../_foundation/source-strategy-brief.md))의 단계별 로드맵을 WBS 컴포넌트(구현)·CF·로드맵 역량에 매핑한다. WBS 본문(6 컴포넌트)은 전략 **1~2단계의 SOP/알림 슬라이스**에 해당하며, 그 이후 단계는 로드맵(CF-7~10)이다.

## §A.1 전략 단계 → WBS·CF

| 전략 단계 | 기간 | 주요 과제 (원문) | WBS / CF | 상태 |
|---|---|---|---|---|
| **1단계** | 26'2Q~3Q | 데이터 수집 체계·OTel 적용·Log/Metric 중앙화 | (전제: SigNoz/OTel upstream) | 전제 |
| | | 핵심 알람 필터링 | SigNoz Ruler + 라우팅 | 전제+CF-3 |
| | | **Runbook/SOP 표준화·자동화** (대상 식별, 실패 시 fallback) | **WBS-1.0·1.1·1.2 / CF-6·1·2** | ★ 구현 |
| **2단계** | 26'3Q~4Q | SigNoz 통합·대시보드 | WBS-1.0 / CF-6 (통합) + upstream | ◐ |
| | | Runbook 검증 | WBS-1.1 / CF-1 (승인·만료 정책) | ◐ |
| | | **반자동 RCA + 메신저 연동** | **WBS-1.2·1.3 / CF-2·3** | ★ 구현 |
| | | Distributed Tracing | (전제: SigNoz upstream) | 전제 |
| | | **미등록 예외 인지·코드연계 분석·자산화** (이상 탐지 1차) | **WBS-1.6 / CF-7(1차)** | ◑ 설계 확정 (planned) |
| | | AI 이상 탐지 모델 학습(기준선) | **CF-7 후속** | ○ planned |
| **3단계** | 26'4Q | 자동화 운영 POC · Webhook/Runbook 자동조치 스크립트 | **CF-8** (승인 기반) | ○ planned |
| (안전장치) | — | Human-in-loop·rollback·승인·PII | WBS-1.4·1.5 / CF-4·5 + CF-2 HITL | ★ 구현 |

## §A.2 별첨1 장기 로드맵 → CF

| 계 | 일정 | 내용 | CF | 상태 |
|---|---|---|---|---|
| 관제 | 26년 3~4Q | 데이터 수집·실시간 감지·Runbook 자동화·반자동 RCA·메신저 연동 | CF-1·2·3·6 (+전제) | ★ 대부분 구현 |
| 산출물/지식화 | 27년 1Q | 산출물 작성 자동화·Incident 구조화·라벨링·RCA 템플릿·보고서 자동 생성·LLM Wiki | **CF-9** | ○ planned |
| ITSM | 27년 3Q | OSS 기반 ITSM·운영 자동화 Workflow | **CF-10** | ○ planned |

## §A.3 해석

- **WBS 본문(WBS-1.0~1.5)** 은 전략 1~2단계의 *SOP→AI 가이드→핸드오프→안전·신뢰* 슬라이스를 구현한 PoC다 (**수행 완료**, 추정 기간 2026-05-25~08-28 — 실제 커밋 일자 아님).
- **2단계 후반 이상탐지(CF-7)는 1차(미등록 예외 대응)가 설계 확정되어 WBS-1.6으로 편입**(planned, [Epic 7](../03-epics/epic-7-unknown-exception.md)). 기준선 학습(FR-CF7.1)은 후속. **3단계(자동조치 CF-8)·별첨1(자산화 CF-9·ITSM CF-10)** 은 미구현 로드맵으로, 기능명세 §6/§9.3과 본 부록에만 명시한다 (저fidelity).
- 성과 목표(50% 리소스 세이브)의 **SM-2·SM-3(상시확인·인시던트분석)** 은 본 WBS 구현분으로 직접 기여, **SM-4·SM-5(보고 자산화·장애처리 자동화)** 는 CF-8·9 완성 시 달성 (기능명세 §3·§9.3).

## §A.4 (참고) 구 Phase 시간선

이전 P0~P5 phase 표기는 본 전략 단계(1/2/3)로 대체됐다. 커밋 시간선은 [`../_foundation/baseline.md`](../_foundation/baseline.md) §4, drift는 [`../_shared/component-source-map.md`](../_shared/component-source-map.md) §9.

## Traceability
→ [`index.md`](index.md) · [`../_shared/traceability.md`](../_shared/traceability.md) · [`../01-prd/index.md`](../01-prd/index.md) §6
