# SOP 등록 → Slack 알림 자동화 테스트 설계

**날짜:** 2026-05-31  
**범위:** Option A — 두 개의 독립적인 단위 테스트  
**목표:** SOP 문서 등록(HTTP 핸들러)과 등록된 SOP의 `sopId`를 레이블로 가진 알림이 실제 Slack webhook으로 전송되는 것을 자동화 테스트로 검증한다.

---

## 배경

DS-APM 프로젝트는 SOP(Standard Operating Procedure) 문서를 등록하고, 해당 SOP의 `sopId`를 alert 레이블(`sop_id`)로 연결하여 Slack 등 채널로 알림을 보내는 흐름을 제공한다.

현재 커버리지:
- SOP 핸들러 단위 테스트 일부 존재 (`handler_test.go`)
- Alertmanager E2E 테스트 존재 (`server_e2e_test.go`) — Slack 채널 없음
- `sop_id` 레이블을 포함한 Slack 알림 전송 테스트 없음

---

## 테스트 범위 (두 개의 독립 함수)

### Test 1 — SOP 등록 핸들러

| 항목 | 내용 |
|---|---|
| 파일 | `pkg/ruler/signozruler/handler_test.go` |
| 함수명 | `TestCreateSOPDocument_PaymentSOPPayload` |
| 레이어 | HTTP 핸들러 (단위 테스트) |
| 스토어 | `memSOPStore` (파일 내 이미 존재) |
| 인증 | `withSOPTestClaims()` 사용 |

**흐름:**
1. `ruletypes.SOPDocument` 구조체를 직접 구성 (`docs/demo/sop_pay.json` 값 기준, 파일 런타임 읽기 아님)
2. `h.CreateSOPDocument(rw, req)` 호출
3. 검증:
   - `HTTP 201 Created`
   - `response.data.sopId == "SOP-PAY-001"`
   - `response.data.version == "2026-05-12.1"`
   - `response.data.approvalStatus == "approved"`
4. `memSOPStore.Get(ctx, orgID, "SOP-PAY-001", "2026-05-12.1")` 호출 → 에러 없음 확인

**기존 패턴 참조:** `TestSOPDocumentHandlersCreateListGetFetchAndBind` (동일 패턴, 실제 demo payload 사용이 차이)

---

### Test 2 — Slack 알림 (sop_id 레이블 포함)

| 항목 | 내용 |
|---|---|
| 파일 | `pkg/alertmanager/alertmanagernotify/slack/slack_test.go` |
| 함수명 | `TestSlackNotifier_SOPAlertLabel` |
| 레이어 | Slack `Notifier.Notify()` (단위 테스트) |
| Slack URL | `os.Getenv("TEST_SLACK_WEBHOOK_URL")` — 없으면 `t.Skip()` |

**흐름:**
1. `TEST_SLACK_WEBHOOK_URL` 환경변수 읽기 → 없으면 `t.Skip("TEST_SLACK_WEBHOOK_URL not set")`
2. `config.SlackConfig{APIURL: webhookURL}` 로 `slack.New()` 호출
3. Alert 구성:
   ```
   alertname:   "PaymentAPI5xx"
   sop_id:      "SOP-PAY-001"
   severity:    "critical"
   environment: "prod"
   ```
4. `notifier.Notify(ctx, alert)` 호출
5. 검증: `err == nil` (Slack으로부터 200 OK 수신)

**Slack 메시지 확인:** template text에 `sop_id` 값을 포함시켜 실제 채널에서 육안 확인 가능.

**기존 패턴 참조:** `TestSlackRedactedURL` — `config.SlackConfig{APIURL: ...}` 패턴 동일

---

## 연결 관계

두 테스트는 독립적이지만 다음 인과 관계를 함께 증명한다:

```
[Test 1] SOP 등록 → sopId: SOP-PAY-001 저장됨
[Test 2] sopId: SOP-PAY-001 레이블의 alert → Slack 전송 성공
```

---

## 환경변수 설정 방법

```bash
# settings/channels에 등록된 Slack webhook URL을 사용
export TEST_SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
go test ./pkg/alertmanager/alertmanagernotify/slack/... -run TestSlackNotifier_SOPAlertLabel -v
```

Test 1은 환경변수 없이 실행 가능:
```bash
go test ./pkg/ruler/signozruler/... -run TestCreateSOPDocument_PaymentSOPPayload -v
```

---

## 비범위 (이번 테스트에서 다루지 않음)

- SOP 등록 → alertmanager 라우팅 → Slack까지의 단일 파이프라인 통합 테스트 (Option B)
- runbook 생성/조회 흐름
- 가짜(mock) Slack 서버를 이용한 payload 내용 검증
- CI 자동 실행 (환경변수 필요로 인해 선택적 실행)

---

## 성공 기준

| 검증 항목 | 방법 |
|---|---|
| SOP 등록 시 201 + 올바른 sopId 반환 | `require.Equal` |
| 등록 후 store에 실제 저장됨 | `store.Get()` 에러 없음 |
| sop_id 레이블 alert → Slack 전송 | `notifier.Notify()` 에러 없음 |
