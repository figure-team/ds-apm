# Case 08. 부모 SOP version 없음

## 상황

운영자가 존재하지 않는 SOP ID 또는 version에 runbook을 붙이려고 한다.
Runbook은 독립 리소스가 아니라 부모 SOP version payload에 embedded로 저장되므로,
부모가 없으면 생성되면 안 된다.

## 사용하는 예시

| 항목 | 파일 |
|---|---|
| Runbook | `docs/sop/codex-sop-runbook-ad-gc-approved.json` |
| 존재하지 않는 SOP ID | `SOP-NOT-FOUND` |
| 존재하지 않는 version | `2099-01-01.1` |
| 기대 동작 | 404 not found |

## 절차

```bash
curl -i -sS -X POST \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-NOT-FOUND/versions/2099-01-01.1/runbooks" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/sop/codex-sop-runbook-ad-gc-approved.json
```

## 기대 결과

- HTTP status는 404다.
- runbook은 어떤 SOP에도 저장되지 않는다.
- 같은 payload를 올바른 SOP/version에 보내면 생성 가능해야 한다.

## 검증

```bash
curl -fsS \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-NOT-FOUND/versions/2099-01-01.1/runbooks" \
  -H "Authorization: Bearer $TOKEN"
```

기대: not found 응답.

## 실패로 봐야 하는 신호

- 부모가 없는데 runbook 생성이 성공한다.
- 빈 SOP 문서가 암묵적으로 만들어진다.
- 에러가 500으로 뭉개져서 사용자에게 원인을 알려주지 못한다.

