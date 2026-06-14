# CF-11 — AI Codebase RCA · 통합 단계(Integration) 설계

> 원 설계: [2026-06-11-cf11-code-rca-design.md](2026-06-11-cf11-code-rca-design.md) (Codex 4라운드 APPROVE).
> 원 설계가 의도적으로 범위에서 제외한 **seam 배선(§11) + M4 표면(§13)** 을 이행하는 후속 설계다.
> 목표 상태: CF-11 `implemented-mvp` → `implemented`, WBS M-7 마일스톤 종결.

## 1. 목표

UJ-5(코드 RCA) 여정을 end-to-end로 가동한다.

```
알람 디스패치(unbound 분기) → coderca 트리거(게이트+admission) → coderca_run 적재(즉시 반환)
→ 워커 claim(lease) → 소스 준비(기준 커밋 pin) → read-only CLI 에이전트 분석
→ 근본원인+수정 제안 파싱 → 핸드오프(CF-3) 전달 + 감사(CF-6)
```

그리고 관리자가 FE에서 저장소 연결·서비스 매핑·기능 토글을 설정하고, 분석 이력(run 목록·보고서)을 조회할 수 있게 한다.

**전제(검증 완료된 사실):**
- 코어(M1~M3: `runstore`(admission·lease·flood-sim)·`sourcestate`·`reporesolver`·`clirunner`·`engine`·parser·`codebaseconfigstore`)는 TDD 완료, 단 **어디에도 배선되지 않음** (`cmd/community`·디스패치 훅에 coderca 참조 0건).
- `coderca.Trigger` 파사드와 `coderca_handler.go`는 **파일 자체가 미작성** — "1줄 배선"만 남은 게 아니라 접합부 코드 작성이 포함된다.
- AI 디스패치 훅(`dispatchhook.Hook`)은 `dispatcher.go:applyAIHook`으로 **실제 배선되어 있음** (패키지 주석 "not yet wired"는 낡은 기술 — 본 작업에서 주석 정정).
- 마이그레이션 082(coderca 테이블 7종)는 이미 존재.

## 2. 진행 방식 — 순차 수직 통합 (A→B→C)

각 단계에 검증 게이트를 두고, 게이트 통과 전 다음 단계에 착수하지 않는다. 원 설계가 M1(비용 제어)을 게이트로 강제한 것과 같은 철학 — 리스크가 가장 큰 서버 lifecycle·디스패치 경로 배선을 먼저 검증한다.

| 단계 | 내용 | 게이트 |
|---|---|---|
| **A** | 백엔드 파이프라인 통합 (Trigger 파사드·워커 루프·seam 배선) | 가짜 CLI e2e + fail-open 테스트 통과 |
| **B** | HTTP API (`coderca_handler.go` + 라우트) | 핸들러 테스트 + secrets 비노출 검증 |
| **C** | FE 설정 + 이력 페이지 | tsc + jest + 수동 확인 |

## 3. A단계 — 백엔드 파이프라인 통합

### 3.1 새 파일 (접합부 코드)

| 파일 | 내용 |
|---|---|
| `pkg/ruler/coderca/trigger.go` | `Trigger` 파사드. `Maybe(ctx, signal)`: cheap pre-check(§3.2) → `runstore.Admit`(dedup·일일 예산·큐 깊이 원자 판정, 원설계 §6.2) → queued run 적재 후 **즉시 반환**. 어떤 실패도 에러로 반환하지 않음(로그만) — FR-CF11.6 fail-open. |
| `pkg/ruler/coderca/worker/worker.go` | 워커 루프. ticker 폴링으로 `engine.ProcessNext` 구동, 서버 lifecycle(start/graceful stop)에 결합. 동시성은 엔진/lease가 DB 기준으로 강제(원설계 §6.3)하므로 워커 수는 폴링 주기만 결정. |
| `pkg/ruler/coderca/delivery/`(보강) | `Deliverer` 구체 구현 — 기존 핸드오프(CF-3)·이력 경로 재사용. 기존 `delivery/delivery.go` 검토 후 부족분만 추가. |
| (기존) `auditor/` | `Auditor`는 기존 구현 재사용 (CF-6 audit 경로, fire-and-forget). |

### 3.2 트리거 게이트 (원설계 §6.1·§10 준수)

순서: **기능 토글 ON → unbound → anomaly → 심각도 → 서비스→저장소 매핑 존재** → Admit(DB). 전부 DB 쓰기 없는 pre-check를 먼저, DB 쓰기는 Admit 1회.

- **unbound 판정**: 디스패치 훅의 SOP 바인딩 결과 재사용 — `hook.Apply`의 unbound 분기(`hook.go` L117-123)가 호출 지점이므로 별도 재판정 불필요.
- **anomaly 판정**: CF-7 anomaly rule 발화 여부. 시그널 구성(라벨/룰타입에서 어떻게 읽을지)은 원설계 §10(trigger precision)을 따른다 — 구현 계획 수립 시 §10과 `anomaly_rule.go`의 라벨 표면을 대조해 확정한다. anomaly 신호 부재 시 **fail-closed(미발화)**.
- 호출 지연 상한: pre-check는 메모리/단일 조회 수준, Admit는 단일 DB tx — 짧은 timeout context로 cap해 디스패처 hot path 영향을 차단.

### 3.3 seam 1줄 배선 (원설계 §11 표 그대로)

| Seam | 파일 | 변경 |
|---|---|---|
| Trigger | `pkg/ruler/aigenerator/dispatchhook/hook.go` (unbound 분기) | `coderca` Trigger 옵셔널 의존성 주입(nil-safe) + unbound 분기에서 `Trigger.Maybe` 호출 |
| Server wiring | `pkg/signoz/signoz.go` · `cmd/community/server.go` | coderca 스토어·엔진·트리거·워커 구성, 디스패치 훅·lifecycle에 주입 |

### 3.4 A 게이트 (검증)

- **e2e (가짜 CLI 바이너리)**: unbound+anomaly 알람 → run 적재 → 워커 claim → 가짜 CLI 실행 → 전달(Deliverer mock) 호출 확인.
- **fail-open**: 트리거 내부 강제 실패(스토어 에러·panic 가드) 시 알람 전달 결과가 배선 전과 동일함을 증명.
- **비발화 케이스**: bound 알람·anomaly 부재·토글 OFF·매핑 부재 각각 미발화 확인.

## 4. B단계 — HTTP API

새 파일 `pkg/ruler/signozruler/coderca_handler.go` (기존 ai_config 핸들러 패턴 미러). 메서드만 구현, 등록은 seam.

| 영역 | 엔드포인트(안) | 비고 |
|---|---|---|
| 저장소 | `GET/POST/PUT/DELETE /api/v1/coderca/repos` | 자격증명 secretbox 암호화 저장, **응답에 평문 비노출** (원설계 AC: "mirrors ai_config") |
| 서비스 매핑 | `GET/POST/PUT/DELETE /api/v1/coderca/service-maps` | |
| 설정 | `GET/PUT /api/v1/coderca/config` | 토글·예산·동시성 |
| 이력 | `GET /api/v1/coderca/runs` (상태·서비스 필터, 페이지) · `GET /api/v1/coderca/runs/{id}` | **`runstore`에 조회 메서드 신설 필요** (현재 Admit/ClaimNext/Finalize만 존재) |

- 권한: 설정·저장소·매핑 = EditAccess, 이력 조회 = ViewAccess.
- run 상세 응답에 기준 커밋(baseline) 포함 — FR-CF11.5.
- seam: `pkg/apiserver/signozapiserver/ruler.go` `addRulerRoutes` 등록 + `pkg/ruler/signozruler/handler.go`·`pkg/signoz/handler.go` `NewHandlers` 배선.

**B 게이트**: 핸들러 단위 테스트(저장→ciphertext, 응답 비노출 / CRUD 왕복 / run 목록·상세) 통과.

## 5. C단계 — FE 설정 + 이력

- **설정 페이지**: `frontend/src/container/AIModuleSettings/` 패턴 미러로 신규 컨테이너 — 저장소 연결(자격증명 입력, 저장 후 마스킹 표시), 서비스 매핑 테이블, 기능 토글·예산.
- **이력 화면**: run 목록(상태·서비스·시각·기준 커밋) + 상세(근본원인·수정 제안 보고서 마크다운 렌더). 설정 페이지의 탭 또는 인접 라우트 — 구현 계획에서 확정.
- **라우팅·메뉴**: FE 라우터 + `menuItems` 등록 (seam).
- **i18n**: 신규 한글 locale namespace (원설계 §5.2). 주의 — 테스트 mock은 t()가 키를 반환하므로 신규 화면 테스트 assertion은 키 기준; `<Trans>` 사용 시 글로벌 test-utils mock 필요.
- **API 클라이언트**: 수기 작성 — 기존 ds 엔드포인트 관례(`api/aiModule/*`, `ApiV2Instance`)를 미러. (원 설계 seam 표의 orval codegen은 이 코드베이스의 ds API 클라이언트 관례와 불일치 — 구현 계획 수립 시 정정.)

**C 게이트**: `tsc --noEmit` + jest + 수동 화면 확인.

## 6. 에러 처리·안전 (변경 없음 — 코어 계약 계승)

- 트리거·워커의 어떤 실패도 알람 경로에 영향 없음 (UJ-3 fail-open 계승, NF-5.2.1).
- 수정 제안은 read-only — 자동 적용 절대 없음 (FR-CF11.4, HITL).
- 암호화 키 부재 + 비공개 저장소 = fail-closed (코어 기구현).
- 폭주 제어(dedup·예산·동시성 캡)는 Admit/lease가 DB 기준으로 강제 (코어 기구현, flood-sim 검증 완료).

## 7. 산출물 정합 (BMAD 체인, PROCESS.md §10)

구현 완료 시:
1. 스토리 `11.1`·`11.2`·`11.6` → done (seam 해소), CF-11 frontmatter `status: implemented`, `open_items` 제거.
2. `traceability.md` §6.1·§8(open 항목 2건 해소)·§5, `05-wbs/index.md` M-7 → 완료, 스토리 일정 갱신.
3. `component-source-map.md` F9/F10 as-built 반영 (기존 follow-up).
4. `dispatchhook` 패키지 낡은 주석 정정.

## 8. 범위 밖 (본 통합에서 하지 않음)

- 운영자 검수 화면(CF-3 open) — 별도 항목.
- DLQ 기본 배선·HMAC(M-4 Beta 격차) — 다음 우선순위로 별도 진행.
- CF-7 학습형 기준선·CF-8~10 로드맵.
- jinhyeok 원안 스캐너 설계(CF-11 부록 이연 대안).

## 9. 미확정(구현 계획 시 확정할 것)

| 항목 | 확정 방법 |
|---|---|
| anomaly 신호의 라벨 표면 (트리거가 무엇을 읽나) | 원설계 §10 + `anomaly_rule.go` 발화 라벨 대조 |
| run 보고서 본문 저장 위치(상세 API 응답 소스) | `runstore.Finalize`·`coderca_run` 스키마 확인 |
| 이력 화면 배치(설정 탭 vs 인접 라우트) | 기존 FE 설정 IA 확인 후 결정 |
