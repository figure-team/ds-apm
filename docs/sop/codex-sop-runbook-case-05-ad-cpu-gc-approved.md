# Case 05. Ad CPU / GC pressure 대응: 승인 runbook 추가

## 상황

adservice CPU가 높고 JVM GC pressure가 함께 관측된다. 운영자는 log 확인,
manual GC mitigation flag, rolling restart를 포함한 승인 runbook을 AD SOP에
연결한다.

## 사용하는 예시

| 항목 | 파일 |
|---|---|
| SOP | `docs/demo/sop_ad.json` |
| Runbook | `docs/sop/codex-sop-runbook-ad-gc-approved.json` |
| SOP ID | `SOP-AD-001` |
| Version | `2026-05-20.1` |
| 기대 status | `approved` |

## 절차

```bash
curl -fsS -X POST "$SIGNOZ_URL/api/v2/ds/sop/documents" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/demo/sop_ad.json

curl -fsS -X POST \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-AD-001/versions/2026-05-20.1/runbooks" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/sop/codex-sop-runbook-ad-gc-approved.json
```

## 기대 결과

- `Reduce adservice JVM GC pressure` runbook이 `approved` 상태로 생성된다.
- description은 한국어 운영 절차로 보인다.
- script는 log 확인, mitigation flag 시도, rollout restart, rollout status 확인을 포함한다.

## 검증

```bash
curl -fsS \
  "$SIGNOZ_URL/api/v2/ds/sop/documents/SOP-AD-001/versions/2026-05-20.1/runbooks?status=approved" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.data.runbooks[] | {title,status,description,executableScript}'
```

## 실패로 봐야 하는 신호

- AD SOP version이 아닌 다른 version에 붙는다.
- description이 운영자가 읽을 수 없는 placeholder 수준이다.
- script가 상태 확인 없이 restart만 수행한다.
