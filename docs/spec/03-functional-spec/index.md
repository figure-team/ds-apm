---
id: SPEC-INDEX
title: DS-APM 기능명세서
type: srs-index
template: ISO/IEC/IEEE 29148-lite + Spec by Example
status: draft
sections_included: [§1, §2, §3, §4, §5]
updated: 2026-06-02
---

# DS-APM 기능명세서 (SRS)

> **템플릿**: ISO/IEC/IEEE 29148:2018 경량형 + Specification by Example (Gherkin acceptance criteria)
> **표기 컨벤션**: `shall` = "~해야 한다", REQ-X.X / NF-X.X 고유 ID
> **상태**: 본문 작성 완료. 모듈별 상세는 `modules/F0~F8.md`에 분산.

## §1. Introduction

### §1.1 Purpose

본 문서는 AIOpsAgent가 SigNoz community 빌드 위에서 **Incident → SOP runbook → Operator handoff** 흐름을 자동화하기 위해 요구하는 기능·비기능 요구사항을 정의한다. ISO/IEC/IEEE 29148:2018 SRS 템플릿을 POC 단계에 맞게 경량화한 형태이며, 각 기능 요구사항은 Gherkin acceptance criteria로 보강된다 (각 모듈 `F*.md`).

본 문서는 산출물 4종 중 하나로, [arc42 Overview](../01-overview/index.md), [Use Case 카탈로그](../02-usecase/index.md), [WBS](../04-wbs/index.md)와 traceability ID로 연결된다.

### §1.2 Document Conventions
- 요구사항은 모두 `shall` = "~해야 한다" 사용
- ID 컨벤션: `REQ-F{모듈}.{번호}` (functional), `NF-{모듈}.{번호}` (non-functional)
- Gherkin acceptance criteria는 영문 (godog 호환)
- 본문 한국어, 코드/필드명/ID 영문

### §1.3 Intended Audience

| 독자 | 사용 목적 |
|---|---|
| **운영자 (Operator)** | UC-001~003 자신 담당 단계(draft 검수, DLQ replay)의 응답 시간·실패 fallback 정책 확인 |
| **SRE** | NF-* 비기능 요건(p95 latency, DLQ persistence, redaction coverage) 충족 여부 확인 |
| **Platform Admin** | tenant policy / AI strategy / audit sink / DLQ sink 운영 설정 (F0, F2, F4, F5) |
| **Security** | PII redaction 정책 (F7), HMAC follow-up (NF-5.3.1), tenant 격리 평가 (F4) |
| **개발자** | 각 모듈 인터페이스·예외 처리 (`F*.md`) + acceptance Gherkin |
| **감리·QA** | traceability matrix와 ID 정합성 검증 (`_shared/traceability.md`) |

### §1.4 Product Scope
- **In Scope**: AIOpsAgent (incident handoff, SOP grounding, AI runbook, notification dispatch, PII redaction, DLQ/replay) — F0~F8 9개 모듈.
- **Out of Scope**: SigNoz upstream 기능, Enterprise 모듈(`ee/`, `cmd/enterprise/`), vector retrieval 기반 SOP grounding, Redis idempotency cache, y2i 관련 기능 (영구 비활성화).

### §1.5 References
- [`../_foundation/baseline.md`](../_foundation/baseline.md) — 변경 표면 + 커밋 ↔ 모듈 매핑
- [`../_shared/traceability.md`](../_shared/traceability.md) — 진실의 원천: UC × F × WBS 매트릭스
- [`../_shared/glossary.md`](../_shared/glossary.md) — 31개 용어

## §2. Overall Description

### §2.1 Product Perspective

**AIOpsAgent**는 SigNoz Community 빌드의 알림 처리 경로에 운영 자동화(SOP 그라운딩·AI 초안·DLQ 재처리) 단계를 추가하는 확장 모듈 그룹이다. `pkg/alertmanager/`, `pkg/ruler/`, `cmd/community/`에 통합된 Go-native MVP이며 SigNoz 없이는 동작하지 않는다.

fork 프레이밍 금지 — AIOpsAgent는 SigNoz의 fork가 아니라 같은 코드라인 위의 확장이다.

| 의존 대상 | 역할 |
|---|---|
| **SigNoz Ruler** | alert evaluation + firing |
| **SigNoz Alertmanager (`dispatch.Dispatcher`)** | dispatcher hot path — F6이 wrapping |
| **PostgreSQL (bun ORM)** | SOP store (마이그레이션 078, `ds_sop_documents`) |
| **LLM Provider** | HTTP/JSON. 200/401/403/429/5xx 표준 status code 필수 |

### §2.2 User Classes

| Class | AIOpsAgent 상호작용 |
|---|---|
| **운영자 (Operator)** | UC-001 draft 검수, UC-002 DLQ replay, UC-003 degraded mode 알림 수신 |
| **SRE** | meta-alert 수신, 자격증명 회전 escalation |
| **Platform Admin** | tenant policy, AI strategy, audit/DLQ sink 설정 |
| **Security** | PII redaction 정책 검토, HMAC 정책 결정 |

### §2.3 Operating Environment

| 카테고리 | 요구사항 |
|---|---|
| **Runtime** | Go 1.x. 단일 binary `cmd/community/`. |
| **Storage (SQL)** | PostgreSQL (bun ORM). 마이그레이션 078의 `ds_sop_documents`. |
| **Storage (Observability)** | ClickHouse — SigNoz upstream 그대로. schema 변경 없음. |
| **LLM Provider** | HTTP-accessible. 401/403/429 표준 응답 필수. |
| **File system** | `var/audit/pilot-events.jsonl`, `var/dlq/*.jsonl` — 50 MiB rotation. |

### §2.4 Design and Implementation Constraints

| ID | 제약 |
|---|---|
| **C-1** | Go 언어로 작성해야 한다 (Python `ds_apm_poc` 폐기). |
| **C-2** | SigNoz upstream internal API를 직접 사용해야 한다. wrapping 허용. |
| **C-3** | Enterprise 모듈(`ee/`, `cmd/enterprise/`)에는 변경을 가하지 않아야 한다. |
| **C-4** | Multi-tenant 격리·PII 처리는 production-ready 아님 (README 명시). |
| **C-5** | y2i 관련 기능은 영구 비활성화. WBS에서도 명시적 OUT OF SCOPE. |

## §3. System Features (모듈별 상세)

| ID | 모듈 | 상태 | 파일 |
|---|---|---|---|
| F0 | 공통 기반 모듈 (Foundation Core) / Pilot Scaffolding | planned | [F0-foundation.md](modules/F0-foundation.md) |
| F1 | SOP Grounding & Store | planned | [F1-sop-grounding.md](modules/F1-sop-grounding.md) |
| F2 | AI Runbook Drafting (with history) | planned | [F2-ai-drafting.md](modules/F2-ai-drafting.md) |
| F3 | AI Quota Controls (fail-open) | planned | [F3-ai-quota.md](modules/F3-ai-quota.md) |
| F4 | Multi-tenant Scope | planned | [F4-multi-tenant.md](modules/F4-multi-tenant.md) |
| F5 | Audit | planned | [F5-audit.md](modules/F5-audit.md) |
| F6 | Notification Dispatch (5 채널) | planned | [F6-notification-dispatch.md](modules/F6-notification-dispatch.md) |
| F7 | PII Redaction | planned | [F7-pii-redaction.md](modules/F7-pii-redaction.md) |
| F8 | DLQ + Replay | planned | [F8-dlq-replay.md](modules/F8-dlq-replay.md) |

각 모듈 파일은 4섹션 구조: 책임 / 인터페이스 요지 / 핵심 동작 / 예외·복구 + Acceptance Criteria + Traceability.

## §4. External Interface Requirements

### §4.1 User Interfaces

| UI | 역할 | 현 구현 | Open Item |
|---|---|---|---|
| **운영자 draft 검수 화면** | UC-001 단계 6~7. `approval_status: pending → approved/rejected`. | SigNoz UI in-app (현재 구현). | frontend 변경 영역 파일별 식별 미완료 — R-5 |
| **DLQ 조회·replay UI/CLI** | UC-002 단계 6~7. | CLI/UI는 F8 범위 밖. ledger + sink만 제공. | follow-up |
| **Slack interactive approve** | UC-001 단계 6 alternate. | 미구현. Teams는 영구 미지원 (research §7.2). | follow-up |

### §4.2 Software Interfaces

| 인터페이스 | 방향 | 비고 |
|---|---|---|
| **SigNoz Alertmanager** | AIOpsAgent → Dispatcher hot path | F6, F8 |
| **LLM Provider** | outbound | HTTP/JSON. UC-003 fail-open 분기 트리거 |
| **PostgreSQL (bun ORM)** | AIOpsAgent ↔ DB | F1 SOP store |

### §4.3 Communications Interfaces

**Inbound**: Alertmanager webhook v4 — `POST /webhook`. 각 alert: `status`, `labels`, `annotations`, `fingerprint`.

**Outbound (5 채널)**:

| 채널 | 프로토콜 | 제약 |
|---|---|---|
| Slack | Incoming Webhook + Block Kit | username/icon override 불가 |
| MS Teams v2 | Adaptive Card v1.4 | `Action.OpenUrl`만. `Action.Submit` 미지원. |
| PagerDuty | Events API v2 | `dedup_key` = idempotency_key |
| Generic Webhook | JSON POST | 자유 schema |
| Email | SMTP MIME | Subject prefix `[SEV-x]` |

## §5. Other Nonfunctional Requirements

### §5.1 Performance
- **NF-5.1.1** webhook 수신 → 채널 2xx 응답 p95 latency 30초 이하 (운영자 approve 시간 제외).
- **NF-5.1.2** dispatcher hot path AI hook p95 latency 1초 이하 (`DefaultGenerateTimeout`).

### §5.2 Safety
- **NF-5.2.1** LLM 실패는 silent drop이 아닌 fallback dispatch 또는 meta-alert 중 하나로 발화해야 한다.
- **NF-5.2.2** AI Engine 도달 전 PII (email, phone, 16자+ secret) 100% redaction 필수.
- **NF-5.2.3** Audit sink 등록 실패는 서버 부팅을 막지 않아야 한다.
- **NF-5.2.4** 채널 dispatch 실패는 dispatcher 자체를 중단시키지 않아야 한다.

### §5.3 Security
- **NF-5.3.1 (HMAC — OPEN)** Replay payload는 HMAC으로 서명되어야 한다. **정책 미정 — open follow-up.** 미해결인 한 production-ready 선언 불가.
- **NF-5.3.2** 모든 outbound 채널 호출은 TLS 1.2 이상.
- **NF-5.3.3** Secret 자격증명은 contract response에 노출되지 않아야 한다.
- **NF-5.3.4** Cross-tenant lookup은 `ErrSOPDocumentNotFound`로 통일.

### §5.4 Software Quality Attributes
- **NF-5.4.1** 재시도 모두 실패 시에도 원본 페이로드와 시도 이력은 DLQ에 보존.
- **NF-5.4.2** 동일 `(fingerprint, channel)` 중복 dispatch 0건.
- **NF-5.4.3** Dispatch/draft/SOP 접근/fail-open 1건당 audit row 1건 이상.

### §5.5 Business Rules
- **NF-5.5.1** 운영자가 받는 알람에서 alert payload + SOP 본문 정보 손실 0건. silent drop이 AI 부정확함보다 나쁘다.
- **NF-5.5.2** Audit log는 reproducibility 토대로만 사용. 개인 책임 추궁 목적 사용 금지.
- **NF-5.5.3** SOP `staleness_days` 90일 초과 시 grounding 보류, raw alert만 전달.

## Traceability
→ [`../_shared/traceability.md`](../_shared/traceability.md)

| 차원 | 매핑 |
|---|---|
| F0~F8 ↔ UC-001~003 | traceability.md §1 |
| F0~F8 ↔ WBS-1.0~1.5 | traceability.md §2 |
| Open items (HMAC, frontend) | traceability.md §6 |
