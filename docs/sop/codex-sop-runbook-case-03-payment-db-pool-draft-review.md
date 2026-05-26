# Case 03. Payment DB pool 고갈 대응: draft runbook 검토

## 상황

Payment API 5xx 원인이 DB connection pool 고갈로 의심된다. AI가 만든 절차를
곧바로 승인하지 않고 `draft` 상태로 저장해 운영자가 script와 설명을 검토한다.

## 사용하는 예시

| 항목 | 파일 |
|---|---|
| SOP | `docs/demo/sop_pay.json` |
| Runbook | `docs/sop/codex-sop-runbook-pay-db-pool-draft.json` |
| SOP ID | `SOP-PAY-001` |
| Version | `2026-05-12.1` |
| 기대 status | `draft` |

## 절차

```bash
curl -fsS -X POST "$SIGNOZ_URL/api/v2/ds/sop/documents" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/demo/sop_pay.json

curl -fsS -X POST \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-PAY-001/versions/2026-05-12.1/runbooks" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/sop/codex-sop-runbook-pay-db-pool-draft.json
```

## 기대 결과

- `Payment DB connection pool 고갈 완화` runbook이 `draft` 상태로 생성된다.
- `aiDraftedBy`가 비어 있지 않아 AI 작성 후보임을 알 수 있다.
- `sourceErrorExamples`가 최대 3개까지 보존된다.
- UI에서는 draft 탭 또는 기본 목록에서 검토 대상으로 보인다.

## 검증

```bash
curl -fsS \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-PAY-001/versions/2026-05-12.1/runbooks?status=draft" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.data.runbooks[] | {title,status,aiDraftedBy,sourceErrorExamples}'
```

## 승인 전 체크리스트

- script가 idempotent한가?
- DB connection 수를 직접 변경하지 않고 완화 조치만 수행하는가?
- log 확인과 rollout status 확인이 포함되어 있는가?
- 운영자가 실행하면 안 되는 destructive command가 없는가?

## 실패로 봐야 하는 신호

- AI draft가 생성 즉시 `approved` 상태가 된다.
- `sourceErrorExamples`가 누락된다.
- script 안에 secret 값이나 환경별 credential이 박혀 있다.
