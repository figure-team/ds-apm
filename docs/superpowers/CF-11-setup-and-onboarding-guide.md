# CF-11 (코드 RCA) — 셋업 & 온보딩 가이드

> 출처: 2026-06-15 세션. 실제 SigNoz backoffice + codex로 CF-11을 처음부터 셋업하고, 화면 "테스트 실행"으로 divide-by-zero 근본원인까지 뽑으며 부딪힌 지점을 정리.
>
> **이 문서의 가치**: CF-11은 사용자가 직접 셋업해야 동작한다(LLM 에이전트, 저장소, 서비스 매핑, 트리거). 그 여정의 *비자명한 함정*과 해법의 집약이며, 향후 가이드 투어·AI 온보딩의 1차 자료. 각 단계의 `🧭 온보딩 포인트`가 투어 후보.
>
> 자매 문서: `CF-2-setup-and-onboarding-guide.md` (AI 1차 분석 초안), `CF-11-testing-guide.md` (Tier별 테스트).

---

## 0. CF-11이 뭐고, 어디에 노출되나

알람이 **SOP에 안 묶일 때(unbound)** 또는 **이상탐지(CF-7) 신호**가 있을 때, AI CLI 에이전트(claude/codex)가 서비스의 **소스 코드를 READ-ONLY로 탐색**해 근본원인·제안수정을 도출한다. HITL — proposedFix는 제안일 뿐 **절대 자동 적용 안 함**.

| 노출 면 | 경로 | 용도 |
|---|---|---|
| 코드 RCA 설정 | `/settings/code-rca` | 설정·저장소·서비스 매핑 + 분석 이력 |
| **온디맨드 테스트** | 분석 이력 탭 **"테스트 실행"** 버튼 | 알람 없이 화면에서 RCA 트리거(2026-06-15 추가) |
| 디스패치(실사용) | unbound/anomaly 알람 → 트리거 → 워커 | 자동 RCA |
| 런 조회 | `GET /api/v2/ds/coderca/runs`, `/runs/{id}` | 결과 |

CF-2와 상보적: **SOP 바운드 → CF-2 대응전략**, **언바운드 → CF-11 코드 RCA**.

---

## 1. 사용자 셋업 여정 (순서대로)

### 1-1. LLM 에이전트 설정 (env)
CF-11은 per-org 설정이 아니라 **배포 레벨 env**로 에이전트를 고른다.

| env | 의미 |
|---|---|
| `DS_APM_CODERCA_AGENT` | `claude` \| `codex` (기본 claude) |
| `DS_APM_CODERCA_MODEL` | 모델명 |
| `DS_APM_CODERCA_MAX_BUDGET_USD` | claude 비용 상한(기본 "0.50") |
| `DS_APM_CODERCA_AUTH_TOKEN` | 에이전트 모델-API 인증(미설정 시 호스트 로컬 CLI 인증 상속) |
| `DS_APM_CODERCA_DIR` | 소스 체크아웃 베이스(기본 `$TMPDIR/ds-coderca`) |

> ⚠️ **함정 (codex 모델)**: ChatGPT 구독 계정 codex는 `gpt-5`·`gpt-5-codex`를 거부한다. `~/.codex/config.toml`의 `model`(예: **`gpt-5.5`**)을 `DS_APM_CODERCA_MODEL`에 줘야 한다.
> ⚠️ **함정 (codex read-only로 코드 읽기)**: codex는 파일을 **셸 명령**(cat/grep)으로 읽는데, 프롬프트가 "셸 금지"였으면 codex가 체크아웃을 못 읽는다(claude의 Read/Grep/Glob 툴 모델 전제). → 에이전트별 read-only 툴링(포트/어댑터)으로 해결: claude=파일툴, codex=read-only 셸. 프롬프트 지시가 CLI 플래그(`-s read-only` / `--allowed-tools`)와 동기.
> ⚠️ **함정 (cli 인증)**: AUTH_TOKEN 미설정이면 워커의 자식 `codex`/`claude`가 호스트 인증(`~/.codex`, claude 로그인)을 상속한다 → 컨테이너에선 인증/바이너리 부재로 실패. 호스트 실행 또는 인증 주입 필요.

🧭 **온보딩 포인트**: 모델명 유효성이 provider/계정에 의존 + 셸/툴 모델 차이 → "테스트 실행"으로 먼저 1건 돌려 검증 유도.

### 1-2. RCA 설정 — `/settings/code-rca` (설정 탭) 또는 `PUT /ds/coderca/config`
`enabled`, `minSeverity`(critical|high|error|warning|info), `cooldownWindowSecs`, `maxRunsPerDay`, `maxQueueDepth`, `maxConcurrentRuns`(0..2), `allowUnboundWithoutAnomaly`.

> ⚠️ **함정 (검증)**: `cooldownWindowSecs`·`maxRunsPerDay`·`maxQueueDepth`는 **>= 1** 필수, `maxConcurrentRuns`는 **0..2**. 0 주면 400. (기본: cooldown 21600, maxRunsPerDay 20, maxQueueDepth 50)

🧭 **온보딩 포인트**: 기본값이 fail-closed(enabled=false). 켜는 것 + 한도 의미(쿨다운=같은 장애 재실행 억제) 설명.

### 1-3. 저장소 등록 — `PUT /ds/coderca/repos`
`CodebaseRepo`: `repoId`, `gitUrl`, `defaultBranch`, `credential`, `enabled`.

> ⚠️ **함정 (gitUrl)**: `scheme://`(https/ssh/git) 또는 scp형(`git@host:path`)이어야 통과. **로컬 테스트는 `file:///abs/path`** 로 클론 가능(git clone --mirror).
> ⚠️ **함정 (credential)**: 암호화 키(`DS_APM_AI_CONFIG_ENCRYPTION_KEY`) 미설정(평문 모드)이면 **비어있지 않은 credential은 거부**(fail-closed). 공개/무자격 저장소는 OK.

🧭 **온보딩 포인트**: source state(fetched/baselineCommit)는 읽기전용 — 첫 런이 클론·해석하기 전엔 비어 있음.

### 1-4. 서비스 매핑 — `PUT /ds/coderca/service-maps`
`serviceName` → `repoId` (+ monorepo면 `subpath`). 알람의 `service.name`을 저장소로 연결.

🧭 **온보딩 포인트**: 매핑 없으면 RCA가 `SkipNoRepoMapping`으로 스킵(무증상). 매핑이 RCA의 필수 전제.

---

## 2. 트리거 (언제 RCA가 도나)

### 디스패치 경로 (자동)
알람 디스패치 → SOP 바인딩 시도 → **status=Missing(언바운드)** 이면 코드 RCA 게이트로:
`feature_on → anomaly(fail-closed) → severity → service→repo → Admit`.

- **anomaly 신호**: 알람 라벨 `anomaly`(CF-7 이상탐지 룰이 firing 시 스탬프). 없으면 fail-closed로 스킵.
- `allowUnboundWithoutAnomaly=true`면 anomaly 없이 unbound+severity만으로 admit(경고 로그) — **데모/초기엔 이걸 켜야** 알람만으로 트리거됨.

> ⚠️ **함정**: SOP에 **바운드되면 CF-2로 가고 CF-11은 안 돈다**(상보 관계). CF-11은 언바운드 알람 대상.

### 온디맨드 (화면 테스트)
분석 이력 탭 **"테스트 실행"**: 서비스 입력 → `POST /ds/coderca/runs` → 큐 등록(한도 적용) → 워커 처리 → 폴링으로 queued→running→done 표시 → 행 클릭 시 근본원인·제안수정. **알람 불필요.**

🧭 **온보딩 포인트**: 실사용 트리거(언바운드+anomaly)는 조건이 많아 비자명 → "테스트 실행"이 첫 검증 수단.

---

## 3. 워커/엔진 흐름 (참고)

`codercaworker`(5초 폴링) → `ClaimNext`(lease) → **resolve service→repo → 소스 준비(mirror clone + baseline worktree 체크아웃) → BuildPrompt(에이전트별 read-only 지시) → CLIRunner(codex/claude, read-only) → ParseRCAResult → 핸드오프 delivery(HITL) → 감사 → Finalize**. 결과(rootCause/proposedFix/confidence/limitations/baseline)는 `coderca_run`에 저장.

> read-only 강제: claude `--allowed-tools Read,Grep,Glob --disallowed-tools Bash,Write,Edit,...`, codex `-s read-only`. proposedFix는 **절대 적용 안 됨**.

---

## 4. 결손 / 알려진 한계

- **incident 직접 연결 키 부재**: run은 `org/service` 기준(+시간 근사) — 특정 인시던트에 하드 연결 안 됨. (장애보고서 집약에서 service+시간으로 근사, 한계 명시)
- **조치 실행 이력 미연동**: 제안만, 실제 수행 추적 별개.
- codex read-only는 OS 샌드박스라 환경에 따라 민감(셸 의존). 에이전트 툴링으로 정렬했으나 호스트 인증 필요.

---

## 5. 온보딩(driver.js / AI) 설계용 요약

**사용자가 직접 해야 하고 막히기 쉬운 순서**:
1. LLM 에이전트 env → **모델명(codex=gpt-5.5)·셸/툴 모델·cli 인증**에서 막힘
2. RCA 설정 → **한도 >=1 검증**, enabled 기본 off
3. 저장소 등록 → **gitUrl 스킴·credential(암호화 미설정 시 거부)**
4. 서비스 매핑 → **없으면 무증상 스킵**(SkipNoRepoMapping)
5. 트리거 → **언바운드+anomaly 조건**(비자명) vs **"테스트 실행"**(즉시)

무증상 실패(매핑 누락·바인딩되어 CF-2로 감·anomaly 미스탬프)를 **사전 진단/체크리스트**로 바꾸는 게 온보딩 최대 가치. 첫 검증은 **"테스트 실행"** 버튼으로.
