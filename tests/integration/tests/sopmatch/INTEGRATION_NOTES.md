# WT-grounding → 통합단계 메모 (배선·조율)

본 WT(`feat/cf1-grounding`)는 SOP 매칭(그라운딩) 고도화를 구현했다. 아래는 **배타 소유 경계 밖**이라 직접 고치지 않고 메모만 남긴 항목이다.

## 1. WT-ai 조율 — dispatch hook 동작 변화 (필수)

`PreviewSOPDocumentBinding`은 이제 `sop_id` 라벨이 없어도 **service.name + severity + owner_team 라벨 조합**으로 SOP를 매칭한다(라벨 모든 차원이 정확히 일치하면 `Status=bound`, `Resolution=label_match`).

- **타입은 additive만 변경**: 응답에 `candidates []SOPBindingCandidate` 필드, resolution 값 `label_match`/`fallback`, explicit 경로의 staleness 경고 추가. 기존 `Status/SOPID/Version/Title/SourceID` 의미는 불변 → **`hook.go` 코드 수정 불필요**.
- **동작(behavior)은 확장됨**: 이전에는 `sop_id` 없는 알람 → unbound → AI 생성기 미호출. 지금은 라벨 조합이 SOP와 정확히 일치하면 → bound → **AI 생성기가 호출됨**. 이게 본 과제의 헤드라인("explicit 라벨 단일매칭을 넘어선다")이다.

### WT-ai가 갱신해야 할 테스트 (현재 깨짐)
- `pkg/ruler/aigenerator/dispatchhook/hook_test.go::TestApply_UnboundSOPReturnsAnnotationsUnchanged` (L187)
  - 전제("`sop_id` 제거 → unbound")가 더 이상 성립 안 함. 공용 시드(`pkg/types/ruletypes/testdata/ds_ai_sop_demo_seed.json`)의 알람 라벨(`service.name=payment-api`, `owner_team=payments`, `severity=critical`)이 시드 SOP-PAY-001(ownerTeam=payments, tags=[payment-api,prod,critical])과 **정확히 일치**하므로, 이제 `label_match`로 bind 된다.
  - 권장 수정(택1): (a) "라벨도 일치하지 않는" 라벨셋으로 바꿔 진짜 unbound 경로를 검증, 또는 (b) `sop_id` 없이 `label_match` bind가 일어나는 동작을 명시적으로 검증하도록 의도 변경.
- **결정 위임**: hook이 `sop_id` 없이 라벨매칭만으로 AI를 자동 그라운딩하는 것을 기본 ON으로 둘지, opt-in 게이트를 둘지는 WT-ai 판단. (비용/부하 영향 있음 — 매칭되는 알람마다 생성기 호출.)

## 2. 라우터/배선 (해당 없음 — 신규 없음)
- 매칭은 기존 `PreviewSOPDocumentBinding` 엔드포인트(`/api/v2/ds/sop/bindings/preview`)를 그대로 사용. 신규 핸들러/라우트 추가 없음.
- batch 핸들러도 기존 `CreateSOPDocumentBatch`(`/api/v2/ds/sop/documents/batch`)에 in-batch 중복(sopId+version) 충돌 검출만 추가 → 신규 배선 없음.

## 3. 의존성
- go.mod/package.json **변경 없음**. 신규 라이브러리 도입 없음.

## 3b. T1 실행 이연 (통합단계 처리)
- `01_match_and_batch.py`(이 폴더)는 **작성·정적검증 완료, 실행 미실시**(사용자 결정: 통합단계 이연 — 2026-06-11).
- 블로커: 개발 셸에서 `uv sync`가 `psycopg2` 빌드 실패(`pg_config`/`libpq-dev` 미설치, sudo 불가). 정규 통합 환경에선 `libpq-dev` 존재하면 해결. 필요 시 `pyproject.toml`의 `psycopg2`→`psycopg2-binary` 전환 검토(공유 config = 통합단계 조율 대상).
- 실행: `cd tests && uv run pytest --basetemp=./tmp/ -vv integration/tests/sopmatch/` (첫 구동은 SigNoz 컨테이너 빌드로 느림).
- 동작 로직 자체는 T0 Go 단위테스트(green)로 이미 증명됨 — T1은 동일 경로를 HTTP로 재확인하는 회귀 안전망.

## 4. 매칭 규칙 요약 (참고)
- 차원: `owner_team`(우선순위 4) > `service.name`(2) > `severity`(1). 정렬 = 일치 개수 desc → 우선순위합 desc → version desc → sopID asc.
- service.name/severity는 SOP `Tags`와 매칭(bare 값 또는 `key:value`/`key=value`). owner_team은 SOP `OwnerTeam`와 매칭.
- staleness: `UpdatedAt` 90일 초과 SOP는 라벨매칭 후보에서 제외(FR-CF1.5). explicit `sop_id`는 bind 유지 + 경고.
- tenant scope(project_id+environment) 미충족/범위 밖 SOP는 후보 제외.
