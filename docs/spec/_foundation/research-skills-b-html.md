# 01-B. HTML 산출물 빌드 패턴 리서치

> Agent B 결과물 (분할). A·C 완료 후 `research-skills.md`로 통합 예정.
> 작성일: 2026-05-28

## 핵심 발견 (TL;DR)

**Anthropic의 Thariq Shihipar가 공개한 [html-effectiveness](https://thariqs.github.io/html-effectiveness/)는 그냥 "HTML이 좋다"는 주장이 아니라 _구체적 디자인 시스템 한 벌_을 보여줍니다.** 20개 예시 전부가 동일한 CSS 변수 팔레트(ivory/clay/oat/olive)·동일한 폰트 스택(`ui-serif`/`system-ui`/`ui-monospace`)·동일한 레이아웃 그리드를 공유합니다. zero deps, no build, no CDN. **DS-APM 4종 산출물의 베이스로 이 시스템을 그대로 흡수하는 것이 1순위 권장**입니다.

다이어그램은 Mermaid v11 (CDN ESM) 1순위, 인라인 SVG 2순위. PlantUML/D2/Excalidraw는 단일 파일 self-contained 요구사항을 깨므로 비추천.

---

## 1. thariqs/html-effectiveness 실측 패턴 분석

[github.com/ThariqS/html-effectiveness](https://github.com/ThariqS/html-effectiveness) 리포지토리의 실제 파일들(`12-incident-timeline.html`, `15-research-concept-explainer.html`, `16-implementation-plan.html`, `index.html`)을 직접 fetch해 확인한 내용입니다.

### 1.1 공통 디자인 토큰 (Anthropic Claude 브랜드 팔레트)

모든 예시 파일이 동일한 `:root` 변수를 씁니다:

```css
:root {
  --ivory:    #FAF9F5;   /* 페이지 배경 (warm off-white) */
  --slate:    #141413;   /* 본문 다크 텍스트 */
  --clay:     #D97757;   /* 액센트 (Anthropic의 brand orange) */
  --oat:      #E3DACC;   /* 보조 베이지 */
  --olive:    #788C5D;   /* "성공" 그린 */
  --sky:      #6A8CAF;   /* 보조 블루 */
  --gray-150: #F0EEE6;
  --gray-300: #D1CFC5;
  --gray-500: #87867F;
  --gray-700: #3D3D3A;

  --serif: ui-serif, Georgia, 'Times New Roman', serif;
  --sans:  system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif;
  --mono:  ui-monospace, 'SF Mono', Menlo, Monaco, monospace;
}
```

**핵심 관찰**: 헤더는 `serif`, 본문은 `sans`, 라벨/배지/코드는 `mono`로 강하게 분리. clay 컬러는 액센트(링크, 강조, "in-progress" dot)에만 등장하고 본문에 침범하지 않음.

### 1.2 공통 레이아웃 패턴

| 요소 | 값 |
|------|-----|
| 페이지 max-width | `1100~1120px` |
| body padding | `56px 32px 120px` (위, 좌우, 아래) |
| 본문 line-height | `1.55~1.65` |
| h1 크기 | `clamp(38px, 5.4vw, 62px)` (index), 정적 33~38px (서브) |
| 본문 폰트 크기 | `14.5~16.5px` |
| 카드 border | `1.5px solid var(--gray-300)` |
| 카드 border-radius | `10~14px` |
| 섹션 간격 | `margin-bottom: 64px` |
| 컬러 폰트-스무딩 | `-webkit-font-smoothing: antialiased` |

### 1.3 반복 사용되는 컴포넌트 패턴

**(A) Eyebrow + Serif H1 헤더** (모든 페이지 공통):
```html
<div class="eyebrow">Implementation plan · Acme web client</div>
<h1>Comment threads on task cards</h1>
```

**(B) Section 번호 헤더**:
```html
<div class="sec-head">
  <span class="num">01</span>
  <h2>Milestones</h2>
</div>
<p class="sec-intro">Ship in four slices, each independently reviewable...</p>
```

**(C) Summary 4-셀 그리드** (Overview 산출물에 그대로 쓸 수 있음):
```html
<div class="summary">
  <div class="cell"><div class="k">Effort</div><div class="v accent">~2 weeks</div></div>
  <div class="cell"><div class="k">Surfaces touched</div><div class="v">3 packages</div></div>
  ...
</div>
```
CSS: `grid-template-columns: repeat(4, 1fr)`, 모바일에서는 `repeat(2, 1fr)`.

**(D) Milestone 타임라인** (WBS에 직접 활용 가능):
```html
<div class="milestone">
  <div class="when">Week 1 · Mon–Tue</div>
  <div class="dot-col"><span class="dot done"></span><span class="line"></span></div>
  <div class="body">
    <h3>Schema & API contract</h3>
    <p>...</p>
    <div class="tags"><span class="tag">packages/db</span></div>
  </div>
</div>
```
CSS는 `grid-template-columns: 120px 28px 1fr`, dot/line이 세로축 진행을 만듦. `.dot.done`은 olive로 채워짐.

**(E) Risk 테이블** (3-컬럼 grid, 행마다 severity 배지):
```html
<div class="risks">
  <div class="row">
    <div class="cell">Realtime duplicate: socket append races...</div>
    <div class="cell"><span class="sev high">HIGH</span></div>
    <div class="cell">Dedupe on server-assigned id...</div>
  </div>
</div>
```
배지 컬러: `sev.high`(연어색 배경), `sev.med`(oat), `sev.low`(연한 olive).

**(F) Open Questions 카드** (clay 좌측 보더):
```css
.q { border-left: 4px solid var(--clay); border-radius: 10px; padding: 16px 20px; }
```

**(G) Sidebar Glossary** (15번 예시):
```css
aside {
  position: sticky;
  top: 32px;
  align-self: start;
}
.page {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 240px;
  gap: 48px;
}
```
모바일(`max-width: 960px`)에서는 1컬럼 + `aside { order: 2; }`.

### 1.4 다이어그램은 거의 100% 인라인 SVG

`16-implementation-plan.html`의 데이터플로우 다이어그램 — Mermaid 없이 **순수 인라인 SVG**:
- `<defs><marker id="arrow">` 으로 화살촉 정의
- `<rect>` 박스 + `<text>` 라벨
- 실선 = HTTP 요청/응답, 점선 clay 컬러 (`stroke-dasharray="5 4"`) = 비동기/이벤트
- 곡선 `<path d="M660 162 C 540 120, 280 120, 205 162">` 로 reconcile 흐름

**15번 예시(consistent hashing)는 vanilla JS로 동적 SVG 생성**: `<input type="range">`로 노드 수 조절, JS가 `<circle>`/`<path>`를 그림. 외부 라이브러리 zero.

### 1.5 자바스크립트 패턴

| 패턴 | 구현 |
|-----|------|
| 슬라이더 → 시각화 갱신 | `input.oninput = () => render()` (vanilla) |
| 용어 hover → glossary 강조 | `data-term` 속성 + `mouseenter` → glossary `dt`에 `.hl` 클래스 |
| `details/summary` | 사용은 적음, 보통 정적 펼침 |
| 다크모드 | **예시들은 다크모드 없음** — light-only로 출하 |

### 1.6 의존성

확인 결과: **모든 예시 파일이 외부 CDN 0개, npm 0개, build step 0개**. 시스템 폰트만 사용. CSS+JS 인라인 only. 가장 큰 파일도 30KB 미만.

---

## 2. 다이어그램 라이브러리 비교

DS-APM 콘텐츠 요구사항: 아키텍처 다이어그램(Overview), 상태 전이/시퀀스(Use Case), 모듈 의존 그래프(WBS).

| 라이브러리 | self-contained 단일 파일 가능 | 번들 크기 | 지원 다이어그램 | 학습곡선 | 시각 품질 | 추천도 |
|-----------|---------|----------|----------------|--------|---------|--------|
| **Mermaid v11** | YES (CDN ESM 1줄) | full ~2.8MB unminified, tiny 패키지 약 절반 ([sidharth.dev](https://www.sidharth.dev/posts/shrinking-mermaid/)) | flowchart, sequence, state, class, ER, C4, gantt, gitgraph, mindmap, timeline, sankey, block, pie, quadrant, requirement, journey, xychart, architecture (20+) | 낮음 (LLM 친숙도 압도적) | 무난, theme 커스텀 가능 | **1순위** (다이어그램 종류 다양 필요 시) |
| **인라인 SVG (no lib)** | YES (네이티브) | 0 KB | 모든 그림 (직접 그림) | 중간 (좌표 계산) | 디자인 일관성 최고 | **1순위** (html-effectiveness가 채택한 방식) |
| **nomnoml** | YES (graphre + nomnoml 두 스크립트, ~50KB+) | 약 graphre 의존 | 클래스/컴포넌트/플로우차트 (state는 classifier 토큰만, 부족함) | 낮음 | 손그림 느낌 (호불호) | 보조용으로만 |
| **D2** | NO (서버 렌더 or WASM 추가 작업) | N/A in browser | 모두 가능, 자동 레이아웃 우수 | 중간 | 최고 수준 | **비추천** (단일 파일 깨짐) |
| **PlantUML** | 부분적 (plantuml.js는 CheerpJ 사용, `file://`로 안 열림 — 로컬 웹서버 필수) | 매우 큼 (Java 포팅) | 가장 다양 | 중간 | 클래식 UML | **비추천** (file:// 깨짐) |
| **Excalidraw** | NO (React + 폰트 에셋 디렉토리 필수) | 큼 | 손그림 화이트보드 | 낮음 | 손그림 캐주얼 | **비추천** |

**결론**: html-effectiveness가 보여준 _인라인 SVG_가 시각적 일관성과 자유도에서 최강이지만 직접 좌표를 짜는 비용이 큼. **DS-APM의 8개 모듈 명세서나 시퀀스 다이어그램처럼 구조가 정형화된 곳은 Mermaid v11 ESM 1줄 임포트로 처리하고, Overview의 아키텍처 다이어그램이나 분기점 시각화처럼 디자인 톤이 중요한 곳은 인라인 SVG로** 하이브리드.

---

## 3. 추천 도구 스택

### 1순위 (DS-APM 4종 통일 베이스)
- **CSS 디자인 시스템**: html-effectiveness의 `:root` 토큰 + 시스템 폰트 스택 그대로 채택
- **레이아웃**: CSS Grid (`max-width: 1120px`, `margin: 0 auto`)
- **JS**: vanilla (외부 의존 0)
- **다이어그램**:
  - 단순 박스+화살표/타임라인/상태도 → **인라인 SVG**
  - 복잡한 시퀀스/ER/state machine → **Mermaid v11 ESM (CDN 1줄)**
- **테이블/카드/배지**: html-effectiveness 패턴 그대로
- **다크모드**: **v1은 light-only 출하 (예시들과 일치)**, v2에서 `prefers-color-scheme` + `light-dark()` 도입

### 2순위 (대체안)
- Mermaid 대신 nomnoml만 쓰면 → 클래스/모듈 다이어그램은 OK지만 시퀀스/상태 부족
- 인라인 SVG가 부담스러우면 → 모든 다이어그램 Mermaid로 통일 (단, 톤이 깨질 수 있음)

---

## 4. 콘텐츠 → 도구 매핑

| 산출물 | 콘텐츠 블록 | 권장 패턴 |
|--------|------------|-----------|
| **Overview** | 한 페이지 요약 | html-effectiveness의 4-cell `summary` 그리드 (Effort / Modules / Risks / Status) |
| | 아키텍처 다이어그램 | **인라인 SVG** (16번 예시처럼 `<defs>` 마커 + `<rect>` + 곡선 path) |
| | 분기점 표 | `risks` 패턴 3-컬럼 (조건 / 결정 / 영향), `sev` 배지 |
| **Use Case** | 상태 전이도 | **Mermaid `stateDiagram-v2`** (노드 수 많고 분기 복잡할 때) 또는 인라인 SVG (단순할 때) |
| | 시퀀스 다이어그램 | **Mermaid `sequenceDiagram`** (가장 LLM 친숙) |
| | 채널 페이로드 before/after | 2-컬럼 grid + `code` 패널, 라벨로 BEFORE/AFTER 표기 |
| | PII redact diff | `<pre>` + 빨강/초록 토큰 컬러 (`.del { background: #F3D9CC; }`, `.add { background: #E4E9DC; }`) |
| | 에러 분기 | "Open Questions" 카드 패턴 (clay 좌측 보더) 변형 — `q` → `branch` 클래스로 |
| **기능명세서 (8 모듈)** | 모듈별 동일 템플릿 | section 5개 고정 (인터페이스 / 데이터 모델 / 상태 전이 / 예외 / 비기능), 각 section에 번호 배지 |
| | 인터페이스 | 표 (메서드 / 인풋 / 아웃풋 / 비고) + `mono` 폰트 |
| | 데이터 모델 | code 패널 (SQL/타입), `code .kw` / `.fn` / `.str` 토큰 컬러 |
| | 상태 전이 | Mermaid `stateDiagram-v2` 또는 인라인 SVG |
| | 예외 | risks 패턴 (예외 / 심각도 / 처리) |
| | 비기능 | summary 셀 그리드 (SLA / 처리량 / 보안) |
| | 8개 모듈 네비 | sticky aside (15번 예시처럼) 또는 상단 chip 목록 |
| **WBS** | phase 타임라인 | Milestone 컴포넌트 그대로 (`grid: 120px 28px 1fr`, dot/line) |
| | 의존 그래프 | **Mermaid `flowchart LR`** (자동 레이아웃 가치 큼) |
| | deliverable matrix | `risks` 4-컬럼 변형 (deliverable / owner / phase / status) |

---

## 5. 샘플 패턴 (카피 가능 스니펫)

### 5.1 디자인 토큰 (전 산출물 공통, 최상단)

```css
:root {
  --ivory:#FAF9F5; --slate:#141413; --clay:#D97757; --oat:#E3DACC;
  --olive:#788C5D; --sky:#6A8CAF;
  --g100:#F0EEE6; --g300:#D1CFC5; --g500:#87867F; --g700:#3D3D3A;
  --serif: ui-serif, Georgia, 'Times New Roman', serif;
  --sans:  system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif;
  --mono:  ui-monospace, 'SF Mono', Menlo, Monaco, monospace;
}
* { box-sizing: border-box; margin: 0; padding: 0; }
body {
  background: var(--ivory); color: var(--g700);
  font-family: var(--sans); line-height: 1.55;
  padding: 56px 32px 120px; -webkit-font-smoothing: antialiased;
}
.page { max-width: 1120px; margin: 0 auto; }
```

### 5.2 Section 헤더 (번호 배지 + serif h2)

```css
.sec-head { display: flex; align-items: baseline; gap: 14px; margin-bottom: 8px; }
.sec-head .num {
  font-family: var(--mono); font-size: 12px;
  background: var(--oat); color: var(--slate);
  padding: 3px 9px; border-radius: 8px;
}
.sec-head h2 {
  font-family: var(--serif); font-weight: 500;
  font-size: 26px; color: var(--slate); letter-spacing: -0.01em;
}
.sec-intro { font-size: 14.5px; color: var(--g500); max-width: 720px; margin-bottom: 28px; }
```

### 5.3 Milestone 타임라인 (WBS 직접 사용)

```css
.milestones { display: flex; flex-direction: column; }
.milestone {
  display: grid; grid-template-columns: 120px 28px 1fr; gap: 0 18px;
}
.milestone .when { text-align: right; font-family: var(--mono); font-size: 12px; color: var(--g500); padding-top: 4px; }
.milestone .dot-col { display: flex; flex-direction: column; align-items: center; }
.milestone .dot {
  width: 14px; height: 14px; border-radius: 50%;
  background: #fff; border: 3px solid var(--clay); margin-top: 4px;
}
.milestone .dot.done { background: var(--olive); border-color: var(--olive); }
.milestone .line { width: 2px; flex: 1; background: var(--g300); margin: 4px 0; }
.milestone:last-child .line { display: none; }
.milestone .body { padding-bottom: 36px; }
```

### 5.4 Severity 배지 (예외/리스크용)

```css
.sev { display: inline-block; font-family: var(--mono); font-size: 11px;
       padding: 2px 8px; border-radius: 6px; font-weight: 600; }
.sev.high { background: #F3D9CC; color: #8A3B1E; }
.sev.med  { background: var(--oat); color: var(--slate); }
.sev.low  { background: #E4E9DC; color: #4B5C39; }
```

### 5.5 Mermaid 임포트 (1줄)

```html
<script type="module">
  import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
  mermaid.initialize({
    startOnLoad: true,
    theme: 'base',
    themeVariables: {
      primaryColor: '#FAF9F5',
      primaryTextColor: '#141413',
      primaryBorderColor: '#D1CFC5',
      lineColor: '#87867F',
      secondaryColor: '#E3DACC',
      tertiaryColor: '#F0EEE6'
    },
    fontFamily: 'ui-sans-serif, system-ui, sans-serif'
  });
</script>
<pre class="mermaid">
stateDiagram-v2
  [*] --> Idle
  Idle --> Validating: payload received
  Validating --> Persisted: ok
  Validating --> Rejected: schema fail
  Persisted --> [*]
</pre>
```

Mermaid를 themeVariables로 html-effectiveness 팔레트에 맞춰야 톤이 안 깨집니다.

### 5.6 인라인 SVG 박스+화살표 (16번 예시 축약)

```html
<svg viewBox="0 0 860 200" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5"
            markerWidth="7" markerHeight="7" orient="auto-start-reverse">
      <path d="M0,0 L10,5 L0,10 z" fill="#87867F"/>
    </marker>
  </defs>
  <g font-size="12" fill="#141413" font-family="ui-sans-serif">
    <rect x="20" y="20" width="180" height="54" rx="10"
          fill="#FFFFFF" stroke="#D1CFC5" stroke-width="1.5"/>
    <text x="110" y="43" text-anchor="middle" font-weight="600">Ingest</text>
    <text x="110" y="60" text-anchor="middle" fill="#87867F" font-size="10.5">OTLP HTTP</text>
  </g>
  <path d="M200 47 L340 47" stroke="#87867F" stroke-width="1.5" fill="none" marker-end="url(#arrow)"/>
  <!-- 점선 비동기 흐름은 stroke-dasharray="5 4" + clay 컬러 -->
</svg>
```

### 5.7 `details/summary`로 접힘 섹션 (zero JS)

```html
<details class="expand">
  <summary>예외 케이스 상세 (12건)</summary>
  <div class="body">...</div>
</details>
<style>
  details.expand { border: 1.5px solid var(--g300); border-radius: 10px; padding: 14px 18px; margin-bottom: 12px; background: #fff; }
  details.expand summary { cursor: pointer; font-weight: 600; color: var(--slate); }
  details.expand[open] { background: var(--g100); }
  details.expand .body { margin-top: 12px; font-size: 13.5px; color: var(--g700); }
</style>
```

### 5.8 Print/PDF 호환 스타일 (필수)

```css
@page { size: A4; margin: 18mm 15mm; }
@media print {
  body { background: #fff; padding: 0; }
  .page { max-width: 100%; }
  section { break-before: page; page-break-before: page; }
  section:first-of-type { break-before: auto; page-break-before: auto; }
  .milestone, .risks .row, .q, .summary .cell,
  pre, table, svg { break-inside: avoid; page-break-inside: avoid; }
  details > summary { list-style: none; }
  details:not([open]) > *:not(summary) { display: none; }
  aside { position: static; }
  a::after { content: " (" attr(href) ")"; font-size: 10px; color: #888; }
}
```

`break-*`를 우선, `page-break-*`를 폴백으로.

### 5.9 다크모드 (v2 추가 시) — light-dark() 함수

```css
:root {
  color-scheme: light dark;
  --bg:   light-dark(#FAF9F5, #1A1A18);
  --fg:   light-dark(#141413, #E8E6DE);
  --card: light-dark(#FFFFFF, #232320);
  --border: light-dark(#D1CFC5, #3D3D3A);
  --accent: light-dark(#D97757, #E8906F);
}
body { background: var(--bg); color: var(--fg); }
```

### 5.10 본문 sticky TOC (모듈명세서 8개 모듈 네비)

```html
<div class="page">
  <main>...본문...</main>
  <aside class="toc">
    <div class="label">Modules</div>
    <ol>
      <li><a href="#m-ingest">01 · Ingest</a></li>
      <li><a href="#m-route">02 · Route</a></li>
    </ol>
  </aside>
</div>
<style>
  .page { display: grid; grid-template-columns: minmax(0,1fr) 220px; gap: 48px; }
  .toc { position: sticky; top: 32px; align-self: start;
         border: 1.5px solid var(--g300); border-radius: 12px;
         background: #fff; padding: 16px; }
  .toc ol { list-style: none; padding: 0; counter-reset: m; }
  .toc li { font-size: 13px; padding: 6px 0; }
  .toc a { color: var(--g700); text-decoration: none; }
  .toc a:hover { color: var(--clay); }
  @media (max-width: 960px) { .page { grid-template-columns: 1fr; } .toc { position: static; } }
  html { scroll-behavior: smooth; }
</style>
```

---

## 6. 금기 패턴 (AI Slop 회피 + Self-contained 위배)

| 패턴 | 왜 금기인가 |
|------|------------|
| **Tailwind CDN 임포트** | 200KB+ 부담, single-file 정신 위배, 톤도 generic해짐 |
| **Inter 폰트 + 보라/파랑 그라데이션** | 전형적 AI slop. `system-ui`/`ui-serif`가 훨씬 인상적 |
| **Font Awesome / Material Icons CDN** | 외부 의존. 필요하면 인라인 SVG 아이콘 (Lucide 등에서 복사) |
| **모든 카드에 다른 그림자** | "depth cues behave like decoration" — html-effectiveness는 그림자 거의 안 씀, `border` 1.5px로 통일 |
| **PlantUML / D2 / Excalidraw 임베드** | `file://`로 안 열리거나 React/폰트 디렉토리 필요. self-contained 깨짐 |
| **Mermaid `theme: 'default'` 그대로 사용** | 보라색 + Inter로 톤 깨짐. 반드시 `themeVariables`로 clay/oat 팔레트로 오버라이드 |
| **Markdown 같은 `>` blockquote** | HTML에서는 의미 없음. `.q` 좌측 보더 카드가 훨씬 효과적 |
| **버튼/링크 `transition: all` 0.3s ease** | 너무 무거움, "snap" 느낌이 AI 같음. `transition: background 150ms` 정도로 짧게 |
| **다크모드 + 라이트모드 토글 v1 출시** | 디자인 결정 2배. v1은 light-only (Anthropic 예시들도 light-only) |
| **`<table>` 대신 `<div>`로 3-컬럼 grid를 짤 때 헤더 누락** | 접근성/인쇄/스크린리더 깨짐. 진짜 표는 `<table>` 쓰기 |
| **CDN 이모지를 헤더 장식으로** | 톤 깨짐. `eyebrow::before { content: ""; width: 24px; height: 1.5px; background: var(--clay); }` 같은 미니멀 장식 사용 |
| **`em`/`%` 단위로 print 마진 지정** | PDF 엔진마다 다름. `mm`/`cm`/`in` 절대단위 |
| **자동 fade-in 스크롤 애니메이션** | 인쇄 시 텍스트 안 보임. v1에 넣지 말 것 |

---

## 7. 참고 링크

### Primary
- [The unreasonable effectiveness of HTML — examples (thariqs.github.io)](https://thariqs.github.io/html-effectiveness/) — 원전 갤러리 20개
- [ThariqS/html-effectiveness (GitHub)](https://github.com/ThariqS/html-effectiveness) — 리포지토리, Apache-2.0
- [Simon Willison: Using Claude Code: The Unreasonable Effectiveness of HTML](https://simonwillison.net/2026/May/8/unreasonable-effectiveness-of-html/)
- [HTML vs Markdown for AI Agents (beam.ai)](https://beam.ai/agentic-insights/html-vs-markdown-which-format-actually-makes-ai-agents-more-useful)

### Diagram Libraries
- [Mermaid official](https://mermaid.js.org/) / [Usage](https://mermaid.js.org/config/usage.html) / [Diagram types](https://deepwiki.com/mermaid-js/mermaid/3-diagram-types)
- [Mermaid bundle size analysis (sidharth.dev)](https://www.sidharth.dev/posts/shrinking-mermaid/)
- [Text-to-diagram comparison: D2 vs Mermaid vs PlantUML](https://text-to-diagram.com/?example=text)
- [nomnoml GitHub](https://github.com/skanaar/nomnoml) / [nomnoml.com](https://www.nomnoml.com/)
- [plantuml.js GitHub](https://github.com/plantuml/plantuml.js/)

### CSS Patterns
- [MDN: prefers-color-scheme](https://developer.mozilla.org/en-US/docs/Web/CSS/@media/prefers-color-scheme)
- [CSS-Tricks: Complete Guide to Dark Mode on the Web](https://css-tricks.com/a-complete-guide-to-dark-mode-on-the-web/)
- [Print CSS Cheatsheet (customjs.space)](https://www.customjs.space/blog/print-css-cheatsheet/)
- [CSS print styles for PDF generation (pdf4.dev)](https://pdf4.dev/blog/css-print-styles-pdf-guide)

### AI Slop Avoidance
- [925studios: AI Slop Web Design Complete Guide 2026](https://www.925studios.co/blog/ai-slop-web-design-guide)
- [MindStudio: How to Avoid AI Slop When Using Claude Design](https://www.mindstudio.ai/blog/claude-design-avoid-ai-slop-design-system)

---

## 실행 권장사항 (DS-APM 적용)

1. **공통 베이스 CSS 파일 한 벌 합의** — §5.1~5.4 토큰과 컴포넌트를 4종 산출물 템플릿에 박제
2. **다이어그램 정책 명문화**: "상태도/시퀀스는 Mermaid v11, 아키텍처는 인라인 SVG, 두 도구 외 사용 금지"
3. **Mermaid theme override는 한 번만 정의** (§5.5)
4. **인쇄 스타일을 처음부터 넣기** (§5.8)
5. **v1은 light-only로 출하**. 다크모드는 v2 follow-up
6. **8개 모듈 명세서는 단일 HTML 1개에 sticky aside + anchor 링크**로 통합 권장
