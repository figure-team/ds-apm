---
id: USECASE-INDEX
title: DS-APM Use Case 카탈로그
type: usecase-index
template: cockburn-fully-dressed + uml-sequence + gherkin
status: draft
updated: 2026-05-29
---

# DS-APM Use Case 카탈로그

> **템플릿**: Cockburn Fully Dressed (메인 골격) + Gherkin (복잡한 에러 케이스) + UML Sequence Diagram (`alt`/`break` 프래그먼트)
> **Level 통일**: 모든 UC는 Cockburn **User-goal (Sea) level**
> **상태**: 본문 작성 완료. UC-001~003 cases 파일 채워짐.

## Use Case 일람

| ID | 제목 | 유형 | Primary Actor | 상태 |
|---|---|---|---|---|
| [UC-001](cases/UC-001-incident-to-channel.md) | Incident에서 채널 전달까지 (Golden Path) | golden path | 운영자 | planned |
| [UC-002](cases/UC-002-channel-failure-dlq.md) | 채널 전달 실패 → DLQ → Replay | failure path | 운영자 / 시스템 | planned |
| [UC-003](cases/UC-003-llm-auth-fail-open.md) | LLM 인증 실패 → quota fail-open → SOP 원문 fallback | failure path | 시스템 | planned |

## 액터 매트릭스

행 = actor, 열 = Use Case. 셀 의미: **P** = Primary, **S** = Supporting, **—** = 관여 없음.

| Actor | 분류 | UC-001 Golden Path | UC-002 DLQ Failure | UC-003 LLM Fail-open |
|---|---|:---:|:---:|:---:|
| **운영자 (Operator)** | 사람 | P | P | S (degraded 알림 수신) |
| **SRE** | 사람 | — | S (DLQ depth meta-alert) | S (자격증명 회전, fail-open storm escalation) |
| **SigNoz / Alertmanager** | 시스템 | S (alert source) | — | — |
| **AIOpsAgent Ingress** | 시스템 | S | — | — |
| **PII 마스킹 필터 (F7)** | 시스템 | S | — | — |
| **Multi-tenant Scope (F4)** | 시스템 | S | — | — |
| **SOP Store (F1)** | 시스템 | S | — | S (raw fallback fetch) |
| **AI Engine (F2)** | 시스템 | S | — | S |
| **AI Quota Controller (F3)** | 시스템 | — | — | **P** (fail-open 결정) |
| **LLM Provider** | 외부 | S | — | S (실패 트리거) |
| **Dispatcher (F6)** | 시스템 | S | S | S |
| **Channel (Slack / Teams / PD / Webhook / Email)** | 외부 | S | S (실패 트리거) | S |
| **DLQ JSONL Sink (F8)** | 시스템 | — | S | — |
| **Replay Ledger (F8)** | 시스템 | — | S | — |
| **Audit Sink (F5)** | 시스템 | S | S | S |

UML use case diagram 1장 (액터 스틱맨 + UC 타원 + 매트릭스 위 cell `P`/`S` 매핑)은 HTML 빌드 단계에서 별도 SVG 렌더링.

## 다이어그램 표준
- **Swimlane Sequence**: SigNoz / Ingress / AI Engine / Operator / Dispatcher / Channel (5~6 lane)
- **State Machine**: Alert lifecycle (`received → grounded → draft_pending → approved → dispatching → delivered / failed_dlq / replayed`)
- **Activity Diagram**: 분기 많은 sad path (필요 시)

각 UC 파일에 Mermaid `sequenceDiagram` 1장 + `stateDiagram-v2` 1장 임베드. UC-003은 sequenceDiagram에 `alt/else` 프래그먼트로 401/403/429 vs 5xx vs malformed 분기를 한 장에 그림.

## 공통 가정 (Background Given)

모든 UC에 공통으로 적용되는 환경 조건. 각 UC의 Preconditions 위에 implicit하게 깔린다.

1. **SigNoz Ruler alert rule 활성** — 대상 alert rule이 평가 중이고, `for` hold 시간 경과 시 firing 상태로 전이해야 한다.
2. **Alertmanager webhook v4 호환** — 페이로드는 `alertname`, `severity`, `status`, `startsAt`, `labels`, `annotations.runbook_url`, `fingerprint`, `generatorURL` 8개 필수 필드를 갖는다 (research §13.1).
3. **대상 채널 1개 이상 등록·healthy** — Slack / MS Teams v2 / PagerDuty / Webhook / Email 중 1+ 채널이 적어도 직전 health check를 통과한 상태.
4. **SOP store 인덱싱 완료** — 대응 SOP 1건 이상이 `approval_status: approved`이고 `staleness_days ≤ 90`. SOP는 explicit-label binding (`signoz_pilot_sop_id`)으로 매칭된다 (F1.1).
5. **AI Strategy 활성 + Quota Controller 정상** — 해당 tenant에 대해 AI Strategy 1건이 active이고, Quota Controller는 fail-open 정책으로 설정 (UC-003 분기에서 활용).
6. **PII 마스킹 필터 활성** — `pkg/types/alertmanagertypes/incident_payload.go`의 redactor가 AI 초안 매니저 호출 직전에 항상 호출 가능 상태.
7. **Tenant Policy 적용** — `project_id` × `environment` 기반 tenant scope이 적용되어 cross-tenant SOP/strategy leakage가 차단된 상태 (F4, NF-F1.1).
8. **Audit JSONL sink 등록** — `cmd/community/main.go`가 부팅 시 `var/audit/pilot-events.jsonl`을 등록한 상태. 실패 시 `NopPilotAuditEventSink` fallback (F0 NF-F0.2).
9. **DLQ JSONL sink + Replay Ledger** — 디스크 쓰기 가능 상태. UC-002에서 본격 활용.
10. **운영자 1인 이상 on-call** — UC-001 단계 7의 draft 승인을 담당할 운영자가 rotation에 등재.

상기 가정이 깨지면 각 UC의 Extension으로 분기 (e.g., UC-001 4a tenant policy 미발견, UC-002 4a DLQ sink 쓰기 실패).

## Traceability
- Use Cases → Features → WBS 매트릭스: [`../_shared/traceability.md`](../_shared/traceability.md)
- UC-001 ←→ F0, F1, F2, F4, F5, F6, F7 / WBS-1.0~1.4
- UC-002 ←→ F6, F8 / WBS-1.3, 1.5 (UC-001 단계 8 Extension `8a`로부터 진입)
- UC-003 ←→ F2, F3 / WBS-1.2 (UC-001 단계 5 Extension `5b`로부터 진입). F1은 raw fallback fetch의 전제로 참여.
