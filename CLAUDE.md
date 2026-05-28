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

## 다른 영역

TODO — 코드 작성·테스트·배포 관련 규칙은 별도 sub 파일로 추가 시 본 파일에 pointer만 추가.
