# Case 10. 상태 전이 충돌

## 상황

Runbook status update가 허용되지 않는 전이를 시도한다. 서버는 같은 status로의
no-op update와 `deprecated -> approved` 직접 전환을 막아야 한다.

## 사용하는 예시

| 항목 | 파일 |
|---|---|
| Deprecated Runbook | `docs/sop/codex-sop-runbook-cart-legacy-deprecated.json` |
| Parent SOP | `SOP-CART-001` / `2026-05-20.1` |
| 기대 동작 | invalid status transition 거절 |

## 절차

먼저 deprecated runbook을 만든다.

```bash
curl -fsS -X POST "$SIGNOZ_URL/api/v2/ds/sop/documents" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/demo/sop_cart.json

created="$(
  curl -fsS -X POST \
    "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-CART-001/versions/2026-05-20.1/runbooks" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $TOKEN" \
    -d @docs/sop/codex-sop-runbook-cart-legacy-deprecated.json
)"

runbook_id="$(printf '%s' "$created" | jq -r '.data.id // .id')"
```

`deprecated -> approved` 직접 전환을 시도한다.

```bash
tmp="$(mktemp)"
jq '.status = "approved"' docs/sop/codex-sop-runbook-cart-legacy-deprecated.json > "$tmp"

curl -i -sS -X PUT \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-CART-001/versions/2026-05-20.1/runbooks/$runbook_id" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @"$tmp"
```

## 기대 결과

- 직접 전환은 거절된다.
- runbook은 계속 `deprecated` 상태로 남는다.
- 다시 살리려면 `deprecated -> draft -> approved` 순서를 거쳐야 한다.

## 검증

```bash
curl -fsS \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-CART-001/versions/2026-05-20.1/runbooks?status=deprecated" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.runbooks[] | {id,title,status}'
```

## 실패로 봐야 하는 신호

- `deprecated -> approved`가 바로 성공한다.
- 같은 status update가 성공해서 client bug를 숨긴다.
- 실패 후에도 runbook 상태가 부분 변경된다.

