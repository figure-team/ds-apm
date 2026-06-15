# CF-2 (AI 1차 분석 초안) — 셋업 & 온보딩 가이드

> 출처: 2026-06-15 세션. 실제 SigNoz backoffice + codex로 CF-2를 처음부터 셋업하고 슬랙까지 풀 초안을 내보내며 부딪힌 모든 지점을 정리한 것.
>
> **이 문서의 가치**: CF-2는 사용자가 **직접 셋업해야 동작**한다(AI provider, SOP, 채널, 룰). 그 셋업 여정에는 *비자명한 함정*이 많다 — 이 문서는 그 함정과 해법의 집약이며, 향후 **가이드 투어(driver.js)나 AI 온보딩**을 붙일 때의 1차 자료다. 각 단계 끝의 `🧭 온보딩 포인트`가 투어 후보 지점.

---

## 0. CF-2가 뭐고, 어디에 노출되나

알람 발생 시 **SOP + 알람 컨텍스트 + 증거**로 LLM이 사고대응 1차 초안을 생성: 헤드라인·가설·초동조치·**고객 공지 초안·벤더(PG) 요청 초안**·한계. HITL — 자동 실행/발송이 아니라 담당자가 다듬는 출발점.

| 노출 면 | 경로 | 용도 |
|---|---|---|
| AI 모듈 설정 | `/settings/ai-module` | provider/모델 설정 + **테스트 버튼**(초안 1건 생성→headline) |
| 디스패치(실사용) | 알람 firing → dispatcher → **AI 훅** → 알람 annotations 병합 → 알림 채널(슬랙 등) | 풀 초안이 알림에 실려 나감 |
| Preview | `POST /api/v2/ds/ai/strategy/preview` | SOP+알람+증거 넣으면 풀 전략 반환 + history 저장 (FE 미배선) |
| History | `GET /api/v2/ds/ai/strategy/history/latest` | 마지막 생성 초안 조회 |

---

## 1. 사용자 셋업 여정 (순서대로)

### 1-1. AI provider 설정 — `/settings/ai-module`
provider=`llm`, llmProvider=`claude`|`codex`, transport=`api`|`cli`, model, (api면 apiKey / cli면 OAuth는 로컬 인증 상속).

> ⚠️ **함정 (codex)**: ChatGPT 구독 계정 codex는 `gpt-5`·`gpt-5-codex`를 **거부**한다(`"model is not supported when using Codex with a ChatGPT account"`). `~/.codex/config.toml`의 `model` 값(예: **`gpt-5.5`**)을 써야 한다. 모델명 틀리면 생성이 즉시 실패.
> ⚠️ **함정 (cli transport)**: cli는 서버 호스트의 로컬 CLI 인증(`~/.codex`, `claude` 로그인)을 상속한다. 컨테이너 안에서 돌리면 인증이 없어 실패 → 호스트 실행 또는 인증 마운트 필요.

🧭 **온보딩 포인트**: provider별로 요구하는 게 다름(api=키, cli=로컬 인증). 모델명 유효성은 provider/계정 종류에 의존 → 투어에서 "테스트 먼저 눌러 검증" 유도.

### 1-2. 테스트 버튼으로 검증
설정 후 **테스트** 클릭 → 백엔드가 `cannedPaymentRequest`(SOP+evidence 내장 결제 시나리오)로 초안 생성 → 성공 시 **headline 토스트**(검정=성공, 빨강=실패).

> ⚠️ **함정**: canned 시나리오에 SOP/evidence가 없으면 실 LLM의 "ready" 초안이 검증(sopId·evidence·인용 필수)을 통과 못 해 항상 실패한다 → 오늘 `cannedPaymentRequest`에 SOP+evidence를 넣어 해결(`ai_config_handler.go`).

🧭 **온보딩 포인트**: "검정 토스트 = 성공"이 직관적이지 않음(사용자가 에러로 오해) → 색/문구 개선 또는 투어 설명 필요.

### 1-3. SOP 등록 (디스패치에 필수)
디스패치 훅은 알람에 **SOP를 바인딩**해야 생성한다. SOP가 없으면(또는 안 바인딩되면) 초안 없이 알람만 나간다(또는 CF-11 코드RCA로 분기).

SOP가 **바인딩되려면** (`PreviewSOPDocumentBinding` 기준):
- `Source.SourceID` **필수** (없으면 바인딩 응답 검증 실패 → 훅이 조용히 스킵). ← 오늘 가장 헷갈린 함정
- `ApprovalStatus` ≠ `disabled` (draft/approved/deprecated는 OK)
- **테넌트 정책**: 알람 라벨에 `project_id`+`environment`가 있어야 하고, SOP `TenantScope.ProjectIDs/Environments`가 그 값을 포함(또는 `"*"`)
- 바인딩 방식: 알람 라벨 `sop_id=<SOPID>`(명시적) **또는** 라벨조합(service.name/severity/team) 매칭

🧭 **온보딩 포인트**: "바인딩 안 됨"이 **무증상**(훅이 조용히 원본 annotations 반환)이라 디버깅이 어렵다. `sourceId 누락`·`tenant 라벨 누락`·`approval=disabled` 3대 사유 → 바인딩 미리보기 UI(`/ds/sop/bindings/preview`)로 진단 유도가 핵심 온보딩 가치.

### 1-4. 알림 채널 + 알람 룰 (실제 "나가게" 하려면)
- 채널: `POST /api/v1/channels` (Slack=Incoming Webhook URL `https://hooks.slack.com/services/...`)
- 룰: `POST /api/v2/rules` — payment 라벨 + `sop_id` + 채널 라우팅
- **AI 훅은 "firing 알람이 디스패치될 때만" 발화**한다. rule **테스트 알림(TestAlert)은 훅을 안 탄다** → 풀 초안을 보려면 진짜 firing 필요.

🧭 **온보딩 포인트**: "테스트 알림으론 AI 초안이 안 보인다"는 매우 비자명 → 명시 안내 필수.

---

## 2. 두 생성 경로의 차이 (중요)

| | Preview / 테스트 | 디스패치(실알람) |
|---|---|---|
| 증거(evidence) | 요청에 실림 → **풀 그라운딩** 초안(가설·조치·고객/벤더 초안) | **v0.1: 안 실림**(`hook.go` "no evidence collector yet") → low_confidence, 내용 빈약 |
| 트리거 | 버튼/ API | firing 알람 |
| 상태 | ready 가능 | 보통 low_confidence |

→ **알려진 갭**: 디스패치 경로 evidence 미전달 = v0.2 과제(EvidenceCollector 배선). 현재 실알람 초안이 빈약한 근본 원인.

---

## 3. 설정 노브 (env)

| env | 의미 |
|---|---|
| `DS_APM_AI_GENERATOR` | `llm`이면 LLM 사용(아니면 local 결정론적) |
| `DS_APM_LLM_PROVIDER` / `_TRANSPORT` / `_MODEL` | claude\|codex / api\|cli / 모델명 |
| `DS_APM_LLM_TIMEOUT_SECONDS` | 생성기 per-call 타임아웃(느린 LLM은 크게) |
| `DS_APM_AI_DISPATCH_TIMEOUT_SECONDS` | **훅 타임아웃**(기본 5초 — 느린 LLM은 디스패치에서 타임아웃하니 키워야. 오늘 추가한 노브) |
| `DS_APM_AI_CONFIG_ENCRYPTION_KEY` | per-org AI config의 apiKey 암호화(미설정 시 평문 경고) |

> ⚠️ 기본 디스패치 타임아웃 5초는 **로컬/빠른 생성기 기준**이다. codex(~14초)는 그 안에 못 끝나 best-effort로 스킵된다 — 이건 "알람 지연 방지" 의도된 설계. 느린 LLM을 디스패치에 쓰려면 노브를 키우거나 **비동기 생성**(발송 후 incident에 후첨)으로 가야 함.

---

## 4. 검증 규칙 (초안이 "유효"하려면)

`ValidateAIStrategy` — status=`ready`일 때: `sopId`·`sopVersion`·hypotheses≥1·firstActions≥1·evidenceRefs≥1 필수. **항상**: 각 hypothesis는 evidenceRefs/sopStepRefs 중 하나 인용, 각 firstAction은 sopStepRef/evidenceRefs 중 하나 인용 + `requiresHumanApproval=true`. non-ready면 `limitations` 필수.

> ⚠️ **함정 (codex 비결정성)**: LLM이 이 스키마를 매번 못 지킨다(오늘 codex 1/3만 통과) → 오늘 생성기에 **검증 실패 시 재시도**(maxParseAttempts=3) 추가 → 3/3 통과.

---

## 5. 고객 공지 초안 = 공지문 양식 (한국 시장)

줄글 ❌. 제목 + `■ 라벨: 내용` 섹션 양식:
```
[결제 서비스 이용 장애 안내]

■ 발생 현황: …
■ 영향 범위: …
■ 조치 사항: …
■ 향후 안내: …
■ 문의처: …
```
가드: 장애 원인 단정 / 배상·법적책임 / 확정 ETA **금지**, 근거 부족 항목은 "확인 중". (프롬프트에 강제)

> 추후(B): SOP에 **comms 템플릿** 섹션을 얹어 LLM이 슬롯만 채우게 → 조직 승인 문구로 통제. 별도 설계 노트: `specs/2026-06-15-cf2-comms-template-grounding-design.md`.

---

## 6. 오늘 한 코드 변경 (참고)

| 파일 | 변경 |
|---|---|
| `llmaigenerator/llm.go` | 검증 실패 재시도(maxParseAttempts) |
| `llmaigenerator/prompt.go` | 인용 의무 + 고객공지 **공지문 양식**·가드 |
| `signozruler/ai_config_handler.go` | 테스트 canned 시나리오에 SOP+evidence |
| `signoz.go` | `DS_APM_AI_DISPATCH_TIMEOUT_SECONDS` 노브 |
| (참고) `coderca/*` | CF-11 에이전트별 read-only 툴링(포트/어댑터) — codex가 셸로 코드 읽게 |

---

## 7. 온보딩(driver.js / AI) 설계용 요약

**사용자가 직접 해야 하고, 막히기 쉬운 순서**:
1. AI provider 설정 → **모델명 유효성**(codex=gpt-5.5)에서 막힘
2. 테스트 버튼 → **검정=성공** 오해
3. SOP 등록 → **sourceId/tenant라벨/approval** 3대 무증상 바인딩 실패
4. 채널+룰 → **테스트 알림은 훅 안 탐**, firing 필요
5. 실알람 초안 빈약 → **evidence 갭(v0.1)** 인지

각 항목이 투어 스텝 + 인라인 진단(바인딩 프리뷰, 테스트 결과 해설) 후보. 무증상 실패(특히 3)를 **사전 진단/체크리스트**로 바꾸는 게 온보딩의 최대 가치.
