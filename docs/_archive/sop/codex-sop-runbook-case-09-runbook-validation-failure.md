# Case 09. Runbook payload validation 실패

## 상황

Runbook 생성 요청 payload 자체가 잘못됐다. 시스템은 저장 전에 domain validation을
실행해서 title, status, confidence, script 제약을 지켜야 한다.

## 사용하는 예시

| 항목 | 파일 |
|---|---|
| Invalid runbook | `docs/sop/codex-sop-runbook-invalid-payload.json` |
| Parent SOP | `SOP-PAY-001` / `2026-05-12.1` |
| 기대 동작 | 4xx invalid input, 저장 없음 |

## 절차

```bash
curl -fsS -X POST "$SIGNOZ_URL/api/v2/ds/sop/documents" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/demo/sop_pay.json

curl -i -sS -X POST \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-PAY-001/versions/2026-05-12.1/runbooks" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/sop/codex-sop-runbook-invalid-payload.json
```

## 기대 결과

- 요청은 invalid input으로 거절된다.
- 비어 있는 title은 저장되지 않는다.
- `status=running` 같은 알 수 없는 status는 저장되지 않는다.
- `confidence=1.2`처럼 범위를 벗어난 값은 저장되지 않는다.

## 검증

```bash
curl -fsS \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-PAY-001/versions/2026-05-12.1/runbooks?status=all" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.runbooks'
```

기대: invalid payload의 title 또는 script가 목록에 없어야 한다.

## 실패로 봐야 하는 신호

- 빈 title runbook이 생성된다.
- unknown status가 저장된다.
- confidence 범위 밖 값이 그대로 저장된다.

