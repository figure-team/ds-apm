---
id: TRACEABILITY
title: CF × User Journey × WBS Traceability Matrix
type: traceability
status: living
axis: CF (Capability Feature)
updated: 2026-06-14
---

# Traceability Matrix (CF 축)

> BMAD 정렬: 산출물 체인 **PRD → 에픽 → 스토리 → WBS (+ Architecture)**. **User Journey(UJ)는 PRD(`01-prd/index.md` §5)에 내장** — 별도 use-case 문서 없음.
> 각 셀 ID는 frontmatter(`implements_features`/`covers_features`/`implements_uj`)와 일치해야 한다 — desync 검출의 진실의 원천.
> top-down 출처(JTBD/UJ): [`../_foundation/source-strategy-brief.md`](../_foundation/source-strategy-brief.md). 코드 매핑(F0~F10): [`component-source-map.md`](component-source-map.md).
>
> **2026-06-12 번호 정합 (통합)**: 우리 구현 번호로 정렬했다. **CF-7 = 메트릭 이상 탐지**(`anomaly_rule.go`, FR-CF7.1, implemented), **CF-11 = AI 코드베이스 RCA**(`pkg/ruler/coderca`, implemented, 마이그레이션 082·083). jinhyeok 브랜치가 코드 RCA를 "CF-7 미등록 예외 대응"으로 표기했던 충돌을 해소 — 해당 스캐너 원안은 [CF-11 §부록(이연 대안)](../01-prd/features/CF-11-code-rca.md)에 보존.

## 1. CF × User Journey

| CF | Title | UJ-1 골든 | UJ-2 DLQ | UJ-3 fail-open | UJ-4 로드맵 | UJ-5 코드RCA |
|---|---|:---:|:---:|:---:|:---:|:---:|
| CF-1 | SOP 자동 연계 + 테넌트 격리 | ✓ | | (전제)¹ | | (전제)² |
| CF-2 | AI 대응 가이드 + 안전 | ✓ | | ✓ | | (전제)² |
| CF-3 | 멀티채널 핸드오프 | ✓ | ✓ | | | (전제)² |
| CF-4 | 민감정보 비노출 (PII) | ✓ | | | | (전제)² |
| CF-5 | 무유실·멱등 재처리 (DLQ) | | ✓ | | | (전제)² |
| CF-6 | 정책·감사 기반 | ✓ | ✓ | ✓ | | (전제)² |
| CF-7 | 메트릭 이상 탐지 *(implemented)* | (트리거)³ | | | | (트리거)⁴ |
| CF-11 | AI 코드베이스 RCA *(implemented)* | | | | | ✓ |
| CF-8~10 | 로드맵(자동조치·자산화·ITSM) + CF-7 학습형 후속 | | | | ✓ | |

> ¹ **(전제)** = 해당 UJ의 *전제*일 뿐 implements 링크 아님(frontmatter `implements_uj` 미포함). **✓ 만 implements**. CF-1은 UJ-3에서 SOP fallback 원문의 전제 역할이나, UJ-3이 구현하는 기능은 CF-2·6.
> ² UJ-5(코드 RCA)를 구현하는 기능은 CF-11. CF-1~6은 UJ-5가 **재사용하는 기존 경로**(unbound 판정·핸드오프·PII·DLQ·감사) — implements 아님.
> ³ **(트리거)** = CF-7 이상 탐지는 **알람을 생성하는 탐지 기능**이다. 해당 여정의 *발화 소스*일 뿐 가치 전달(implements) 아님 → frontmatter `implements_uj` 미포함.
> ⁴ CF-11 트리거 술어의 `anomaly` 조건 = CF-7. CF-7 신호가 없으면 CF-11은 **fail-closed(미발화)**. CF-11은 CF-7 부재에도 CF-1(unbound)만으로 의미 있게 동작하도록 `AnomalySignal` 인터페이스로 분리(설계 §3·§10).

## 2. CF × WBS Component

| CF | WBS-1.0 공통기반 | WBS-1.1 SOP | WBS-1.2 AI초안 | WBS-1.3 디스패처 | WBS-1.4 PII | WBS-1.5 DLQ | WBS-1.6 이상탐지 | WBS-1.7 코드RCA |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| CF-1 | ✓ (테넌트) | ✓ | | | | | | |
| CF-2 | | | ✓ | | | | | |
| CF-3 | | | | ✓ | | | | |
| CF-4 | | | | | ✓ | | | |
| CF-5 | | | | | | ✓ | | |
| CF-6 | ✓ | | | | | | | |
| CF-7 *(implemented)* | | | | | | | ✓ | |
| CF-11 *(implemented)* | | | | | | | | ✓ |

> 유일한 교차: CF-1의 테넌트 격리 FR(FR-CF1.3·1.4)가 코드상 공통 기반(WBS-1.0)에 위치. 나머지는 CF↔WBS 1:1.

## 3. User Journey × WBS

| UJ | WBS-1.0 | WBS-1.1 | WBS-1.2 | WBS-1.3 | WBS-1.4 | WBS-1.5 | WBS-1.6 | WBS-1.7 |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| UJ-1 골든패스 | ✓ | ✓ | ✓ | ✓ | ✓ | | (트리거) | |
| UJ-2 실패·복구 | ✓ | | | ✓ | | ✓ | | |
| UJ-3 fail-open | ✓ | | ✓ | | | | | |
| UJ-5 코드 RCA *(implemented)* | | | | | | | (트리거) | ✓ |

> UJ-5의 implements = WBS-1.7(CF-11). WBS-1.6(CF-7 이상 탐지)은 UJ-1·UJ-5의 **알람 트리거 소스**. UJ-5는 CF-1(unbound)·CF-3(핸드오프 전달)·CF-6(감사) 경로도 재사용한다(implements 아님).

## 4. JTBD ↔ CF ↔ UJ (top-down)

| CF | Jobs-to-be-Done | 참여 UJ |
|---|---|---|
| CF-1 | JTBD-1, 3 | UJ-1 (UJ-3 전제) |
| CF-2 | JTBD-2, 3, 4 | UJ-1, UJ-3 |
| CF-3 | JTBD-1, 3 | UJ-1, UJ-2 |
| CF-4 | JTBD-4 | UJ-1 |
| CF-5 | JTBD-4 | UJ-2 |
| CF-6 | JTBD-6 | UJ-1, UJ-2, UJ-3 |
| CF-7 *(implemented)* | JTBD-1 | UJ-1·UJ-5 (트리거) |
| CF-11 *(implemented)* | JTBD-1, 2, 4 | UJ-5 |

> UJ-1=골든(정상), UJ-2=DLQ 실패·복구, UJ-3=LLM fail-open, UJ-4=사전대응·자산화(로드맵), UJ-5=코드 RCA(unbound∧anomaly→AI 코드 분석→HITL, implemented). 상세 내러티브: [`../01-prd/index.md`](../01-prd/index.md) §5.

## 5. 코드(모듈) ↔ CF ↔ WBS (구현 매핑)

> 구현 커밋 이력은 작업용 nested repo(`var/signoz`)에 있고, 공개 `ds-apm`은 **single squash**라 개별 커밋·SHA가 없다. 코드 경로↔CF는 [`component-source-map.md`](component-source-map.md), CF↔WBS는 §2 참조.

| 구 모듈(F) | 기능 | CF | WBS |
|---|---|---|---|
| F0 | foundation / pilot scaffolding | CF-6 | WBS-1.0 |
| F1 | SOP grounding & store | CF-1 | WBS-1.1 |
| F2 | AI strategy history | CF-2 | WBS-1.2 |
| F3 | AI quota controls (fail-open) | CF-2 | WBS-1.2 |
| F4 | multi-tenant scope | CF-1 (테넌트) | WBS-1.0 |
| F5 | audit | CF-6 | WBS-1.0 |
| F6 | notification dispatch | CF-3 | WBS-1.3 |
| F7 | PII redaction | CF-4 | WBS-1.4 |
| F8 | JSONL DLQ + replay | CF-5 | WBS-1.5 |
| F9 *(implemented)* | metric anomaly detection (`pkg/query-service/rules/anomaly_rule.go` — z-score moving-avg ± k·σ) | CF-7 | WBS-1.6 |
| F10 *(implemented)* | AI codebase RCA (`pkg/ruler/coderca`: `engine`·`clirunner`·`runstore`(+`query.go`)·`sourcestate`·`reporesolver`·`codebaseconfigstore`(+`sqlcodebasercaconfigstore`)·`trigger`·`worker`·`delivery/amsink.go`·`auditor/dssink.go`; HTTP `signozruler/coderca_handler.go`; 디스패치 훅 `dispatchhook/hook.go`; 서버 `signoz.go`; FE `container/CodeRcaSettings`·`api/codeRca`; 타입 `ruletypes/codebase_rca_config.go`) | CF-11 | WBS-1.7 |

> 마이그레이션: 078(`ds_sop_documents`·`ds_ai_strategy_history`), 079(`ds_ai_config`), 080(oauth 컬럼), 081(`alter ds_ai_strategy_history multi`), **082**(`082_add_ds_codebase_config.go` — `ds_codebase_repo`·`ds_codebase_service_map`·`coderca_run`·`coderca_admission`·`coderca_budget`·`coderca_capacity`·`coderca_skip_stat`, F10/coderca), **083**(`083_update_ds_codebase_config.go` — `ds_codebase_config` 테이블 + `coderca_run` 보고서 4컬럼 `root_cause`·`proposed_fix`·`confidence`·`limitations`). 이상 탐지(F9)는 **신규 테이블 없이** 기존 rule 엔진을 확장(rule JSON의 anomaly 타입)하며 발화 알람에 `anomaly=true` 라벨 스탬프(CF-11 트리거 신호). F9/F10은 [`component-source-map.md`](component-source-map.md)에 as-built 반영 필요(follow-up).
> **이연 대안(미구현)**: jinhyeok 원안의 스캐너 설계(`exceptionscanner`·`codeanalyzer`·`vcs` + 가칭 `ds_exception_signatures`·`ds_exception_reports`·`ds_vcs_config`)는 **코드 미작성**이며, 그 원안이 가정한 "마이그레이션 081 = 예외 스토어"는 실제와 다르다(실 081 = `ds_ai_strategy_history multi`). 설계는 [CF-11 §부록](../01-prd/features/CF-11-code-rca.md)에 보존.

## 6. FR 커버리지 (CF → FR)

FR 단위 매핑은 [`../01-prd/index.md`](../01-prd/index.md) §7 Coverage Map. CF별 `fr_ids`(frontmatter)와 일치:

| CF | FR | 개수 |
|---|---|---|
| CF-1 | FR-CF1.1~1.5 | 5 |
| CF-2 | FR-CF2.1~2.6 | 6 |
| CF-3 | FR-CF3.1~3.3 | 3 |
| CF-4 | FR-CF4.1 | 1 |
| CF-5 | FR-CF5.1~5.3 (5.3 open) | 3 |
| CF-6 | FR-CF6.1~6.3 | 3 |
| **구현 핵심 합** | | **21** |
| CF-7 이상 탐지 *(implemented)* | FR-CF7.1 | 1 |
| CF-11 코드 RCA *(implemented)* | FR-CF11.1~11.6 | 6 |
| 로드맵 *(CF-7 학습형 후속·CF-8~10)* | FR-CF7.2·8.1·9.1·9.2·10.1 | 5 (저fidelity) |

## 6.1 Epic × Story × FR (작업 추적)

에픽/스토리 = 작업 정의 원본([`../03-epics/`](../03-epics/index.md)·[`../04-stories/`](../04-stories/)), FR = 요구. 스토리 **28건(27 done + 1 planned)**. 일정은 [`../05-wbs/index.md`](../05-wbs/index.md) §스토리 일정.

| Epic | Story | 제목 | FR | 상태 |
|---|---|---|---|---|
| 1 | [1.1](../04-stories/1.1.story.md) | SOP 자동 연계 | FR-CF1.1 | done |
| 1 | [1.2](../04-stories/1.2.story.md) | SOP 보관·매칭 | FR-CF1.2 | done |
| 1 | [1.3](../04-stories/1.3.story.md) | 테넌트 격리 | FR-CF1.3 | done |
| 1 | [1.4](../04-stories/1.4.story.md) | SOP 존재 비노출 | FR-CF1.4 | done |
| 1 | [1.5](../04-stories/1.5.story.md) | 비활성·만료 미적용 | FR-CF1.5 | done |
| 2 | [2.1](../04-stories/2.1.story.md) | AI 대응 가이드 생성 | FR-CF2.1 | done |
| 2 | [2.2](../04-stories/2.2.story.md) | 전문가 없이 1차 대응 | FR-CF2.2 | done |
| 2 | [2.3](../04-stories/2.3.story.md) | 사람 승인 강제(HITL) | FR-CF2.3 | done |
| 2 | [2.4](../04-stories/2.4.story.md) | AI 실패에도 전달(fail-open) | FR-CF2.4 | done |
| 2 | [2.5](../04-stories/2.5.story.md) | 사용량 제어 | FR-CF2.5 | done |
| 2 | [2.6](../04-stories/2.6.story.md) | 과거 대응 이력 참조 | FR-CF2.6 | done |
| 3 | [3.1](../04-stories/3.1.story.md) | SOP·AI 5채널 수신 | FR-CF3.1 | done |
| 3 | [3.2](../04-stories/3.2.story.md) | 알림 템플릿 정의 | FR-CF3.2 | done |
| 3 | [3.3](../04-stories/3.3.story.md) | 채널 실패 무중단 | FR-CF3.3 | done |
| 4 | [4.1](../04-stories/4.1.story.md) | 민감정보 마스킹 | FR-CF4.1 | done |
| 5 | [5.1](../04-stories/5.1.story.md) | 무유실 보존 | FR-CF5.1 | done |
| 5 | [5.2](../04-stories/5.2.story.md) | 멱등 재발송 | FR-CF5.2 | done |
| 5 | [5.3](../04-stories/5.3.story.md) | 재발송 HMAC | FR-CF5.3 | **planned (open)** |
| 6 | [6.1](../04-stories/6.1.story.md) | 팀별 정책 설정 | FR-CF6.1 | done |
| 6 | [6.2](../04-stories/6.2.story.md) | 행위 1건당 감사 기록 | FR-CF6.2 | done |
| 6 | [6.3](../04-stories/6.3.story.md) | 감사 실패 무중단 | FR-CF6.3 | done |
| 7 | [7.1](../04-stories/7.1.story.md) | 메트릭 이상 탐지 (z-score 기준선) | FR-CF7.1 | done |
| 11 | [11.1](../04-stories/11.1.story.md) | 트리거 게이트 + AI 코드 RCA 보고서 | FR-CF11.1 | done |
| 11 | [11.2](../04-stories/11.2.story.md) | 저장소·서비스 매핑·기능 토글 설정 | FR-CF11.2 | done |
| 11 | [11.3](../04-stories/11.3.story.md) | 폭주 비용·볼륨 제어 (admission·lease) | FR-CF11.3 | done |
| 11 | [11.4](../04-stories/11.4.story.md) | HITL 수정 제안·read-only 샌드박스 | FR-CF11.4 | done |
| 11 | [11.5](../04-stories/11.5.story.md) | 분석 기준 커밋 pin·echo | FR-CF11.5 | done |
| 11 | [11.6](../04-stories/11.6.story.md) | 분석 실패에도 알람 무영향 (fail-open) | FR-CF11.6 | done |

> Epic N = CF-N(1:1). **Epic 7 = CF-7(이상 탐지), Epic 11 = CF-11(코드 RCA).** Story FR = PRD §7 인수 기준. 코드 근거는 각 story Dev Notes. Epic↔WBS 컴포넌트는 §2. CF-11 코어(M1~M3) TDD 완료 + **HTTP·FE·디스패치 훅 트리거·서버 배선 전체 완료**(설계 §11) → CF-11 status=implemented.

## 7. 검증 가이드 (desync 검출)

이 표가 진실. 각 stub frontmatter는 이 표와 일치해야 한다.
1. `01-prd/features/CF-N.md`의 `implements_uj`·`covered_by_wbs`·`fr_ids` ↔ §1·§2·§6
2. `01-prd/index.md` §5 UJ 목록 ↔ §1·§3 (UJ는 PRD 내장)
3. `05-wbs/index.md` 컴포넌트 트리의 Covers(CF) ↔ §2 · `03-epics/epic-N.md`의 `covers_feature` ↔ §1(에픽=CF 1:1)
4. 새 커밋은 §5에 append (CF·WBS 매핑 포함)
5. `04-stories/{N.M}.story.md`의 `epic`·`fr`·`covers_feature` ↔ §6.1 + 해당 `03-epics/epic-N.md`의 `stories` 목록

## 8. Open / Missing 항목

| 항목 | 상태 | CF | 비고 |
|---|---|---|---|
| HMAC 서명 정책 (NF-5.3.1, FR-CF5.3) | open | CF-5 | replay payload 서명 미정 |
| DLQ 기본 배선 nil | open | CF-5 | `server.go` sink 주입 미연결 |
| Idempotency 키 (fingerprint,channel) 확장 | 권장 | CF-5 | 현재 fingerprint만 |
| Replay API/UI | 범위 밖 | CF-5 | ledger·sink만 제공 |
| Multi-tenant RLS | non-goal(MVP) | CF-1 | label filter only |
| PII OTel Collector 단 이동 | open | CF-4 | 현재 ingress 단일 지점 |
| Redaction rate metric | 미구현 | CF-4 | |
| Frontend 운영자 검수 화면 | open | CF-3 | 변경 영역 미식별 |
| CF-11 통합 seam (디스패치 훅 트리거·라우트 등록·서버 배선·FE) | 해소됨 | CF-11 | 전체 배선·테스트 완료 (설계 §11) |
| CF-11 HTTP 핸들러 + 설정 FE | 해소됨 | CF-11 | coderca_handler.go(10 엔드포인트) + CodeRcaSettings 구현 (M4, 설계 §13) |
| CF-11 잔여 follow-up (EvidenceCollector·OS 샌드박스·monorepo clone·수동 smoke) | open | CF-11 | CF-11 open_items 참조 |
| CF-7 학습형 기준선(seasonal/learned) | planned | CF-7 | v1=통계(z-score) 구현, 학습형은 후속 (FR-CF7.2) |
| 로드맵 CF-8~10 | planned | CF-8~10 | 자동조치·자산화·ITSM |
| 이연 대안: 스캐너 기반 미등록 예외 대응 | deferred | CF-11 | jinhyeok 원안 — 미구현, [CF-11 §부록](../01-prd/features/CF-11-code-rca.md) 보존 |
| drift: `sop_document_file_store.go` | 정정됨 | CF-1 | 코드 삭제, DB store가 영속화 |
| drift: opsgenie 6번째 채널 | 정정됨 | CF-3 | 보강 없음, "5채널"은 보강 기준 |

## 9. 미생성 (BMAD 범위 밖, 현 시점)

- **01-overview / 00-brief** — 별도 산출물로 만들지 않음(필요 시 PRD에서 파생).
- **02-usecase** — 폐지. User Journey는 PRD §5에 내장(BMAD).
- **Architecture(ERD 등)** — BMAD Solutioning 단계로 별도 진행 예정.
