---
id: GLOSSARY
title: DS-APM 용어집
type: glossary
status: draft
updated: 2026-05-29
---

# 용어집 (Glossary)

> arc42 §12에 해당. 4종 산출물 어디서든 호버/anchor로 참조.
> 본문 한국어 + 영문 키워드(`alert`, `fingerprint`, `dedup_key` 등)는 그대로 유지. baseline.md, research-skills-c-domain.md에서 확정된 사항만 정의.

## A

- **alert** — 모니터링 시스템이 임계치 초과·이상 패턴 감지 시 발생시키는 이벤트 객체. DS-APM에서는 SigNoz/Alertmanager가 webhook으로 `POST`하는 `firing`/`resolved` 상태의 페이로드를 가리키며, 항상 `fingerprint`·`labels`·`annotations`를 동반한다.
- **Alertmanager** — Prometheus 생태계의 alert dedup/inhibit/routing 컴포넌트. DS-APM은 Alertmanager가 보내는 webhook v4 스키마(`status`, `alerts[]`, `groupLabels`, `commonLabels`, `groupKey`)를 ingress 표준으로 받는다. SigNoz가 내장한 Alertmanager fork(`pkg/alertmanager/`)가 dispatch 진입점.
- **AI strategy** — 특정 tenant·SOP 컨텍스트에서 어떤 AI 모델·prompt·temperature 조합으로 runbook draft를 생성할지 정의하는 전략 객체. DS-APM에서는 `pkg/types/ruletypes/ai_strategy*.go`로 모델링되며, tenant 단위로 접근이 격리된다 (커밋 `3fa604e03`).
- **AI strategy history** — AI strategy의 버전별 변경 이력. 최신 strategy만이 아니라 직전 strategy까지 persist해 draft가 어떤 정책으로 생성됐는지 reproducibility를 보장한다. `pkg/types/ruletypes/ai_strategy_history*.go`에 정의 (커밋 `cb29d2a59`).
- **annotations.runbook_url** — Prometheus 관례 annotation으로, 알람과 짝지어진 SOP/runbook URL을 가리킨다. DS-APM의 SOP grounding이 1차 키로 사용하며, 이 URL이 비어있거나 staleness가 90일을 넘으면 draft 생성을 보류하고 raw alert만 운영자에게 전달한다.
- **audit sink** — SOP 접근·draft 생성·dispatch 결과 등 감사 대상 이벤트를 영속화하는 저장 채널. `pkg/types/ruletypes/pilot_audit_sink*.go`로 모델링되며, SOP 접근을 auditable하게 만든 커밋 `8a55208ef`에서 도입됐다 (F5 Audit).

## C

- **channel adapter** — DS-APM의 canonical dispatch payload를 채널별 외부 포맷(Slack Block Kit, MS Teams Adaptive Card, PagerDuty Events v2, Email, generic Webhook)으로 변환하는 어댑터. `pkg/alertmanager/alertmanagernotify/` 하위 5채널 구현(F6 Notification Dispatch). 어댑터별 제약(예: MS Teams는 `Action.Submit` 불가)을 흡수한다.
- **correlation_id** — 같은 incident 맥락에 속하는 여러 alert·dispatch·draft를 묶는 DS-APM 자체 식별자. `fingerprint`가 단일 alert의 idempotency 시드라면, `correlation_id`는 incident 단위 timeline 추적에 사용된다.

## D

- **dedup_key** — PagerDuty Events API v2 최상위 필드로, 같은 키로 재전송된 이벤트를 기존 alert에 병합한다. DS-APM의 idempotency key와 1:1 매핑되어 channel adapter가 payload에 그대로 박는다.
- **dispatch round** — 동일 (alert, channel) 쌍에 대해 운영자가 manual replay를 트리거할 때마다 증가하는 round 번호. idempotency key 산출식의 세 번째 컴포넌트(`dispatch.round_no`)로, replay마다 새로운 key가 생성돼 같은 라운드의 중복은 차단하고 의도적 재전송은 허용한다.
- **DLQ (Dead Letter Queue)** — 채널 dispatch가 retry budget을 모두 소진하거나 4xx(429 제외)로 즉시 실패했을 때 원본 payload·시도 timestamp·응답 코드·에러 메시지를 보존하는 큐. DS-APM은 JSONL 파일 기반 DLQ를 사용하며 (커밋 `ade174bb8`), dispatcher 와이어링은 `pkg/alertmanager/alertmanagerserver/dispatcher.go` (커밋 `91b9ff5db`)에서 완성됐다 (F8).
- **draft** — AI Engine이 SOP grounding 결과를 바탕으로 생성한 runbook 초안 객체. 운영자 검수 전 상태로, `approval_status: pending`을 가진다. → **runbook draft** 참조.

## F

- **fail-open** — 보호 메커니즘(quota, validation, dependency)이 실패해도 시스템 전체가 정지하지 않고 degraded 모드로 계속 동작하는 정책. DS-APM에서는 LLM auth/quota 실패 시 (커밋 `a6757136e`, F3) AI draft를 포기하고 SOP 원문 + raw alert만 운영자에게 전달한다.
- **fingerprint** — Alertmanager가 alert별로 자동 생성하는 고정 길이 해시. 같은 alert 인스턴스를 식별하는 자연 키이며, DS-APM의 idempotency key의 첫 번째 시드로 사용된다.
- **fork base commit** — DS-APM이 SigNoz upstream에서 분기한 시점의 커밋(`feea9e9b3 refactor: remove light mode styles ... (#11080)`). 산출물의 변경 표면은 이 base 이후 11개 DS-APM 커밋만 대상으로 한다.

## I

- **Idempotency Key** — 같은 이벤트의 중복 처리를 막기 위한 deterministic 키. DS-APM에서는 `sha256(alert.fingerprint || channel.id || dispatch.round_no)`를 사용. 랜덤 UUID를 쓰지 않고 페이로드 자연 키를 결합해, 동일 alert가 동일 채널로 같은 라운드에 재유입되면 같은 key가 산출돼 dispatch가 차단된다.
- **Incident** — 사용자 가시 영향이 있거나, 두 팀 이상의 협업이 필요하거나, 1시간 집중 분석으로도 미해결인 사건. PagerDuty 정의 기준 SEV-2 이상은 모두 major incident. DS-APM은 alert 클러스터를 `correlation_id`로 묶어 incident 단위로 추적한다.

## M

- **MVP (Minimum Viable Product)** — DS-APM의 현 상태 분류. README 명시대로 멀티테넌트 격리·PII 처리는 production-ready 아니며, 초기 11개 커밋(`+12,632 LOC`) 범위만 구현됐다.
- **multi-tenant** — 한 DS-APM 인스턴스가 여러 tenant의 SOP·AI strategy·audit log를 격리하여 다루는 운영 모델. tenant policy(`pkg/types/ruletypes/tenant_policy`)로 SOP strategy 접근이 tenant 단위로 scoping된다 (커밋 `3fa604e03`, F4). 단 README는 격리가 production-ready 아님을 명시.

## O

- **Operator (운영자)** — DS-APM의 primary actor. AI가 생성한 runbook draft를 검수해 approve/reject하고, dispatch 결과를 모니터링하며, DLQ 항목의 manual replay를 트리거한다. PagerDuty Incident Command 모델의 Incident Commander(IC) + Operations Lead 역할을 겸한다.
- **OTel (OpenTelemetry)** — CNCF 관측 표준. trace/metric/log 시그널과 resource semantic attribute(`service.name`, `deployment.environment`, `k8s.*` 등)를 정의한다. SigNoz가 OTel-native이므로 DS-APM의 alert payload는 OTel resource attribute를 자연스럽게 캐리한다.

## P

- **PII (Personally Identifiable Information)** — 개인 식별 정보. DS-APM은 이메일·전화·long secret을 redaction 대상으로 분류한다 (커밋 `3e9dfa557`, F7). OTel 가이드 원칙대로 가능한 한 이른 단계(ingress)에서 거른다.

## R

- **redaction** — payload에서 민감 정보를 제거·마스킹·해시·truncation 처리하는 동작. DS-APM은 `pkg/types/alertmanagertypes/incident_payload.go`에서 PII redaction을 수행하며, AI Engine 진입 전에 100% 적용돼야 한다.
- **replay ledger** — DLQ 항목을 manual replay 할 때 같은 dispatch가 중복 실행되지 않도록 idempotent하게 기록하는 원장. 커밋 `ade174bb8`에서 JSONL DLQ와 함께 도입됐으며, replay마다 `dispatch.round_no`가 증가해 새로운 idempotency key를 산출한다.
- **runbook** — 특정 alert/symptom에 대해 운영자가 따라야 할 단계별 대응 절차 문서. GitLab 원칙 "as short as possible, complete enough to be executed without further research"를 따르며, DS-APM의 SOP grounding 대상이다.
- **runbook draft** — AI Engine이 retrieval된 SOP를 컨텍스트로 생성한 runbook 초안. `draft_id`, `runbook_ids[]`(grounding source), `model.*`, `prompt_template_*`, `citations[]`, `approval_status` 필드를 가진다. 운영자가 approve해야 dispatch 단계로 진행된다.

## S

- **severity (P1~P4)** — alert의 응답 시급성 등급. DS-APM은 PagerDuty 표준(SEV-1~SEV-5)과 Prometheus 관례(`critical`/`warning`/`info`)의 합집합을 사용하며, ITIL Priority Matrix(Impact × Urgency)로 P1~P4를 derive할 수 있다. SEV-2 이상은 모두 major incident로 다룬다.
- **SigNoz** — OpenTelemetry-native 오픈소스 관측 플랫폼 (MIT). DS-APM은 SigNoz **community 빌드** 위에 SOP grounding/AI draft/handoff 레이어를 얹는다. Enterprise 모듈(`ee/`, `cmd/enterprise/`)은 범위 밖.
- **SOP (Standard Operating Procedure)** — 표준 운영 절차. DS-APM에서는 운영자가 업로드한 운영 문서를 가리키며, vector store에 인덱싱되어 alert 발생 시 grounding 대상이 된다. 파일 영속화는 `pkg/types/ruletypes/sop_document*.go`와 `sop_document_file_store.go`에서 처리 (커밋 `72944ecac`, `c7f4fd330`, F1).
- **SRE (Site Reliability Engineering)** — Google이 정립한 운영·신뢰성 엔지니어링 분과. DS-APM의 alert/incident/runbook/postmortem 개념 모델은 Google SRE Book(Ch.6/12/14/15) 권장 사항을 base로 한다.

## T

- **tenant** — DS-APM이 격리해서 다루는 운영 단위(고객/팀/조직). 각 tenant는 자체 SOP·AI strategy·audit log를 가지며, tenant policy로 접근이 scoping된다. → **multi-tenant** 참조.
