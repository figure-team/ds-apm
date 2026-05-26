# Case 01. Payment 5xx 대응: 승인 runbook 생성

## 상황

Payment API 5xx 비율이 높아지고 payment-service pod 재시작으로 즉시 완화할 수
있는 상황이다. 운영자는 이미 검증된 수동 runbook을 SOP에 붙이고, incident
중에는 script를 복사해서 직접 실행한다.

## 사용하는 예시

| 항목 | 파일 |
|---|---|
| SOP | `docs/demo/sop_pay.json` |
| Runbook | `docs/demo/runbook_pay_restart.json` |
| SOP ID | `SOP-PAY-001` |
| Version | `2026-05-12.1` |
| 기대 status | `approved` |

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
  -d @docs/demo/runbook_pay_restart.json
```

## 기대 결과

- `Rolling restart payment-service` runbook이 생성된다.
- 기본 runbook 목록 조회에서 보인다. 기본 조회는 `approved,draft`를 대상으로 한다.
- UI에서는 SOP row를 펼쳤을 때 Runbooks 영역에 표시된다.
- 운영자가 script를 복사할 수 있어야 한다.

## 검증

```bash
curl -fsS \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-PAY-001/versions/2026-05-12.1/runbooks" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.runbooks[] | {title,status,updatedBy}'
```

기대값:

```json
{
  "title": "Rolling restart payment-service",
  "status": "approved",
  "updatedBy": "demo-seed"
}
```

## 실패로 봐야 하는 신호

- runbook이 생성됐지만 `status`가 `approved`가 아니다.
- 목록 응답에 runbook이 없거나 다른 SOP version에 붙었다.
- script가 비어 있거나 `kubectl rollout status` 확인 절차가 빠져 있다.

