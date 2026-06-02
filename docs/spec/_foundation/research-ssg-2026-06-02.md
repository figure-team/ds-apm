---
id: RESEARCH-SSG-2026-06-02
title: 산출물 HTML 생성 전략 리서치 — SSG vs md→HTML 변환기 vs LLM 생성(thariqs 계보)
type: research
date: 2026-06-02
status: living
audience: 다음 세션 / SSG 결정 담당
updates: research-skills-b-html.md (§핵심발견 "thariqs 1순위 권장" 갱신)
related: handoff-2026-06-02.md §3.1
updated: 2026-06-02
---

# 산출물 HTML 생성 전략 리서치 — 2026-06-02

> **목적**: handoff §3.1 "SSG 도입, 더 비교" 요청에서 출발한 조사 결과를 결정 보류 상태로 박아둔다.
> **결론은 아직 미확정** (사용자: "결정 보류, 더 볼 것"). 본 문서는 결정 컨텍스트 보존용 audit 자료.
> **선행 자료**: `research-skills-b-html.md` (2026-05-28, thariqs html-effectiveness를 "1순위 권장"으로 결론). 본 문서가 그 결론을 **용도 분리로 정정**.

---

## 0. TL;DR — 기존 결론을 바꾸는 3가지 사실

1. **축이 틀렸었다.** "어떤 docs SSG(MkDocs vs Starlight)냐"가 아니라 **두 철학 중 무엇이냐**의 문제다 — (A) md=source → 빌드 도구가 HTML 생성, (B) Claude가 self-contained HTML을 직접 생성(thariqs 계보). handoff §3.1·§3.2 전체가 A 진영을 암묵 전제로 짜여 있었다.
2. **handoff §3.1 "1위 추천 MkDocs Material"은 이미 낡았다.** Material for MkDocs는 **2025-11-05부터 유지보수 모드**(9.7.0이 마지막 기능 릴리스, 버그·보안 패치만 2026-11까지). 메인테이너는 차세대 SSG **Zensical**로 이동. 신규 프로젝트 1순위로 박을 도구 아님.
3. **thariqs가 이 흐름의 진원지다** (research-skills-b-html.md가 인용한 그 출처). 영향력이 커서 **Anthropic이 Claude Code 일부 산출물 기본값을 HTML로 전환**. §3.1이 thariqs를 "over-apply한 잘못된 축"으로 **전면 기각한 것은 과했다** — 기각이 아니라 **용도 분리**가 맞다.

---

## 1. 진짜 결정 축 — 두 철학 (양립 안 함)

| | **A. 빌드 파이프라인** (md=source → 도구 → HTML) | **B. LLM-as-generator** (thariqs 계보) |
|---|---|---|
| 생성기 | SSG/변환기 (mdbook·Marmite·marky·MkDocs…) | **Claude 자신** (`dogum/html-artifacts` 스킬), 빌드툴 0 |
| 다이어그램 | mermaid 원문 유지 → JS 또는 빌드타임 SVG | **inline SVG 직접 박음** (mermaid.js·CDN 0) |
| md source | 진실의 원천 유지, HTML은 산출물 | HTML이 1차 산출물 (md source 보존은 별도 관리) |
| 지속 편집 | md 한 줄 고치고 재빌드 | 매 편집 시 HTML 통째 재생성 |
| 일관성 | 전 문서 자동 동일 | 문서마다 Claude가 새로 짬 (드리프트 위험) |
| PROCESS와 정합 | **§5 원칙 4 "md=SSOT, HTML=산출물"과 일치** | md=SSOT 원칙과 충돌 (HTML이 1차가 되므로) |

> 두 철학 모두 **"CDN 의존 0 + mermaid이 SVG `<text>`로 남아 LLM이 읽음"** 이라는 §3.1 실질 요구는 충족한다. 핵심 차이는 **편집 모델**과 **md=truth 보존**.

---

## 2. thariqs 계보 (GitHub 실측)

1. **Thariq Shihipar (Anthropic Claude Code 팀)** — *"The Unreasonable Effectiveness of HTML"* (2026-05). HTML self-contained 산출물이 markdown을 구조적으로 이기는 **9개 카테고리**(비교표·구현계획·코드리뷰·디자인시스템·**다이어그램**·리포트·슬라이드·커스텀에디터…) 제시.
   - `ThariqS/html-effectiveness` = **도구가 아니라 예제 갤러리** (자체완결 `.html` 20개 + index, 빌드·의존성 0). research-skills-b-html.md가 베이스로 흡수 권장한 그 디자인 시스템.
2. **`dogum/html-artifacts`** (Apache 2.0) — thariqs 휴리스틱을 **Claude 스킬로 operationalize**. Claude가 적합한 작업에서 markdown 대신 self-contained HTML 직접 생성, 다이어그램은 **inline SVG**. 짧은 대화/코드전용/단순요약엔 carve-out. claude.ai·Claude Code·API 설치 가능. = B 진영의 실체 도구.

---

## 3. 후보 도구 지형 (2026-06 기준)

### 3.1 md→HTML 변환기 (사용자가 가리킨 "HTML로 만들어주는" 신규 OSS, A 진영 경량)

| 도구 | 언어/설치 | mermaid 처리 | 적합도 |
|---|---|---|---|
| **mdbook + mdbook-mermaid-ssr** | Rust 바이너리 | **빌드타임 SVG (headless chrome, JS·CDN 0)** | §3.1 "빌드타임 SVG" 이상형에 정확히 부합. nav·검색 무료 |
| **Marmite** | Rust 단일 바이너리, zero-config | mermaid 임베드(client-side) | .md 폴더 → 사이트 자동. 가장 단순 |
| **marky** | Rust `cargo install` 단일 바이너리 | mermaid+LaTeX 임베드(client-side) | 순수 "md→테마 HTML" 변환 |
| **Quarto** | Pandoc 기반(설치 무거움) | HTML=client JS / PDF=빌드타임 PNG | 장문·다국어·PDF 동시 필요 시 |
| **Pandoc** | 성숙·범용 | 필터로 mermaid | `--embed-resources --standalone` 단일 HTML |

### 3.2 풀 docs SSG 프레임워크 (handoff §3.1이 비교하던 축 — 우리엔 과함)
- **MkDocs Material** — **유지보수 모드(2025-11)**, 신규 1순위 부적합. 후속 = Zensical.
- **Astro Starlight** — Pagefind 검색 내장, i18n. Node 스택. State of JS 2025 docs SSG 만족도 최상위.
- **Fumadocs** — Next.js, **LLM-first 기능 내장**(llms.txt/llms-full.txt, `.md` append 시 raw 반환, AI search). 단 Next.js/React 스택 무거움. mermaid는 remark 플러그인.
- VitePress / Docusaurus / Nextra / rspress — 리서치상 우리 용도 우위 없음/과함.

---

## 4. 샘플 빌드 실측 (2026-06-02, throwaway)

`01-overview/index.md`(mermaid flowchart 2 + sequence 1)를 양 철학으로 빌드해 Tailscale 비교.
**샘플 산출물은 `docs/spec/_ssg-samples/`에 두되 `.gitignore` 처리(커밋 안 함, 평가용 일회성).**

| | Pure-B (B 진영) | Hybrid (A 진영) |
|---|---|---|
| 경로 | `_ssg-samples/pure-b/overview.html` | `_ssg-samples/hybrid/overview.html` |
| 생성기 | Claude 직접 (html-artifacts 방식) | mdbook + mdbook-mermaid-ssr |
| 다이어그램 | hand-authored inline SVG ×3 | mermaid 원문 → 빌드타임 SVG ×3 |
| 외부 의존 | **0** (20KB 단일 파일, JS·CSS·CDN 0) | CDN 0이나 **로컬 JS 번들 존재**(nav·검색·테마) |
| md=truth | HTML이 1차 (md 별도 관리) | **md 유지**, HTML은 산출물 |
| 검증 | 200, svg 3·text 56, console error 0 | 200, svg(diagram) 3·text 58, mermaid div 0, CDN 0 |

### 4.1 mdbook-mermaid-ssr 셋업 마찰 (PROCESS 박을 때 전제)
- **mdbook 0.5.x 필수** — 이 도구는 신규 split 크레이트 `mdbook-preprocessor 0.5.0`에 의존. (0.4.x로 내리면 "Unable to parse the input"으로 실패. 처음 0.4로 내린 건 오판이었음.)
- **시스템 chrome + `CHROME` env 필수** — `supports` 단계가 headless chrome을 띄워 검증하는데, chrome 못 찾으면 `Error: can't mermaid renderer for given renderer`로 **위장된 오류**. 본 환경은 puppeteer 캐시 chrome 사용: `CHROME=~/.cache/puppeteer/chrome/linux-148.0.7778.97/chrome-linux64/chrome`.
- **SSR 번들 mermaid가 엄격** — flowchart 노드 라벨의 괄호·`/`·`+`를 따옴표로 감싸야 렌더(`ID["라벨"]`). 기존 md 다이어그램 2개가 그래서 처음 "render returned null". 도입 시 기존 다이어그램 일괄 따옴표 처리 필요(일회성).

---

## 5. 추천 (미확정, 결정 보류 중)

우리 산출물은 둘로 갈리고, 이번 비교가 그 분리를 확증한다:

- **상세본 4종 (01~04)** — 계속 진화 + `md=SSOT`가 PROCESS §5·§8의 하드 요구 + 다이어그램 다수 → **A 진영(mdbook-mermaid-ssr)**. §3.1 "빌드타임 SVG" 이상형이 실제 구현됨. PROCESS §8("상세본 md-only")을 "상세본 md=source + SSG가 HTML 빌드"로 재개정 필요.
- **요약본 (00-brief 5종)** — 한 장짜리 전달용 → **B 진영(Claude 생성 self-contained HTML)** = thariqs 방식. 현 00-brief HTML이 사실상 이미 이 방식. research-skills-b-html.md의 디자인 시스템 흡수 권장은 **요약본에 한정하면 유효**.

→ research-skills-b-html.md의 "thariqs 1순위 권장(4종 전체 베이스)"은 **요약본에만 적용**으로 좁혀 정정.

---

## 6. 미해결 / 다음 단계

- [ ] **결정 보류** — 사용자가 "더 볼 것" 요청. 다음: **`04-wbs/index.md`의 gantt**(md-raw에서 텍스트로 깨진 게 §3.1 원래 발단)를 양 철학으로 빌드해 추가 비교.
- [ ] 채택 시 적용 범위 확정(용도 분리 vs 전체 통합) → `PROCESS.md §8` 재개정 + handoff §3.1 종결.
- [ ] handoff §3.2 수정배치 목록(milestone 통일 + WBS work package 일정)의 HTML 처리 방식이 SSG 결정에 종속.
- [ ] 샘플(`_ssg-samples/`, `hybrid-book/`)은 throwaway — 평가 끝나면 삭제 (현재 gitignore됨).

---

## 7. 출처

- thariqs: [html-effectiveness](https://github.com/ThariqS/html-effectiveness) · [dogum/html-artifacts](https://github.com/dogum/html-artifacts) · [Anthropic 전환 해설](https://pasqualepillitteri.it/en/news/2243/html-vs-markdown-claude-code-thariq-anthropic)
- MkDocs maintenance-mode: [docsio 리뷰](https://docsio.co/blog/mkdocs-material)
- 변환기: [mdbook-mermaid-ssr](https://github.com/commanderstorm/mdbook-mermaid-ssr) · [Marmite](https://github.com/rochacbruno/marmite) · [marky](https://github.com/metafates/marky) · [Quarto diagrams](https://quarto.org/docs/authoring/diagrams.html)
- SSG: [Starlight](https://docsio.co/blog/starlight-docs) · [Fumadocs LLM](https://www.fumadocs.dev/docs/integrations/llms) · [Fumadocs vs Nextra vs Starlight](https://www.pkgpulse.com/guides/fumadocs-vs-nextra-v4-vs-starlight-documentation-sites-2026)

---

## 8. 변경 기록

| 날짜 | 변경 | 사유 |
|---|---|---|
| 2026-06-02 | 최초 작성 | handoff §3.1 "더 비교" → 두 철학 재프레임·MkDocs maintenance-mode·thariqs 계보·샘플 2종 실측. 결정 보류로 자료 정리 |
