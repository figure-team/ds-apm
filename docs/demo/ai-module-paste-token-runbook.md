# DS-APM AI Module — Paste-Token 설정 가이드

> 운영자 대상. 로컬에서 발급한 OAuth 토큰을 Settings UI에 붙여넣어 CLI 기반 LLM 백엔드(Claude / Codex)를 활성화하는 절차.

## 언제 이 문서를 보나

- AI Module을 `provider=LLM` + `transport=CLI` 로 운영하려는 경우.
- 컨테이너 내부에서 `claude auth login` / `codex login` 을 직접 돌릴 수 없는 환경 (대부분의 서버/원격 배포).
- `provider=mock` 또는 `provider=local`만 쓸 거라면 **이 문서 불필요** — Settings에서 그냥 토글하면 됨.

## 한눈에 보기

```
[1] 로컬 머신에서 토큰 발급 (claude setup-token / codex login)
        ↓
[2] Settings → AI Module 진입
        ↓
[3] Provider=LLM, Vendor 선택, Transport=CLI, 토큰 paste
        ↓
[4] Test → Save
        ↓
[5] 알림 트리거 → 실제 strategy 생성 확인
```

전체 소요 ≈ 10분 (토큰 발급 5분 + UI 설정/검증 5분).

## Prereqs

| 항목 | 확인 |
|---|---|
| 운영자가 admin 권한으로 SigNoz UI에 로그인할 수 있다 | Settings 페이지 좌측 메뉴에 "AI Module" 노출 |
| 컨테이너 PATH에 `claude` 또는 `codex` CLI가 설치되어 있다 | `docker exec signoz which claude codex` |
| 운영자 로컬 머신에 동일 CLI가 설치되어 있고 브라우저 인증이 가능하다 | `claude auth status` / `codex login status` |

> **주의**: 컨테이너 안의 CLI 버전과 운영자 로컬의 CLI 버전이 다르면 토큰 형식이 호환되지 않을 수 있습니다. 운영 컨테이너 빌드 시 사용한 npm 패키지 버전(예: `@anthropic-ai/claude-code@x.y.z`)을 운영자 로컬에서도 동일하게 맞추는 것을 권장합니다.

---

## Step 1A — Claude 토큰 발급 (운영자 로컬)

```bash
# 운영자 로컬 머신에서:
claude setup-token
```

진행 흐름:
1. 명령이 브라우저를 띄움 (또는 URL 출력 — 헤드리스 환경이면 URL을 다른 머신에서 열기).
2. Anthropic 계정으로 로그인 (Claude 구독 필요 — Console API 키와 다름).
3. CLI가 long-lived OAuth 토큰을 stdout에 출력함. 예: `sk-ant-oat01-...`
4. 토큰을 안전하게 클립보드에 복사. **이 토큰은 ANTHROPIC_API_KEY가 아니라 OAuth 토큰**. 환경변수 이름은 런타임에서 `CLAUDE_CODE_OAUTH_TOKEN`으로 주입된다.

대안: Anthropic Console API 키 (`sk-ant-api03-...`) 도 같은 필드에 paste 가능 — Claude Code CLI가 둘 다 지원함. Console 키는 사용량 과금, OAuth 토큰은 구독 기반.

## Step 1B — Codex 토큰 발급 (운영자 로컬)

Codex 는 **두 가지 paste 형식**을 지원합니다 — 운영 환경에 맞는 쪽을 선택하세요.

### 옵션 1: OPENAI_API_KEY (단일 라인) — 권장

API key를 가진 사용자 / 사용량 과금 모델.

```bash
# 운영자 로컬 또는 OpenAI 대시보드에서:
echo "$OPENAI_API_KEY"
# sk-...
```

장점: 단순, 만료 없음 (수동 revoke 전까지 유효), 단일 라인 paste.

### 옵션 2: ~/.codex/auth.json (멀티 라인 JSON) — ChatGPT 구독

ChatGPT Plus/Pro 구독 사용자가 API 키 없이 인증할 때.

```bash
# 운영자 로컬 머신에서:
codex login
# 브라우저 OAuth 완료 후 — 전체 파일을 클립보드에 복사:
cat ~/.codex/auth.json
```

서버는 paste된 JSON 을 인식하면 컨테이너 내부 임시 디렉터리에 `auth.json` 으로 materialize 한 뒤 `CODEX_HOME=<tmpdir>` env 로 codex 자식 프로세스에 주입합니다. 매 호출마다 tempdir 이 새로 생성되어 호출 후 자동 삭제 — 호스트 파일시스템에는 잔재가 남지 않습니다.

> **만료**: ChatGPT 구독의 access_token 은 약 30일에 한 번 refresh 가 필요합니다. v0.1 에서는 자동 refresh 가 없으므로 만료 시 운영자가 다시 `codex login` 후 새 auth.json 을 paste 해야 합니다. UI 의 Settings → AI Module 페이지는 다음 Test 시도에서 auth 에러를 감지하면 노란 배너로 알림을 띄웁니다.

---

## Step 2 — Settings → AI Module 진입

1. SigNoz UI → 우측 상단 프로필 → **Settings**.
2. 좌측 메뉴 **AI Module** 클릭.
3. 현재 설정값과 `Last updated` 타임스탬프가 노출됨. (한 번도 저장 안 했다면 빈 폼.)

---

## Step 3 — 폼 입력

| 필드 | 값 |
|---|---|
| **Provider** | `LLM` |
| **LLM Provider** | `Claude` 또는 `Codex` |
| **Transport** | `CLI` |
| **Model** | (비워두면 패키지 기본값 — Claude: `claude-sonnet-4-6`, Codex: `gpt-5`) |
| **OAuth Token** | Step 1A 또는 1B 에서 복사한 토큰을 paste. Codex 의 경우 단일 라인 API key 또는 멀티 라인 auth.json JSON 모두 허용 — Settings UI 가 자동 인식 |
| **Binary path** | (비워두면 PATH에서 `claude` / `codex` 찾음) |
| **Timeout (seconds)** | (비워두면 15초 기본값) |

저장된 토큰은 **AES-GCM (AES-256) 으로 암호화**되어 `ds_ai_config.oauth_token_ciphertext` 컬럼에 저장. 응답에는 항상 `<unchanged>` 센티넬로 마스킹되므로 GET으로 평문이 노출되지 않음.

> 마스크 동작: 필드를 클릭(focus)하면 빈 상태로 클리어되고, 빈 상태로 Save하면 기존 토큰 유지. 새 값을 타이핑하고 Save해야 토큰이 갱신됨.

---

## Step 4 — Test

`Test` 버튼은 현재 폼 내용으로 throwaway generator 를 구성해 가상의 Payment incident 페이로드를 LLM에 던지고, 응답 headline + model을 반환합니다.

- 성공: `Connection OK` (또는 strategy headline) 토스트.
- 실패: 에러 메시지에서 원인 식별:
  - `claudecli: run claude: exec ... no such file or directory` → 컨테이너에 binary 미설치. Binary path 명시하거나 이미지 재빌드.
  - `stderr: Authentication error` → 토큰 잘못됨 또는 만료. Step 1 재실행.
  - `context deadline exceeded` → CLI가 느림. Timeout을 30–60초로 키워볼 것.

**Save 전에 Test가 통과해야 운영 안전.** Test에서 `<unchanged>` 센티넬은 자동으로 저장된 토큰으로 치환되므로, 이미 한 번 저장된 토큰을 새로 타이핑하지 않고 Test만 다시 돌리는 것도 가능.

---

## Step 5 — Save & 실제 알림에서 확인

1. `Save changes` → "AI module configuration saved" 토스트 → `Last updated` 타임스탬프 갱신.
2. AI Strategy generator는 다음 알림 dispatch 시점부터 새 설정으로 동작 (per-org cache가 무효화됨).
3. SOP 가 시드된 룰에 대해 알림을 트리거 → Slack 채널 또는 `/api/v2/ds/ai/strategies` 히스토리에서 LLM이 생성한 strategy가 들어왔는지 확인.

빠른 검증 (트리거 없이):
```bash
# 직전 N건의 AI strategy 히스토리 조회
curl -sS http://localhost:8080/api/v2/ds/ai/strategies?limit=5 \
  -H "Authorization: Bearer $JWT" | jq '.data[] | {createdAt,model,headline}'
```
새 결과의 `model` 필드가 `claude-sonnet-4-6` (또는 codex의 `gpt-5`) 로 들어오면 LLM 라우팅 OK.

---

## Troubleshooting

| 증상 | 원인 | 대응 |
|---|---|---|
| Save에서 400 `oauthToken: only allowed when provider="llm" and transport="cli"` | Transport가 `api`인데 OAuth 토큰이 폼에 남아있음 | Transport를 `cli`로 바꾸거나 OAuth 필드를 비우고 다시 Save |
| Test에서 `exec ... no such file or directory` | 이미지가 dynamic-linked binary로 빌드됨 (CGO_ENABLED=1) — alpine/musl과 비호환 | `make docker-build-community-amd64` 재실행 (Makefile에 CGO_ENABLED=0 박혀있음) |
| Test에서 `Authentication error` | 토큰 만료/오기재 | Step 1 재발급 후 paste. UI 상단에 노란 "Authentication issue detected" 배너가 노출되어 어느 secret 을 갱신해야 하는지 안내함 |
| Codex JSON paste 후 Test가 `oauthToken: looked like JSON but failed to parse` | `~/.codex/auth.json` 의 일부만 복사했거나 JSON 이 깨짐 | `cat ~/.codex/auth.json` 으로 전체 내용을 통째로 복사 — 첫 `{` 부터 마지막 `}` 까지 |
| Strategy 히스토리에 mock 응답이 계속 들어옴 | 다른 컴포넌트가 `DS_APM_AI_GENERATOR=mock` env를 우선시 중 | `.omc/demo/docker-compose.dsapm-override.yaml` 에서 `DS_APM_AI_GENERATOR` 제거 후 compose 재기동 |
| `Binary path` 가 비어있는데도 CLI 안 잡힘 | 컨테이너 PATH에 binary 없음 | `Binary path`에 절대경로 입력 (예: `/usr/local/bin/claude`) 또는 이미지 빌드 시 npm으로 CLI 설치 |
| 운영자 머신과 컨테이너의 CLI 버전 불일치로 토큰 거부 | 버전 mismatch | 양쪽 CLI 버전 핀 (예: `npm i -g @anthropic-ai/claude-code@1.x.y`) |

---

## 보안 메모

- 토큰은 AES-256-GCM 으로 컬럼 암호화. 마스터 키는 `DS_APM_AI_CONFIG_ENCRYPTION_KEY` 환경변수 (base64-encoded 32-byte). 미설정 시 plaintext 저장 fallback — **운영 환경에서는 반드시 설정**.
- GET 응답에서 토큰은 `<unchanged>` 센티넬로 항상 스크럽됨. UI는 마스크 상태를 ref로 추적하며, 빈 입력 + masked 상태일 때만 센티넬을 다시 전송.
- 토큰은 CLI 자식 프로세스의 env로만 흐른다 (`CLAUDE_CODE_OAUTH_TOKEN` / `OPENAI_API_KEY`). 로그/응답에 평문이 노출되는 경로는 현재 없음.
- 멀티 테넌트가 아닌 instance-wide 단일 토큰을 가정한다면, 같은 토큰을 모든 org config에 paste하는 운영도 가능 (단, 감사 추적이 어려워짐).

## 관련 문서

- 구현 플랜: `docs/superpowers/plans/2026-05-21-ai-cli-oauth-paste-token.md`
- 데모 전체 흐름: `docs/demo/2026-05-21-runbook.md`
- 백엔드 코드 진입점: `pkg/ruler/signozruler/ai_config_handler.go`, `pkg/ruler/aigenerator/aigenerator.go`
