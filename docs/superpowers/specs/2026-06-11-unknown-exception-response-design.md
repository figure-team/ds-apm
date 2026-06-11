---
id: SPEC-CF7-UNKNOWN-EXCEPTION
title: 미등록 예외 대응 (Unknown Exception Response) — 설계서
type: design-spec
status: approved-draft
target_cf: CF-7 (가칭 "미등록 예외 인지·코드연계 분석·대응")
updated: 2026-06-11
---

# 미등록 예외 대응 (Unknown Exception Response) — 설계서

> 본 문서는 brainstorming 산출 설계서이며, 승인 후 `docs/spec/` BMAD 체인(PRD CF-7 + 에픽 + 스토리 + WBS)으로 전개한다.
> 작성 규칙은 [docs/spec/PROCESS.md](../../spec/PROCESS.md) 준수 — 본문 한국 SI 문체, ID·Gherkin 키워드 영문.

---

## §0. 문제 정의

현재 DS-APM은 **Alert Rule에 등록된 현상**만 대응한다. 알람 firing 시 `sop_id` 라벨로 SOP에 연계(CF-1)하고 AI 가이드(CF-2)를 생성하는 구조이므로, **Alert Rule에 등록되지 않은 예외(unexpected exception)는 인지조차 되지 않는다.**

본 기능의 목표:

1. 미등록 예외를 시스템이 **자동 인지**하여야 한다.
2. 오류 로그만으로는 원인 추론 정확도가 낮으므로, **git/svn 코드베이스(소스·blame·커밋 이력)를 연계**하여 AI가 오류를 분석하여야 한다.
3. **초기 대응 가이드 + 보고서 초안**을 자동 생성하고 알림을 전송하여야 한다.
4. 사후처리로 해당 예외를 **신규 SOP/Runbook으로 등록(자산화)**하여, 이후 동일 예외는 기존 CF-1 경로로 대응되어야 한다.

## §1. 확정 결정 사항 (brainstorming 결과)

| # | 결정 항목 | 결정 | 근거 |
|---|---|---|---|
| 1 | 인지 방식 | **텔레메트리 스캐너** — ClickHouse 예외·에러로그 주기 스캔, 기존 rule/SOP 매핑 시그니처 제외 | Alert Rule이 아예 없는 예외도 인지해야 함 (catch-all rule로는 불가) |
| 2 | 분석 방식 | **스택트레이스 기반 컨텍스트 추출** (결정적) — 에이전트 tool-use 아님 | 토큰·시간 예측 가능, 기존 quota·fail-open 철학과 정합 |
| 3 | VCS 연결 | **로컬 미러** — 읽기전용 자격증명으로 주기 fetch(git `clone --mirror`, svn checkout) | VCS 서버 부하·rate limit 없음, 오프라인 분석 |
| 4 | 코드 보안 | **기존 LLM 경로 + org별 "코드 컨텍스트 포함" 정책 스위치** — OFF 시 로그·메타만으로 분석 | MVP 현실안, 기존 secretbox 암호화·감사 체계 재사용 |
| 5 | 범위 | **4단계 전체를 한 CF로** (인지→분석→보고·알림→사후등록) | end-to-end 사용자 가치 단위 (BMAD) |
| 6 | 아키텍처 | **C안: 2단 알림 하이브리드** (아래 §2) | silent drop 0·fail-open 불변 원칙 계승 + 분석 시간 제약 해소 |

## §2. 아키텍처 — 2단 알림 하이브리드

핵심 긴장점: 코드베이스 연계 LLM 분석은 수십 초가 소요되나, 기존 알림 핫패스의 AI hook은 **1초 fail-open budget**(CF-2)이다. 이를 분리하기 위해 알림을 2단으로 나눈다.

```
Exception Scanner (주기, org별)
  │  ClickHouse 예외·에러로그 스캔 → 시그니처 정규화 → 신규 클러스터 판정
  ├─▶ [즉시·동기] 1차 알림: synthetic alert 주입
  │     PutAlerts(ctx, orgID, alerts)            ← pkg/alertmanager/signozalertmanager/provider.go:104
  │     라벨: ds_unregistered_exception=true, service, severity
  │     → 기존 dispatcher → PII 마스킹(CF-4) → 5채널(CF-3) → DLQ(CF-5) → 감사(CF-6) 전부 재사용
  │
  └─▶ [비동기] 분석 워커 (codeanalyzer)
        ① 스택트레이스 파싱 (파일/라인/프레임)
        ② 서비스명 → repo 매핑(ds_vcs_config) → 로컬 미러 조회
        ③ 컨텍스트 추출: 소스 스니펫(±30라인) + git blame + 최근 커밋 log
        ④ LLM 분석 (기존 aigenerator provider, 코드포함 스위치 반영)
        ⑤ 보고서 초안 저장(ds_exception_reports)
        ⑥ 2차 알림: 보고서 요약 + UI 링크 (동일 5채널)
        ⑦ (운영자, UI) 보고서 검수 → "SOP/Runbook으로 등록" → draft 생성 (HITL)
```

- **분석 실패·LLM 장애 시**: 1차 알림은 이미 전달됨 — 정보 손실 0 (UJ-3 fail-open 계승). 2차는 best-effort이며 실패는 감사 기록 + 시그니처 상태에 표시.
- **PRD §9.1 Non-goal(메트릭/벡터 자동 SOP 라우팅 금지)과의 관계**: 본 기능은 *기존 SOP로의 자동 라우팅이 아니라* **신규 예외의 발견과 등록 제안**이다. SOP 연계는 운영자 승인(등록) 후 explicit `sop_id` 라벨로만 발생 — 원칙 유지.

## §3. 컴포넌트 설계

### §3.1 Exception Scanner — `pkg/ruler/exceptionscanner/` (신규)

| 항목 | 내용 |
|---|---|
| 역할 | org별 주기 스캔(기본 60초, 설정 가능)으로 ClickHouse의 트레이스 예외(span events)·에러 로그를 조회, 신규 예외 클러스터 인지 |
| 시그니처 정규화 | `fingerprint = hash(예외 타입 + 최상위 애플리케이션 프레임(파일:함수) + 서비스명)` — 메시지의 가변부(ID·숫자·타임스탬프)는 제외하여 동일 원인의 재발을 한 시그니처로 묶음 |
| 제외(known) 판정 | ① 기존 Alert Rule이 커버하는 시그니처 ② 이미 SOP에 연결된(linked_sop_id 존재) 시그니처 ③ dismissed 상태 시그니처 ④ 재알림 억제 윈도우(기본 24h) 내 기보고 시그니처 — count·last_seen만 갱신 |
| 출력 | 신규 fingerprint → `ds_exception_signatures` insert(status=new) → 1차 알림 발화(status=notified) → 분석 큐 적재 |
| 구현 형태 | 기존 factory/provider 컨벤션(docs/contributing/go/provider.md)에 따른 백그라운드 서비스. ClickHouse 조회는 query-service의 기존 error/exception 리더 경로(pkg/query-service/app/clickhouseReader) 재사용 |

### §3.2 분석 워커 — `pkg/ruler/codeanalyzer/` (신규)

| 항목 | 내용 |
|---|---|
| 입력 | 시그니처(status=notified) + 예외 샘플(스택트레이스 포함, 최대 3건 — 기존 `RunbookDraftRequest.ErrorExamples` 상한 관행 준수) |
| 스택트레이스 파서 | 언어별(Java/Go/Python/Node 우선) 프레임 추출 → 파일 경로·라인·함수. 파싱 실패 시 코드 컨텍스트 없이 로그 기반 분석으로 degrade |
| 코드 컨텍스트 추출 | 로컬 미러에서: ① 해당 파일 스니펫 ±30라인 ② `git blame` 해당 라인 ③ 해당 파일 최근 커밋 5건(`git log`). svn은 `svn blame`/`svn log` 동등 구현. **총량 상한**(기본 32KB)으로 프롬프트 폭주 방지 |
| LLM 호출 | 기존 `pkg/ruler/aigenerator/` provider(llm/local/mock) 재사용 — quota·사용량 이력·감사 그대로 적용. org 스위치 `include_code_context=false`면 코드 컨텍스트 생략 |
| 출력 | 보고서 초안(`ds_exception_reports`): 원인 추론(가설+신뢰도), 근거 커밋 목록, 초기 대응 가이드, Runbook 초안(§3.4에서 사용), LLM 사용량 메타. 저장 후 status=analyzed + 2차 알림 |
| 실행 모델 | 단일 워커 goroutine + DB 기반 큐(status 컬럼 폴링) — MVP에서는 외부 큐 인프라 도입하지 않음 |

### §3.3 VCS 미러 매니저 — `pkg/vcs/` (신규)

| 항목 | 내용 |
|---|---|
| 설정 | `ds_vcs_config`(org당 1행): repos JSON 배열 — 각 항목 `{name, type: git|svn, url, 서비스명 매핑(service.name 패턴), 기본 브랜치}` + 자격증명 암호문(기존 secretbox 패턴, ds_ai_config과 동일) + `include_code_context` 스위치 |
| 미러 동작 | git: `clone --mirror` 후 주기 `fetch`(기본 10분). svn: checkout 후 주기 `update`. 저장 위치 `var/vcs/{org}/{repo}` (기존 `var/` 파일 영역 관행) |
| 격리 | org별 디렉터리 분리. 자격증명은 읽기전용 계정 권장(문서화). 미러 실패는 감사 기록 + 분석 시 "코드 컨텍스트 불가" degrade |

### §3.4 사후 SOP/Runbook 등록 (기존 확장)

- 보고서 상세 UI에서 운영자가 **"SOP/Runbook으로 등록"** 실행 → ① 신규 SOP 생성 또는 기존 SOP 선택 ② 보고서의 Runbook 초안을 기존 `RunbookDrafter` 출력 형식으로 변환해 **draft 상태**로 임베드 ③ 시그니처에 `linked_sop_id` 기록.
- 승인은 **기존 상태머신**(draft→approved, [runbook.go:110-124](../../../pkg/types/ruletypes/runbook.go)) 그대로 — 본 기능은 새 승인 절차를 만들지 않는다 (HITL 불변).
- 등록 후 운영자가 해당 현상의 Alert Rule을 생성하고 `sop_id` 라벨을 달면 이후 재발은 CF-1 골든패스로 흡수된다. (Alert Rule 자동 생성은 본 CF 범위 외 — Open Item)
- frontend 기존 `container/Runbooks/RunbookDraftFromError.tsx`(오류→AI 런북 초안) 패턴을 재사용·확장.

## §4. 데이터 모델 — 마이그레이션 081 (PostgreSQL/bun, 078~080 패턴)

```
ds_exception_signatures
  org_id TEXT PK · fingerprint TEXT PK
  service TEXT · exception_type TEXT · sample_message TEXT(가변부 제거)
  first_seen / last_seen TEXT · count INT
  status TEXT  -- new → notified → analyzed → registered | dismissed
  linked_sop_id TEXT NULL
  payload TEXT  -- JSON: 예외 샘플(스택트레이스, 최대 3건, PII 마스킹 적용)

ds_exception_reports
  org_id TEXT PK · fingerprint TEXT PK · version TEXT PK
  contract_version TEXT · created_at TEXT
  payload TEXT  -- JSON: 원인추론·근거커밋·초기대응가이드·runbook초안·LLM사용량

ds_vcs_config
  org_id TEXT PK
  repos TEXT  -- JSON 배열 (name/type/url/서비스매핑/브랜치)
  credential_ciphertext TEXT  -- secretbox
  include_code_context BOOL · mirror_interval_seconds INT
  updated_at TEXT
```

상태머신: `new→notified`(1차 알림 성공) `→analyzed`(보고서 저장) `→registered`(SOP 연결) / 어느 단계에서든 `→dismissed`(운영자 무시 처리, 재알림 차단). `dismissed→new` 직접 전이 금지(재발 시 count만 갱신).

## §5. API & Frontend

### §5.1 REST (`/api/v2/ds/*`, 기존 handler/module 컨벤션)

| 경로 | 동작 |
|---|---|
| `GET /api/v2/ds/exceptions` | 시그니처 목록 (상태·서비스 필터, 페이징) |
| `GET /api/v2/ds/exceptions/{fingerprint}/report` | 보고서 조회 |
| `POST /api/v2/ds/exceptions/{fingerprint}/dismiss` | 무시 처리 |
| `POST /api/v2/ds/exceptions/{fingerprint}/register` | SOP/Runbook draft 등록 (신규 SOP 생성 또는 기존 SOP 지정) |
| `GET/PUT /api/v2/ds/vcs/config` | VCS 설정 CRUD (자격증명은 write-only, 응답 마스킹) |

전 엔드포인트 org 스코프 + 기존 authz + 감사 기록(CF-6).

### §5.2 Frontend

1. **미등록 예외 페이지** (`pages/UnregisteredExceptions/` 가칭): 시그니처 목록(상태 뱃지·발생수·서비스·최근 발생) + 상세 drawer(스택트레이스 → 보고서: 원인 추론·근거 커밋·대응 가이드) + 액션(무시 / SOP 등록)
2. **SOP 등록 모달**: 신규 SOP 생성 or 기존 SOP 선택 → Runbook 초안 미리보기·수정 → draft 저장 (`RunbookForm` 재사용)
3. **설정 > VCS 연결**: repo 목록 CRUD + 서비스 매핑 + 코드전송 스위치 (기존 AI 모듈 설정 페이지 패턴)
4. i18n: 신규 문구는 en/ko 양 locale에 키 추가 (현행 i18n 작업 관행 준수)

## §6. 안전·비기능 요구 (불변 원칙 계승)

| 원칙 | 적용 |
|---|---|
| silent drop 0 | 1차 알림은 분석과 무관하게 즉시·동기. 2차 알림·보고서는 best-effort. 1차 알림 실패는 기존 DLQ로 보존 |
| HITL | 시스템은 어떤 조치도 실행하지 않음. SOP/Runbook은 draft로만 생성, 기존 승인 상태머신 필수 통과 |
| 테넌트 격리 | 전 테이블·스캐너·워커·미러 디렉터리 org 단위. 타 org 시그니처 존재 여부 비노출 (FR-CF1.4 동등) |
| PII | 예외 샘플·알림 페이로드에 기존 CF-4 마스킹 적용. 보고서 저장 전에도 적용 |
| LLM 통제 | 기존 quota·사용량 이력·감사 재사용. 코드 컨텍스트 총량 상한 32KB. org 스위치 OFF 시 코드 미전송 |
| 부하 | 스캐너 쿼리는 lookback 윈도우(기본 5분, 주기보다 넓게 — 경계 유실 방지) + LIMIT. ClickHouse 부하 영향 측정 후 주기 조정 가능 |
| 노이즈 | fingerprint 재알림 억제 24h + dismissed 영구 차단. 폭주 시(주기당 신규 N건 초과) 묶음 알림 1건으로 축약 |

## §7. 테스트 전략

- **유닛**: 시그니처 정규화(가변부 제거·해시 안정성), 언어별 스택트레이스 파서(픽스처: Java/Go/Python/Node), 상태머신 전이(금지 전이 포함), 코드 컨텍스트 상한
- **통합**: 스캐너 제외 로직(기존 rule/SOP/dismissed/억제윈도우) 테이블 테스트, `PutAlerts` 주입→dispatcher 경유 검증, sqlsopstore 패턴의 store 테스트
- **LLM**: `mockaigenerator` 패턴으로 보고서 파이프라인 결정적 검증 + degrade 경로(LLM 실패→1차 알림만)
- **인수(Gherkin/godog, 한글 스텝)**: UJ-5 골든패스 / 분석 실패 분기 / dismiss / SOP 등록 후 재발 시 CF-1 경로 흡수
- **frontend**: Jest+Testing Library, 기존 글로벌 i18n mock 관행(키 반환) 준수

## §8. Open Items (본 CF 범위 외, 후속 결정)

- Alert Rule 자동 생성 제안(등록 시 rule 초안까지) — CF-8 영역과 경계 확인 필요
- 스캐너의 메트릭 이상 탐지 확장(예외가 아닌 지표 이상) — CF-7 로드맵 원안과의 통합 여부
- 사내/자체 LLM endpoint 강제 옵션 — 보안 정책 강화 시
- 보고서의 정식 장애보고서(RCA) 템플릿 승격 — CF-9 연계

## §9. BMAD 체인 전개 계획

본 설계 승인 후 [PROCESS.md](../../spec/PROCESS.md) §10 절차로:

1. `01-prd/index.md` — Feature Map에 CF-7 등재(★ planned→in-progress), UJ-5 추가(§5), Coverage Map에 FR-CF7.1~7.N 추가
2. `01-prd/features/CF-7-unknown-exception.md` — FR 상세(고객 voice) + Given/When/Then
3. `03-epics/epic-7-unknown-exception.md` + `04-stories/7.1~7.N.story.md` — 스토리 분해(스캐너/VCS/분석워커/알림/UI/등록)
4. `05-wbs/` 일정 반영 + `_shared/traceability.md`·`component-source-map.md` 갱신
