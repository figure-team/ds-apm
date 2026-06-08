---
id: COMPONENT-SOURCE-MAP
title: 컴포넌트 구성·소스 매핑 (개발 현장 용어 기준)
type: reference
status: living
updated: 2026-06-05
source_of_truth: 실제 코드(pkg/) + _foundation/baseline.md
---

# DS-APM 컴포넌트 구성 및 소스 매핑

> 본 문서는 **SigNoz community 빌드 위에 자체 개발한 소스**를 기준으로, 6개 컴포넌트와
> 컴포넌트 상세, 그리고 기능명세(F0~F8)·WBS 매핑을 한 곳에 정리한다.
> 컴포넌트 상세 용어는 **한국 개발 현장에서 통용되는 표현**으로 통일했다(원 표현 대조는 §8 부록).
> 소스는 모두 현재 코드(`pkg/`)로 검증했으며, 문서·코드 불일치는 §9에 명시한다.

---

## 0. 소스 구분 (3 분류)

per-commit 이력은 `suUdong/signoz` fork(branch `ds-apm/native-mvp-foundation`, 커밋 11건,
+12,632 LOC)에 있고, 본 공개 스냅샷은 squash본이다. 파일 기준 구분은 아래 3가지.

| 표기 | 구분 | 의미 |
|---|---|---|
| 🟢 | **자체 개발(신규)** | SigNoz엔 없던 우리가 새로 만든 파일 |
| 🟡 | **확장(원본 패치)** | SigNoz 원본 파일에 우리 로직을 끼워 넣은 것 |
| ⚪ | **SigNoz 원본** | 변경 없이 그대로 사용 |

핵심 자체 개발 영역: `pkg/ruler/{sopstore, aigenerator, aihistorystore, aiconfigstore,
runbookdrafter}` 5개 신규 패키지 + `pkg/alertmanager/alertmanagernotify/dlq` +
`pkg/types/ruletypes/{pilot,sop,ai_strategy,tenant,notification}*` 28개 파일 + 마이그레이션 `078`.

---

## 1. 공통 기반 모듈 — `WBS-1.0`

**기능명세:** F0 Foundation + F4 멀티테넌시 + F5 감사

> 모든 하위 컴포넌트가 공유하는 기반 데이터 구조·정책·기록 통로.

| 상세 (개발 현장 용어) | 소스 |
|---|---|
| 테넌트별 설정(계약) 스키마 정의 | 🟢 `pkg/types/ruletypes/pilot_contract.go` — `PilotConfiguration` 외 5종 구조체 + 유효성 검증 |
| 표준 본문(Markdown) 포맷 정의 | 🟢 `pkg/types/ruletypes/pilot_managed_markdown.go` |
| 멀티테넌시 격리 정책 | 🟢 `pkg/types/ruletypes/tenant_policy.go` — 라벨 기반 테넌트 판별·권한 검사 |
| 감사 로그 기록 인터페이스(추상화) + 파일(JSONL) 구현 | 🟢 `pkg/types/ruletypes/pilot_audit_sink.go`, `pilot_audit_sink_jsonl.go` |
| DB 스키마 마이그레이션 | 🟢 `pkg/sqlmigration/078_add_ds_apm_stores.go` — `ds_sop_documents` 테이블 |
| 단일 실행 파일 진입점 연동 | 🟡 `cmd/community/` — 감사 기록기 등록 후 SigNoz 바이너리에 연결 |

---

## 2. SOP 연계(그라운딩) 서비스 — `WBS-1.1`

**기능명세:** F1 SOP Grounding & Store

> 알람을 운영 절차서(SOP)에 자동으로 연결.

| 상세 (개발 현장 용어) | 소스 |
|---|---|
| SOP 도메인 모델(타입) | 🟢 `pkg/types/ruletypes/sop_document.go`, `sop_preview.go`, `storable_sop_document.go` |
| SOP 저장소 인터페이스 | 🟢 `pkg/types/ruletypes/sop_store.go` — 등록(upsert)/조회/목록/삭제 |
| DB 저장소 구현체 | 🟢 `pkg/ruler/sopstore/sqlsopstore/sop.go` |
| 알람–SOP 연계(매칭) 로직 | 🟢 `sop_document.go: PreviewSOPDocumentBinding` — 명시적 `sop_id` 라벨 기반 매칭 |
| SOP 등록·조회 HTTP 엔드포인트 | 🟡 `pkg/ruler/signozruler/handler.go` — POST/GET/GET (※ SOP **삭제 엔드포인트는 미노출**) |
| ⚠️ 파일 저장 구현체 | ❌ `signozruler/sop_document_file_store.go` — **현재 코드에 없음**(삭제됨, §9 참조). 영속화는 DB 저장소가 담당 |

---

## 3. AI 초안 매니저 — `WBS-1.2`

**기능명세:** F2 AI Drafting + F3 사용량 제어(장애 시 우회)

> 운영 절차서를 참고해 AI가 조치 초안을 생성·저장하고, 사용량 초과 시 안전하게 우회.

| 상세 (개발 현장 용어) | 소스 |
|---|---|
| AI 전략 도메인 모델 | 🟢 `pkg/types/ruletypes/ai_strategy.go`, `ai_strategy_generator.go` |
| 생성형 AI(LLM) 연동 어댑터 | 🟢 `pkg/ruler/aigenerator/llmaigenerator/{claudeapi, claudecli, codexapi, codexcli}` — **claude(Anthropic)·codex(OpenAI) 2종**, API/CLI 두 경로 |
| 전략 생성 및 저장(영속화) | 🟢 `pkg/ruler/aigenerator/{aigenerator, storeaware}.go`, `aiconfigstore/sqlaiconfigstore` |
| AI 호출 이력 기록 | 🟢 `pkg/types/ruletypes/ai_strategy_history*.go`, `aihistorystore/sqlaihistorystore/history.go` — 장애별 **최신 1건 덮어쓰기(upsert)** |
| 사용량(쿼터) 제어 — 장애 시 우회 | 🟢 `ai_strategy.go`(쿼터·타임아웃·라이선스·제공사 4종 제어) + `aigenerator/dispatchhook/hook.go` |
| AI 설정 엔드포인트 | 🟢 `pkg/ruler/signozruler/ai_config_handler.go` + `aiconfigstore/secretbox`(자격증명 암호화) |
| (참고) 런북 작성기 | 🟢 `pkg/ruler/runbookdrafter/llmrunbookdrafter/` |

---

## 4. 알림 발송기(디스패처) — `WBS-1.3`

**기능명세:** F6 Notification Dispatch

> SigNoz 발송 경로를 감싸 AI·SOP 정보를 합쳐 5개 채널로 전송.

| 상세 (개발 현장 용어) | 소스 |
|---|---|
| 발송기 래핑(확장) | 🟡 `pkg/alertmanager/alertmanagerserver/dispatcher.go` — 발송 직전 `applyAIHook` 삽입 |
| AI·SOP 정보 주입(전파) | 🟢 `aigenerator/dispatchhook/hook.go`, `pkg/types/ruletypes/notification_template_preview.go` |
| 채널 연동 어댑터 | 🟡 `pkg/alertmanager/alertmanagernotify/{slack, msteamsv2, pagerduty, webhook, email}` — **AI·SOP 보강 5종** / ⚪ `opsgenie`(보강 없는 6번째 채널) |

---

## 5. 개인정보(PII) 마스킹 필터 — `WBS-1.4`

**기능명세:** F7 PII Redaction

> 외부 발송 전 민감 정보를 가림.

| 상세 (개발 현장 용어) | 소스 |
|---|---|
| 마스킹 규칙 엔진 | 🟡 `pkg/types/alertmanagertypes/incident_payload.go` — 이메일·국내 전화번호·긴 시크릿 + **JWT·Bearer 토큰·URL 민감 키**까지 |
| 장애 데이터 마스킹 적용 | 🟡 `incident_payload.go: SanitizeIncidentInfo` — 값만 마스킹, 알람 식별 라벨은 보존 |

---

## 6. 실패 큐(DLQ) 재처리 서비스 — `WBS-1.5`

**기능명세:** F8 DLQ + Replay

> 채널 발송 최종 실패 건을 안전 보관하고 재발송.

| 상세 (개발 현장 용어) | 소스 |
|---|---|
| 실패 큐(DLQ) 파일 저장기 | 🟢 `pkg/alertmanager/alertmanagernotify/dlq/dlq.go` |
| 멱등 재발송 관리대장(중복 방지) | 🟢 `pkg/alertmanager/alertmanagernotify/dlq/ledger.go` — `MarkIfNew` |
| 발송기 연동(실패 시 적재) | 🟡 `dispatcher.go: recordTerminalFailure` → DLQ (※ `server.go` 배선이 현재 `nil` — 기본값 미연결) |
| 재발송 API / 재처리 상태 관리 / HMAC 서명 검증 | ❌ **미구현(계획)** — F8·WBS-1.5 `status: planned`, 설계만 존재 |

---

## 7. 기능명세(F0~F8) ↔ 컴포넌트 매핑 요약

| 기능모듈 | 컴포넌트 |
|---|---|
| F0 Foundation | 1. 공통 기반 모듈 |
| F4 멀티테넌시 | 1. 공통 기반 모듈 |
| F5 감사 | 1. 공통 기반 모듈 |
| F1 SOP 연계·저장 | 2. SOP 연계 서비스 |
| F2 AI 초안 | 3. AI 초안 매니저 |
| F3 사용량 제어 | 3. AI 초안 매니저 |
| F6 알림 발송 | 4. 알림 발송기 |
| F7 PII 마스킹 | 5. PII 마스킹 필터 |
| F8 DLQ·재발송 | 6. 실패 큐 재처리 |

> 기능명세는 **9개 모듈(F0~F8)**, WBS·발표덱은 **6개 컴포넌트** 축이다. F0·F4·F5 → 컴포넌트 1,
> F2·F3 → 컴포넌트 3으로 합쳐진다.

---

## 8. 부록 — 용어 대조 (원 표현 → 개발 현장 표현)

| 원 표현 | 개발 현장 표현 | 비고 |
|---|---|---|
| Grounding | 알람–SOP 연계(매칭) | 학술/외래어라 현장에서 잘 안 씀 |
| Sink | 기록기 / 저장기 (출력 인터페이스) | "감사 Sink" → "감사 로그 기록기" |
| Idempotent | 멱등 (중복 방지) | 멱등은 통용 |
| Ledger | 관리대장 / 장부 | "Replay Ledger" → "재발송 관리대장" |
| fail-open | 장애 시 우회 (차단하지 않고 통과) | |
| Redaction | (개인정보) 마스킹 | 현장 표준 표현 |
| Replay | 재발송 / 재처리 | |
| Dispatcher wrapping | 발송기 래핑(확장) | |
| persistence | 영속화 (저장) | |
| propagation | 주입 / 전파 | |
| payload | 페이로드 (전송 데이터) | 통용 |
| Store | 저장소 | |
| Provider adapter | 연동 어댑터 | |
| route / handler | 엔드포인트 / 핸들러 | 통용 |
| DLQ (Dead Letter Queue) | 실패 큐 (미전송 큐) | 약어는 병기 유지 |

> 인터페이스·어댑터·핸들러·스키마·API·마이그레이션·멱등·영속화·페이로드 등 이미 현장에서
> 통용되는 외래어는 그대로 두고, **잘 안 쓰이는 표현만 교체**했다.

---

## 9. 알려진 불일치 (drift)

| 항목 | 내용 |
|---|---|
| `sop_document_file_store.go` | F1 `source_paths`·`baseline.md §3`에는 "신규"로 남아 있으나 **현재 코드에 없음**(삭제됨). SOP 영속화는 DB 저장소가 담당. |
| Replay API / 상태 머신 / HMAC | F8·WBS-1.5에 일부 현재형으로 기술되어 있으나 코드 미구현(`status: planned`). 본 문서는 "미구현(계획)"으로 표기. |
| DLQ 발송기 배선 | `dispatcher.go`에 적재 로직은 있으나 `server.go`의 sink 주입이 `nil`(기본 미연결). |

> 본 문서는 **코드를 진실의 원천**으로 작성했다. 위 불일치는 F1·F8·baseline 문서 측 정정이 필요하다.
