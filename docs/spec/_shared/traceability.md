---
id: TRACEABILITY
title: UC × Feature × WBS Traceability Matrix
type: traceability
status: living
updated: 2026-05-28
---

# Traceability Matrix

> 산출물 4종(Overview / Use Case / 기능명세 / WBS)의 ID를 한 표로 묶는다.
> 각 셀의 ID는 해당 파일의 frontmatter `implements_uc` / `covered_by_wbs` / `covers_features` 와 일치해야 한다 — desync 검출은 이 표가 진실의 원천.

## 1. Feature × Use Case

| Feature | Title | UC-001 (Golden) | UC-002 (DLQ) | UC-003 (LLM fail) |
|---|---|:---:|:---:|:---:|
| F0 | Foundation | ✓ | | |
| F1 | SOP Grounding & Store | ✓ | | ✓ (전제) |
| F2 | AI Runbook Drafting | ✓ | | ✓ |
| F3 | AI Quota Controls | | | ✓ |
| F4 | Multi-tenant Scope | ✓ | | |
| F5 | Audit | ✓ | ✓ | ✓ |
| F6 | Notification Dispatch | ✓ | ✓ | |
| F7 | PII Redaction | ✓ | | |
| F8 | DLQ + Replay | | ✓ | |

## 2. Feature × WBS Component

| Feature | WBS-1.0 Foundation | WBS-1.1 SOP | WBS-1.2 AI | WBS-1.3 Notif | WBS-1.4 PII | WBS-1.5 DLQ |
|---|:---:|:---:|:---:|:---:|:---:|:---:|
| F0 | ✓ | | | | | |
| F1 | | ✓ | | | | |
| F2 | | | ✓ | | | |
| F3 | | | ✓ | | | |
| F4 | ✓ | | | | | |
| F5 | ✓ | | | | | |
| F6 | | | | ✓ | | |
| F7 | | | | | ✓ | |
| F8 | | | | | | ✓ |

## 3. Use Case × WBS

| UC | WBS-1.0 | WBS-1.1 | WBS-1.2 | WBS-1.3 | WBS-1.4 | WBS-1.5 |
|---|:---:|:---:|:---:|:---:|:---:|:---:|
| UC-001 Golden Path | ✓ | ✓ | ✓ | ✓ | ✓ | |
| UC-002 DLQ Failure | | | | ✓ | | ✓ |
| UC-003 LLM Fail-open | | | ✓ | | | |

## 4. 커밋 ↔ Feature ↔ WBS (역사 추적)

| 커밋 (SHA prefix) | 메시지 요약 | Feature | WBS |
|---|---|---|---|
| `026863650` | native MVP foundation pilot scaffolding | F0 | WBS-1.0 |
| `72944ecac` | ground alerts in uploaded SOPs | F1 | WBS-1.1 |
| `8a55208ef` | make SOP access auditable | F5 | WBS-1.0 |
| `3fa604e03` | scope SOP strategy access by tenant | F4 | WBS-1.0 |
| `a6757136e` | fail open AI quota controls | F3 | WBS-1.2 |
| `cb29d2a59` | persist latest AI strategy history | F2 | WBS-1.2 |
| `5c036c806` | propagate SOP AI context to channels | F6 | WBS-1.3 |
| `c7f4fd330` | persist SOP documents to file | F1 | WBS-1.1 |
| `3e9dfa557` | redact email/phone/long secrets in payload | F7 | WBS-1.4 |
| `ade174bb8` | JSONL DLQ + idempotent replay ledger | F8 | WBS-1.5 |
| `91b9ff5db` | wire DLQ into alertmanager dispatcher | F8 | WBS-1.5 |

## 5. 검증 가이드 (desync 검출)

이 표가 진실. 각 stub 파일의 frontmatter는 이 표와 일치해야 한다.

검증 방법 (수동/스크립트 가능):
1. `03-functional-spec/modules/F{n}.md`의 `implements_uc`, `covered_by_wbs` ↔ §1, §2
2. `02-usecase/cases/UC-NNN.md`의 `implements_features`, `related_wbs` ↔ §1, §3
3. `04-wbs/packages/WBS-1-N.md`의 `covers_features` ↔ §2

frontmatter 변경 시 이 표도 함께 업데이트 (PR 체크리스트).

## 6. Open / Missing 항목

| 항목 | 상태 | 비고 |
|---|---|---|
| HMAC 정책 (NF-5.3.1, F8) | open | replay 서명 정책 미정 |
| Frontend 변경 영역 | open | F6, UC-001 단계 6 (운영자 검수 화면) 매핑 미확정 |
| 기존 `docs/_archive/` 자료 재사용 판단 | open | usecase·sop·merge_usecase 초안 |
