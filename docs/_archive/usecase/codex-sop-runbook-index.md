# Codex SOP / Runbook 유스케이스

이 문서는 DS-APM의 SOP / Runbook 기능을 **사용자 행위 기준**으로 정리한
유스케이스 인덱스다. 장애별 상세 대응 절차와 curl 실행 예시는 유스케이스가
아니라 SOP/runbook 예시이므로 `docs/sop/` 아래로 분리했다.

## 범위

여기서 유스케이스는 “운영자 또는 뷰어가 시스템으로 달성하려는 목표”만 다룬다.

- 포함: SOP 등록, runbook 등록, 등록되지 않은 에러 패턴 처리, AI 초안 생성,
  초안 검토/승인, 폐기, 조회
- 제외: Payment 5xx를 어떻게 조치할지, Redis cache를 어떻게 flush할지 같은
  장애별 상세 절차

상세 SOP/runbook 예시는 `docs/sop/codex-sop-runbook-case-*.md`를 참고한다.

## 액터

| 액터 | 설명 |
|---|---|
| 운영자 / SRE | SOP와 runbook을 만들고 수정한다. AI draft를 검토하고 승인한다. |
| 뷰어 | 등록된 SOP와 runbook을 조회한다. 실행 가능한 script를 직접 변경하지 않는다. |
| LLM Provider | error example과 SOP context를 받아 runbook 초안을 생성한다. |

## 유스케이스 목록

| ID | 유스케이스 | 주 액터 | 목표 | 성공 기준 |
|---|---|---|---|---|
| UC-01 | SOP 문서를 등록한다 | 운영자 / SRE | 서비스 장애 패턴에 연결할 SOP를 시스템에 저장한다 | SOP ID와 version으로 조회 가능하다 |
| UC-02 | SOP 문서를 조회한다 | 운영자 / SRE, 뷰어 | incident 중 관련 SOP를 찾는다 | SOP 목록/상세/특정 version을 읽을 수 있다 |
| UC-03 | Runbook을 등록한다 | 운영자 / SRE | SOP version에 실행 가능한 대응 절차를 붙인다 | runbook이 부모 SOP version의 목록에 표시된다 |
| UC-04 | Runbook AI 초안을 생성한다 | 운영자 / SRE | error example으로 runbook 후보를 만든다 | 초안이 반환되지만 자동 저장되지는 않는다 |
| UC-05 | Draft runbook을 검토하고 승인한다 | 운영자 / SRE | AI 또는 사람이 만든 draft를 운영 가능한 절차로 확정한다 | status가 `draft`에서 `approved`로 전환된다 |
| UC-06 | Runbook을 폐기한다 | 운영자 / SRE | 더 이상 쓰지 않는 절차를 기본 목록에서 숨긴다 | status가 `deprecated`로 전환되고 deprecated 조회에서만 보인다 |
| UC-07 | Status별 runbook을 조회한다 | 운영자 / SRE, 뷰어 | 승인/초안/폐기 상태를 구분해 본다 | `status` query로 필요한 목록만 조회된다 |
| UC-08 | 잘못된 요청을 거절받는다 | 운영자 / SRE | 누락된 입력이나 잘못된 payload를 저장 전에 확인한다 | 시스템이 4xx 또는 typed failure envelope로 원인을 반환한다 |
| UC-09 | 등록되지 않은 에러 패턴을 처리한다 | 운영자 / SRE | 기존 runbook이 없는 에러를 임시 대응하고 지식화한다 | 관련 SOP를 찾거나 초안을 만든 뒤 draft로 남긴다 |

## 유스케이스 상세

### UC-01. SOP 문서를 등록한다

운영자는 서비스별 장애 패턴을 설명하는 SOP 문서를 등록한다.

입력:

- `sopId`
- `version`
- `title`
- `bodyMarkdown`
- `tenantScope`
- `approvalStatus`

대표 endpoint:

```text
POST /api/v2/ds/sop/documents
```

성공 후:

- 같은 `sopId`와 `version`으로 조회된다.
- alert label 또는 AI strategy가 해당 SOP를 참조할 수 있다.

### UC-02. SOP 문서를 조회한다

운영자와 뷰어는 incident 대응 중 관련 SOP를 찾는다.

대표 endpoint:

```text
GET /api/v2/ds/sop/documents
GET /api/v2/ds/sop/documents/{sopId}
GET /api/v2/ds/sop/documents/{sopId}/versions/{version}
```

성공 후:

- SOP title, body, owner team, tenant scope를 확인할 수 있다.
- 특정 version 기준으로 연결된 runbook을 이어서 조회할 수 있다.

### UC-03. Runbook을 등록한다

운영자는 특정 SOP version에 runbook을 등록한다. v0.1에서는 시스템이 script를
실행하지 않고, 운영자가 검토 후 직접 복사해서 실행한다.

대표 endpoint:

```text
POST /api/v2/ds/sop/documents/{sopId}/versions/{version}/runbooks
```

성공 후:

- runbook ID가 서버에서 부여된다.
- 기본 status가 `approved`이거나 요청 payload의 status가 반영된다.
- 같은 SOP version의 runbook 목록에서 보인다.

### UC-04. Runbook AI 초안을 생성한다

운영자는 error example을 붙여넣어 runbook 후보를 생성한다.

대표 endpoint:

```text
POST /api/v2/ds/runbooks/draft
```

성공 후:

- draft runbook 후보가 반환된다.
- 반환된 초안은 자동 저장되지 않는다.
- 운영자가 검토 후 별도 생성 요청을 보내야 저장된다.

### UC-05. Draft runbook을 검토하고 승인한다

운영자는 draft 상태의 runbook을 읽고 script 안전성, 설명, 검증 절차를 확인한 뒤
승인한다.

대표 endpoint:

```text
PUT /api/v2/ds/sop/documents/{sopId}/versions/{version}/runbooks/{runbookId}
```

성공 후:

- status가 `draft`에서 `approved`로 바뀐다.
- 기본 목록과 approved filter에서 보인다.

### UC-06. Runbook을 폐기한다

운영자는 오래되었거나 위험한 절차를 hard delete하지 않고 `deprecated`로 내린다.

대표 endpoint:

```text
PUT /api/v2/ds/sop/documents/{sopId}/versions/{version}/runbooks/{runbookId}
```

성공 후:

- 기본 목록에서 빠진다.
- `status=deprecated` 조회에서 확인된다.
- 다시 살릴 때는 `deprecated -> draft -> approved` 순서를 거친다.

### UC-07. Status별 runbook을 조회한다

운영자와 뷰어는 필요한 상태의 runbook만 조회한다.

대표 endpoint:

```text
GET /api/v2/ds/sop/documents/{sopId}/versions/{version}/runbooks?status=approved,draft
GET /api/v2/ds/sop/documents/{sopId}/versions/{version}/runbooks?status=deprecated
GET /api/v2/ds/sop/documents/{sopId}/versions/{version}/runbooks?status=all
```

성공 후:

- 기본 조회는 `approved,draft`만 반환한다.
- 폐기된 절차는 명시적으로 요청해야 보인다.

### UC-08. 잘못된 요청을 거절받는다

운영자는 누락된 입력, 잘못된 status, 없는 SOP version 같은 요청이 저장 전에
차단되는지 확인한다.

대표 실패:

- draft 요청에 `sopId` 또는 `version` 누락
- `errorExamples` 0개 또는 3개 초과
- 없는 SOP version에 runbook 생성
- 빈 title, 알 수 없는 status, confidence 범위 오류
- 허용되지 않는 status transition

성공 기준:

- 잘못된 요청은 저장되지 않는다.
- 사용자가 원인을 알 수 있는 오류 형태로 반환된다.

### UC-09. 등록되지 않은 에러 패턴을 처리한다

운영자는 incident 중 기존 runbook으로 바로 매칭되지 않는 에러를 만난다. 이때
시스템은 “없는 절차를 자동 실행”하지 않고, 운영자가 관련 SOP를 찾고 error
example을 바탕으로 draft runbook을 만들 수 있게 돕는다.

Trigger:

- alert에는 `sop_id`가 없거나 잘못된 값이 들어 있다.
- SOP는 있지만 해당 에러에 맞는 runbook이 없다.
- runbook 목록 조회 결과가 비어 있다.
- 기존 runbook은 있지만 모두 `deprecated` 상태다.

운영자 목표:

- 관련 SOP가 있는지 확인한다.
- 없으면 새 SOP를 등록하거나 기존 SOP를 보강 대상으로 표시한다.
- error example으로 AI draft를 생성한다.
- draft를 자동 승인하지 않고 검토 대기 상태로 남긴다.

대표 흐름:

```text
1. 에러/alert label로 관련 SOP를 조회한다.
2. SOP가 없으면 새 SOP 등록 대상으로 분류한다.
3. SOP는 있지만 runbook이 없으면 errorExamples로 AI draft를 요청한다.
4. 반환된 후보를 draft runbook으로 저장한다.
5. 운영자가 검토한 뒤 approved 또는 deprecated로 전환한다.
```

대표 endpoint:

```text
GET  /api/v2/ds/sop/documents
GET  /api/v2/ds/sop/documents/{sopId}/versions/{version}/runbooks
POST /api/v2/ds/runbooks/draft
POST /api/v2/ds/sop/documents/{sopId}/versions/{version}/runbooks
```

성공 후:

- 등록되지 않은 에러가 “처리 불가”로 끝나지 않고 draft 지식으로 남는다.
- draft는 기본 목록에서 검토 대상으로 보인다.
- 승인 전까지 자동 실행되거나 approved로 취급되지 않는다.

실패/예외 기준:

- 관련 SOP가 없으면 runbook draft 요청은 not found로 실패할 수 있다.
- LLM 설정 문제가 있으면 typed failure envelope로 실패 원인을 반환한다.
- error example이 부족하거나 너무 많으면 invalid input으로 거절된다.
