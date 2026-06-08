# DS-APM — Project Guide for Claude

본 파일은 매 conversation 시작 시 로드된다. 토큰 절약을 위해 짧게 유지하고, 상세 규칙은 서브 파일을 가리키는 pointer로 둔다. Claude는 해당 영역 작업을 시작할 때만 서브 파일을 Read한다.

---

## 산출물 작성 (`docs/spec/`)

DS-APM 산출물을 손대기 전에 **반드시 먼저** `docs/spec/PROCESS.md`를 Read해서 규칙을 따른다. 산출물 폴더 안의 어떤 `.md`라도 수정·생성하는 작업이면 본 절차 대상.

### 산출물 (BMAD 양식)
- **기능명세 = PRD** (`01-prd/`) — Vision·JTBD·**User Journey(UJ)**·Feature(CF)·FR. UJ는 PRD 내장.
- **에픽** (`03-epics/`) — BMAD 에픽(목표 + 스토리 목록). **스토리** (`04-stories/`) — 서술형 스토리 + 인수기준 + Tasks(별도 파일 `{N.M}.story.md`). **애자일** 작업 정의.
- **WBS** (`05-wbs/`) — PMI 컴포넌트·일별 Excel 일정·간트. **에픽 ≠ WBS**, 별도 산출물(같은 작업 두 관점).
- **Architecture** (`02-architecture/`) — C4 + ERD. **후행 stub**(BMAD Solutioning 단계).
- 01-overview·00-brief·02-usecase는 만들지 않음(폐지/미생성).

### 진실의 원천 (3개)
- `docs/spec/_foundation/source-strategy-brief.md` — ★ 원본 사업 전략서 (top-down 출발점)
- `docs/spec/PROCESS.md` — 작성 규칙·결정·frontmatter 스키마·변경 절차
- `docs/spec/_shared/traceability.md` — **CF × UJ × WBS** 매핑 매트릭스

### 핵심 원칙 (PROCESS.md 요지)
- 분해는 **top-down**(전략서 → JTBD → UJ → Feature(CF) → FR). 코드(F0~F8, `_shared/component-source-map.md`)는 *구현 근거*로 강등하고 FR 본문은 고객 voice. 범위 hybrid(구현 CF 상세 + 로드맵 CF 저fidelity).
- **Markdown이 source of truth, HTML은 큐레이션 뷰.** HTML 직접 편집 금지.
- **Stable ID는 frontmatter + 파일명에 박힘** (`CF-1`, `FR-CF1.1`, `UJ-1`, `EPIC-N`, `Story N.M`, `WBS-1.1`). 제목 바뀌어도 ID 유지.
- **모든 frontmatter는 traceability.md(CF×UJ×WBS + Epic×Story×FR)와 일치해야** 한다 (desync 검출 가능).
- **BMAD 영어/애자일 어휘는 한국 SI 문체로 변환**해 쓴다 (PRD=기능명세서, Story=**서술형** "{역할}는 ~한다", Acceptance=인수/검증기준; **에픽 ≠ WBS 작업패키지**; ID·Gherkin 키워드는 영문 유지). 변환표·문체 규칙은 PROCESS.md §2.8.

상세는 PROCESS.md.

---

## 수정 배치모드 (트리거: `수정배치모드`)

사용자가 산출물을 직접 검토하며 수정점을 모았다가 **한 번에 병렬로** 반영하기 위한 모드.

- **진입**: 사용자가 `수정배치모드`라고 입력하면 Claude는 **"수정할 곳을 알려주세요."** 라고만 답하고 대기한다. 먼저 수정하지 않는다.
- **수집**: 사용자가 항목을 던질 때마다 **즉시 수정하지 않고** 번호 매겨 리스트에 누적한다. 매 턴 현재까지의 `📝 수정 목록`을 보여준다. 각 항목은 의도만 기록하고, 영향 파일·라인은 적용 시점에 grep으로 찾는다. 사용자가 결정을 못 한 항목은 옵션 표로 좁혀 선택받되 **수정은 보류**.
- **적용**: 사용자가 `적용`(또는 `apply`/`go`)이라고 하면 그때 실행한다.
  1. grep으로 전체 영향 범위를 먼저 스캔
  2. 영역(`prd` / `architecture` / `epics` / `wbs` / `_shared`)별로 **파일 겹침 없이** 파티션해 서브에이전트 병렬 dispatch — 공통 변환 규칙을 prompt에 박아 일관성 보장
  3. **Markdown source + HTML 산출물을 함께** 갱신 (자동 빌드 파이프라인 없음)
  4. 완료 후 grep으로 잔재 0건 + Tailscale URL 200 전수 검증, 결과를 표로 보고
- **종료**: 적용이 끝나면 모드 해제. 사용자가 명시적으로 끝내면 미적용 목록은 보존한 채 대기 해제.

`_foundation/`(source-strategy-brief·baseline)은 입력/audit 자료이므로 별도 지시 없으면 배치 대상에서 제외.

---

## 다른 영역

TODO — 코드 작성·테스트·배포 관련 규칙은 별도 sub 파일로 추가 시 본 파일에 pointer만 추가.
