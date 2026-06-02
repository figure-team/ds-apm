---
id: WBS-INDEX
title: DS-APM WBS
type: wbs-index
template: PMI WBS 2nd ed. + Agile (hybrid)
decomposition_logic: component-oriented
status: draft
updated: 2026-06-02
---

# DS-APM WBS

> **템플릿**: PMI Practice Standard for WBS 2nd ed. (상위 Lv1-3, 100% rule, deliverable-oriented) + Agile (하위 Lv4-5, Epic→Story)
> **Lv2 분해 논리**: component-oriented (실제 `pkg/` 디렉토리에 1:1 매핑)
> **시간선(Phase)은 부록 §A로 분리** (역사 추적용)
> **상태**: 본문 작성 완료. Work package 6건은 `packages/`에 분산.

## 100% Rule

DS-APM 범위 = `WBS-1.0 ∪ WBS-1.1 ∪ ... ∪ WBS-1.5` (자식 합 = 부모 100%, 누락·중복 없음)
**Excluded scope** (OUT OF SCOPE)는 §Excluded Scope 참조.

## WBS Tree (Component-oriented)

```
WBS-1   DS-APM Project (root)
├─ WBS-1.0  공통 기반 모듈 (Foundation Core)              — pilot contract, managed markdown, audit sink, tenant policy
│           Covers: F0, F4, F5
├─ WBS-1.1  SOP 그라운딩 서비스 (SOP Grounding Service)   — SOP store, grounding, file persistence
│           Covers: F1
├─ WBS-1.2  AI 초안 매니저 (AI Drafter Manager)          — runbook drafter, AI generator, quota controls, strategy history
│           Covers: F2, F3
├─ WBS-1.3  알림 디스패처 (Notification Dispatcher)      — 5 채널 adapter + dispatcher
│           Covers: F6
├─ WBS-1.4  PII 마스킹 필터 (PII Masking Filter)         — incident payload redaction
│           Covers: F7
└─ WBS-1.5  DLQ 재처리 서비스 (DLQ Replay Service)       — JSONL DLQ, idempotent replay ledger
            Covers: F8
```

## Work Package 일람

| ID | 제목 | 상태 | 커버 Feature | 파일 |
|---|---|---|---|---|
| WBS-1.0 | 공통 기반 모듈 (Foundation Core) | planned | F0, F4, F5 | [WBS-1-0-foundation.md](packages/WBS-1-0-foundation.md) |
| WBS-1.1 | SOP 그라운딩 서비스 (SOP Grounding Service) | planned | F1 | [WBS-1-1-sop-engine.md](packages/WBS-1-1-sop-engine.md) |
| WBS-1.2 | AI 초안 매니저 (AI Drafter Manager) | planned | F2, F3 | [WBS-1-2-ai-drafter.md](packages/WBS-1-2-ai-drafter.md) |
| WBS-1.3 | 알림 디스패처 (Notification Dispatcher) | planned | F6 | [WBS-1-3-notification-dispatcher.md](packages/WBS-1-3-notification-dispatcher.md) |
| WBS-1.4 | PII 마스킹 필터 (PII Masking Filter) | planned | F7 | [WBS-1-4-pii-redactor.md](packages/WBS-1-4-pii-redactor.md) |
| WBS-1.5 | DLQ 재처리 서비스 (DLQ Replay Service) | planned | F8 | [WBS-1-5-dlq-replay.md](packages/WBS-1-5-dlq-replay.md) |

## 구축 일정

전체 기간: **2026-05-25 ~ 2026-08-28 (약 14주)**

| 컴포넌트 | 기간 | 시작 | 종료 | 의존 |
|---|---|---|---|---|
| WBS-1.0 공통 기반 모듈 | 3주 | 2026-05-25 | 2026-06-12 | (선행, 전체 의존) |
| WBS-1.1 SOP 그라운딩 서비스 | 3주 | 2026-06-15 | 2026-07-03 | WBS-1.0 |
| WBS-1.2 AI 초안 매니저 | 4주 | 2026-06-15 | 2026-07-10 | WBS-1.0 (1.1과 병렬) |
| WBS-1.3 알림 디스패처 | 3주 | 2026-07-13 | 2026-07-31 | WBS-1.1, 1.2 |
| WBS-1.4 PII 마스킹 필터 | 2주 | 2026-07-13 | 2026-07-24 | WBS-1.0 (1.3과 병렬) |
| WBS-1.5 DLQ 재처리 서비스 | 3주 | 2026-08-03 | 2026-08-21 | WBS-1.3 |
| 통합·안정화 버퍼 | 1주 | 2026-08-24 | 2026-08-28 | 전체 (E2E·HMAC 결정·검수) |

### Work Package 단위 일정 (일 단위, 영업일 기준)

| WP ID | 작업명 | 선행 | 시작일 | 종료일 | 기간(영업일) |
|---|---|---|---|---|---|
| 1.0.1 | Pilot 계약 스키마 | — | 2026-05-25 | 2026-05-27 | 3 |
| 1.0.2 | 관리형 Markdown 페이로드 | 1.0.1 | 2026-05-28 | 2026-06-01 | 3 |
| 1.0.3 | 테넌트 격리 정책 | 1.0.2 | 2026-06-02 | 2026-06-04 | 3 |
| 1.0.4 | 감사 Sink 추상화 | 1.0.3 | 2026-06-05 | 2026-06-08 | 2 |
| 1.0.5 | JSONL 감사 Sink 구현 | 1.0.4 | 2026-06-09 | 2026-06-10 | 2 |
| 1.0.6 | community 진입점 와이어업 | 1.0.5 | 2026-06-11 | 2026-06-12 | 2 |
| 1.1.1 | SOP Store 인터페이스 정의 | 1.0.6 | 2026-06-15 | 2026-06-17 | 3 |
| 1.1.2 | SQL 스토어 구현체 | 1.1.1 | 2026-06-18 | 2026-06-22 | 3 |
| 1.1.3 | 파일 영속화 구현체 | 1.1.2 | 2026-06-23 | 2026-06-25 | 3 |
| 1.1.4 | SOP 도메인 타입 | 1.1.3 | 2026-06-26 | 2026-06-29 | 2 |
| 1.1.5 | Grounding 로직 | 1.1.4 | 2026-06-30 | 2026-07-01 | 2 |
| 1.1.6 | Runbook Handler SOP 라우트 | 1.1.5 | 2026-07-02 | 2026-07-03 | 2 |
| 1.2.1 | AI Strategy 도메인 타입 | 1.0.6 | 2026-06-15 | 2026-06-18 | 4 |
| 1.2.2 | LLM Provider 어댑터 | 1.2.1 | 2026-06-19 | 2026-06-24 | 4 |
| 1.2.3 | Strategy 생성·persistence | 1.2.2 | 2026-06-25 | 2026-06-29 | 3 |
| 1.2.4 | Strategy History append | 1.2.3 | 2026-06-30 | 2026-07-02 | 3 |
| 1.2.5 | Quota Controller (fail-open) | 1.2.4 | 2026-07-03 | 2026-07-07 | 3 |
| 1.2.6 | Dispatch Hook Integration | 1.2.5 | 2026-07-08 | 2026-07-10 | 3 |
| 1.3.1 | Dispatcher wrapping | 1.2.6 | 2026-07-13 | 2026-07-15 | 3 |
| 1.3.2 | AI context propagation | 1.3.1 | 2026-07-16 | 2026-07-20 | 3 |
| 1.3.3 | Slack + MS Teams v2 adapter | 1.3.2 | 2026-07-21 | 2026-07-23 | 3 |
| 1.3.4 | PagerDuty adapter | 1.3.3 | 2026-07-24 | 2026-07-27 | 2 |
| 1.3.5 | Webhook + Email adapter | 1.3.4 | 2026-07-28 | 2026-07-29 | 2 |
| 1.3.6 | 5채널 통합 라우팅·전송 검증 | 1.3.5 | 2026-07-30 | 2026-07-31 | 2 |
| 1.4.1 | Redaction rule engine | 1.0.6 | 2026-07-13 | 2026-07-14 | 2 |
| 1.4.2 | Incident payload redaction 적용 | 1.4.1 | 2026-07-15 | 2026-07-16 | 2 |
| 1.4.3 | Audit sink 연동 | 1.4.2 | 2026-07-17 | 2026-07-20 | 2 |
| 1.4.4 | Tenant별 룰 확장 훅 | 1.4.3 | 2026-07-21 | 2026-07-22 | 2 |
| 1.4.5 | OTel Collector 단 이동 검토 | 1.4.4 | 2026-07-23 | 2026-07-24 | 2 |
| 1.5.1 | JSONL DLQ Sink | 1.3.6 | 2026-08-03 | 2026-08-05 | 3 |
| 1.5.2 | Idempotent Replay Ledger | 1.5.1 | 2026-08-06 | 2026-08-10 | 3 |
| 1.5.3 | Dispatcher 통합 | 1.5.2 | 2026-08-11 | 2026-08-13 | 3 |
| 1.5.4 | Replay API 엔드포인트 | 1.5.3 | 2026-08-14 | 2026-08-17 | 2 |
| 1.5.5 | Replay 상태 머신 | 1.5.4 | 2026-08-18 | 2026-08-19 | 2 |
| 1.5.6 | HMAC 정책 (scaffolding only) | 1.5.5 | 2026-08-20 | 2026-08-21 | 2 |

```mermaid
gantt
    title AIOpsAgent WBS 일정 (work package 단위, 2026-05-25 ~ 08-28)
    dateFormat YYYY-MM-DD
    axisFormat %m-%d
    excludes weekends
    section WBS-1.0 공통 기반 모듈
    1.0.1 Pilot 계약 스키마 :t101, 2026-05-25, 3d
    1.0.2 관리형 Markdown 페이로드 :t102, after t101, 3d
    1.0.3 테넌트 격리 정책 :t103, after t102, 3d
    1.0.4 감사 Sink 추상화 :t104, after t103, 2d
    1.0.5 JSONL 감사 Sink 구현 :t105, after t104, 2d
    1.0.6 community 진입점 와이어업 :t106, after t105, 2d
    section WBS-1.1 SOP 그라운딩
    1.1.1 SOP Store 인터페이스 정의 :t111, after t106, 3d
    1.1.2 SQL 스토어 구현체 :t112, after t111, 3d
    1.1.3 파일 영속화 구현체 :t113, after t112, 3d
    1.1.4 SOP 도메인 타입 :t114, after t113, 2d
    1.1.5 Grounding 로직 :t115, after t114, 2d
    1.1.6 Runbook Handler SOP 라우트 :t116, after t115, 2d
    section WBS-1.2 AI 초안 매니저
    1.2.1 AI Strategy 도메인 타입 :t121, after t106, 4d
    1.2.2 LLM Provider 어댑터 :t122, after t121, 4d
    1.2.3 Strategy 생성·persistence :t123, after t122, 3d
    1.2.4 Strategy History append :t124, after t123, 3d
    1.2.5 Quota Controller (fail-open) :t125, after t124, 3d
    1.2.6 Dispatch Hook Integration :t126, after t125, 3d
    section WBS-1.3 알림 디스패처
    1.3.1 Dispatcher wrapping :t131, after t126, 3d
    1.3.2 AI context propagation :t132, after t131, 3d
    1.3.3 Slack + MS Teams v2 adapter :t133, after t132, 3d
    1.3.4 PagerDuty adapter :t134, after t133, 2d
    1.3.5 Webhook + Email adapter :t135, after t134, 2d
    1.3.6 5채널 통합 라우팅·전송 검증 :t136, after t135, 2d
    section WBS-1.4 PII 마스킹
    1.4.1 Redaction rule engine :t141, after t106, 2d
    1.4.2 Incident payload redaction 적용 :t142, after t141, 2d
    1.4.3 Audit sink 연동 :t143, after t142, 2d
    1.4.4 Tenant별 룰 확장 훅 :t144, after t143, 2d
    1.4.5 OTel Collector 단 이동 검토 :t145, after t144, 2d
    section WBS-1.5 DLQ 재처리
    1.5.1 JSONL DLQ Sink :t151, after t136, 3d
    1.5.2 Idempotent Replay Ledger :t152, after t151, 3d
    1.5.3 Dispatcher 통합 :t153, after t152, 3d
    1.5.4 Replay API 엔드포인트 :t154, after t153, 2d
    1.5.5 Replay 상태 머신 :t155, after t154, 2d
    1.5.6 HMAC 정책 (scaffolding only) :t156, after t155, 2d
    section 통합
    통합·안정화 버퍼 :tbuf, after t156, 5d
```

## Excluded Scope (명시적 OUT OF SCOPE)

- **SigNoz upstream 기능 자체** — AIOpsAgent 범위 밖
- **Enterprise 모듈** (`ee/`, `cmd/enterprise/`) — 별도 라이선스
- **y2i 관련 기능** — 영구 비활성화 (메모리 정책)

## WBS Dictionary

각 work package의 Deliverable / Acceptance / Owner / Effort / Dependencies / Verification은 `packages/` 안 개별 파일에서 관리.

## Milestones

WBS 자체는 정적 scope 문서이며 진행률을 박지 않는다 (research-skills-a-methods.md §4.4). 다음 milestone은 시간선 부록 §A의 phase와 다르게 **production-readiness gate** 기준으로 분리한다.

| Milestone | 목표일 | 기준 | 현재 상태 | 의존 |
|---|---|---|---|---|
| **M-1 기반 완료** | 2026-06-12 | WBS-1.0 acceptance 통과. F0~F8 9 모듈 정의, UC-001~003 본문·시퀀스·NFR 작성, 산출물 4종 합의. | **예정** (Phase 0 진입) | WBS-1.0 |
| **M-2 도메인 엔진 완료** | 2026-07-10 | WBS-1.1 + WBS-1.2 acceptance 통과. SOP grounding + AI 초안 생성 E2E 동작. | **미진입** | M-1 |
| **M-3 전달·안전 완료** | 2026-07-31 | WBS-1.3 + WBS-1.4 acceptance 통과. 5채널 dispatch + PII 마스킹 E2E 동작. | **미진입** | M-2 |
| **M-4 신뢰성·Beta GA** | 2026-08-21 | WBS-1.5 acceptance 통과. HMAC 정책 결정 (NF-5.3.1). DLQ 운영 UI/CLI 최소 1종 제공. Frontend 변경 영역 식별·문서화. | **미진입 (NF-5.3.1 미해결, frontend R-5 open)** | M-3 + ADR-003 결정 |
| **M-5 Production-readiness** | 2026-08-28 | 통합·안정화 버퍼 소화. Multi-tenant 격리 강화. PII Collector 단 이동 결정. HMAC 정책 운영 검증. | **미진입** | M-4 + R-3, R-4, R-7 follow-up 클리어 |

추가 milestone 후보 (현재 미일정):
- **M-X Vector retrieval 도입** — explicit-label binding을 vector retrieval로 확장 (현재 OUT OF SCOPE).
- **M-X 6번째 채널 추가** — ADR-002 (channel adapter pattern) 결정 트리거.

## Appendix
- [Phase 시간선 (P0~P5)](appendix-phases.md) — 착수 전 사전 검토 단계의 시간선 분할 (역사 추적용)

## Traceability
- WBS × Feature × Use Case 매트릭스: [`../_shared/traceability.md`](../_shared/traceability.md)
- Open items (HMAC, frontend, archive 자료): traceability.md §6
