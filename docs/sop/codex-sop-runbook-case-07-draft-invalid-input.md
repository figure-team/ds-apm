# Case 07. Draft 요청 입력 오류

## 상황

AI draft preview 요청 payload가 잘못됐다. 시스템은 LLM 호출 전에 필수값과
error example 개수를 검증해야 한다.

## 사용하는 예시

| 항목 | 파일 |
|---|---|
| Version 누락 요청 | `docs/sop/codex-sop-runbook-invalid-draft-missing-version.json` |
| Error example 초과 요청 | `docs/sop/codex-sop-runbook-invalid-draft-too-many-examples.json` |
| 관련 endpoint | `POST /api/v2/ds/runbooks/draft` |
| 기대 동작 | LLM 호출 전 4xx invalid input |

## 절차

Version 누락:

```bash
curl -i -sS -X POST "$SIGNOZ_URL/api/v2/ds/runbooks/draft" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/sop/codex-sop-runbook-invalid-draft-missing-version.json
```

Error example 4개:

```bash
curl -i -sS -X POST "$SIGNOZ_URL/api/v2/ds/runbooks/draft" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d @docs/sop/codex-sop-runbook-invalid-draft-too-many-examples.json
```

## 기대 결과

- `sopId` 또는 `version`이 비면 invalid input으로 거절된다.
- `errorExamples`가 0개이거나 3개를 초과하면 invalid input으로 거절된다.
- LLM provider를 호출하지 않는다.
- 부모 SOP에 아무 runbook도 저장하지 않는다.

## 검증

응답 body에 다음 의미가 들어 있어야 한다.

```text
sopId and version required
errorExamples: at most 3 entries
```

## 실패로 봐야 하는 신호

- 입력이 틀렸는데 LLM 호출까지 진행된다.
- 4개 이상의 `sourceErrorExamples`가 저장된다.
- 누락된 version이 빈 문자열 version으로 저장된다.

