# 00. Baseline — DS-APM 산출물 작성을 위한 출발점

> 본 문서는 산출물 4종(Overview / Use Case / 기능명세서 / WBS) 작성의 **사실 기반(baseline)** 이다.
> 산출물은 모두 HTML로 만들지만, 본 문서는 작업 입력용 메모이므로 Markdown으로 유지한다.

작성일: 2026-05-28
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
| SigNoz upstream 분기점 (parent) | `feea9e9b3 refactor: remove light mode styles ... (#11080)` |
| DS-APM 시작 커밋 | `026863650 feat(ds-apm): add native mvp foundation pilot scaffolding` |
| DS-APM 마지막 커밋 (HEAD) | `91b9ff5db feat(ds-apm): wire dead-letter sink into alertmanager dispatcher` |
| 브랜치명 | `ds-apm/native-mvp-foundation` |
| nested repo 경로 | `workspace_archive/ds-apm/var/signoz` |
| 원격 (세 곳 HEAD 동일) | `origin=SigNoz/signoz`, `sudong=suUdong/signoz`, `product=suUdong/signoz-product` |

> 정식 diff 명령:
> `git diff 026863650^..ds-apm/native-mvp-foundation`

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
| `pkg/ruler/signozruler/` | `handler.go` +467, `sop_document_file_store.go` 신규, `handler_test.go` +660 |
| `pkg/types/alertmanagertypes/` | `incident.go`, `incident_payload.go` (PII redaction 포함) |
| `pkg/types/ruletypes/` (대거 신규) | `ai_strategy*`, `ai_strategy_history*`, `sop_document*`, `sop_preview*`, `pilot_contract*`, `pilot_managed_markdown*`, `pilot_audit_sink*`, `tenant_policy`, `notification_template_preview*` |
| `pkg/query-service/rules/` | `prom_rule.go`, `threshold_rule.go` 소폭 |
| testdata | `ds_ai_sop_demo_seed.json` |
| `cmd/community`, `cmd/enterprise` | 진입점 통합 |
| `frontend/src`, `frontend/public` | UI 변경 — **범위 별도 확인 필요 (Open Item #1)** |

## 4. 커밋 ↔ 기능 모듈 매핑

| # | 커밋 | 기능 | 모듈 키 |
|---|---|---|---|
| 1 | `026863650` | Native MVP pilot scaffolding | F0 Foundation |
| 2 | `72944ecac` | Ground alerts in uploaded SOPs | F1 SOP Grounding |
| 3 | `8a55208ef` | Make SOP access auditable | F5 Audit |
| 4 | `3fa604e03` | Scope SOP strategy access by tenant | F4 Multi-tenant |
| 5 | `a6757136e` | Fail open AI quota controls | F3 AI Quota |
| 6 | `cb29d2a59` | Persist latest AI strategy history | F2 AI Drafting (history) |
| 7 | `5c036c806` | Propagate SOP AI context to channels | F6 Notification Dispatch |
| 8 | `c7f4fd330` | Persist SOP documents to file | F1 SOP Store |
| 9 | `3e9dfa557` | Redact PII (email, phone, long secrets) | F7 PII Redaction |
| 10 | `ade174bb8` | JSONL DLQ + idempotent replay ledger | F8 DLQ + Replay |
| 11 | `91b9ff5db` | Wire DLQ into alertmanager dispatcher | F8 DLQ + Replay |

## 5. 산출물 4종 범위 락인

### 5.1 Overview (`01-overview.html`)
- 정체, 위치, 분기점, 표면 수치 (위 §1~§3)
- 핵심 가치: "관측 알람 → 운영 SOP → 운영자 핸드오프" 자동화
- 스코프 아웃: SigNoz upstream 자체 기능, Enterprise 모듈

### 5.2 Use Case (`02-usecase.html`)
- **메인 로직 흐름** (7 도메인 직렬):
  1. SOP 업로드/그라운딩 → 2. AI runbook 초안 → 3. Quota/Tenant 보호 → 4. Strategy history 기록 → 5. 채널 dispatch → 6. PII redaction → 7. DLQ/Replay
- **상세 케이스 2건** (이미 `docs/sop/`에 시나리오 10건 존재. 그 중 선정):
  - **Case A (golden path)** — `case-01 payment 5xx approved`: 결제 5xx 알람 → SOP 그라운딩 → AI draft approved → 채널 dispatch
  - **Case B (failure path)** — 후보 둘 중 택1:
    - `case-09 runbook validation failure` → DLQ → replay
    - `case-06 LLM auth failure` → quota fail-open → 운영자 알림
- HTML 시각화 강점 활용: 상태 전이 다이어그램, 채널 페이로드 before/after, PII redact diff

### 5.3 기능명세서 (`03-spec.html`)
8개 모듈 × 동일 템플릿 (인터페이스 / 데이터 모델 / 상태 전이 / 예외·복구 / 비기능 요건):

| ID | 모듈 | 근거 커밋 |
|---|---|---|
| F0 | Foundation / Pilot scaffolding | `026863650` |
| F1 | SOP Grounding & Store | `72944ecac`, `c7f4fd330` |
| F2 | AI Runbook Drafting (with history) | `cb29d2a59` |
| F3 | AI Quota Controls (fail-open) | `a6757136e` |
| F4 | Multi-tenant Scope | `3fa604e03` |
| F5 | Audit | `8a55208ef` |
| F6 | Notification Dispatch (5채널) | `5c036c806` |
| F7 | PII Redaction | `3e9dfa557` |
| F8 | DLQ + Replay | `ade174bb8`, `91b9ff5db` |

### 5.4 WBS (`04-wbs.html`)
커밋 시간선 그대로 Phase화:

| Phase | 내용 | 커밋 |
|---|---|---|
| P0 | Foundation | `026863650` |
| P1 | SOP Layer | `72944ecac` → `8a55208ef` → `3fa604e03` → `c7f4fd330` |
| P2 | AI Layer | `a6757136e` → `cb29d2a59` |
| P3 | Notification | `5c036c806` |
| P4 | Safety (PII) | `3e9dfa557` |
| P5 | Reliability (DLQ/Replay) | `ade174bb8` → `91b9ff5db` |

각 Phase 항목: 작업 / 입력 / 산출 / 검증 / 의존 / 소요.

## 6. 기존 산출물 (선검토 대상)

이미 `docs/` 하위에 초안이 존재 — 본 산출물 작성 전 신뢰도/재사용 가능성 판정 필요:

| 경로 | 내용 |
|---|---|
| `docs/usecase/claude-ds-apm-{operator-,}usecase.{md,html}` | Claude가 작성한 유즈케이스 초안 |
| `docs/usecase/codex-sop-runbook-index.md` | Codex가 작성한 SOP 인덱스 |
| `docs/merge_usecase/{claude,codex}-ds-apm-usecase-final.{md,html}` | 두 모델 산출 머지본 ("final"이라고 명시) |
| `docs/sop/codex-sop-runbook-case-01 ~ case-10*` | 시나리오 10건 + JSON 페이로드 — Use Case의 상세 케이스 입력으로 활용 가능 |

## 7. Open Items (산출물 작성 전 정리 필요)

1. **Frontend 변경 범위** — `frontend/src`, `frontend/public` 변경 파일 목록·기능 식별 (운영자 화면 흐름이 Use Case의 핵심)
2. **기존 산출물 재사용 판단** — `docs/usecase`·`docs/merge_usecase`·`docs/sop` 콘텐츠를 입력으로 쓸지, 무시할지
3. **HMAC 정책 follow-up** (미해결) — DLQ replay 시 시그니처 정책이 미정. 기능명세 F8에 "미해결" 표기 필요
4. **SigNoz/OTel/ITIL 외부 리서치** — Overview의 비교/근거, 기능명세의 업계 표준 매핑용 (context7, WebFetch)

---

## 참조

- 공개 스냅샷: <https://github.com/figure-team/ds-apm>
- 개인 fork (signoz): <https://github.com/suUdong/signoz>
- 회사용 fork (signoz-product): <https://github.com/suUdong/signoz-product>
- SigNoz upstream: <https://github.com/SigNoz/signoz>
- HTML 효과성 참조: <https://thariqs.github.io/html-effectiveness/>
