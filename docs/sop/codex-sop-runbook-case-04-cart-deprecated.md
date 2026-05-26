# Case 04. Cart legacy 절차 폐기: deprecated runbook

## 상황

과거에는 cart-service pod를 직접 삭제하는 방식으로 latency를 완화했지만,
현재는 Redis cache flush 후 deployment rolling restart를 수행하는 절차가 더
안전하다. 기존 절차는 삭제하지 않고 `deprecated`로 내려 UI와 API 노출 방식을
확인한다.

## 사용하는 예시

| 항목 | 파일 |
|---|---|
| SOP | `docs/demo/sop_cart.json` |
| 권장 Runbook | `docs/demo/runbook_cart_redis.json` |
| Deprecated Runbook | `docs/sop/codex-sop-runbook-cart-legacy-deprecated.json` |
| SOP ID | `SOP-CART-001` |
| Version | `2026-05-20.1` |
| 기대 status | `deprecated` |

## 절차

```bash
curl -fsS -X POST "$SIGNOZ_URL/api/v2/ds/sop/documents" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/demo/sop_cart.json

curl -fsS -X POST \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-CART-001/versions/2026-05-20.1/runbooks" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/demo/runbook_cart_redis.json

curl -fsS -X POST \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-CART-001/versions/2026-05-20.1/runbooks" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/sop/codex-sop-runbook-cart-legacy-deprecated.json
```

## 기대 결과

- 기본 목록에는 권장 runbook만 보인다.
- deprecated 목록에는 legacy runbook이 보인다.
- legacy runbook 설명에 대체 절차가 명시되어 있다.

## 검증

기본 목록:

```bash
curl -fsS \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-CART-001/versions/2026-05-20.1/runbooks" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.runbooks[] | {title,status}'
```

Deprecated 목록:

```bash
curl -fsS \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-CART-001/versions/2026-05-20.1/runbooks?status=deprecated" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.runbooks[] | {title,status}'
```

## 실패로 봐야 하는 신호

- deprecated runbook이 기본 목록에 섞여 나온다.
- legacy runbook에 대체 절차가 없다.
- hard delete 없이 상태만 바꾸는 흐름을 검증할 수 없다.
