# Case 02. Cart Redis latency 대응: AI draft preview

## 상황

Cart service p99 latency가 급증했고 Redis GETEX timeout과 hit ratio 하락이
같이 보인다. 운영자는 error example을 붙여넣어 AI가 runbook 후보를 만들 수
있는지 확인한다. 이 단계는 preview이므로 저장하지 않는다.

## 사용하는 예시

| 항목 | 파일 |
|---|---|
| SOP | `docs/demo/sop_cart.json` |
| Draft request | `docs/sop/codex-sop-runbook-cart-redis-draft-request.json` |
| SOP ID | `SOP-CART-001` |
| Version | `2026-05-20.1` |
| 기대 동작 | 저장 없이 후보 runbook 반환 |

## 절차

```bash
curl -fsS -X POST "$SIGNOZ_URL/api/v2/ds/sop/documents" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/demo/sop_cart.json

curl -fsS -X POST "$SIGNOZ_URL/api/v2/ds/runbooks/draft" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/sop/codex-sop-runbook-cart-redis-draft-request.json | jq '.'
```

## 기대 결과

- 응답은 runbook 후보를 반환한다.
- 후보 runbook의 `status`는 `draft`다.
- 후보에는 `title`, `description`, `executableScript`, `confidence`,
  `sourceErrorExamples`가 포함된다.
- 이 호출만으로 부모 SOP의 runbooks 목록에는 저장되지 않는다.

## 검증

Preview 응답 확인:

```bash
curl -fsS -X POST "$SIGNOZ_URL/api/v2/ds/runbooks/draft" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/sop/codex-sop-runbook-cart-redis-draft-request.json \
  | jq '.data // .'
```

저장되지 않았는지 확인:

```bash
curl -fsS \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-CART-001/versions/2026-05-20.1/runbooks?status=draft" \
  -H "Authorization: Bearer $TOKEN" | jq '.data.runbooks'
```

## 실패로 봐야 하는 신호

- preview 호출 뒤 목록 조회에 새 runbook이 자동 저장되어 있다.
- `errorExamples`가 응답의 `sourceErrorExamples`에 반영되지 않는다.
- auth 또는 timeout 오류가 일반 성공 runbook처럼 보인다.
