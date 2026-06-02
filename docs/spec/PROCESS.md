---
id: SPEC-PROCESS
title: Spec Documents — Process & Rules
type: process
status: living
updated: 2026-06-02
---

# Spec Documents — Process & Rules

> 이 파일은 `docs/spec/` 산출물 작성 시 적용되는 **모든 규칙·결정·구조**를 한 곳에 모은다.
> CLAUDE.md는 본 파일을 가리키는 pointer만 가지며, 본 파일은 산출물 작업 시작 시점에 1회 Read하면 충분하다.

## 0. TL;DR — 4가지만 기억하면 됨

1. 산출물 4종(Overview / Use Case / 기능명세서 / WBS)은 **현재 구현된 코드를 reverse-engineering**해서 작성. 미래 계획 아님. 단 동결 아니고 계속 진화.
2. **Markdown이 source of truth**. **상세본(01~04)은 md-only** — HTML 빌드 안 함. **요약본(00-brief)만 HTML 유지** — 비기술 의사결정자용. HTML 직접 손대지 않는다.
3. **stable ID는 frontmatter + 파일명에 박힘** (`F1`, `UC-001`, `WBS-1.1`). 제목이 바뀌어도 ID는 유지.
4. **`_shared/traceability.md`가 진실의 원천**. 모든 frontmatter는 이 매트릭스와 일치해야 한다.

---

## 1. 산출물 4종

| # | 산출물 | 위치 | 표준 |
|---|---|---|---|
| 1 | Overview | `01-overview/` | arc42 v9.0 + C4 |
| 2 | Use Case | `02-usecase/` | Cockburn Fully Dressed + UML sequence + Gherkin |
| 3 | 기능명세서 (SRS) | `03-functional-spec/` | ISO/IEC/IEEE 29148-lite + Spec by Example |
| 4 | WBS | `04-wbs/` | PMI WBS 2nd ed. (component-oriented Lv2) + Agile (Lv3+) |

상세본(01~04)은 `.md` 파일이 source이자 최종 산출물(md-only). LLM·개발자는 `.md`를 직접 읽는다. HTML 빌드 대상은 요약본(`00-brief/`)만이다.

---

## 2. 확정된 결정 (5가지)

| # | 항목 | 결정 |
|---|---|---|
| 1 | WBS Lv2 골격 | **Component-oriented** (공통 기반 모듈 / SOP 그라운딩 서비스 / AI 초안 매니저 / 알림 디스패처 / PII 마스킹 필터 / DLQ 재처리 서비스). Phase 시간선은 부록 `appendix-phases.md`로만 |
| 2 | 상세 에러 케이스 2건 | **UC-002** (Channel 4xx/5xx → DLQ → Replay), **UC-003** (LLM auth/quota fail-open → SOP fallback) |
| 3 | 언어 톤 | 본문 한국어 + ID/코드 스키마 영문. `shall` → "~해야 한다" 일관 |
| 4 | Gherkin 키워드 | **영문** (`Given/When/Then`) — godog 호환 |
| 5 | arc42 버전 | **v9.0** (2025-07, §10 Quality Tree+Scenarios 재구조화 포함) |

---

## 3. 변경 용이 구조 — 5대 원칙

1. **Atomic — 1 파일 1 개념**: UC 1건 = 1 파일, 모듈 1개 = 1 파일, WBS package 1개 = 1 파일
2. **Stable ID를 파일명에 박기**: `F1-sop-grounding.md`에서 `F1`은 영구. 제목이 "Runbook Indexing"으로 바뀌어도 `F1` 유지
3. **YAML frontmatter로 메타데이터**: status, commits, source_paths, implements_uc/covered_by_wbs 등 모두 frontmatter
4. **Single Source of Truth = Markdown**, HTML은 빌드 산출물. HTML 직접 편집 ✗
5. **코드를 ground truth로 역참조**: 모든 모듈/WBS의 `source_paths` + `commits`로 실제 코드 가리킴. 코드 변경 시 grep으로 영향 산출 자동 추적

---

## 4. 폴더 구조

```
docs/spec/
├─ PROCESS.md                          # 본 파일
├─ _foundation/                        # 작업 입력 (baseline + research)
│  ├─ baseline.md                      # 구현 현황 (커밋·LOC·diff)
│  └─ research-skills-{a,b,c}*.md      # 표준·HTML·도메인 리서치
├─ _shared/                            # 공통 자산
│  ├─ traceability.md                  # ★ 진실의 원천: UC × F × WBS 매트릭스
│  ├─ glossary.md                      # 용어집 (anchor target)
│  └─ design-tokens.css                # (예정) HTML 빌드 공통 CSS
├─ 01-overview/
│  ├─ index.md
│  ├─ adr/ADR-NNN-*.md
│  └─ diagrams/
├─ 02-usecase/
│  ├─ index.md
│  ├─ cases/UC-NNN-*.md
│  └─ diagrams/
├─ 03-functional-spec/
│  ├─ index.md
│  └─ modules/F{0..8}-*.md
└─ 04-wbs/
   ├─ index.md
   ├─ appendix-phases.md               # P0~P5 시간선 (역사)
   └─ packages/WBS-1-{0..5}-*.md
```

---

## 5. Frontmatter 스키마

### 5.1 Use Case (`02-usecase/cases/UC-NNN-*.md`)

```yaml
---
id: UC-NNN                       # stable, 영구
title: ...
type: usecase
level: User-goal                 # Cockburn level 통일
scope: DS-APM System
status: implemented | draft | deprecated | planned
primary_actor: ...
supporting_actors: [...]
implements_features: [F0, F1]    # _shared/traceability.md §1과 일치
related_wbs: [WBS-1.0, WBS-1.1]  # _shared/traceability.md §3과 일치
priority: P1 | P2 | P3 | P4
updated: YYYY-MM-DD
---
```

### 5.2 Feature Module (`03-functional-spec/modules/FN-*.md`)

```yaml
---
id: FN                           # F0~F8, stable
title: ...
status: implemented | implemented-mvp | draft | deprecated | planned
commits: [SHA1, SHA2]            # 11개 DS-APM 커밋 중 어느 것
source_paths:                    # 실제 pkg/ 경로
  - pkg/...
implements_uc: [UC-NNN]          # _shared/traceability.md §1과 일치
covered_by_wbs: [WBS-1.N]        # _shared/traceability.md §2와 일치
updated: YYYY-MM-DD
caveats: "..."                   # 선택: README 경고 등
open_items: [...]                # 선택: 미해결 follow-up
---
```

### 5.3 WBS Package (`04-wbs/packages/WBS-1-N-*.md`)

```yaml
---
id: WBS-1.N
title: ...
parent: WBS-1
status: implemented | implemented-mvp | implemented-{caveat}-pending | planned | draft
covers_features: [Fx, Fy]        # _shared/traceability.md §2와 일치
source_paths: [pkg/...]
acceptance: pending | passed
estimated_effort: completed | TBD | <hours>
commits: [...]
updated: YYYY-MM-DD
---
```

### 5.4 ADR (`01-overview/adr/ADR-NNN-*.md`)

```yaml
---
id: ADR-NNN
title: ...
status: proposed | accepted | superseded | deprecated
date: YYYY-MM-DD
deciders: [...]
supersedes: [ADR-XXX] | []
superseded_by: ADR-XXX | null
updated: YYYY-MM-DD
---
```

---

## 6. Traceability 검증

`_shared/traceability.md`가 진실의 원천. 모든 stub의 frontmatter는 이 매트릭스와 일치해야 한다.

### 6.1 검증 체크리스트 (변경 시)

- [ ] `FN.md`의 `implements_uc` ↔ traceability.md §1 (Feature × UC)
- [ ] `FN.md`의 `covered_by_wbs` ↔ traceability.md §2 (Feature × WBS)
- [ ] `UC-NNN.md`의 `implements_features` ↔ traceability.md §1
- [ ] `UC-NNN.md`의 `related_wbs` ↔ traceability.md §3 (UC × WBS)
- [ ] `WBS-1-N.md`의 `covers_features` ↔ traceability.md §2
- [ ] 새 커밋은 traceability.md §4 (커밋 ↔ Feature ↔ WBS)에 append

### 6.2 desync 발생 시
1. 진실의 원천(`traceability.md`)을 먼저 수정
2. 영향받는 frontmatter들을 일괄 갱신
3. 양방향 link (e.g., F1.md의 `implements_uc` ↔ UC-001.md의 `implements_features`) 모두 확인

---

## 7. TODO 마커 규칙

미작성 섹션은 `TODO` 또는 `TODO — 설명`. grep 한 번에 검출 가능:

```bash
grep -rn "^TODO\|: TODO\|- TODO" docs/spec/
```

- 작성 완료 시 TODO 토큰 모두 제거
- 부분 작성 시 `TODO (partial: 무엇이 남았는지)`
- 미해결 follow-up은 TODO가 아니라 frontmatter `open_items:`로 관리 (영구 추적)

---

## 8. HTML 빌드 정책

### 8.1 빌드 대상 범위 (결정 확정)

| 영역 | HTML 빌드 | 이유 |
|---|:---:|---|
| 상세본 (`01-overview/`, `02-usecase/`, `03-functional-spec/`, `04-wbs/`) | **아님** | LLM·개발자는 `.md`를 직접 읽음. `.md` = source of truth = 최종 산출물 |
| 요약본 (`00-brief/`) | **함** | 비기술 의사결정자 프레젠테이션용. 5종 HTML 파일 유지 |

- **상세본은 md-only**. 상세본 4종 HTML 파일은 삭제됐으며 재생성하지 않는다.
- **사람은 `.md`만 편집**. HTML은 절대 직접 손대지 않음 (desync 방지).
- `.md` → HTML 변환은 요약본(`00-brief/`)에만 명시적으로 실행.

### 8.2 요약본 HTML 파일명

`00-brief/` 폴더 내 HTML만 유지. 파일명 예: `brief.html`, `brief-spec.html` 등 요약본 식별자 기준.

### 8.3 빌드 베이스 (요약본 전용)
- CSS: `_shared/design-tokens.css` (예정 — research-skills-b-html.md §5.1 토큰 그대로)
- 다이어그램: Mermaid v11 ESM CDN 1줄 + 인라인 SVG (자유도 필요 시)
- JS: vanilla (외부 의존 0)
- 인쇄 호환: `@media print` 블록 항상 포함 (research-skills-b-html.md §5.8)

### 8.4 금기
research-skills-b-html.md §6의 AI slop 회피 규칙 그대로 적용 (Tailwind CDN, Inter+그라데이션, transition all, dark/light 토글 v1 등).

---

## 9. 작성 순서 권장 (TODO 채우기 우선순위)

1. **`_shared/glossary.md`** — 다른 문서가 모두 anchor로 참조. 용어부터.
2. **`03-functional-spec/modules/F*.md`** — 코드(`source_paths`)를 직접 읽고 작성. F0 → F8 순.
3. **`02-usecase/cases/UC-*.md`** — Cockburn template 채움. UC-001 → 002 → 003.
4. **`04-wbs/packages/WBS-*.md`** — Deliverable / Acceptance / Verification.
5. **`01-overview/index.md`** — 위 자료를 요약. 마지막에 작성 (전체 합쳐야 정확).
6. **요약본 HTML 빌드** — 상세본 완성 후 `00-brief/` 한정. `_shared/design-tokens.css` 준비 후 요약본 HTML만 생성 (상세본 HTML 빌드 안 함).

---

## 10. 변경 시 절차 (PR/커밋 체크리스트)

1. **코드 변경** → 영향받는 `.md` 파일을 `source_paths` 역색인으로 찾기
   ```bash
   grep -rln "변경된_경로" docs/spec/
   ```
2. **frontmatter 갱신**
   - `commits:` 새 SHA append
   - `status:` 변경 (e.g., `draft` → `implemented`)
   - `updated:` 오늘 날짜
3. **traceability.md 갱신**: 새 커밋은 §4에, 영향받는 매핑은 §1~3에
4. **TODO 검토**: 새 코드로 채워졌으면 TODO 제거
5. **본 PROCESS.md의 규칙 위배 없는지** 확인
6. **요약본 HTML 재빌드** (`00-brief/` 내용이 바뀌었을 때만. 상세본 변경은 HTML 빌드 없음)

---

## 11. 참조 문서 (필요 시 Read)

- [`_foundation/baseline.md`](_foundation/baseline.md) — 11 commits / 100 files / +12.6k LOC, fork base commit, 커밋 ↔ 모듈 매핑
- [`_foundation/research-skills-a-methods.md`](_foundation/research-skills-a-methods.md) — arc42 / Cockburn / 29148 / PMI 깊이
- [`_foundation/research-skills-b-html.md`](_foundation/research-skills-b-html.md) — 디자인 토큰, Mermaid, 인라인 SVG, print CSS, AI slop 회피
- [`_foundation/research-skills-c-domain.md`](_foundation/research-skills-c-domain.md) — Google SRE / PagerDuty / OTel / Alertmanager / DLQ idempotency

본 PROCESS.md는 위 4문서의 결정·요약본. 깊이 필요 시 위 4문서로 drill-down.

---

## 12. 본 파일의 진화

본 PROCESS.md도 stub들과 마찬가지로 living document.
- 규칙 추가/변경 시 `updated:` frontmatter 갱신
- 큰 결정 변경 시 ADR로 별도 기록 후 본 파일 참조 갱신
- 본 파일 변경이 잦으면 CLAUDE.md의 pointer만 가리키므로 토큰 비용 없음
