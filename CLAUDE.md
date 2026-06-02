# DS-APM — Project Guide for Claude

본 파일은 매 conversation 시작 시 로드된다. 토큰 절약을 위해 짧게 유지하고, 상세 규칙은 서브 파일을 가리키는 pointer로 둔다. Claude는 해당 영역 작업을 시작할 때만 서브 파일을 Read한다.

---

## 산출물 작성 (`docs/spec/`)

DS-APM 산출물 4종(Overview / Use Case / 기능명세서 / WBS)을 손대기 전에 **반드시 먼저** `docs/spec/PROCESS.md`를 Read해서 규칙을 따른다. 산출물 폴더 안의 어떤 `.md`라도 수정·생성하는 작업이면 본 절차 대상.

### 진실의 원천 (3개)
- `docs/spec/PROCESS.md` — 작성 규칙·결정·frontmatter 스키마·변경 절차
- `docs/spec/_shared/traceability.md` — UC × Feature × WBS 매핑 매트릭스
- `docs/spec/_foundation/baseline.md` — 현재 구현 상태 (커밋·LOC·변경 표면)

### 핵심 원칙 (PROCESS.md 요지)
- 산출물은 **현재 구현된 코드를 reverse-engineering**. 미래 계획 아님. 단 동결 아님 — 계속 진화.
- **Markdown이 source of truth, HTML은 빌드 산출물.** HTML 직접 편집 금지.
- **Stable ID는 frontmatter + 파일명에 박힘** (`F1`, `UC-001`, `WBS-1.1`). 제목 바뀌어도 ID 유지.
- **모든 frontmatter는 traceability.md와 일치해야** 한다 (desync 검출 가능).

상세는 PROCESS.md.

---

## 수정 배치모드 (트리거: `수정배치모드`)

사용자가 산출물을 직접 검토하며 수정점을 모았다가 **한 번에 병렬로** 반영하기 위한 모드.

- **진입**: 사용자가 `수정배치모드`라고 입력하면 Claude는 **"수정할 곳을 알려주세요."** 라고만 답하고 대기한다. 먼저 수정하지 않는다.
- **수집**: 사용자가 항목을 던질 때마다 **즉시 수정하지 않고** 번호 매겨 리스트에 누적한다. 매 턴 현재까지의 `📝 수정 목록`을 보여준다. 각 항목은 의도만 기록하고, 영향 파일·라인은 적용 시점에 grep으로 찾는다. 사용자가 결정을 못 한 항목은 옵션 표로 좁혀 선택받되 **수정은 보류**.
- **적용**: 사용자가 `적용`(또는 `apply`/`go`)이라고 하면 그때 실행한다.
  1. grep으로 전체 영향 범위를 먼저 스캔
  2. 영역(`00-brief` / `01-overview` / `02-usecase` / `03-functional-spec` / `04-wbs` / `_shared`)별로 **파일 겹침 없이** 파티션해 서브에이전트 병렬 dispatch — 공통 변환 규칙을 prompt에 박아 일관성 보장
  3. **Markdown source + HTML 산출물을 함께** 갱신 (자동 빌드 파이프라인 없음)
  4. 완료 후 grep으로 잔재 0건 + Tailscale URL 200 전수 검증, 결과를 표로 보고
- **종료**: 적용이 끝나면 모드 해제. 사용자가 명시적으로 끝내면 미적용 목록은 보존한 채 대기 해제.

`_foundation/`(baseline·handoff·research)은 audit 자료이므로 별도 지시 없으면 배치 대상에서 제외.

---

## 다른 영역

TODO — 코드 작성·테스트·배포 관련 규칙은 별도 sub 파일로 추가 시 본 파일에 pointer만 추가.
