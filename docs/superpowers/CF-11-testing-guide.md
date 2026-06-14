# CF-11 (AI 코드베이스 RCA) 테스트 가이드

> SOP가 없는 이상 알람이 떴을 때 AI가 소스코드를 분석해 근본원인+수정 제안 보고서를 만드는 기능(UJ-5)을 **단계별로 검증**한다.
> 3개 티어로 나뉜다 — 위에서부터 차례로 하면 된다. **Tier 0은 서버 없이** 바로 가능하고, Tier 2는 실제 LLM CLI + 서버가 필요하다.

---

## 0. 사전 개념 — 무엇이 분석을 발화시키나 (게이트 조건)

알람 한 건이 코드 RCA로 넘어가려면 **다음을 전부** 만족해야 한다 (`pkg/ruler/coderca/trigger/trigger.go`). 하나라도 빠지면 미발화(fail-closed):

| # | 조건 | 의미 |
|---|---|---|
| 1 | **기능 ON** | 해당 org의 코드 RCA 설정 `enabled=true` (기본 OFF, opt-in) |
| 2 | **unbound** | 알람에 매칭되는 SOP가 없음 (SOP 바인딩 상태 = `missing`) |
| 3 | **anomaly** | 알람 라벨 또는 주석에 `anomaly=true`(또는 `1`) — CF-7 이상탐지 룰이 자동 스탬프 |
| 4 | **심각도** | `severity` 라벨이 설정의 `minSeverity` 이상 (기본 `high` → high·critical) |
| 5 | **서비스→저장소 매핑** | `service.name` 라벨의 서비스가 저장소에 매핑돼 있음 |
| 6 | **admission** | dedup(쿨다운)·일일 예산·큐 깊이 통과 |

전부 통과하면 `coderca_run`이 `queued`로 적재되고, 워커가 집어 CLI 에이전트를 read-only로 구동한 뒤 보고서를 만들어 **핸드오프(메타-알람)** 로 전달하고 이력에 남긴다.

---

## Tier 0 — 서버 없이 로직 검증 (가장 빠름, 지금 바로 가능)

LLM·서버·DB 없이 전체 파이프라인 로직이 맞는지 자동화 테스트로 확인한다.

### 0.1 백엔드 단위·통합 테스트

```bash
cd /c/Users/KTDS/git/ds-apm

# 핵심 e2e: 트리거→큐→워커→핸드오프 전 경로 + fail-open + fail-closed
go test ./pkg/ruler/coderca/ -run TestUJ5 -count=1 -v

# 코어 전체 (admission·lease·flood-sim·trigger·worker·engine·sinks·stores)
go test ./pkg/ruler/coderca/... -count=1

# HTTP 핸들러 (자격증명 암호화·평문 비노출·타 org 404)
go test ./pkg/ruler/signozruler/ -count=1

# CF-7 anomaly 라벨 스탬프 + 설정 타입 + 마이그레이션
go test ./pkg/query-service/rules/ ./pkg/types/ruletypes/ ./pkg/sqlmigration/ -count=1
```

**기대**: `TestUJ5EndToEnd`·`TestUJ5FailOpen`·`TestUJ5FailClosed` 포함 전부 `PASS`.

> ⚠️ 이 Windows 머신에서는 `pkg/ruler/coderca/clirunner`와 `...codexcli`/`claudecli` 테스트가 **사전존재 플랫폼 이슈**(`syscall.Kill` 미지원·CLI 바이너리 부재)로 실패한다 — CF-11 구현과 무관하니 무시. Linux/Mac에서는 통과한다.

### 0.2 프론트엔드 테스트

```bash
cd frontend

# 설정·이력 화면 컴포넌트 테스트 (6건)
node node_modules/jest/bin/jest.js --silent "CodeRcaSettings"

# 타입 체크 (coderca 관련 파일에 에러 없어야 함)
node node_modules/typescript/bin/tsc --noEmit -p tsconfig.json 2>&1 | grep -i "codeRca\|CodeRcaSettings" | head
```

**기대**: jest 6/6 `PASS`, tsc grep 결과 비어 있음(에러 0).

여기까지 통과하면 **로직은 검증된 것**이다. 실제 동작(LLM 호출 포함)을 보려면 Tier 1·2로.

---

## Tier 1 — 서버 띄우고 설정·API 검증 (LLM 불필요)

설정 저장·조회, 저장소·매핑 CRUD, 이력 조회 API가 실제로 동작하는지 확인한다. **실제 분석(LLM)은 아직 안 한다.**

### 1.1 서버 기동

> 이 프로젝트의 backend는 ClickHouse(텔레메트리) + SQLite(설정)를 쓴다. 로컬은 보통 다음 순서:

```bash
# (1) 의존 인프라 — ClickHouse + OTel collector
make devenv-up

# (2) 커뮤니티 백엔드 (별도 터미널) — coderca 환경변수 포함
SIGNOZ_INSTRUMENTATION_LOGS_LEVEL=debug \
SIGNOZ_SQLSTORE_SQLITE_PATH=signoz.db \
SIGNOZ_ALERTMANAGER_PROVIDER=signoz \
SIGNOZ_TELEMETRYSTORE_PROVIDER=clickhouse \
SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN=tcp://127.0.0.1:9000 \
SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_CLUSTER=cluster \
SIGNOZ_TOKENIZER_JWT_SECRET=secret \
DS_APM_AI_GENERATOR=mock \
DS_APM_AI_CONFIG_ENCRYPTION_KEY=$(openssl rand -hex 32) \
make go-run-community
```

> **중요**:
> - `DS_APM_AI_GENERATOR`(또는 `DS_APM_LLM_PROVIDER`)를 설정해야 AI 디스패치 훅이 생성되고, 거기에 코드 RCA 트리거가 주입된다. Tier 1에서는 `mock`이면 충분.
> - `DS_APM_AI_CONFIG_ENCRYPTION_KEY`(32바이트 hex)가 있어야 **비공개 저장소 자격증명**을 저장할 수 있다(없으면 공개 저장소만 허용 — fail-closed).
> - **이 Windows 환경에서는** `cmd/community` 최종 링크가 사전존재 `bytedance/sonic` 의존성 비호환으로 실패한다. **Linux/Mac 또는 Docker**에서 띄워라(`deploy/docker/docker-compose.local.yaml` 참고). 라이브러리 컴파일·테스트(Tier 0)는 영향 없다.

마이그레이션 082·083은 부팅 시 자동 적용된다(`provider.go`에 등록됨). 부팅 로그에 `add_ds_codebase_config`·`update_ds_codebase_config`가 보이면 OK.

### 1.2 화면(UI)으로 설정 — 권장

브라우저에서 **`/settings/code-rca`** 진입. (좌측 설정 메뉴에 "Code RCA" 항목)

> ⚠️ 현재 진입 권한은 **ADMIN 전용**(AI Module 페이지와 동일). 운영자(viewer)도 이력을 보게 하려면 권한을 넓혀야 한다 — 가이드 맨 아래 "권한 넓히기" 참조.

**설정 탭**:
1. **기능·임계값 카드** — `기능 사용` ON, `최소 심각도`=high, 나머지 기본값(쿨다운 21600초, 일일 20, 큐 50, 동시 1).
2. **저장소 카드** — `저장소 추가`: 분석할 Git 저장소 등록.
   - 공개 repo면 자격증명 비워둠 (예: `https://github.com/<org>/<small-repo>.git`).
   - 비공개면 읽기 PAT 입력 (암호화 저장됨).
   - `사용(enabled)` 체크.
3. **서비스 매핑 카드** — `매핑 추가`: 알람의 `service.name` → 위 저장소 ID. (예: 서비스명 `pay` → repo `repo-1`)

저장 후 새로고침 → 값이 유지되고, 저장소의 자격증명 칸이 `<unchanged>`로 마스킹돼 보이면 정상(평문 비노출).

**이력 탭**: 아직 분석이 없으니 비어 있음 — Tier 2에서 채워진다.

### 1.3 API로 설정·검증 — UI 대신 curl

org 스코프는 인증 토큰(claims)에서 온다. 아래 `$TOKEN`·`$BASE`는 환경에 맞게.

```bash
BASE=http://localhost:8080/api/v2
AUTH="Authorization: Bearer $TOKEN"

# 설정 조회 (미설정 시 기본값 enabled=false 반환)
curl -s -H "$AUTH" $BASE/ds/coderca/config | jq

# 기능 켜기 + 임계값
curl -s -X PUT -H "$AUTH" -H 'Content-Type: application/json' $BASE/ds/coderca/config -d '{
  "enabled": true, "minSeverity": "high",
  "cooldownWindowSecs": 21600, "maxRunsPerDay": 20,
  "maxQueueDepth": 50, "maxConcurrentRuns": 1,
  "allowUnboundWithoutAnomaly": false
}'

# 저장소 등록 (공개 repo, 자격증명 없음)
curl -s -X PUT -H "$AUTH" -H 'Content-Type: application/json' $BASE/ds/coderca/repos -d '{
  "repoId": "repo-1", "gitUrl": "https://github.com/<org>/<small-repo>.git",
  "defaultBranch": "main", "enabled": true, "credential": ""
}'

# 저장소 목록 (credential은 "<unchanged>"로 마스킹돼 와야 함 — 평문 비노출 검증)
curl -s -H "$AUTH" $BASE/ds/coderca/repos | jq

# 서비스→저장소 매핑
curl -s -X PUT -H "$AUTH" -H 'Content-Type: application/json' $BASE/ds/coderca/service-maps -d '{
  "serviceName": "pay", "repoId": "repo-1", "subpath": ""
}'

# 이력 목록 (아직 비어 있음)
curl -s -H "$AUTH" "$BASE/ds/coderca/runs?limit=20" | jq
```

**검증 포인트**:
- 설정 PUT → `204`, 재조회 시 반영.
- 저장소 목록의 `credential`이 절대 평문으로 안 나옴(`<unchanged>` 또는 빈 값).
- 잘못된 값(`minSeverity:"nonsense"`)으로 PUT → `400`.

---

## Tier 2 — 실제 end-to-end (LLM CLI + 알람 트리거)

진짜로 알람 → 분석 → 보고서까지 도는지 본다. **비용이 발생**하니 작은 repo + 낮은 예산으로.

### 2.1 추가 준비

1. **CLI 코딩 에이전트 설치 + 인증** — `claude` 또는 `codex` 바이너리가 서버의 `PATH`에 있어야 한다.
2. **서버 환경변수에 에이전트 설정 추가**(1.1의 기동 명령에 더해):

```bash
DS_APM_CODERCA_AGENT=claude \              # 또는 codex (기본: claude)
DS_APM_CODERCA_MODEL=claude-fable-5 \       # 에이전트 모델
DS_APM_CODERCA_MAX_BUDGET_USD=0.20 \        # 1회 분석 하드 상한 (기본 0.50)
DS_APM_CODERCA_AUTH_TOKEN=<model-api-key> \ # 미설정 시 LLM provider 키로 폴백
DS_APM_CODERCA_DIR=/var/tmp/ds-coderca \    # 미러 클론·체크아웃 캐시 (기본 OS temp)
# ... (1.1의 나머지 + 진짜 LLM provider) ...
```

> 에이전트는 **read-only**로 강제된다(claude: 도구 allow/deny + `--max-budget-usd`; codex: `-s read-only`). 코드를 절대 수정하지 않는다.

### 2.2 "unbound + anomaly" 알람 만들기 — 두 가지 방법

#### 방법 A: 메트릭 이상탐지 룰 (실전 경로)

1. SigNoz에서 **anomaly 타입 알림 룰**을 만든다(특정 메트릭에 z-score 기준선).
2. 그 룰이 매칭할 `service.name`은 2.1에서 매핑한 서비스(`pay`)로 하고, **그 서비스용 SOP는 등록하지 않는다**(unbound 유지).
3. 메트릭이 기준선을 벗어나 룰이 firing → CF-7이 알람에 `anomaly=true`를 자동 스탬프 → 디스패치 → 코드 RCA 트리거.

> 메트릭을 인위적으로 튀게 만들기 번거로우면 방법 B가 빠르다.

#### 방법 B: 알람 직접 주입 (가장 빠른 수동 트리거)

코드 RCA는 **디스패치 경로의 unbound 분기**에서 발화한다. 매칭 SOP가 없고 `anomaly=true`인 알람을 alertmanager로 밀어넣으면 된다:

```bash
curl -s -X POST -H "$AUTH" -H 'Content-Type: application/json' \
  http://localhost:8080/api/v2/alerts -d '[{
    "labels": {
      "alertname": "PayServiceError",
      "service.name": "pay",
      "severity": "critical",
      "anomaly": "true"
    },
    "annotations": { "summary": "결제 서비스 5xx 급증(수동 트리거 테스트)" }
  }]'
```

> 라벨 4개(`alertname`·`service.name`·`severity`·`error_class`)가 dedup 시그니처를 이룬다. 같은 라벨로 다시 쏘면 쿨다운(기본 6시간) 동안 1건으로 합쳐진다 — 두 번째 분석을 바로 보려면 `alertname`을 바꿔라.
> ⚠️ 실제 엔드포인트 경로/페이로드 스키마는 이 배포의 alertmanager 설정에 따라 다를 수 있다. 안 되면 방법 A(룰 기반)로.

### 2.3 결과 확인

1. **서버 로그**: `coderca trigger: run queued` → 워커가 claim → `delivered` (또는 실패 시 `failed`/`timeout`).
2. **이력 탭** (`/settings/code-rca` → 분석 이력) 또는 API:
   ```bash
   curl -s -H "$AUTH" "$BASE/ds/coderca/runs?limit=20" | jq
   # 상태 전이: queued → running → done(또는 failed/timeout)
   ```
   `done`인 run을 클릭(또는 `GET /ds/coderca/runs/{runId}`)하면 **근본원인·수정 제안·신뢰도·기준 커밋(baseline)** 이 보인다.
3. **메타-알람**: 평소 알림 채널(Slack 등)에 `alertname=CodeRCASuggestion`, `severity=info`, `coderca=true` 라벨의 "Code RCA suggestion" 알림이 도착. 본문에 "AI 생성 제안 — 적용 전 사람 검토 필요"(HITL) 경고 포함.

### 2.4 안전장치(fail-open) 확인 — 선택

- **분석이 실패해도 원래 알람 전달은 멀쩡해야 한다.** 일부러 실패시키려면: 매핑된 repo의 `gitUrl`을 접근 불가 주소로 바꾸고 트리거 → run은 `failed`로 기록되지만 원본 알람 디스패치는 정상.
- **미발화 확인**: `enabled=false`로 끄거나 / `anomaly` 라벨을 빼거나 / 매핑을 지우고 트리거 → `coderca_run`이 안 생긴다(이력 그대로).

---

## 부록 — 권한 넓히기 (운영자에게 이력 노출)

현재 `/settings/code-rca` 진입은 **ADMIN 전용**이다. 운영자(viewer)도 분석 이력을 보게 하려면:

`frontend/src/utils/permission/index.ts`에서
```ts
CODE_RCA_SETTINGS: ['ADMIN'],
```
을
```ts
CODE_RCA_SETTINGS: ['ADMIN', 'EDITOR', 'VIEWER'],
```
로 바꾼다. 백엔드가 쓰기(설정·저장소·매핑 변경)는 EditAccess로 막고, 화면의 편집 컨트롤은 `isAdmin`으로 숨기므로 viewer는 **조회만** 가능하다.

---

## 빠른 체크리스트

- [ ] Tier 0: `go test ./pkg/ruler/coderca/ -run TestUJ5 -v` 통과 + FE jest 6/6
- [ ] Tier 1: 서버 기동(AI generator + 암호화 키 env) → `/settings/code-rca`에서 기능 ON·저장소·매핑 등록 → 새로고침 유지·자격증명 마스킹
- [ ] Tier 2: CLI 에이전트 PATH 등록 → unbound+anomaly 알람 트리거 → 이력에서 `done` + 보고서 → 채널에 메타-알람
- [ ] 안전: 실패해도 원본 알람 무영향 / 게이트 미충족 시 미발화

문제가 생기면 어느 티어·어느 단계에서 멈췄는지 알려주면 같이 짚어보자.
