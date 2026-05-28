---
id: SPEC-INDEX
title: DS-APM 기능명세서
type: srs-index
template: ISO/IEC/IEEE 29148-lite + Spec by Example
status: draft
sections_included: [§1, §2, §3, §4, §5]
updated: 2026-05-29
---

# DS-APM 기능명세서 (SRS)

> **템플릿**: ISO/IEC/IEEE 29148:2018 경량형 + Specification by Example (Gherkin acceptance criteria)
> **표기 컨벤션**: `shall` = "~해야 한다", REQ-X.X / NF-X.X 고유 ID
> **상태**: 본문 작성 완료. 모듈별 상세는 `modules/F0~F8.md`에 분산.

## §1. Introduction

### §1.1 Purpose

본 문서는 DS-APM이 SigNoz community 빌드 위에서 **Incident → SOP runbook → Operator handoff** 흐름을 자동화하기 위해 요구하는 기능·비기능 요구사항을 정의해야 한다. 산출물은 ISO/IEC/IEEE 29148:2018의 SRS 템플릿을 POC 단계에 맞게 경량화(`§1, §2, §3, §4, §5`)한 형태이며, 각 기능 요구사항은 Specification by Example의 Living Documentation 원칙에 따라 Gherkin acceptance criteria로 보강된다 (각 모듈 `F*.md` §F*.7).

본 문서는 산출물 4종 중 하나로, [arc42 Overview](../01-overview/index.md), [Use Case 카탈로그](../02-usecase/index.md), [WBS](../04-wbs/index.md)와 traceability ID로 연결된다.

### §1.2 Document Conventions
- 요구사항은 모두 `shall` = "~해야 한다" 사용
- ID 컨벤션: `REQ-F{모듈}.{번호}` (functional), `NF-{모듈}.{번호}` (non-functional)
- Gherkin acceptance criteria는 영문 (godog 호환)
- 본문 한국어, 코드/필드명/ID 영문

### §1.3 Intended Audience

| 독자 | 사용 목적 |
|---|---|
| **운영자 (Operator)** | UC-001~003에서 자신이 담당하는 단계(draft 검수, DLQ replay)의 정확한 응답 시간·실패 fallback 정책 확인 |
| **SRE** | NF-* 비기능 요건(p95 latency, DLQ persistence, redaction coverage) 충족 여부 확인 + meta-alert 정책 (F3, F7, F8) |
| **Platform Admin** | tenant policy / AI strategy / audit sink / DLQ sink의 운영 설정 (F0, F4, F5) |
| **Security** | PII redaction policy (F7), HMAC follow-up (NF-5.3.1), tenant 격리 NFR (NF-F1.1, NF-F4.*) |
| **개발자** | 각 모듈의 인터페이스·데이터 모델·예외 처리 (각 `F*.md`의 §2~§6) + acceptance Gherkin (§*.7) |
| **감리·QA** | traceability matrix와 ID 정합성 검증 (`_shared/traceability.md`) |

### §1.4 Product Scope
- **In Scope**: DS-APM 확장 레이어 (incident handoff, SOP grounding, AI runbook, notification dispatch, PII redaction, DLQ/replay) — F0~F8 9개 모듈 전부.
- **Out of Scope**: SigNoz upstream 기능 자체, Enterprise 모듈(`ee/`, `cmd/enterprise/`), vector retrieval 기반 SOP grounding (현재 explicit-label binding만), Redis 기반 idempotency cache (현재 파일 기반 JSONL), y2i 관련 기능 (영구 비활성화).

### §1.5 References
- [`../_foundation/baseline.md`](../_foundation/baseline.md) — 11 커밋·100 파일·+12,632 LOC 변경 표면 + 커밋 ↔ 모듈 매핑
- [`../_foundation/research-skills-a-methods.md`](../_foundation/research-skills-a-methods.md) §3 — ISO/IEC/IEEE 29148-lite + Spec by Example 채택 근거
- [`../_foundation/research-skills-c-domain.md`](../_foundation/research-skills-c-domain.md) — Alertmanager v4 / PagerDuty Events v2 / Slack Block Kit / MS Teams Adaptive Card v1.4 / DLQ + idempotency / PII OTel processor
- [`../_shared/traceability.md`](../_shared/traceability.md) — 진실의 원천: UC × F × WBS 매트릭스
- [`../_shared/glossary.md`](../_shared/glossary.md) — 31개 용어

## §2. Overall Description

### §2.1 Product Perspective

DS-APM은 **SigNoz community 빌드의 확장 레이어**다. 별도 서비스가 아니라 SigNoz의 `pkg/alertmanager/`, `pkg/ruler/`, `pkg/types/ruletypes/`, `cmd/community/`에 직접 흡수된 Go-native MVP이며, SigNoz 없이는 동작하지 않는다 (ADR-001).

| 의존 대상 | 역할 | 변경 가능성 |
|---|---|---|
| **SigNoz Ruler** | alert evaluation + firing | upstream API 변경 시 DS-APM 영향 — 모니터링 필요 (R-6) |
| **SigNoz Alertmanager (`dispatch.Dispatcher`)** | dispatcher hot path | F6은 이를 wrapping해서 `aiHook` + `dlqSink`를 끼워넣음 |
| **SigNoz `notify.Stage`** | 채널별 notification stage | 5채널 어댑터 (Slack / Teams v2 / PagerDuty / Webhook / Email)를 패치 |
| **SigNoz ClickHouse 연결** | observability backend | DS-APM은 metrics/logs sink로만 활용, 직접 schema 변경 안 함 |

fork 프레이밍 **금지** — DS-APM은 SigNoz의 fork가 아니라 같은 코드라인 위의 흡수 확장이다 (메모리 정책 "var/signoz는 우리 코드").

### §2.2 Product Features (high-level)
→ §3 (모듈별 상세). F0~F8 9 모듈 = WBS-1.0~1.5 6 components.

### §2.3 User Classes

| Class | 식별 | DS-APM 상호작용 |
|---|---|---|
| **운영자 (Operator)** | on-call rotation 등재 | UC-001 draft 검수 (단계 6~7), UC-002 DLQ replay (단계 6~7), UC-003 degraded mode 알림 수신 |
| **SRE** | service ownership | meta-alert 수신 (F3 fail-open / F7 redaction spike / F8 DLQ depth), 자격증명 회전 escalation (UC-003 7a) |
| **Platform Admin** | tenant·정책 권한 | tenant policy 등록 (F4), AI strategy 활성화 (F2), audit/DLQ sink 운영 설정 (F0, F5, F8) |
| **Security** | 감사 권한 | PII redaction 정책 검토 (F7), HMAC 정책 결정 (NF-5.3.1 — open), tenant 격리 production-readiness 평가 (R-3) |

### §2.4 Operating Environment

| 카테고리 | 요구사항 |
|---|---|
| **Runtime** | Go 1.x (SigNoz community 호환 버전). 단일 binary `cmd/community/`. |
| **Storage (SQL)** | PostgreSQL (bun ORM). SOP store는 마이그레이션 078의 `ds_sop_documents` 테이블 사용 (F1). |
| **Storage (Observability)** | ClickHouse — SigNoz upstream 그대로. DS-APM은 schema 변경 안 함. |
| **Storage (Cache, optional)** | Redis — 현재 idempotency cache로 미사용. ledger는 파일 기반 (F8). future R-7 follow-up. |
| **LLM Provider** | HTTP-accessible. 자체 호스팅 모델 또는 외부 SaaS (OpenAI, Anthropic, Bedrock 등). 401/403/429 응답을 표준 status code로 반환해야 함 (F3, UC-003). |
| **File system** | `var/audit/pilot-events.jsonl` (audit), `var/dlq/*.jsonl` (DLQ) 50 MiB rotation. fsync 정책으로 프로세스 crash 시 1초 이내 마지막 N개 entry 손실 허용 (MVP). |
| **OS / Container** | Linux container. SigNoz upstream과 동일 base image. |

### §2.5 Design and Implementation Constraints

| ID | 제약 | 출처 |
|---|---|---|
| **C-1** | Go 언어로 작성해야 한다 (Python `ds_apm_poc` 폐기, ADR-001). |
| **C-2** | SigNoz upstream의 internal API (`dispatch.Dispatcher`, `notify.Stage`, `provider.Alerts`)를 직접 사용해야 한다. wrapping은 허용. |
| **C-3** | Enterprise 모듈(`ee/`, `cmd/enterprise/`)에는 변경을 가하지 않아야 한다 — 별도 라이선스. |
| **C-4** | Multi-tenant 격리·PII 처리는 **production-ready 아님** (README 명시). 본 SRS의 모든 NFR은 MVP 기준이며, production hardening은 별도 milestone에서. |
| **C-5** | DS-APM 산출물은 `figure-team/ds-apm` 공개 스냅샷에 single squash commit으로 노출되지만, 실제 작업 히스토리는 `workspace_archive/ds-apm/var/signoz` nested repo에서 관리된다. nested repo의 자체 `.git` 확인이 모든 변경의 필수 절차다 (R-1). |
| **C-6** | y2i 관련 기능은 영구 비활성화 (메모리 정책). WBS에서도 명시적 OUT OF SCOPE. |

### §2.6 User Documentation

- 운영자/SRE 대상 운영 가이드는 본 산출물 범위 밖. 본 SRS는 시스템 외부 인터페이스·NFR을 정의한다.
- Acceptance Gherkin은 각 모듈 `F*.md` §F*.7에서 godog 호환 형태로 제공.

### §2.7 Assumptions and Dependencies

| 가정 | 영향 시나리오 |
|---|---|
| SigNoz community build가 OpenTelemetry-네이티브로 alert label과 resource attribute를 보존해야 한다. | OTel attribute drop 시 SOP grounding key가 누락되어 UC-001 5a Extension으로 전이 |
| Alertmanager webhook v4 스키마가 안정적이어야 한다 (`alertname`, `status`, `fingerprint`, `labels`, `annotations.runbook_url`, `generatorURL`). | 스키마 변경 시 UC-001 단계 1의 schema validation reject 증가 (Extension 2a) |
| LLM Provider가 표준 HTTP status code (200 / 401 / 403 / 429 / 5xx)를 일관되게 반환해야 한다. | 비표준 응답 시 UC-003의 fail-open 분류가 흔들림 (research §13.4) |
| 운영자 1인 이상이 on-call rotation에 등재돼 있어야 한다. | 미등재 시 UC-001 단계 6 (draft 검수) 진행 불가 → SOP raw fallback으로 degrade |
| 디스크 쓰기 가능 + sufficient space (audit/DLQ 각 50 MiB rotation 기준). | 쓰기 불가 시 F0.5 fail-open (audit) / UC-002 4a Extension (DLQ) |

## §3. System Features (모듈별 상세)

| ID | 모듈 | 상태 | 파일 |
|---|---|---|---|
| F0 | Foundation / Pilot scaffolding | implemented | [F0-foundation.md](modules/F0-foundation.md) |
| F1 | SOP Grounding & Store | implemented | [F1-sop-grounding.md](modules/F1-sop-grounding.md) |
| F2 | AI Runbook Drafting (with history) | implemented | [F2-ai-drafting.md](modules/F2-ai-drafting.md) |
| F3 | AI Quota Controls (fail-open) | implemented | [F3-ai-quota.md](modules/F3-ai-quota.md) |
| F4 | Multi-tenant Scope | implemented | [F4-multi-tenant.md](modules/F4-multi-tenant.md) |
| F5 | Audit | implemented | [F5-audit.md](modules/F5-audit.md) |
| F6 | Notification Dispatch (5 채널) | implemented | [F6-notification-dispatch.md](modules/F6-notification-dispatch.md) |
| F7 | PII Redaction | implemented | [F7-pii-redaction.md](modules/F7-pii-redaction.md) |
| F8 | DLQ + Replay | implemented (HMAC pending) | [F8-dlq-replay.md](modules/F8-dlq-replay.md) |

각 모듈 파일은 동일 템플릿: §F*.1 개요 / §F*.2 인터페이스 / §F*.3 데이터 모델 / §F*.4 상태 전이 / §F*.5 예외·복구 / §F*.6 NFR / §F*.7 Acceptance (Gherkin) / §F*.8 Traceability.

## §4. External Interface Requirements

### §4.1 User Interfaces

| UI | 역할 | 현 구현 | Open Item |
|---|---|---|---|
| **운영자 draft 검수 화면** | UC-001 단계 6~7. `approval_status: pending → approved/rejected` 전이. | SigNoz UI in-app (현재 구현). `frontend/src`, `frontend/public` 변경 영역 일부. | **TODO (partial: frontend 변경 영역 파일별 식별 미완료 — R-5, traceability.md §6 open item)** |
| **DLQ 조회·replay UI/CLI** | UC-002 단계 6~7. DLQ entry 조회, manual replay 발행, bulk replay. | CLI/UI 인터페이스는 F8 모듈 범위 밖. ledger와 sink만 제공 (F8.9 open). | follow-up |
| **AI strategy / tenant policy 관리 화면** | F2, F4의 admin 인터페이스. | 현재 in-app 화면 일부. 정확한 매핑 미확정. | follow-up |
| **Slack interactive approve button** | UC-001 단계 6 alternate (Slack에서 직접 approve). | **미구현**. Teams는 `Action.Submit` 제약으로 영구 미지원 (research §7.2). | follow-up |

### §4.2 Hardware Interfaces
N/A — DS-APM은 컨테이너 software-only.

### §4.3 Software Interfaces

| 인터페이스 | 방향 | 프로토콜 / API | 비고 |
|---|---|---|---|
| **SigNoz Ruler API** | DS-APM ↔ Ruler | Go internal — `pkg/apiserver/signozapiserver/ruler.go` (+182), `pkg/ruler/signozruler/handler.go` (+467) | F1, F2 wiring |
| **SigNoz Alertmanager (`dispatch.Dispatcher`)** | DS-APM → Dispatcher hot path | Go internal — `pkg/alertmanager/alertmanagerserver/dispatcher.go` wrapping (+56) | F6, F8 |
| **OTel Collector** | inbound | OpenTelemetry resource semantic attribute carry-through | research §4.1 — `service.name`, `deployment.environment`, `k8s.*` 등 자동 전파 |
| **LLM Provider** | outbound | HTTP / JSON. POST `/chat` (or equivalent). status code: 200 / 401 / 403 / 429 / 5xx | UC-003 fail-open 분기의 트리거 |
| **ClickHouse** | DS-APM → CK | SigNoz upstream observability path 그대로 (metrics, logs) | schema 변경 없음 |
| **PostgreSQL (bun ORM)** | DS-APM ↔ DB | bun ORM, table `ds_sop_documents` (마이그레이션 078). 복합 키 `(org_id, sop_id, version)` | F1 SOP store |
| **Redis** | (planned) | TTL-native cache — idempotency ledger 후속 | 현재 미사용, F8 R-7 follow-up |

### §4.4 Communications Interfaces

#### Inbound

| ID | 인터페이스 | 스키마 |
|---|---|---|
| **COMM-IN-1** | Alertmanager webhook v4 | `POST /webhook` body: `{receiver, status, alerts[], groupLabels, commonLabels, externalURL, version=4, groupKey}`. 각 alert는 `status`, `labels`, `annotations`, `startsAt`, `endsAt`, `generatorURL`, `fingerprint`. (research §5.2 참조) |

#### Outbound (5 채널)

| ID | 채널 | 프로토콜 / 스키마 | 제약 |
|---|---|---|---|
| **COMM-OUT-1** | Slack | Incoming Webhook + Block Kit (`blocks: [header, section, divider, actions]`) | username/icon override 불가 |
| **COMM-OUT-2** | MS Teams v2 | Adaptive Card v1.4 (`type: AdaptiveCard, body: [TextBlock, FactSet]`) | `Action.OpenUrl`만 지원. `Action.Submit` 작동 안 함 (research §7.2). v0.1은 양방향 버튼 미지원. |
| **COMM-OUT-3** | PagerDuty | Events API v2 (`routing_key`, `event_action: trigger`, `dedup_key`, `payload`) | `dedup_key`는 DS-APM의 idempotency_key와 1:1 매핑 |
| **COMM-OUT-4** | Generic Webhook | JSON POST. 자유 schema. | `severity`, `service`, `sop_url`, `ai_*` 필드 포함 |
| **COMM-OUT-5** | Email | SMTP MIME | Subject prefix에 `[SEV-x]`. SMTP queue가 자체 retry 보유 |

채널별 retry 정책은 UC-002 Sub-Variations 참조: Slack max 3회/1s base, Teams v2 max 3회/2s base, PagerDuty max 5회 (dedup 활용), Webhook max 3회/1s, Email max 2회/5s.

## §5. Other Nonfunctional Requirements

### §5.1 Performance

- **NF-5.1.1** 시스템은 Alertmanager webhook 수신 → 채널 2xx 응답까지 p95 latency 30초를 초과하지 않아야 한다 (운영자 approve 시간 제외). (arc42 QS-PERF-1)
- **NF-5.1.2** 시스템은 dispatcher hot path의 AI hook 호출을 동기적으로 처리하되 p95 latency 1초 (`DefaultGenerateTimeout`)를 초과하지 않아야 한다. (F6 NF-F6.1)
- **NF-5.1.3** 시스템은 LLM 응답 수신 → fail-open 결정까지 p95 latency 1초를 초과하지 않아야 한다. (UC-003 NFR)
- **NF-5.1.4** 시스템은 fail-open 발동 → SOP raw fallback의 Dispatcher 전달까지 p95 latency 3초를 초과하지 않아야 한다. (UC-003 NFR)
- **NF-5.1.5** 시스템은 dispatcher maintenance ticker로 30초 주기 empty aggregation group GC를 수행해야 한다. (F6 NF-F6.2)

### §5.2 Safety

- **NF-5.2.1 (Fail-open over silent drop)** LLM Provider 실패는 절대 운영자에게 silent drop으로 나타나지 않아야 한다. fallback dispatch 또는 meta-alert 중 하나는 반드시 발화해야 한다. (UC-003 Minimal Guarantee)
- **NF-5.2.2 (PII before AI)** AI Engine에 도달하기 전 페이로드의 PII (email, phone, 16자 이상 secret)는 100% redact돼야 한다. PII가 redact되지 않은 페이로드는 절대 AI Engine·외부 채널로 송신되지 않아야 한다. (UC-001 Minimal Guarantee, F7)
- **NF-5.2.3 (Audit boot fail-open)** Audit sink 등록 실패는 서버 부팅을 막지 않아야 한다. (F0 NF-F0.2)
- **NF-5.2.4 (Dispatcher robustness)** 채널 dispatch 실패는 dispatcher 자체를 중단시키지 않아야 한다. error는 항상 log + (선택적으로) DLQ로 흡수돼야 한다. (F6 NF-F6.5)
- **NF-5.2.5 (DLQ best-effort)** DLQ persistence는 best-effort로, dispatcher hot path를 절대 막지 않아야 한다. sink write 실패는 `WarnContext` 로깅 후 dispatcher가 계속 진행해야 한다. (F8 NF-F8.1)

### §5.3 Security

- **NF-5.3.1 (HMAC — OPEN)** Replay payload는 HMAC으로 서명되어야 한다. **정책 미정 — open follow-up** (F8 `open_items`, traceability.md §6, R-2). 본 항목이 미해결인 한 production-ready 선언 불가.
- **NF-5.3.2 (TLS)** 모든 outbound 채널 호출 (5채널 + LLM Provider)은 TLS 1.2 이상으로 수행돼야 한다.
- **NF-5.3.3 (Secret management)** SOP source 자격증명은 contract response에 노출되지 않아야 한다. `SecretRefVisible`, `CredentialDetailsVisible`, `BrowserCredentialsUsed`는 항상 `false`여야 한다 (F0 NF-F0.3). pilot contract validator가 `token=`, `client_secret`, `api_key`, JWT-like 패턴을 차단해야 한다 (F0.5).
- **NF-5.3.4 (Cross-tenant opaqueness)** Cross-tenant SOP/strategy lookup은 `ErrSOPDocumentNotFound`로 통일되어 존재 누설 없이 반환돼야 한다 (F1 NF-F1.1, F4).
- **NF-5.3.5 (Display URL safety)** `DisplayURL`은 `http`/`https`만 허용되며 sensitive query parameter는 자동 제거돼야 한다 (`safeDisplayURL`, F1 NF-F1.4).
- **NF-5.3.6 (Contract version frozen)** Contract version 문자열은 절대 자동 변경되지 않아야 한다 — downstream desync 방지 (F0 NF-F0.1).

### §5.4 Software Quality Attributes

arc42 §10 Quality Tree와 cross-reference. 본 절은 SRS-side ID로 다시 묶은 것.

- **NF-5.4.1 (Reliability — dispatch 손실 0)** 모든 재시도가 실패하더라도 원본 페이로드와 시도 이력은 손실 없이 DLQ에 보존돼야 한다 (UC-002 Minimal Guarantee, arc42 QS-REL-1).
- **NF-5.4.2 (Reliability — idempotency)** 동일 `(alert.fingerprint, channel.id, dispatch.round_no)` 튜플 기준 중복 dispatch는 0건이어야 한다. (UC-001 NFR, arc42 QS-REL-2). 단, 현 구현은 `EventID = alert.fingerprint`만 사용 — `(fingerprint, channel)` 튜플 확장은 F8 follow-up.
- **NF-5.4.3 (Maintainability — audit)** Dispatch / draft / SOP 접근 / fail-open 발동 1건당 audit row 1건 이상이 영속 기록돼야 하며, 누락률은 0%여야 한다. (arc42 QS-MNT-1)
- **NF-5.4.4 (Maintainability — `pkg/` 1:1 매핑)** WBS Level 2의 6 컴포넌트는 `pkg/` 디렉토리에 1:1로 매핑돼야 한다. (WBS index §WBS Tree)
- **NF-5.4.5 (Availability — single region MVP)** Ingress 99.9% 가용성 목표 (single-region MVP). multi-region failover는 본 SRS 범위 밖.

### §5.5 Business Rules

- **NF-5.5.1 (정보 손실 0)** 운영자가 받는 알람에서 alert payload + SOP 본문 정보 손실은 0건이어야 한다. AI draft 부정확함보다 silent drop이 더 나쁘다 (UC-003 NFR).
- **NF-5.5.2 (Blameless culture)** Audit log는 reproducibility 토대로만 사용돼야 하며, 개인 책임 추궁 목적으로 사용되지 않아야 한다 (SRE Postmortem Culture, research-skills-c-domain.md §1.4).
- **NF-5.5.3 (Handoff acknowledgment)** 운영자 교대 시 명시적 인수와 firm acknowledgment 수신 전까지 콜에서 나가지 않아야 한다 (PagerDuty Incident Command 모델, research §1.2). 본 룰은 운영 프로세스 권고로, DS-APM 시스템은 audit trail로만 지원.
- **NF-5.5.4 (90일 SOP staleness)** SOP의 `staleness_days`가 90일을 초과하면 grounding 결과를 보류하고 운영자에게 raw alert만 전달해야 한다 (UC-001 5a Extension, F1).
- **NF-5.5.5 (SEV-2 이상 = major incident)** SEV-2 이상은 모두 major incident로 다루며 PagerDuty paging + Slack/Teams broadcast 동시 발송해야 한다 (UC-001 Sub-Variations, research §2.2).

## Traceability
→ [`../_shared/traceability.md`](../_shared/traceability.md)

| 차원 | 매핑 |
|---|---|
| F0~F8 ↔ UC-001~003 | traceability.md §1 (Feature × UC) |
| F0~F8 ↔ WBS-1.0~1.5 | traceability.md §2 (Feature × WBS Component) |
| UC-001~003 ↔ WBS-1.0~1.5 | traceability.md §3 (Use Case × WBS) |
| 11 commits ↔ F0~F8 ↔ WBS-1.0~1.5 | traceability.md §4 (역사 추적) |
| Open items (HMAC, frontend, archive 자료) | traceability.md §6 |
