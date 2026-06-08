# Case 06. LLM 설정 없음 또는 인증 실패

## 상황

운영자가 AI draft preview를 실행했지만 LLM provider 설정이 없거나 token/API key가
잘못되어 LLM 호출이 실패한다. 이 경우 UI가 일반 HTTP 실패처럼 터지면 안 되고,
사용자가 설정 문제를 이해할 수 있는 typed envelope를 받아야 한다.

## 사용하는 예시

| 항목 | 파일 |
|---|---|
| SOP | `docs/demo/sop_cart.json` |
| Draft request | `docs/sop/codex-sop-runbook-cart-redis-draft-request.json` |
| 관련 endpoint | `POST /api/v2/ds/runbooks/draft` |
| 기대 동작 | HTTP 200 + `ok:false` + `errorKind` |

## 절차

LLM 설정이 없거나 잘못된 상태를 만든 뒤 preview를 호출한다. 예를 들어
provider token을 비우거나 만료된 token을 설정한 뒤 실행한다.

```bash
curl -fsS -X POST "$SIGNOZ_URL/api/v2/ds/runbooks/draft" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/sop/codex-sop-runbook-cart-redis-draft-request.json | jq '.'
```

## 기대 결과

- HTTP status는 200이다.
- body는 성공 runbook이 아니라 실패 envelope다.
- `ok`는 `false`다.
- `errorKind`는 `auth`, `timeout`, `other` 중 하나다.
- 실패한 draft는 부모 SOP에 저장되지 않는다.

## 검증

```bash
curl -fsS -X POST "$SIGNOZ_URL/api/v2/ds/runbooks/draft" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/sop/codex-sop-runbook-cart-redis-draft-request.json \
  | jq '{ok,error,errorKind}'
```

기대 형태:

```json
{
  "ok": false,
  "error": "...",
  "errorKind": "auth"
}
```

## 실패로 봐야 하는 신호

- LLM 실패가 500 HTML 또는 untyped text로 반환된다.
- `ok:false` 없이 성공 runbook처럼 보인다.
- 실패했는데 runbook 목록에 draft가 저장된다.

