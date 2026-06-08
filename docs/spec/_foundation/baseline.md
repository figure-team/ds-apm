# 00. Baseline — DS-APM as-built 사실 기반

> 본 문서는 산출물(현 구조: PRD→Architecture→Epic→Story→WBS)의 **as-built 사실 기반(baseline)** 이다. §1~4는 불변 사실 스냅샷, §5~6은 2026-06-08 현행화.
> 작업 입력용 메모이므로 Markdown으로 유지한다.

작성일: 2026-05-28 (사실 스냅샷) · 현행화: 2026-06-08
저장소: `figure-team/ds-apm` (공개 스냅샷, single squash commit)
실제 작업 히스토리: `workspace_archive/ds-apm/var/signoz` (nested repo)

---

## 1. 프로젝트 정체

- **DS-APM** = SigNoz **community** 빌드 위에 얹은 *Incident → SOP runbook → Operator handoff* 확장 레이어
- SigNoz 자체는 OpenTelemetry-네이티브 관측 플랫폼 (MIT)
- **Enterprise 모듈(`ee/`, `cmd/enterprise/`) 은 범위 밖** — SigNoz의 별도 Enterprise License를 따르므로 본 산출물에서 다루지 않는다
- 상태: 초기 MVP. 멀티테넌트 격리, PII 처리는 **production-ready 아님**(README 명시)

## 2. 베이스라인 위치

| 항목 | 값 |
|---|---|
| SigNoz upstream 분기점 (parent) | `refactor: remove light mode styles ... (#11080)` |
| DS-APM 시작 커밋 | native mvp foundation pilot scaffolding |
| DS-APM 마지막 커밋 (HEAD) | wire dead-letter sink into alertmanager dispatcher |
| 브랜치명 | `ds-apm/native-mvp-foundation` |
| nested repo 경로 | `workspace_archive/ds-apm/var/signoz` |
| 원격 (세 곳 HEAD 동일) | `origin=SigNoz/signoz`, `sudong=suUdong/signoz`, `product=suUdong/signoz-product` |

> 구현 이력(11 커밋)은 nested repo `var/signoz` 기준 — 공개 `ds-apm`은 single squash라 개별 커밋·SHA가 없다.

## 3. 변경 표면 요약

- **DS-APM 커밋 11건**
- **변경 파일 100개, +12,632 / -110 LOC** (거의 전부 추가)
- 변경 영역 (top-level)

| 영역 | 핵심 변경 |
|---|---|
| `pkg/alertmanager/alertmanagernotify/` | Slack / MS Teams v2 / PagerDuty / Webhook / Email **5채널 모두 패치** |
| `pkg/alertmanager/alertmanagerserver/` | `dispatcher.go` (+56, DLQ 와이어), `dispatcher_dlq_test.go` (신규) |
| `pkg/alertmanager/alertmanagertemplate/` | 템플릿 확장 |
| `pkg/apiserver/signozapiserver/ruler.go` | API 라우트 +182 |
| `pkg/ruler/signozruler/` | `handler.go` +467, `sop_document_file_store.go` (당시 신규, **현재 삭제됨** — §5), `handler_test.go` +660 |
| `pkg/types/alertmanagertypes/` | `incident.go`, `incident_payload.go` (PII redaction 포함) |
| `pkg/types/ruletypes/` (대거 신규) | `ai_strategy*`, `ai_strategy_history*`, `sop_document*`, `sop_preview*`, `pilot_contract*`, `pilot_managed_markdown*`, `pilot_audit_sink*`, `tenant_policy`, `notification_template_preview*` |
| `pkg/query-service/rules/` | `prom_rule.go`, `threshold_rule.go` 소폭 |
| testdata | `ds_ai_sop_demo_seed.json` |
| `cmd/community`, `cmd/enterprise` | 진입점 통합 |
| `frontend/src`, `frontend/public` | UI 변경 — **범위 별도 확인 필요** (§6 open) |

## 4. 기능 모듈(F0~F8) ↔ CF ↔ WBS

> 공개 `ds-apm`은 squash라 개별 커밋이 없다. nested repo의 11 커밋이 구현한 모듈↔CF↔WBS 매핑(코드 경로는 [`../_shared/component-source-map.md`](../_shared/component-source-map.md)):

| 모듈 | 기능 | CF | WBS |
|---|---|---|---|
| F0 | Foundation / pilot scaffolding | CF-6 | WBS-1.0 |
| F1 | SOP Grounding & Store | CF-1 | WBS-1.1 |
| F2 | AI Runbook Drafting (history) | CF-2 | WBS-1.2 |
| F3 | AI Quota Controls (fail-open) | CF-2 | WBS-1.2 |
| F4 | Multi-tenant Scope | CF-1 | WBS-1.0 |
| F5 | Audit | CF-6 | WBS-1.0 |
| F6 | Notification Dispatch (5채널) | CF-3 | WBS-1.3 |
| F7 | PII Redaction | CF-4 | WBS-1.4 |
| F8 | DLQ + Replay | CF-5 | WBS-1.5 |

> 마이그레이션: 078(`ds_sop_documents`·`ds_ai_strategy_history`), 079(`ds_ai_config`), 080(ai oauth 컬럼).

## 5. 현 산출물 구조 (2026-06-08 현행화)

> §1~4는 as-built 사실(불변). 초기 "산출물 4종(Overview / Use Case / 기능명세 / WBS) HTML" 계획은 **폐기**됨. 현 구조는 BMAD 체인:

- **PRD** `01-prd/`(CF·FR) → **Architecture** `02-architecture/`(C4·ERD) → **Epic** `03-epics/` → **Story** `04-stories/` → **WBS** `05-wbs/`. 규칙: [`../PROCESS.md`](../PROCESS.md).
- 코드↔CF↔WBS 매핑·drift(예: §3의 `sop_document_file_store.go`는 당시 신규였으나 현재 삭제됨 — SOP 영속화는 DB store)는 [`../_shared/component-source-map.md`](../_shared/component-source-map.md).
- §4의 F0~F8은 **코드 모듈 단위**(매핑용). 산출물 분해 축은 **CF(사용자 가치)** — F→CF 매핑은 [`../_shared/traceability.md`](../_shared/traceability.md) §5.

## 6. Open Items (현 추적 위치)

- **HMAC 정책**(DLQ replay 서명 미정) — Story 5.3 / PRD §9.3.
- **Frontend 운영자 검수 화면** 변경 영역 미식별 — PRD §9.3.
- (해결) 기존 `docs/` 초안 재사용·외부 표준(SigNoz/OTel/ITIL) 리서치 → 본 spec에 반영 완료.

---

## 참조

- 공개 스냅샷: <https://github.com/figure-team/ds-apm>
- 개인 fork (signoz): <https://github.com/suUdong/signoz>
- 회사용 fork (signoz-product): <https://github.com/suUdong/signoz-product>
- SigNoz upstream: <https://github.com/SigNoz/signoz>
