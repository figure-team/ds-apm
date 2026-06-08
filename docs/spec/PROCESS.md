---
id: SPEC-PROCESS
title: Spec Documents — Process & Rules
type: process
status: living
updated: 2026-06-08
---

# Spec Documents — Process & Rules

> `docs/spec/` 산출물 작성 시 적용되는 **모든 규칙·결정·구조**를 한 곳에 모은다. CLAUDE.md는 본 파일 pointer만 가지며, 작업 시작 시 1회 Read하면 충분하다.
> **2026-06-08 개정 (BMAD 정렬)**: 산출물을 BMAD 체인 **PRD → 에픽 → 스토리 → WBS (+ Architecture)** 로 재편했다. **사업 전략서에서 top-down**으로 derive하고, 기능명세는 **Capability Feature(CF) 축 + FR**로 분해하며, **User Journey(UJ)는 PRD에 내장**(별도 use-case 문서 폐지).

## 0. TL;DR — 5가지만 기억하면 됨

1. 산출물은 **사업 전략서(`_foundation/source-strategy-brief.md`)에서 top-down**: Vision → Target User·Jobs(JTBD) → User Journey(UJ) → Feature(CF) → FR. 코드(F0~F8)는 전략의 **현재 구현 표면**이며 FR의 *구현 근거*로 강등(FR 본문은 고객 voice).
2. 기능명세 분해 축은 **CF(Capability Feature, 사용자 가치)**. 구 F0~F8은 코드 매핑(component-source-map)으로만 남고, 산출물 ID는 `CF-1..N` + `FR-CFn.m`.
3. **범위 hybrid**: 구현 핵심(CF-1~6)은 FR 상세 + Given/When/Then. 미구현 로드맵(CF-7~10)은 저fidelity FR + 단계 태그.
4. **Markdown = source of truth**. `.md`(풍부) + `.html`(사람용 큐레이션 뷰) 공존. md↔html 1:1 동기 불필요 — 사실(숫자·날짜·명명·상태·ID) 모순만 금지.
5. **stable ID는 frontmatter + 파일명에 박힘** (`CF-1`, `WBS-1.1`, `FR-CF1.1`, `UJ-1`). 진실의 원천 = `source-strategy-brief.md`(top-down) + `_shared/traceability.md`(CF×UJ×WBS) + 코드(`component-source-map.md`).

---

## 1. 산출물 (BMAD 정렬)

| # | 산출물 | 위치 | 표준 | BMAD 대응 |
|---|---|---|---|---|
| 1 | **기능명세 (PRD)** | `01-prd/` | BMAD PRD(Vision→JTBD→UJ→Features→FR) × 29148-lite × Spec by Example | PRD |
| 2 | **에픽/스토리** | `03-epics/` + `04-stories/` | 에픽(목표+스토리 목록) + **스토리 파일**(서술형 + 인수기준 + Tasks/Subtasks) — 애자일 작업 정의 | Epics & Stories |
| 3 | **WBS** | `05-wbs/` | PMI WBS 2nd(component Lv2) + 일별 일정·간트 — **에픽/스토리에서 파생** | (에픽과 별도) |
| 4 | **Architecture** | `02-architecture/` | C4 + ERD/데이터모델 | Architecture (Solutioning) — 후행 stub |

- **에픽 ≠ WBS**: 에픽/스토리=애자일 작업 정의(사용자 가치), WBS=PMI 컴포넌트·일정 분해. 둘 다 PRD에서 파생, CF/컴포넌트로 매핑.
- **User Journey(UJ)는 PRD(`01-prd/index.md` §5)에 내장** — 별도 `02-usecase/` 문서 폐지(BMAD는 journey를 PRD에 둠).
- **01-overview / 00-brief는 별도 산출물로 만들지 않음** — 필요 시 PRD에서 파생.
- `.md`(source) + `.html`(큐레이션 뷰) 공존. HTML은 자동 빌드 아니라 큐레이션이므로 깊이는 달라도 되고 사실만 일치.

---

## 2. 확정된 결정 (7가지)

| # | 항목 | 결정 |
|---|---|---|
| 1 | **분해 방향** | **Top-down** — 전략서 Vision/JTBD/UJ → FR. 코드 역설계는 *구현 근거* 확인용. |
| 2 | **분해 축** | **CF(사용자 가치)**. BMAD "기술 레이어 금지, 사용자 가치로 묶기". 구 F0~F8은 매핑으로만. |
| 3 | **FR 표기** | "[액터]는 ~한다/보장받는다"(고객 voice). 코드 어휘는 `구현 근거`로 분리. testable. |
| 4 | **범위** | **Hybrid** — 구현 CF는 FR+G/W/T 상세, 로드맵 CF는 저fidelity + 단계 태그. |
| 5 | **산출물** | **PRD · 에픽 · 스토리 · WBS · Architecture** (`prd`/`epics`/`stories`/`wbs`/`architecture`). UJ는 PRD 내장. UseCase/Overview/Brief 폐지. |
| 6 | 언어 / Gherkin | 본문 한국어 + ID/코드 영문. Gherkin 키워드 영문(`Given/When/Then`, godog), 스텝 한글. |
| 7 | WBS·여정 | WBS Lv2 = component(6, CF와 1:1) + 일별 Excel 스케줄. 에러 여정 2건: **UJ-2**(채널 실패→DLQ→Replay), **UJ-3**(LLM fail-open→SOP fallback). |

### 2.8 BMAD ↔ 한국 SI 용어·문체 변환 (필수)

BMAD는 영어·애자일 어휘다. 산출물 본문은 **한국 SI 문체**로 변환해 쓴다 — 요구는 `shall` = "~하여야 한다", 설명은 명사·동명사 종결("~기능 제공", "~검증함"), **표 중심**(항목ID·내용·근거·검증), 구어체·경어 배제(개조식).

| BMAD (영어식) | 한국 SI 용어 |
|---|---|
| PRD | 기능명세서 / 요구사항정의서 |
| Vision | 추진 배경·목적 |
| Target User / Persona | 대상 사용자 / 이해관계자 |
| Jobs-to-be-Done | 업무 요구 / 현행 문제점 |
| User Journey (UJ) | 업무 흐름 / 업무 시나리오 |
| Feature (CF) | (단위)기능 / 기능 영역 |
| Functional Requirement (FR) | 기능 요구사항 (`FR-CF`) |
| Acceptance Criteria (Given/When/Then) | **인수 기준 / 검증 기준** |
| Epic | 에픽 (애자일 작업 묶음 — WBS 작업 패키지와 구분) |
| Story | 스토리 / 사용자 스토리 (서술형: 역할·요구·목적) |
| NFR | 비기능 요구사항 |
| Milestone | 주요 일정 / 마일스톤 |
| Sprint / Backlog | 해당 없음 (단계·차수 / 잔여작업) |

- **ID·코드는 영문 유지**(`CF-1`·`FR-CF1.1`·`WBS-1.1`·`UJ-1`), 본문·라벨·제목은 SI 문체.
- Gherkin 키워드는 영문 유지(godog), 스텝은 한글 SI 문체.
- **스토리 문장**: BMAD `As a / I want / So that`는 **한국어 서술형**으로 — "{역할}는 {요구}한다. (목적: {benefit})". 영한 혼용 금지.
- 근거 표준: §11의 CBD SW 표준 산출물 가이드 · NIPA SW 산출물 작성 가이드 · 행정기관 정보화사업 추진 매뉴얼.

---

## 3. 변경 용이 구조 — 5대 원칙

1. **Atomic — 1 파일 1 개념**: CF feature 1개 = 1 파일, **스토리 1개 = 1 파일**(`04-stories/N.M.story.md`).
2. **Stable ID를 파일명에 박기**: `CF-1`·`FR-CF1.1`·`EPIC-N`·**Story `N.M`**(`04-stories/1.1.story.md`) 영구. UJ-1~4는 PRD §5.
3. **YAML frontmatter**: CF(status·jtbd·maps_modules·source_paths·implements_uj·covered_by_wbs·fr_ids) · 에픽(covers_feature·maps_wbs·stories) · 스토리(epic·fr·covers_feature·wbs_component·status).
4. **Single Source of Truth = Markdown**, HTML은 큐레이션 뷰. HTML 직접 편집 ✗.
5. **코드를 ground truth로**: 각 FR의 `구현 근거` + CF의 `source_paths`. 코드 변경 시 grep 영향 추적.

---

## 4. 폴더 구조

```
docs/spec/
├─ PROCESS.md                          # 본 파일
├─ _foundation/                        # 작업 입력 (top-down 진실원천)
│  ├─ source-strategy-brief.md         # ★ 원본 사업 전략서 (top-down 출발점)
│  └─ baseline.md                      # as-built 사실 기반 (변경표면·LOC·모듈↔CF 매핑)
├─ _shared/                            # 공통 자산
│  ├─ traceability.md                  # ★ CF × UJ × WBS 매트릭스 (매핑 진실원천)
│  ├─ component-source-map.md          # ★ 6컴포넌트 ↔ 코드 ↔ F0~F8 (as-built, drift)
│  ├─ glossary.md · design-tokens.css   # 용어집 · HTML 토큰
├─ 01-prd/                               # ★ PRD (BMAD)
│  ├─ index.md                         # Vision·JTBD·SM·UJ(§5)·Feature Map·Coverage Map·NFR·Non-goals
│  └─ features/CF-{1..N}-*.md          # CF별: 개요·FR(무엇을/Acceptance/구현근거)·NFR·예외·Open
├─ 02-architecture/                      # ★ Architecture (C4+ERD) — 후행 stub
│  └─ index.md
├─ 03-epics/                             # ★ BMAD 에픽 (목표 + 스토리 목록 링크)
│  ├─ index.md
│  └─ epic-{1..6}-*.md                 # Epic 목표 + 스토리 표(→ 04-stories/)
├─ 04-stories/                           # ★ BMAD 스토리 (별도 파일, 애자일 작업 정의)
│  └─ {epic}.{story}.story.md          # 서술형 스토리 + 인수기준 + Tasks/Subtasks + Dev Notes
└─ 05-wbs/                               # ★ PMI WBS (컴포넌트·일정, 에픽/스토리에서 파생)
   ├─ index.md                         # 컴포넌트 트리·100%rule·스토리 일정·gantt·마일스톤
   └─ appendix-phases.md               # 전략 로드맵 연계
```

> 폐지: `02-usecase/`(UJ는 PRD §5로), `01-overview/`·`00-brief/`(미생성).

---

## 5. Frontmatter 스키마

### 5.1 Capability Feature (`01-prd/features/CF-N-*.md`)

```yaml
---
id: CF-N                          # CF-1..N, stable (사용자 가치 단위)
title: ...
status: implemented | implemented-mvp | planned | draft
jtbd: [JTBD-1, JTBD-3]            # source-strategy-brief Jobs-to-be-Done
maps_modules: [F1, F4]            # 구 코드 단위 (component-source-map과 일치)
source_paths: [pkg/...]
implements_uj: [UJ-1]             # PRD §5 User Journey (traceability §1과 일치)
covered_by_wbs: [WBS-1.N]         # traceability.md §2와 일치
fr_ids: [FR-CFN.1, FR-CFN.2]      # 이 CF의 FR (index §7 Coverage Map과 일치)
updated: YYYY-MM-DD
caveats: "..."                    # 선택
open_items: [...]                 # 선택
---
```

> User Journey(UJ)는 **PRD `index.md` §5에 내장**(별도 파일·frontmatter 없음). FR 한 줄(고객 voice)+매핑은 index §7, FR별 G/W/T + 구현 근거는 CF 파일.

### 5.2 에픽 (`03-epics/epic-N-*.md`)

```yaml
---
id: EPIC-N                        # 에픽 = 1 CF (사용자 가치)
title: ...
type: epic
covers_feature: CF-N
maps_wbs: WBS-1.x                 # 대응 WBS 컴포넌트
realizes_uj: [UJ-x]
stories: [N.1, N.2, ...]          # 소속 스토리 IDs (→ 04-stories/)
status: implemented | implemented-mvp | planned
updated: YYYY-MM-DD
---
```
> 에픽 본문 = 목표(Goal) + **스토리 목록 표**(→ `04-stories/`). **스토리 본문 인라인 금지**(별도 파일).

### 5.3 스토리 (`04-stories/{epic}.{story}.story.md`)

```yaml
---
id: STORY-N.M
epic: EPIC-N
covers_feature: CF-N
fr: FR-CFn.m                      # 1 스토리 ↔ 1 FR
wbs_component: WBS-1.x
status: done | planned
updated: YYYY-MM-DD
---
```
> 본문 = BMAD 스토리 템플릿: `Status` · `## Story`(서술형 "{역할}는 ~한다. (목적:…)") · `## Acceptance Criteria`(FR 인수기준) · `## Tasks / Subtasks`(코드 기준 도출, AC 참조) · `## Dev Notes`(소스 경로·테스트). 영한 혼용 금지(§2.8).

### 5.4 ADR (Architecture 단계 진입 시) — `02-architecture/adr/ADR-NNN-*.md`
`id / title / status / date / deciders / supersedes / superseded_by / updated`. (현재 미생성.)

---

## 6. Traceability 검증

`_shared/traceability.md`가 매핑의 진실의 원천. **CF 축**으로 CF×UJ×WBS를 묶고, JTBD·코드(모듈)도 부속표로 둔다.

### 6.1 검증 체크리스트 (변경 시)
- [ ] `CF-N.md`의 `implements_uj`·`covered_by_wbs`·`fr_ids` ↔ traceability §1·§2·§6
- [ ] `index.md` §5 UJ 목록 ↔ traceability §1·§3 (UJ는 PRD 내장)
- [ ] `WBS-1-N.md`의 `covers_features`(CF) ↔ traceability §2
- [ ] 새 코드(모듈)는 traceability §5 + component-source-map에 반영 (CF·WBS 매핑 포함)
- [ ] `04-stories/N.M.story.md`의 `epic`·`fr` ↔ traceability §6.1 + 에픽 `stories` 목록

### 6.2 desync 시
1. `traceability.md` 먼저 수정 → 2. 영향 frontmatter 일괄 갱신 → 3. 양방향 link 확인.

---

## 7. TODO 마커 규칙

미작성은 `TODO` / `TODO — 설명` (`grep -rn "^TODO\|: TODO\|- TODO" docs/spec/`). 완료 시 제거, 부분은 `TODO (partial: 남은 것)`. 미해결 follow-up은 frontmatter `open_items:`.

---

## 8. HTML / Markdown 정책

### 8.1 역할 분리
| | 역할 | 특성 |
|---|---|---|
| **`.md`** | source of truth · LLM·개발자 소비 | 풍부·완전 (FR·구현 근거·전체 상세) |
| **`.html`** | 사람용 뷰 (보고·열람) | 큐레이션·슬림 (핵심 노출 + `<details>` 접기) |

- 자동 빌드 아닌 큐레이션 뷰(깊이 달라도 됨). 1:1 동기 불필요, 사실 모순만 금지.
- 링크: html→html은 `.html`, _foundation 등 HTML 없는 대상은 `.md`.

### 8.2 빌드 베이스
- CSS: `_shared/design-tokens.css` 토큰(인라인). 다이어그램: Mermaid v11 ESM CDN 1줄(`<pre class="mermaid">`), 라벨에 `()`·`/`·`+` 있으면 `ID["라벨"]` 따옴표, 점선 `A -. text .-> B`. JS 외부 의존 0, `@media print` 포함.

### 8.3 톤앤매너 (요약본)
핵심만 노출 → 곁가지는 `<details class="expand">`. 나열은 `.item`. 상세 링크는 `.html`.

### 8.4 금기
AI slop 회피: Tailwind CDN · Inter+그라데이션 · transition all · dark/light 토글 등 금지.

---

## 9. 작성 순서 권장

1. **`_foundation/source-strategy-brief.md`** — top-down 출발점. ★ 존재.
2. **`01-prd/index.md` + `features/CF-*.md`** — Feature Map + Coverage Map 먼저(BMAD step-02 = 승인 게이트), 그 뒤 FR 상세(step-03). ★ 완료.
3. **`03-epics/`(에픽: 목표+스토리목록) + `04-stories/`(스토리 21: 서술형+인수기준+Tasks) + `05-wbs/`(PMI WBS: 스토리 파생 일정)** — 에픽≠WBS. ★ 완료.
4. **`_shared/traceability.md`** — CF×UJ×WBS. ★ 완료.
5. **HTML 뷰 생성** — PRD(index+CF×N) + WBS. §8 참조.
6. (후행) **Architecture** — C4 + ERD/데이터모델 (BMAD Solutioning).

---

## 10. 변경 시 절차 (PR/커밋 체크리스트)

1. **코드 변경** → 영향 `.md`를 `source_paths`/`구현 근거` 역색인: `grep -rln "변경경로" docs/spec/`
2. **frontmatter 갱신**: `status:`, `updated:` 오늘.
3. **traceability.md 갱신**: 영향 매핑(CF×UJ×WBS) + §5 코드(모듈)↔CF.
4. **index §7 Coverage Map ↔ CF `fr_ids` 정합** 확인.
5. **TODO 검토** + 본 규칙 위배 확인.
6. **HTML 재빌드** (해당 산출물 변경 시).

---

## 11. 참조 문서

- [`_foundation/source-strategy-brief.md`](_foundation/source-strategy-brief.md) — ★ 원본 사업 전략서
- [`_foundation/baseline.md`](_foundation/baseline.md) — as-built 사실 (변경표면 +12.6k LOC · 모듈↔CF)
- [`_shared/component-source-map.md`](_shared/component-source-map.md) — 6컴포넌트↔코드↔F0~F8, drift
- BMAD-METHOD `bmm-skills/2-plan-workflows/bmad-prd` + `3-solutioning/bmad-create-epics-and-stories`
- 한국 SI 산출물 문체·양식 (§2.8 변환 규칙 근거): CBD SW 표준 산출물 관리 가이드(cisp.or.kr/kpmo.or.kr) · NIPA SW 산출물 작성 가이드(swbank.kr) · 행정기관 정보화사업 추진 매뉴얼(정부통합전산센터 nirs.go.kr)

---

## 12. 본 파일의 진화

- 2026-06-08: **스토리 별도 파일화 + WBS 스토리 파생**. 에픽=목표+스토리목록(slim), 스토리=`04-stories/{N.M}.story.md` 21개(서술형·인수기준·Tasks, BMAD 템플릿), WBS 항목은 에픽/스토리에서 파생(완료 항목 유지). 스토리 영한 혼용 금지(§2.8).
- 2026-06-08: **BMAD 프레임워크 install 안 함 — 철학만 차용**. `npx bmad-method install`(`_bmad/`·dev-story→code-review 루프·`_bmad-output/`)은 *전방위 코드생성* 프레임워크라, **reverse-eng + 한국 SI 산출물** 목적엔 부적합·중복. 본 PROCESS.md가 BMAD config-of-record 역할(PRD→Feature→FR·샤딩·Spec-by-Example 인코딩). → 재도입 제안 불필요.
- 2026-06-08: **BMAD 정렬** — 산출물 PRD(03)+WBS(04) 2종으로 축소, UJ를 PRD 내장(02-usecase 폐지), 01-overview·00-brief 미생성. Architecture는 Solutioning 단계 후행.
- 2026-06-08(초): top-down(source-strategy-brief) + CF 축 + FR 고객 voice + hybrid 범위 전환. 구 F0~F8 모듈 축 폐기(코드 매핑으로만 잔존).
