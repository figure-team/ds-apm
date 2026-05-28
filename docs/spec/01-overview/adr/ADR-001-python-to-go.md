---
id: ADR-001
title: Python ds_apm_poc 폐기 → Go + SigNoz로 통합
status: accepted
date: 2026-05-19
deciders: [team]
supersedes: []
superseded_by: null
updated: 2026-05-29
---

# ADR-001 — Python ds_apm_poc 폐기 → Go + SigNoz로 통합

## Status
**Accepted** (2026-05-19)

## Context

DS-APM 초기 prototype은 `workspace_archive/ds-apm`의 Python orchestrator (`ds_apm_poc` 패키지)로 시작했다. 이 단계의 아키텍처는 다음과 같았다.

- **별도 프로세스 2개**: SigNoz community 빌드와 Python `ds_apm_poc`가 각각 독립 컨테이너로 동작.
- **Bridge layer**: 두 시스템 사이를 잇는 Python bridge가 incident routing / SOP grounding / AI runbook drafting / retry+DLQ / notification dispatch를 직접 구현.
- **중복 구현**: Alertmanager가 이미 보유한 Slack / MS Teams / PagerDuty / Webhook / Email 5채널 dispatch를 Python에서 다시 만들어야 했고, retry budget·DLQ 인덱스도 Python 자체 자료구조로 별도 유지.

이 구조는 두 가지 비용을 발생시켰다.

1. **운영 비용 이중화** — Go (SigNoz) + Python (ds_apm_poc) 두 런타임을 모두 모니터링·배포·롤백해야 함.
2. **Interface drift** — alert payload 스키마, OTel resource attribute, idempotency key 규약이 두 시스템 간 미세하게 어긋나기 시작했고, bridge가 매번 매핑을 다시 검증해야 했음.

결정 시점(2026-05-19)에 archive 브랜치 `orchestrator-to-signoz-migration`에서 마이그레이션 작업이 완료되어 종결 커밋 `bc7e491 docs(migration): close orchestrator-to-signoz migration with verification evidence`가 기록됐다.

## Decision

Python `ds_apm_poc` 런타임을 **폐기**하고, DS-APM의 모든 기능(F0~F8)을 **SigNoz community 코드베이스에 Go-native MVP로 흡수**한다. DS-APM은 별도 서비스가 아니라 SigNoz community 빌드 위의 **확장 레이어**로 존재한다. fork 프레이밍 (`SigNoz fork`)은 쓰지 않으며, `pkg/alertmanager/`, `pkg/ruler/`, `pkg/types/ruletypes/`, `cmd/community/` 등 기존 패키지에 직접 변경을 가한다.

운영 코드는 `workspace_archive/ds-apm/var/signoz` nested repo(`ds-apm/native-mvp-foundation` 브랜치, fork base `feea9e9b3` 이후 11 커밋)에 위치하며 `figure-team/ds-apm`에 single squash commit으로 공개 스냅샷이 노출된다.

## Consequences

### Positive
- **Single binary 배포** — `cmd/community` 1개 진입점만 부팅·롤백. 운영 surface가 절반으로 축소.
- **OTel-native** — SigNoz가 OpenTelemetry resource semantic attribute를 그대로 캐리하므로, DS-APM의 alert payload는 instrumentation에서 dispatcher까지 schema-mapping 없이 흐른다.
- **Alertmanager 5채널 재사용** — Slack / MS Teams v2 / PagerDuty / Webhook / Email 어댑터를 0줄에서 다시 짤 필요가 없고, DS-APM의 SOP/AI annotation을 기존 채널 template hook에 끼워넣는 형태로 흡수 (F6).
- **Dispatch hot path 단순화** — Python bridge가 처리하던 retry/DLQ를 Alertmanager dispatcher hot path 안의 `aiHook` + `dlqSink`로 통합 (F6, F8). DLQ는 별도 큐 백엔드 없이 JSONL sink로 구현 (`ade174bb8`).
- **Interface drift 종결** — alert payload, idempotency key, audit event 3개의 contract version 문자열을 `pilot_contract.go` 한 곳에서 frozen string으로 export하여 두 시스템 간 desync 가능성을 제거.

### Negative
- **SigNoz upstream 종속** — community 빌드의 internal API (e.g., `dispatch.Dispatcher`, `notify.Stage`) 변경 시 DS-APM도 영향 받음. upstream merge 비용 발생.
- **Go 재작성 비용** — Python `ds_apm_poc`의 SOP grounding / AI drafting / quota / PII / DLQ 모듈 약 100 파일, **+12,632 LOC** 신규 작성 (baseline §3).
- **Nested repo 운영 위험** — 운영 fork가 `workspace_archive/ds-apm/var/signoz`라는 nested 위치에 있어, 상위 repo (`workspace_archive/ds-apm`)에서 `git status`만 보면 변경이 가려진다. 메모리 항목 "var/signoz는 우리 코드" 정책으로 항상 nested repo의 자체 `.git`도 확인해야 한다.
- **Enterprise 모듈 경계** — `ee/`, `cmd/enterprise/`는 SigNoz Enterprise License 적용이므로 DS-APM 산출물 범위 밖. community 빌드에만 머무는 자기 제약이 생긴다.

## Alternatives Considered

### A. Python orchestrator + SigNoz bridge 유지
- **이점**: 기존 Python 코드 보존, 팀 일부의 Python 숙련도 활용 가능.
- **이유 미채택**: 두 런타임 운영 비용 + bridge가 짊어진 interface drift 비용이 누적. POC 단계에서 이미 ds_apm_poc와 SigNoz alert schema가 어긋나기 시작.

### B. 외부 Go 서비스로 SigNoz와 통합
- **개요**: SigNoz는 그대로 두고, 별도 Go 서비스가 webhook 수신 + SOP grounding + AI drafting + 채널 dispatch를 담당.
- **이유 미채택**: Alertmanager 5채널 dispatch 코드 (Slack Block Kit, MS Teams Adaptive Card v1.4, PagerDuty Events API v2 등)를 새 서비스에서 재구현해야 함. SigNoz `pkg/alertmanager/alertmanagernotify/` 자산을 버리는 것이 손실 과다.

### C. 현 상태 유지 (Python only)
- **이유 미채택**: SigNoz community의 OTel-native ingress 자산을 활용하지 못함. 자체 OTel collector 운영 비용이 polynomially 증가.

## References
- 메모리 항목: "Orchestrator → SigNoz 마이그레이션 (2026-05-19)" — Python ds_apm_poc 폐기 + PR #1 (DLQ 활성화 + HMAC 정책 follow-up 미해결)
- Archive 브랜치: `orchestrator-to-signoz-migration` (`workspace_archive/ds-apm`)
- 마이그레이션 종결 커밋 (archive): `bc7e491 docs(migration): close orchestrator-to-signoz migration with verification evidence`
- DS-APM 시작 커밋: `026863650 feat(ds-apm): add native mvp foundation pilot scaffolding`
- 변경 표면: 100 파일, +12,632 / -110 LOC ([`../../_foundation/baseline.md`](../../_foundation/baseline.md) §3)
- 메모리 항목: "var/signoz는 우리 코드 (nested repo)" — fork 프레이밍 금지 + nested repo 자체 `.git` 확인 정책
