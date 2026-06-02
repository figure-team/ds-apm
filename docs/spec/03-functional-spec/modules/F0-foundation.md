---
id: F0
title: 공통 기반 모듈 (Foundation Core) / Pilot Scaffolding
status: planned
commits: [026863650]
source_paths:
  - cmd/community/
  - pkg/types/ruletypes/pilot_contract.go
  - pkg/types/ruletypes/pilot_managed_markdown.go
implements_uc: [UC-001]
covered_by_wbs: [WBS-1.0]
updated: 2026-06-02
---

# F0 — 공통 기반 모듈 (Foundation Core) / Pilot Scaffolding

> **상태**: 착수 예정 (착수보고 기준)
> AIOpsAgent 네이티브 MVP의 토대 — pilot contract, managed markdown, community 진입점 통합.

## F0.1 개요

본 모듈은 AIOpsAgent가 SigNoz community build 위에서 동작하기 위해 필요한 **계약(contract) 정의·진입점·기본 sink 등록**을 담당한다. 핵심은 다음 세 가지다.

1. **Pilot Contract 정의** — SOP source / health / audit / service account / configuration 5종 contract의 버전 문자열과 enum, validator를 한 곳에 모은다 (`pilot_contract.go`).
2. **Managed Markdown SOP source** — AIOpsAgent가 v0.1에서 지원하는 유일한 SOP source kind. 운영자가 markdown 본문을 직접 등록하는 minimal path.
3. **Community 진입점** — `cmd/community/main.go`가 부팅 시 pilot audit JSONL sink를 등록한다 (F5와 연결).

본 모듈은 다른 F1~F8 모듈이 의존하는 contract 식별자(`ContractVersion` 문자열)와 enum(`PilotSOPSourceKind*`, `PilotAuditOutcome*` 등)을 모두 export한다.

## F0.2 인터페이스

진입점은 `cmd/community/main.go`의 `main()` 함수다.

```go
func main() {
    logger := instrumentation.NewLogger(instrumentation.Config{...})

    if jsonlSink, err := ruletypes.NewPilotAuditEventJSONLSink(
        "var/audit/pilot-events.jsonl",
        ruletypes.DefaultPilotAuditJSONLMaxSizeBytes,
    ); err != nil {
        zap.L().Warn("pilot audit JSONL sink init failed; falling back to nop", zap.Error(err))
    } else {
        ruletypes.RegisterPilotAuditEventSink(jsonlSink)
    }

    registerServer(cmd.RootCmd, logger)
    cmd.RegisterGenerate(cmd.RootCmd, logger)
    cmd.Execute(logger)
}
```

검증 인터페이스는 `pilot_contract.go`의 다섯 validator 함수:

```go
func ValidatePilotSOPSourceCatalog(resp PilotSOPSourceCatalogResponse) error
func ValidatePilotSOPSourceHealth(resp PilotSOPSourceHealthResponse) error
func ValidatePilotAuditEvent(event PilotAuditEvent) error
func ValidatePilotServiceAccountProfile(profile PilotServiceAccountProfile) error
func ValidatePilotConfiguration(config PilotConfiguration) error
```

## F0.3 데이터 모델

핵심 contract 버전 (전부 frozen string):

| Constant | 값 |
|---|---|
| `PilotSOPSourceCatalogContractVersion` | `ds-apm.sop-source-catalog.v1` |
| `PilotSOPSourceHealthContractVersion` | `ds-apm.sop-source-health.v1` |
| `PilotAuditEventContractVersion` | `ds-apm.audit-event.v1` |
| `PilotServiceAccountProfileContractVersion` | `ds-apm.service-account-profile.v1` |
| `PilotConfigurationContractVersion` | `ds-apm.pilot-configuration.v1` |

주요 struct (요약):

```go
type PilotSOPSource struct {
    SourceID, DisplayName, Kind, AuthMode, Status string
    LastHealthCheckAt, LastSyncAt                 string
    Capabilities                                  PilotSOPSourceCapabilities
    ServiceAccountProfile                         string
    SecretRefVisible                              bool   // 항상 false
    ConfiguredBy                                  string
    Warnings                                      []string
}

type PilotManagedMarkdownDocument struct {
    SOPID, Version, Title, BodyMarkdown, DisplayURL, UpdatedAt string
    Tags                                                       []string
}

type PilotConfiguration struct {
    ContractVersion, ProjectID, Environment, ServiceName string
    SelectedSources                                       []PilotSelectedSource
    AllowedCapabilities                                   PilotAllowedCapabilities
    AuditMode                                             string  // required | deferred | disabled
    Enabled                                               bool
    RolloutID                                             string
}
```

SOP source kind enum: `url_registry | managed_markdown | git_markdown | confluence | notion | sharepoint | custom_connector`. **v0.1에서는 `managed_markdown` 외 모두 미구현.**

Body markdown 상한: `PilotSOPFetchBodyMarkdownMaxBytes = 256 KiB`.

## F0.4 상태 전이

해당 없음. 본 모듈은 contract 정의·진입점·sink 등록만 담당하며, runtime state machine을 보유하지 않는다.

## F0.5 예외 및 복구

- **JSONL sink 초기화 실패** — `cmd/community/main.go`는 fail-open. `zap.L().Warn(...)` 후 default `NopPilotAuditEventSink`로 진행한다. 서버 부팅은 절대 막지 않는다 (운영 가용성 우선).
- **Contract validation 실패** — validator는 `errors.Join(errs...)`로 모든 위반을 한꺼번에 보고. 호출자가 fail-closed/fail-open을 선택.
- **Secret-like 값 검출** — `hasPilotSecretLikeValue()`가 `token=`, `client_secret`, `api_key`, JWT-like 패턴 등을 contract 응답에서 차단. 위반 시 validation error 반환.

## F0.6 비기능 요건 (NFR)

- **NF-F0.1** 시스템은 contract version 문자열을 절대 자동 변경하지 않아야 한다 (downstream desync 방지).
- **NF-F0.2** Audit sink 등록 실패는 서버 부팅을 막지 않아야 한다 (fail-open).
- **NF-F0.3** 모든 browser-visible response (`SecretRefVisible`, `CredentialDetailsVisible`, `BrowserCredentialsUsed`)는 항상 `false`여야 한다.
- **NF-F0.4** Body markdown payload는 256 KiB를 초과하지 않아야 한다.

## F0.7 Acceptance Criteria (Gherkin)

```gherkin
Feature: Foundation contract validation
  Background:
    Given the AIOpsAgent contract package is initialized

  Scenario: Pilot SOP source catalog rejects credential leakage
    Given a PilotSOPSource with secretRefVisible set to true
    When ValidatePilotSOPSourceCatalog is called
    Then the validator returns an error mentioning "secretRefVisible"

  Scenario: Audit JSONL sink failure does not abort boot
    Given the audit log directory is read-only
    When the community main() initializes the JSONL sink
    Then the warning is logged and NopPilotAuditEventSink remains active
    And the server proceeds to register commands
```

## F0.8 Traceability
- Implements UC: UC-001
- Covered by WBS: WBS-1.0
- Source: `cmd/community/`, `pkg/types/ruletypes/pilot_contract.go`, `pkg/types/ruletypes/pilot_managed_markdown.go`
- Commits: `026863650`
