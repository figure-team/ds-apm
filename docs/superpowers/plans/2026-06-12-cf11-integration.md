# CF-11 통합 (seam 배선 + M4 표면) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** TDD 완료된 coderca 코어를 실제 알람 경로·HTTP API·FE 설정/이력 화면에 배선해 CF-11을 end-to-end로 가동한다 (UJ-5).

**Architecture:** A(백엔드 파이프라인: per-org 설정 스토어 신설 → 보고서 영속 → Trigger 파사드 → 워커 서비스 → 디스패치 훅/서버 seam 배선) → B(coderca_handler HTTP API + 라우트) → C(FE 설정+이력 페이지, ai-module 패턴 미러). 각 단계 끝에 검증 게이트.

**Tech Stack:** Go (bun/SQLite, gorilla/mux, factory.Service), React+antd (RouteTab settings, react-i18next en/ko), 수기 API 클라이언트 (`ApiV2Instance`).

**스펙:** [docs/superpowers/specs/2026-06-12-cf11-integration-design.md](../specs/2026-06-12-cf11-integration-design.md) · 원 설계 [2026-06-11-cf11-code-rca-design.md](../specs/2026-06-11-cf11-code-rca-design.md)

**탐색으로 확정된 사실 (전제):**
- 마이그레이션 082는 `pkg/signoz/provider.go:203`에 **이미 등록됨**. 단 `ds_codebase_config`(per-org 토글·임계값) 테이블은 **없음** — 신규 083 필요.
- `coderca_run`에 보고서 본문 컬럼 없음(`result_ref`만) — 이력 상세 API를 위해 083에서 컬럼 추가 + `Finalize` 확장 필요.
- `dispatchhook.Hook`은 `dispatcher.go:653 applyAIHook`로 실제 배선돼 있음(패키지 주석이 낡음 — 본 작업에서 정정).
- 핸들러/라우트 미러 대상: `pkg/ruler/signozruler/ai_config_handler.go` + `pkg/apiserver/signozapiserver/ruler.go`의 `/api/v2/ds/ai/config` 블록.
- FE 미러 대상: `container/AIModuleSettings` + `api/aiModule/*` + Settings 페이지 등록 5개소. `/settings/*`는 AppRoutes의 `SETTINGS` 라우트(`exact:false`)가 받으므로 AppRoutes 변경 불필요.
- 알람 dedup 시그니처 라벨: `alertname, service.name, severity, error_class` (`signature.go:13`). 서비스 라벨 = `service.name`.
- anomaly 판정(v1, 원 설계 §10): 알람 labels/annotations의 명시적 `anomaly` 키 — fail-closed. CF-7 `anomaly_rule.go`가 발화 시 이 라벨을 직접 찍도록 보강(Task 7).
- 빌드/테스트 명령: Go는 `go test ./pkg/... -count=1`, FE는 frontend에서 `node node_modules/typescript/bin/tsc --noEmit -p tsconfig.json` + `node node_modules/jest/bin/jest.js --silent <pattern>` (yarn은 PATH에 없음).

---

## Phase A — 백엔드 파이프라인 통합

### Task 1: per-org CodebaseRCAConfig 도메인 타입

**Files:**
- Create: `pkg/types/ruletypes/codebase_rca_config.go`
- Create: `pkg/types/ruletypes/codebase_rca_config_store.go`
- Test: `pkg/types/ruletypes/codebase_rca_config_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

```go
package ruletypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCodebaseRCAConfig(t *testing.T) {
	cfg := DefaultCodebaseRCAConfig("org-1")
	assert.Equal(t, "org-1", cfg.OrgID)
	assert.False(t, cfg.Enabled) // 기본 OFF, opt-in (설계 §6.1)
	assert.Equal(t, "high", cfg.MinSeverity)
	assert.Equal(t, 21600, cfg.CooldownWindowSecs) // 6h
	assert.Equal(t, 20, cfg.MaxRunsPerDay)
	assert.Equal(t, 50, cfg.MaxQueueDepth)
	assert.Equal(t, 1, cfg.MaxConcurrentRuns)
	assert.False(t, cfg.AllowUnboundWithoutAnomaly)
}

func TestValidateCodebaseRCAConfig(t *testing.T) {
	valid := DefaultCodebaseRCAConfig("org-1")
	require.NoError(t, ValidateCodebaseRCAConfig(valid))

	bad := valid
	bad.MinSeverity = "nonsense"
	require.Error(t, ValidateCodebaseRCAConfig(bad))

	bad2 := valid
	bad2.MaxRunsPerDay = -1
	require.Error(t, ValidateCodebaseRCAConfig(bad2))

	bad3 := valid
	bad3.OrgID = ""
	require.Error(t, ValidateCodebaseRCAConfig(bad3))
}

func TestSeverityAtLeast(t *testing.T) {
	assert.True(t, SeverityAtLeast("critical", "high"))
	assert.True(t, SeverityAtLeast("HIGH", "high")) // 대소문자 무시
	assert.False(t, SeverityAtLeast("warning", "high"))
	assert.False(t, SeverityAtLeast("", "high"))      // 라벨 부재 → fail-closed
	assert.False(t, SeverityAtLeast("unknown", "high")) // 미지 등급 → fail-closed
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./pkg/types/ruletypes/ -run "TestDefaultCodebaseRCAConfig|TestValidateCodebaseRCAConfig|TestSeverityAtLeast" -count=1`
Expected: FAIL (undefined: DefaultCodebaseRCAConfig 등)

- [ ] **Step 3: 구현**

`codebase_rca_config.go`:

```go
package ruletypes

import (
	"fmt"
	"strings"
)

// CodebaseRCAConfigContractVersion versions the CF-11 per-org config payload.
const CodebaseRCAConfigContractVersion = "ds.codebase_rca_config.v1"

// severityRank orders alert severities for the min-severity gate. Unknown or
// missing severities rank 0 so the gate fails closed (design §10).
var severityRank = map[string]int{
	"critical": 4,
	"high":     3,
	"error":    3,
	"warning":  2,
	"info":     1,
}

// SeverityAtLeast reports whether severity meets the minimum. Comparison is
// case-insensitive; unknown values never pass (fail-closed).
func SeverityAtLeast(severity, min string) bool {
	s := severityRank[strings.ToLower(strings.TrimSpace(severity))]
	m := severityRank[strings.ToLower(strings.TrimSpace(min))]
	if s == 0 || m == 0 {
		return false
	}
	return s >= m
}

// CodebaseRCAConfig is the per-org CF-11 feature toggle + cost thresholds
// (design §6: "all thresholds live in codebase_config, per-org overridable").
// Agent/model/auth are deployment-level (env), not per-org.
type CodebaseRCAConfig struct {
	ContractVersion string `json:"contractVersion"`
	OrgID           string `json:"orgId"`
	Enabled         bool   `json:"enabled"`
	// MinSeverity gates the trigger predicate (default "high" → high|critical).
	MinSeverity        string `json:"minSeverity"`
	CooldownWindowSecs int    `json:"cooldownWindowSecs"`
	MaxRunsPerDay      int    `json:"maxRunsPerDay"`
	MaxQueueDepth      int    `json:"maxQueueDepth"`
	MaxConcurrentRuns  int    `json:"maxConcurrentRuns"`
	// AllowUnboundWithoutAnomaly revives the legacy unbound+severity trigger
	// without an anomaly signal. Off by default; enabling logs a loud warning
	// (design §10).
	AllowUnboundWithoutAnomaly bool   `json:"allowUnboundWithoutAnomaly"`
	UpdatedAt                  string `json:"updatedAt"` // RFC3339
}

// DefaultCodebaseRCAConfig returns the fail-closed defaults from design §6.
func DefaultCodebaseRCAConfig(orgID string) CodebaseRCAConfig {
	return CodebaseRCAConfig{
		ContractVersion:    CodebaseRCAConfigContractVersion,
		OrgID:              orgID,
		Enabled:            false,
		MinSeverity:        "high",
		CooldownWindowSecs: 21600,
		MaxRunsPerDay:      20,
		MaxQueueDepth:      50,
		MaxConcurrentRuns:  1,
	}
}

// ValidateCodebaseRCAConfig validates a config update.
func ValidateCodebaseRCAConfig(cfg CodebaseRCAConfig) error {
	var errs []string
	if strings.TrimSpace(cfg.OrgID) == "" {
		errs = append(errs, "orgId: must not be empty")
	}
	if _, ok := severityRank[strings.ToLower(strings.TrimSpace(cfg.MinSeverity))]; !ok {
		errs = append(errs, fmt.Sprintf("minSeverity: %q is not one of critical|high|error|warning|info", cfg.MinSeverity))
	}
	if cfg.CooldownWindowSecs < 0 {
		errs = append(errs, "cooldownWindowSecs: must be >= 0")
	}
	if cfg.MaxRunsPerDay < 0 {
		errs = append(errs, "maxRunsPerDay: must be >= 0")
	}
	if cfg.MaxQueueDepth < 0 {
		errs = append(errs, "maxQueueDepth: must be >= 0")
	}
	if cfg.MaxConcurrentRuns < 0 || cfg.MaxConcurrentRuns > 2 {
		errs = append(errs, "maxConcurrentRuns: must be 0..2 (design §6.3)")
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("codebase RCA config validation: %s", strings.Join(errs, "; "))
}
```

`codebase_rca_config_store.go`:

```go
package ruletypes

import (
	"context"
	"errors"
)

// ErrCodebaseRCAConfigNotFound is returned when no per-org config row exists;
// callers fall back to DefaultCodebaseRCAConfig.
var ErrCodebaseRCAConfigNotFound = errors.New("codebase RCA config not found")

// CodebaseRCAConfigStore persists the per-org CF-11 toggle + thresholds.
// No secrets — encryption closures are not needed.
type CodebaseRCAConfigStore interface {
	Upsert(ctx context.Context, cfg CodebaseRCAConfig) error
	Get(ctx context.Context, orgID string) (CodebaseRCAConfig, error)
}
```

- [ ] **Step 4: 통과 확인**

Run: `go test ./pkg/types/ruletypes/ -run "TestDefaultCodebaseRCAConfig|TestValidateCodebaseRCAConfig|TestSeverityAtLeast" -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/types/ruletypes/codebase_rca_config.go pkg/types/ruletypes/codebase_rca_config_store.go pkg/types/ruletypes/codebase_rca_config_test.go
git commit -m "[Feat] CF-11 per-org RCA 설정 도메인 타입·스토어 인터페이스 추가"
```

---

### Task 2: 마이그레이션 083 + SQL config 스토어

**Files:**
- Create: `pkg/sqlmigration/083_update_ds_codebase_config.go`
- Create: `pkg/ruler/coderca/codebaseconfigstore/sqlcodebasercaconfigstore/sqlcodebasercaconfigstore.go`
- Test: `pkg/ruler/coderca/codebaseconfigstore/sqlcodebasercaconfigstore/sqlcodebasercaconfigstore_test.go`
- Modify: `pkg/signoz/provider.go:203` 다음 줄에 등록

- [ ] **Step 1: 마이그레이션 작성** — 082(`082_add_ds_codebase_config.go`)와 동일한 구조(트랜잭션·factory) 미러:

```go
package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

// updateDSCodebaseConfig (CF-11 integration stage) adds the per-org config
// table that design §8 named `codebase_config` (migration 082 shipped only the
// repo/map/cost tables) and widens coderca_run with the persisted RCA report
// so the run-history API can serve report bodies.
type updateDSCodebaseConfig struct {
	sqlstore sqlstore.SQLStore
}

func NewUpdateDSCodebaseConfigFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(
		factory.MustNewName("update_ds_codebase_config"),
		func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
			return &updateDSCodebaseConfig{sqlstore: sqlstore}, nil
		},
	)
}

func (migration *updateDSCodebaseConfig) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *updateDSCodebaseConfig) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS ds_codebase_config (
			org_id                        TEXT    NOT NULL PRIMARY KEY,
			enabled                       BOOLEAN NOT NULL DEFAULT FALSE,
			min_severity                  TEXT    NOT NULL DEFAULT 'high',
			cooldown_window_secs          INTEGER NOT NULL DEFAULT 21600,
			max_runs_per_day              INTEGER NOT NULL DEFAULT 20,
			max_queue_depth               INTEGER NOT NULL DEFAULT 50,
			max_concurrent_runs           INTEGER NOT NULL DEFAULT 1,
			allow_unbound_without_anomaly BOOLEAN NOT NULL DEFAULT FALSE,
			updated_at                    TEXT    NOT NULL DEFAULT ''
		)`,
		`ALTER TABLE coderca_run ADD COLUMN root_cause   TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE coderca_run ADD COLUMN proposed_fix TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE coderca_run ADD COLUMN confidence   TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE coderca_run ADD COLUMN limitations  TEXT NOT NULL DEFAULT ''`,
	}
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (migration *updateDSCodebaseConfig) Down(ctx context.Context, db *bun.DB) error {
	return nil // additive only
}
```

- [ ] **Step 2: provider.go 등록 (seam)** — `pkg/signoz/provider.go` 203행 `sqlmigration.NewAddDSCodebaseConfigFactory(sqlstore),` 바로 다음 줄에 추가:

```go
		sqlmigration.NewUpdateDSCodebaseConfigFactory(sqlstore),
```

- [ ] **Step 3: 실패하는 스토어 테스트 작성** — 기존 `sqlcodebaseconfigstore_test.go`의 테스트 하니스(SQLite 인메모리 셋업)를 먼저 Read해 동일 패턴으로 작성. 핵심 케이스:

```go
func TestRCAConfigUpsertGetRoundTrip(t *testing.T) {
	store := newTestStore(t) // 기존 테스트 하니스 미러 (마이그레이션 083 적용 포함)
	ctx := context.Background()

	_, err := store.Get(ctx, "org-1")
	require.ErrorIs(t, err, ruletypes.ErrCodebaseRCAConfigNotFound)

	cfg := ruletypes.DefaultCodebaseRCAConfig("org-1")
	cfg.Enabled = true
	cfg.MaxRunsPerDay = 5
	require.NoError(t, store.Upsert(ctx, cfg))

	got, err := store.Get(ctx, "org-1")
	require.NoError(t, err)
	assert.True(t, got.Enabled)
	assert.Equal(t, 5, got.MaxRunsPerDay)

	// 테넌트 격리: 다른 org는 not-found
	_, err = store.Get(ctx, "org-2")
	require.ErrorIs(t, err, ruletypes.ErrCodebaseRCAConfigNotFound)
}
```

- [ ] **Step 4: 실패 확인**

Run: `go test ./pkg/ruler/coderca/codebaseconfigstore/sqlcodebasercaconfigstore/ -count=1`
Expected: FAIL (패키지 없음)

- [ ] **Step 5: 구현** — `sqlcodebaseconfigstore.go`(repo 스토어)와 동일한 bun raw-SQL 스타일로 Upsert(`INSERT ... ON CONFLICT (org_id) DO UPDATE`)/Get(`sql.ErrNoRows → ErrCodebaseRCAConfigNotFound`) 구현. 구조체는 `Store{sqlstore sqlstore.SQLStore}` + `New(store sqlstore.SQLStore) *Store`. 컴파일 타임 검증 `var _ ruletypes.CodebaseRCAConfigStore = (*Store)(nil)` 포함.

- [ ] **Step 6: 통과 확인 + 전체 빌드**

Run: `go test ./pkg/ruler/coderca/... ./pkg/sqlmigration/ -count=1 && go build ./...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/sqlmigration/083_update_ds_codebase_config.go pkg/ruler/coderca/codebaseconfigstore/sqlcodebasercaconfigstore/ pkg/signoz/provider.go
git commit -m "[Feat] 마이그레이션 083(ds_codebase_config·run 보고서 컬럼) + RCA 설정 SQL 스토어"
```

---

### Task 3: runstore 확장 — 보고서 영속 + 이력 조회

**Files:**
- Modify: `pkg/ruler/coderca/runstore/lease.go` (FinalizeParams + Finalize SQL)
- Create: `pkg/ruler/coderca/runstore/query.go`
- Test: `pkg/ruler/coderca/runstore/query_test.go` (+ 기존 lease_test.go 보강)

- [ ] **Step 1: 실패하는 테스트 작성** — 기존 `lease_test.go`/`runstore_test.go` 하니스 재사용:

```go
func TestFinalizePersistsReportAndBaseline(t *testing.T) {
	store, now := newTestStore(t), time.Unix(1000, 0)
	admit := mustAdmit(t, store, "org-1", "svc-a", "key-1", now) // 기존 헬퍼 미러
	claim := mustClaim(t, store, now)

	ok, err := store.Finalize(context.Background(), runstore.FinalizeParams{
		Scope: "global", RunID: claim.RunID, LeaseToken: claim.LeaseToken,
		Status: coderca.RunStatusDone, ResultRef: "ref-1", Now: now,
		BaselineCommit: "abc123", RootCause: "nil deref in handler",
		ProposedFix: "guard nil", Confidence: "high", Limitations: "single repo",
	})
	require.NoError(t, err)
	require.True(t, ok)

	detail, err := store.GetRun(context.Background(), "org-1", claim.RunID)
	require.NoError(t, err)
	assert.Equal(t, "abc123", detail.BaselineCommit)
	assert.Equal(t, "nil deref in handler", detail.RootCause)
	assert.Equal(t, "guard nil", detail.ProposedFix)
	assert.Equal(t, coderca.RunStatusDone, detail.Status)
}

func TestListRunsFiltersAndTenantIsolation(t *testing.T) {
	// org-1에 2건(1 done, 1 queued), org-2에 1건 적재 후:
	// ListRuns(org-1, {})            → 2건, created_at DESC
	// ListRuns(org-1, {Status:done}) → 1건
	// ListRuns(org-2, {})            → 1건 (org-1 건 비노출)
	// GetRun(org-2, org1RunID)       → ErrRunNotFound (테넌트 격리)
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./pkg/ruler/coderca/runstore/ -run "TestFinalizePersists|TestListRuns" -count=1`
Expected: FAIL

- [ ] **Step 3: 구현**

`lease.go`의 `FinalizeParams`에 필드 추가(기존 필드 유지):

```go
type FinalizeParams struct {
	Scope      string
	RunID      string
	LeaseToken string
	Status     coderca.RunStatus // done | failed | timeout | unparseable
	ResultRef  string
	Now        time.Time

	// Persisted RCA report (integration stage): served by the run-history API.
	BaselineCommit string
	RootCause      string
	ProposedFix    string
	Confidence     string
	Limitations    string
}
```

`Finalize`의 fenced UPDATE 문 확장 (`lease.go:185` 일대):

```go
		r, err := db.ExecContext(ctx,
			`UPDATE coderca_run SET status = ?, result_ref = ?, finished_at = ?, lease_until = 0,
			        baseline_commit = ?, root_cause = ?, proposed_fix = ?, confidence = ?, limitations = ?
			 WHERE run_id = ? AND lease_token = ? AND status = ?`,
			string(p.Status), p.ResultRef, p.Now.Unix(),
			p.BaselineCommit, p.RootCause, p.ProposedFix, p.Confidence, p.Limitations,
			p.RunID, p.LeaseToken, string(coderca.RunStatusRunning),
		)
```

`query.go` 신규:

```go
package runstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
)

// ErrRunNotFound is returned by GetRun for a missing or other-org run.
var ErrRunNotFound = errors.New("coderca run not found")

// RunSummary is one row of the run-history list (no report body).
type RunSummary struct {
	RunID          string            `json:"runId"`
	OrgID          string            `json:"orgId"`
	Service        string            `json:"service"`
	Status         coderca.RunStatus `json:"status"`
	BaselineCommit string            `json:"baselineCommit"`
	CreatedAt      int64             `json:"createdAt"`  // unix seconds
	FinishedAt     int64             `json:"finishedAt"` // 0 = not finished
	Attempts       int               `json:"attempts"`
	ResultRef      string            `json:"resultRef"`
}

// RunDetail is a run with its persisted RCA report.
type RunDetail struct {
	RunSummary
	RootCause   string `json:"rootCause"`
	ProposedFix string `json:"proposedFix"`
	Confidence  string `json:"confidence"`
	Limitations string `json:"limitations"`
}

// ListRunsParams filters the run-history list. Zero values = no filter.
type ListRunsParams struct {
	Status  string
	Service string
	Limit   int // default 50, max 200
	Offset  int
}

// ListRuns returns the org's runs, newest first.
func (s *Store) ListRuns(ctx context.Context, orgID string, p ListRunsParams) ([]RunSummary, error) {
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	q := `SELECT run_id, org_id, service, status, baseline_commit, created_at, finished_at, attempts, result_ref
	      FROM coderca_run WHERE org_id = ?`
	args := []interface{}{orgID}
	if p.Status != "" {
		q += " AND status = ?"
		args = append(args, p.Status)
	}
	if p.Service != "" {
		q += " AND service = ?"
		args = append(args, p.Service)
	}
	q += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, p.Offset)

	rows, err := s.sqlstore.BunDB().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	out := make([]RunSummary, 0, limit)
	for rows.Next() {
		var r RunSummary
		var status string
		if err := rows.Scan(&r.RunID, &r.OrgID, &r.Service, &status, &r.BaselineCommit,
			&r.CreatedAt, &r.FinishedAt, &r.Attempts, &r.ResultRef); err != nil {
			return nil, err
		}
		r.Status = coderca.RunStatus(status)
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetRun returns one run with its report. Tenant-isolated: a run belonging to
// another org returns ErrRunNotFound (existence is not leaked).
func (s *Store) GetRun(ctx context.Context, orgID, runID string) (RunDetail, error) {
	var d RunDetail
	var status string
	err := s.sqlstore.BunDB().QueryRowContext(ctx,
		`SELECT run_id, org_id, service, status, baseline_commit, created_at, finished_at, attempts, result_ref,
		        root_cause, proposed_fix, confidence, limitations
		 FROM coderca_run WHERE org_id = ? AND run_id = ?`,
		orgID, runID,
	).Scan(&d.RunID, &d.OrgID, &d.Service, &status, &d.BaselineCommit,
		&d.CreatedAt, &d.FinishedAt, &d.Attempts, &d.ResultRef,
		&d.RootCause, &d.ProposedFix, &d.Confidence, &d.Limitations)
	if errors.Is(err, sql.ErrNoRows) {
		return RunDetail{}, ErrRunNotFound
	}
	if err != nil {
		return RunDetail{}, err
	}
	d.Status = coderca.RunStatus(status)
	return d, nil
}
```

> `BunDB()` 메서드명이 다르면 기존 runstore.go의 트랜잭션 외 조회 패턴(`s.sqlstore.BunDBCtx(ctx)` — tx 외부에선 동일 메서드 사용 가능)을 따른다. 단순 조회는 트랜잭션 불필요.

- [ ] **Step 4: 통과 확인 (기존 테스트 포함 전체)**

Run: `go test ./pkg/ruler/coderca/runstore/ -count=1`
Expected: PASS (기존 admit/lease/floodsim 포함)

- [ ] **Step 5: Commit**

```bash
git add pkg/ruler/coderca/runstore/
git commit -m "[Feat] runstore 보고서 영속(Finalize 확장) + run 이력 조회(ListRuns/GetRun)"
```

---

### Task 4: engine — 보고서 필드 패스스루

**Files:**
- Modify: `pkg/ruler/coderca/engine/engine.go` (`ProcessNext`/`runOne`)
- Test: `pkg/ruler/coderca/engine/engine_test.go` (기존 보강)

- [ ] **Step 1: 실패하는 테스트 작성** — 기존 engine_test의 fake RunStore가 받은 `FinalizeParams`를 캡처해 단언:

```go
func TestProcessNextPersistsReportOnDone(t *testing.T) {
	// 기존 happy-path 테스트 하니스 재사용. fake CLI가
	// RCAResult{BaselineCommit:"abc123", RootCause:"rc", ProposedFix:"fix",
	//           Confidence:"high", Limitations:"lim"} 반환하도록 설정.
	// 단언: fakeRuns.lastFinalize.RootCause == "rc",
	//       .ProposedFix == "fix", .Confidence == "high",
	//       .BaselineCommit == "abc123" (CLI echo 우선; 비면 source baseline)
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./pkg/ruler/coderca/engine/ -run TestProcessNextPersistsReport -count=1`
Expected: FAIL

- [ ] **Step 3: 구현** — `runOne` 시그니처를 `(coderca.RunStatus, string, string, string, coderca.RCAResult)`(status, resultRef, detail, baseline, result)로 확장하고, `ProcessNext`의 `Finalize` 호출에 채워 넣는다:

```go
	status, resultRef, detail, baseline, result := e.runOne(ctx, claim)
	// ... Audit 동일 ...
	_, ferr := e.deps.Runs.Finalize(ctx, runstore.FinalizeParams{
		Scope:      e.cfg.Scope,
		RunID:      claim.RunID,
		LeaseToken: claim.LeaseToken,
		Status:     status,
		ResultRef:  resultRef,
		Now:        e.deps.Now(),
		BaselineCommit: firstNonEmptyStr(result.BaselineCommit, baseline),
		RootCause:      result.RootCause,
		ProposedFix:    result.ProposedFix,
		Confidence:     result.Confidence,
		Limitations:    result.Limitations,
	})
```

`runOne`의 각 return 지점에 baseline/result 값을 추가한다(실패 지점은 zero RCAResult + 그 시점의 baseline). 파일 하단에 헬퍼:

```go
func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
```

- [ ] **Step 4: 통과 확인**

Run: `go test ./pkg/ruler/coderca/engine/ -count=1`
Expected: PASS (기존 전이 테스트 포함)

- [ ] **Step 5: Commit**

```bash
git add pkg/ruler/coderca/engine/
git commit -m "[Feat] engine: RCA 보고서·기준 커밋을 Finalize로 영속"
```

---

### Task 5: Trigger 파사드 (게이트 체인)

**Files:**
- Create: `pkg/ruler/coderca/trigger/trigger.go`
- Test: `pkg/ruler/coderca/trigger/trigger_test.go`

- [ ] **Step 1: 실패하는 테스트 작성** — 순수 게이트는 fake 의존성으로 테이블 테스트 (원 설계 §10 "exhaustively table-tested"):

```go
package trigger

// fakes: fakeCfgStore(cfg, err), fakeMaps(맵 존재 여부), fakeAdmitter(호출 기록)

func TestMaybeGateChain(t *testing.T) {
	base := func() ruletypes.CodebaseRCAConfig {
		c := ruletypes.DefaultCodebaseRCAConfig("org-1")
		c.Enabled = true
		return c
	}
	labels := map[string]string{
		"alertname": "PayErr", "service.name": "pay", "severity": "critical", "anomaly": "true",
	}

	cases := []struct {
		name       string
		cfg        ruletypes.CodebaseRCAConfig
		cfgErr     error
		labels     map[string]string
		mapped     bool
		wantAdmit  bool
	}{
		{"all gates pass → admit", base(), nil, labels, true, true},
		{"feature off → no admit", ruletypes.DefaultCodebaseRCAConfig("org-1"), nil, labels, true, false},
		{"config store error → fail-closed, no admit", base(), errors.New("db down"), labels, true, false},
		{"no anomaly label → fail-closed", base(), nil, without(labels, "anomaly"), true, false},
		{"anomaly=false → fail-closed", base(), nil, with(labels, "anomaly", "false"), true, false},
		{"below severity → no admit", base(), nil, with(labels, "severity", "warning"), true, false},
		{"severity label absent → fail-closed", base(), nil, without(labels, "severity"), true, false},
		{"no service label → no admit", base(), nil, without(labels, "service.name"), true, false},
		{"unmapped service → no admit (no_repo_mapping)", base(), nil, labels, false, false},
		{"allow_unbound_without_anomaly → anomaly 없이 admit", withFlag(base()), nil, without(labels, "anomaly"), true, true},
	}
	// 각 케이스: fakeAdmitter.called == wantAdmit 단언.
	// admit 케이스는 AdmitParams 단언: DedupKey == coderca.DedupKey("org-1","pay",
	//   coderca.ErrorSignature(labels)), CooldownWindow == 6h, MaxRunsPerDay == 20.
}

func TestMaybeNeverPanicsOrBlocks(t *testing.T) {
	// fakeAdmitter가 panic해도 Maybe는 정상 반환 (recover).
	// fakeCfgStore가 ctx를 무시하고 2초 sleep해도 Maybe는 ~1초 내 반환 (timeout ctx).
}

func TestMaybeAnomalyFromAnnotations(t *testing.T) {
	// labels에 없고 annotations["anomaly"]=="true"여도 통과 (설계 §10: label/annotation).
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./pkg/ruler/coderca/trigger/ -count=1`
Expected: FAIL (패키지 없음)

- [ ] **Step 3: 구현**

```go
// Package trigger is the CF-11 dispatch-side gate (design §10): a fire-and-
// forget facade the dispatch hook calls on the unbound branch. It never
// returns an error and never panics — any failure inside the gate must leave
// the alert path untouched (FR-CF11.6).
package trigger

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// maybeTimeout bounds the trigger's worst-case dispatch-path cost: the cheap
// pre-checks are in-memory; the only DB work is one config Get, one map Get,
// and the single Admit transaction.
const maybeTimeout = time.Second

// AnomalyLabel is the v1 anomaly signal: an explicit label/annotation on the
// alert (design §10, fail-closed). CF-7's anomaly rule stamps it at firing.
const AnomalyLabel = "anomaly"

// Admitter is the runstore admission port (satisfied by *runstore.Store).
type Admitter interface {
	Admit(ctx context.Context, p runstore.AdmitParams) (runstore.AdmitResult, error)
	RecordSkip(ctx context.Context, orgID string, reason coderca.SkipReason, now time.Time) error
}

// Trigger gates and enqueues code-RCA runs from the dispatch path.
type Trigger struct {
	cfgs   ruletypes.CodebaseRCAConfigStore
	maps   ruletypes.CodebaseServiceMapStore
	runs   Admitter
	logger *slog.Logger
	now    func() time.Time
}

// New builds a Trigger. logger may be nil (slog.Default); now may be nil.
func New(cfgs ruletypes.CodebaseRCAConfigStore, maps ruletypes.CodebaseServiceMapStore, runs Admitter, logger *slog.Logger, now func() time.Time) *Trigger {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return &Trigger{cfgs: cfgs, maps: maps, runs: runs, logger: logger.With(slog.String("component", "ds-apm-coderca-trigger")), now: now}
}

// Maybe evaluates the gate chain for an unbound alert and, when every gate
// passes, atomically admits a queued run. It NEVER returns an error or panics;
// it returns quickly (bounded by maybeTimeout). Gate order (design §6.1/§10):
// feature_on → anomaly(fail-closed) → severity → service→repo → Admit.
func (t *Trigger) Maybe(ctx context.Context, orgID string, labels, annotations map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			t.logger.ErrorContext(ctx, "coderca trigger: recovered panic", slog.Any("panic", r))
		}
	}()
	if t == nil || orgID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, maybeTimeout)
	defer cancel()

	cfg, err := t.cfgs.Get(ctx, orgID)
	if err != nil {
		// not-found → 기본값(Enabled=false) = feature off. 그 외 에러 → fail-closed.
		return
	}
	if !cfg.Enabled {
		return // feature_off: DB 기록 없음 (§6.1 — pre-check는 DB 미접촉)
	}

	if !anomalous(labels, annotations) {
		if cfg.AllowUnboundWithoutAnomaly {
			t.logger.WarnContext(ctx, "coderca trigger: admitting WITHOUT anomaly signal (allow_unbound_without_anomaly is ON)", slog.String("orgId", orgID))
		} else {
			return // fail-closed (§10)
		}
	}

	if !ruletypes.SeverityAtLeast(labels["severity"], cfg.MinSeverity) {
		return
	}

	service := strings.TrimSpace(labels["service.name"])
	if service == "" {
		return
	}
	if _, err := t.maps.Get(ctx, orgID, service); err != nil {
		// 미매핑(또는 조회 실패) → skip. no_repo_mapping만 집계 (§6.4).
		_ = t.runs.RecordSkip(ctx, orgID, coderca.SkipNoRepoMapping, t.now())
		return
	}

	sig := coderca.ErrorSignature(labels)
	res, err := t.runs.Admit(ctx, runstore.AdmitParams{
		OrgID:          orgID,
		Service:        service,
		DedupKey:       coderca.DedupKey(orgID, service, sig),
		Now:            t.now(),
		CooldownWindow: time.Duration(cfg.CooldownWindowSecs) * time.Second,
		MaxRunsPerDay:  cfg.MaxRunsPerDay,
		MaxQueueDepth:  cfg.MaxQueueDepth,
	})
	if err != nil {
		t.logger.WarnContext(ctx, "coderca trigger: admit failed", slog.String("orgId", orgID), slog.Any("err", err))
		return
	}
	if res.Admitted {
		t.logger.InfoContext(ctx, "coderca trigger: run queued", slog.String("orgId", orgID), slog.String("service", service), slog.String("runId", res.RunID))
	}
}

// anomalous reports whether the alert carries the explicit anomaly signal.
func anomalous(labels, annotations map[string]string) bool {
	for _, m := range []map[string]string{labels, annotations} {
		v := strings.ToLower(strings.TrimSpace(m[AnomalyLabel]))
		if v == "true" || v == "1" {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: 통과 확인**

Run: `go test ./pkg/ruler/coderca/trigger/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/ruler/coderca/trigger/
git commit -m "[Feat] coderca Trigger 파사드 — fail-closed 게이트 체인 + 원자 admission"
```

---

### Task 6: 워커 서비스 (factory.Service)

**Files:**
- Create: `pkg/ruler/coderca/worker/worker.go`
- Test: `pkg/ruler/coderca/worker/worker_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

```go
package worker

// fakeEngine: ProcessNext 호출 횟수 기록, 처음 N회 processed=true 후 false.
// fakeReaper: Reap 호출 기록.

func TestWorkerDrainsQueueOnTick(t *testing.T) {
	// pollEvery=10ms로 Start(goroutine) → 100ms 대기 → Stop.
	// 단언: ProcessNext 총 호출 ≥ N+1 (큐 소진까지 연속 호출 후 idle 폴링).
}

func TestWorkerReapsPeriodically(t *testing.T) {
	// reapEvery=10ms → Stop 후 fakeReaper.calls ≥ 1, ReapParams.MaxAttempts == 2.
}

func TestWorkerStopUnblocksStart(t *testing.T) {
	// Start는 블로킹; Stop 호출 시 1초 내 Start가 반환.
}

func TestWorkerSurvivesEngineError(t *testing.T) {
	// ProcessNext가 에러 반환해도 루프 지속 (다음 tick에 재호출).
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./pkg/ruler/coderca/worker/ -count=1`
Expected: FAIL

- [ ] **Step 3: 구현**

```go
// Package worker runs the coderca engine as a SigNoz factory.Service: a
// polling loop that drains queued runs and periodically reaps expired leases
// (design §5.1 worker pool / §6.3 reaper).
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
)

const (
	DefaultPollEvery   = 5 * time.Second
	DefaultReapEvery   = time.Minute
	DefaultMaxAttempts = 2 // design §6.3
)

// Engine processes one queued run (satisfied by *engine.Engine).
type Engine interface {
	ProcessNext(ctx context.Context) (bool, error)
}

// Reaper sweeps expired leases (satisfied by *runstore.Store).
type Reaper interface {
	Reap(ctx context.Context, p runstore.ReapParams) (int, error)
}

// Worker is the coderca background service.
type Worker struct {
	engine      Engine
	reaper      Reaper
	scope       string
	maxAttempts int
	pollEvery   time.Duration
	reapEvery   time.Duration
	now         func() time.Time
	logger      *slog.Logger
	stop        chan struct{}
	done        chan struct{}
}

// New builds a Worker. Zero durations fall back to defaults; logger/now may be nil.
func New(engine Engine, reaper Reaper, scope string, pollEvery, reapEvery time.Duration, logger *slog.Logger, now func() time.Time) *Worker {
	if pollEvery <= 0 {
		pollEvery = DefaultPollEvery
	}
	if reapEvery <= 0 {
		reapEvery = DefaultReapEvery
	}
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	if scope == "" {
		scope = "global"
	}
	return &Worker{
		engine: engine, reaper: reaper, scope: scope,
		maxAttempts: DefaultMaxAttempts,
		pollEvery:   pollEvery, reapEvery: reapEvery,
		now:    now,
		logger: logger.With(slog.String("component", "ds-apm-coderca-worker")),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

// Start blocks (factory.Service contract) until Stop or ctx cancel.
func (w *Worker) Start(ctx context.Context) error {
	defer close(w.done)
	poll := time.NewTicker(w.pollEvery)
	defer poll.Stop()
	reap := time.NewTicker(w.reapEvery)
	defer reap.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-w.stop:
			return nil
		case <-poll.C:
			w.drain(ctx)
		case <-reap.C:
			if _, err := w.reaper.Reap(ctx, runstore.ReapParams{Scope: w.scope, Now: w.now(), MaxAttempts: w.maxAttempts}); err != nil {
				w.logger.WarnContext(ctx, "coderca worker: reap failed", slog.Any("err", err))
			}
		}
	}
}

// drain processes queued runs until the queue is empty or capacity is hit.
func (w *Worker) drain(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		default:
		}
		processed, err := w.engine.ProcessNext(ctx)
		if err != nil {
			w.logger.WarnContext(ctx, "coderca worker: process failed", slog.Any("err", err))
			return // 다음 tick에 재시도
		}
		if !processed {
			return
		}
	}
}

// Stop signals Start to return and waits for it.
func (w *Worker) Stop(ctx context.Context) error {
	close(w.stop)
	select {
	case <-w.done:
	case <-ctx.Done():
	}
	return nil
}
```

- [ ] **Step 4: 통과 확인**

Run: `go test ./pkg/ruler/coderca/worker/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/ruler/coderca/worker/
git commit -m "[Feat] coderca 워커 서비스 — 폴링 드레인 + lease reaper (factory.Service)"
```

---

### Task 7: CF-7 anomaly 라벨 스탬프

**Files:**
- Modify: `pkg/query-service/rules/anomaly_rule.go` (라벨 빌드 지점, ~283행 `lb.Set(ruletypes.AlertNameLabel, ...)` 인접)
- Test: `pkg/query-service/rules/anomaly_rule_test.go` (기존 보강)

- [ ] **Step 1: 실패하는 테스트 작성** — 기존 anomaly_rule_test의 알람 생성 경로 테스트를 미러해, 발화된 알람 라벨에 `anomaly=true`가 포함됨을 단언:

```go
func TestAnomalyRuleStampsAnomalyLabel(t *testing.T) {
	// 기존 발화 경로 테스트 하니스 재사용(anomaly_rule_test.go의 firing 케이스).
	// 단언: 생성된 alert.Labels에 {"anomaly": "true"} 포함.
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./pkg/query-service/rules/ -run TestAnomalyRuleStampsAnomalyLabel -count=1`
Expected: FAIL

- [ ] **Step 3: 구현** — `anomaly_rule.go`의 라벨 빌드 블록(`lb.Set(ruletypes.AlertNameLabel, r.Name())` 283행 인접)에 1줄 추가:

```go
		// CF-11 trigger signal (design §10): anomaly alerts carry an explicit
		// marker so the code-RCA gate stays fail-closed for everything else.
		lb.Set("anomaly", "true")
```

- [ ] **Step 4: 통과 확인 (기존 룰 테스트 포함)**

Run: `go test ./pkg/query-service/rules/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/query-service/rules/
git commit -m "[Feat] CF-7 anomaly 알람에 anomaly=true 라벨 스탬프 (CF-11 트리거 신호)"
```

---

### Task 8: 핸드오프 sink (CF-3 메타-알람) + 감사 sink (CF-6)

**Files:**
- Create: `pkg/ruler/coderca/delivery/amsink.go`
- Create: `pkg/ruler/coderca/auditor/dssink.go`
- Test: `pkg/ruler/coderca/delivery/amsink_test.go`, `pkg/ruler/coderca/auditor/dssink_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

```go
// amsink_test.go — fake AlertPutter가 받은 PostableAlerts를 캡처해 단언:
func TestAlertmanagerSinkSubmitsMetaAlert(t *testing.T) {
	// Submit(msg{OrgID:"org-1", Service:"pay", RunID:"r1", BaselineCommit:"abc",
	//            Title:"T", Body:"B"}) 후:
	// putter.orgID == "org-1"
	// labels: alertname=="CodeRCASuggestion", service.name=="pay",
	//         severity=="info", coderca=="true"
	// annotations: summary=="T", description=="B", coderca.run_id=="r1",
	//              coderca.baseline_commit=="abc"
	// 반환 ref == "r1"; putter 에러 시 에러 전파.
}

// dssink_test.go — fake Audit 함수가 받은 AuditEvent 단언:
func TestDSSinkRecordsAuditEvent(t *testing.T) {
	// Record(AuditRecord{...Status: done, Outcome: "success"...}) 후:
	// event.EventName.String() == "coderca.run.updated"
	// event.Body에 runID·orgID·status 포함, event.Timestamp == rec.At
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./pkg/ruler/coderca/delivery/ ./pkg/ruler/coderca/auditor/ -count=1`
Expected: FAIL

- [ ] **Step 3: 구현**

`delivery/amsink.go` — 알람 모델 필드명은 `pkg/types/alertmanagertypes/alert.go`(`PostableAlert = models.PostableAlert`, prometheus/alertmanager v2 models)를 Read해 확인 후 작성:

```go
package delivery

import (
	"context"

	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/prometheus/alertmanager/api/v2/models"
)

// AlertPutter is the narrow slice of alertmanager.Alertmanager the sink needs.
type AlertPutter interface {
	PutAlerts(ctx context.Context, orgID string, alerts alertmanagertypes.PostableAlerts) error
}

// AlertmanagerSink delivers a code-RCA handoff as a meta-alert through the
// normal CF-3 dispatch path (channels, templates, PII filter all reused).
type AlertmanagerSink struct {
	am AlertPutter
}

// NewAlertmanagerSink builds the sink over the running alertmanager.
func NewAlertmanagerSink(am AlertPutter) *AlertmanagerSink {
	return &AlertmanagerSink{am: am}
}

// Submit publishes the handoff as a PostableAlert; ref = run id.
func (s *AlertmanagerSink) Submit(ctx context.Context, msg HandoffMessage) (string, error) {
	alert := &alertmanagertypes.PostableAlert{
		Annotations: models.LabelSet{
			"summary":                 msg.Title,
			"description":             msg.Body,
			"coderca.run_id":          msg.RunID,
			"coderca.baseline_commit": msg.BaselineCommit,
		},
		Alert: models.Alert{
			Labels: models.LabelSet{
				"alertname":    "CodeRCASuggestion",
				"service.name": msg.Service,
				"severity":     "info",
				"coderca":      "true",
			},
		},
	}
	if err := s.am.PutAlerts(ctx, msg.OrgID, alertmanagertypes.PostableAlerts{alert}); err != nil {
		return "", err
	}
	return msg.RunID, nil
}

var _ HandoffSink = (*AlertmanagerSink)(nil)
```

`auditor/dssink.go`:

```go
package auditor

import (
	"context"
	"fmt"

	"github.com/SigNoz/signoz/pkg/types/audittypes"
)

// AuditFunc is the narrow slice of pkg/auditor.Auditor the sink needs
// (fire-and-forget, drop-on-full upstream).
type AuditFunc func(ctx context.Context, event audittypes.AuditEvent)

// DSSink bridges coderca audit records to the CF-6 auditor service.
type DSSink struct {
	audit AuditFunc
}

// NewDSSink builds the sink over auditor.Audit.
func NewDSSink(audit AuditFunc) *DSSink {
	return &DSSink{audit: audit}
}

// Record maps the record onto a ds audit event and emits it.
func (s *DSSink) Record(ctx context.Context, rec AuditRecord) {
	if s.audit == nil {
		return
	}
	s.audit(ctx, audittypes.AuditEvent{
		Timestamp: rec.At,
		EventName: audittypes.NewEventName("coderca.run", audittypes.ActionUpdate),
		Body: fmt.Sprintf("coderca run finalized: run=%s org=%s service=%s status=%s outcome=%s detail=%s",
			rec.RunID, rec.OrgID, rec.Service, rec.Status, rec.Outcome, rec.Detail),
	})
}

var _ Sink = (*DSSink)(nil)
```

- [ ] **Step 4: 통과 확인**

Run: `go test ./pkg/ruler/coderca/delivery/ ./pkg/ruler/coderca/auditor/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/ruler/coderca/delivery/ pkg/ruler/coderca/auditor/
git commit -m "[Feat] coderca 핸드오프(메타-알람)·감사 sink — CF-3/CF-6 경로 재사용"
```

---

### Task 9: 디스패치 훅 seam — Trigger 주입 + unbound 분기 호출

**Files:**
- Modify: `pkg/ruler/aigenerator/dispatchhook/hook.go`
- Test: `pkg/ruler/aigenerator/dispatchhook/hook_test.go` (기존 보강)

- [ ] **Step 1: 실패하는 테스트 작성**

```go
// fakeTrigger records (orgID, labels, annotations) calls.

func TestApplyCallsCodeRCATriggerOnUnbound(t *testing.T) {
	// SOP 미바인딩 상태의 기존 unbound 테스트 케이스 재사용.
	// hook.SetCodeRCATrigger(fake) 후 Apply →
	// 단언: fake 호출 1회, orgID/labels/annotations 전달, 반환 annotations 불변.
}

func TestApplyDoesNotCallTriggerWhenBound(t *testing.T) {
	// SOP 바인딩되는 기존 happy-path 케이스에서 fake 호출 0회.
}

func TestApplyUnchangedWhenTriggerNil(t *testing.T) {
	// SetCodeRCATrigger 미호출(기본 nil) — 기존 unbound 동작 그대로, panic 없음.
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./pkg/ruler/aigenerator/dispatchhook/ -run "CodeRCA|TriggerNil" -count=1`
Expected: FAIL

- [ ] **Step 3: 구현** — `hook.go`:

(1) 인터페이스 + 필드 + setter (coderca 패키지 임포트 없이 — 의존 역전):

```go
// CodeRCATrigger is the CF-11 trigger seam (design §11): called fire-and-
// forget on the unbound branch. Implementations must never panic or block
// beyond their own internal timeout.
type CodeRCATrigger interface {
	Maybe(ctx context.Context, orgID string, labels, annotations map[string]string)
}
```

`Hook` 구조체에 `codeRCA CodeRCATrigger` 필드 추가, 그리고:

```go
// SetCodeRCATrigger injects the CF-11 trigger after construction (the trigger
// depends on stores built later in server wiring). nil-safe; optional.
func (h *Hook) SetCodeRCATrigger(t CodeRCATrigger) { h.codeRCA = t }
```

(2) unbound 분기(117~123행)에서 호출 — **unbound 상태일 때만** (forbidden/disabled/validation은 제외). `ruletypes`의 SOP 바인딩 상태 상수를 grep으로 확인(`grep -n "SOPBindingStatus" pkg/types/ruletypes/*.go`)하고 unbound 상수를 사용:

```go
	if err != nil || binding.Status != ruletypes.SOPBindingStatusBound {
		// Unbound, forbidden, disabled, or validation failure — ... (기존 주석 유지)
		if err == nil && binding.Status == ruletypes.SOPBindingStatusUnbound && h.codeRCA != nil {
			// CF-11 (UJ-5): unbound 알람만 코드 RCA 게이트로 전달. 트리거는
			// fail-open 계약(에러·패닉 무전파, 자체 timeout)을 가지므로 디스패치
			// 경로에 추가 실패 모드를 만들지 않는다.
			h.codeRCA.Maybe(ctx, orgID, labels, annotations)
		}
		return annotations
	}
```

> grep 결과 unbound 상수명이 다르면(예: `SOPBindingStatusUnbound`가 아니라면) 실제 상수명으로 대체. "바인딩 없음"을 뜻하는 상태만 매칭해야 한다.

(3) 패키지 주석 정정 — `hook.go:11-14`의 낡은 "not yet wired" 문단을 다음으로 교체:

```go
// This hook is wired into the dispatcher's notify path via
// pkg/alertmanager/alertmanagerserver/dispatcher.go (applyAIHook) and is
// constructed in pkg/signoz/signoz.go.
```

- [ ] **Step 4: 통과 확인 (기존 훅 테스트 전체)**

Run: `go test ./pkg/ruler/aigenerator/dispatchhook/ -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/ruler/aigenerator/dispatchhook/
git commit -m "[Feat] 디스패치 훅 unbound 분기에 CF-11 트리거 seam 배선 (nil-safe)"
```

---

### Task 10: 서버 배선 (signoz.go) — 스토어·엔진·트리거·워커 구성

**Files:**
- Modify: `pkg/signoz/signoz.go` (aiDispatchHook 구성부 ~420행, alertmanager 구성 후 ~441행, registry ~556행)

- [ ] **Step 1: 구성 코드 추가** — `signoz.go`에서 alertmanager 생성(441행) **이후**, auditor 생성(450행) **이후** 지점에 coderca 구성 블록 삽입. 먼저 `sqlcodebaseconfigstore`/`sqlcodebaseservicemapstore`의 실제 생성자명을 확인(`grep -n "^func New" pkg/ruler/coderca/codebaseconfigstore/*/*.go`)하고 다음을 작성:

```go
	// ── CF-11 code RCA (coderca) — integration wiring (design §11) ──────────
	// Per-org stores + cost-control run store.
	codercaRepoStore := sqlcodebaseconfigstore.New(sqlstore)
	codercaMapStore := sqlcodebaseservicemapstore.New(sqlstore)
	codercaCfgStore := sqlcodebasercaconfigstore.New(sqlstore)
	codercaRunStore := codercarunstore.New(sqlstore)

	// Trigger: injected into the dispatch hook (fail-open, fire-and-forget).
	if aiDispatchHook != nil {
		aiDispatchHook.SetCodeRCATrigger(codercatrigger.New(
			codercaCfgStore, codercaMapStore, codercaRunStore,
			providerSettings.Logger, nil,
		))
	}

	// Engine: deployment-level agent/model/auth from env (per-org thresholds
	// live in ds_codebase_config and are enforced at admission).
	codercaAgent := os.Getenv("DS_APM_CODERCA_AGENT")
	if codercaAgent == "" {
		codercaAgent = "claude"
	}
	codercaBudget := os.Getenv("DS_APM_CODERCA_MAX_BUDGET_USD")
	if codercaBudget == "" {
		codercaBudget = "0.50"
	}
	codercaAuth := os.Getenv("DS_APM_CODERCA_AUTH_TOKEN")
	if codercaAuth == "" {
		codercaAuth = pickAPIKey(os.Getenv("DS_APM_LLM_PROVIDER"))
	}
	codercaBaseDir := os.Getenv("DS_APM_CODERCA_DIR")
	if codercaBaseDir == "" {
		codercaBaseDir = filepath.Join(os.TempDir(), "ds-coderca")
	}
	hostname, _ := os.Hostname()

	gitRunner := codercasourcestate.NewShellGitRunner(filepath.Join(codercaBaseDir, "mirrors"))
	codercaEngine := codercaengine.New(
		codercaengine.Config{
			Scope:        "global",
			InstanceID:   hostname,
			Agent:        clirunner.Agent(codercaAgent),
			Model:        os.Getenv("DS_APM_CODERCA_MODEL"),
			MaxBudgetUSD: codercaBudget,
			AuthToken:    codercaAuth,
		},
		codercaengine.Deps{
			Runs:  codercaRunStore,
			Repos: reporesolver.New(codercaMapStore, codercaRepoStore, aiCipher.DecryptFunc()),
			Source: codercasourcestate.NewManager(gitRunner, filepath.Join(codercaBaseDir, "checkouts")),
			CLI:    clirunner.NewRunner(),
			Deliver: codercadelivery.New(codercadelivery.NewAlertmanagerSink(alertmanager)),
			Auditor: codercaauditor.New(codercaauditor.NewDSSink(auditor.Audit), nil),
		},
	)
	codercaWorker := codercaworker.New(codercaEngine, codercaRunStore, "global", 0, 0, providerSettings.Logger, nil)
```

임포트 별칭(충돌 회피):

```go
	codercaauditor "github.com/SigNoz/signoz/pkg/ruler/coderca/auditor"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/clirunner"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/codebaseconfigstore/sqlcodebaseconfigstore"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/codebaseconfigstore/sqlcodebasercaconfigstore"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/codebaseconfigstore/sqlcodebaseservicemapstore"
	codercadelivery "github.com/SigNoz/signoz/pkg/ruler/coderca/delivery"
	codercaengine "github.com/SigNoz/signoz/pkg/ruler/coderca/engine"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/reporesolver"
	codercarunstore "github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
	codercasourcestate "github.com/SigNoz/signoz/pkg/ruler/coderca/sourcestate"
	codercatrigger "github.com/SigNoz/signoz/pkg/ruler/coderca/trigger"
	codercaworker "github.com/SigNoz/signoz/pkg/ruler/coderca/worker"
```

> 주의: aiDispatchHook 구성(420행)은 alertmanager(432행)보다 앞이므로, **트리거 주입은 alertmanager 생성 뒤에 둬도 무방**(트리거는 alertmanager에 의존하지 않지만, coderca 블록을 한곳에 모으기 위해 450행 이후 단일 블록으로 배치). `auditor.Audit`는 메서드 값으로 전달.

- [ ] **Step 2: registry에 워커 등록** — 556행 `factory.NewRegistry(...)` 목록에 추가:

```go
		factory.NewNamedService(factory.MustNewName("codercaworker"), codercaWorker),
```

> 서비스 이름은 `factory.MustNewName` 규칙(소문자·영숫자)을 따라야 한다 — 기존 이름들(`statsreporter` 등)과 같은 형식.

- [ ] **Step 3: 빌드 + 기존 테스트**

Run: `go build ./... && go test ./pkg/signoz/ -count=1`
Expected: PASS (컴파일 OK; handler_test 등 기존 테스트 영향 없음 — NewHandlers는 B에서 변경)

- [ ] **Step 4: Commit**

```bash
git add pkg/signoz/signoz.go
git commit -m "[Feat] 서버 배선 — coderca 스토어·엔진·트리거·워커 구성 및 서비스 등록"
```

---

### Task 11: A 게이트 — e2e 통합 테스트 (가짜 CLI)

**Files:**
- Create: `pkg/ruler/coderca/integration_test.go`

- [ ] **Step 1: 테스트 작성** — 실제 SQLite 테스트 스토어(기존 runstore 테스트 하니스 재사용) + fake CLI/git/sink로 트리거→워커 전체 경로 검증:

```go
package coderca_test

// 구성: sqlite 스토어(마이그레이션 082+083 적용) 위에
//   cfgStore(Enabled=true 업서트), mapStore(svc→repo 업서트), repoStore(enabled repo),
//   trigger.New(...), engine.New(... CLI=fakeCLI, Source=fakeSource, Deliver=캡처 sink, Auditor=캡처 sink)

func TestUJ5EndToEnd_UnboundAnomalyAlertProducesHandoff(t *testing.T) {
	// 1) trigger.Maybe(ctx, "org-1", labels{anomaly:true, severity:critical,
	//    service.name:pay, alertname:PayErr}, nil)
	// 2) 단언: coderca_run 1건 queued (runstore.ListRuns)
	// 3) engine.ProcessNext 1회 → processed=true
	// 4) 단언: 캡처 sink가 HandoffMessage 수신 (Service=="pay", Body에 RootCause 포함)
	// 5) 단언: run status==done, root_cause 영속 (GetRun)
	// 6) 같은 labels로 Maybe 재호출 → 쿨다운 dedup, run 총 1건 유지
}

func TestUJ5FailOpen_TriggerNeverBlocksAlertPath(t *testing.T) {
	// dispatchhook.Hook + SetCodeRCATrigger(panic하는 trigger / 2초 sleep하는 cfgStore)
	// → hook.Apply(unbound 알람)가 annotations 불변으로 정상 반환 (panic·블로킹 없음)
}

func TestUJ5FailClosed_NoFiringWithoutGates(t *testing.T) {
	// (a) anomaly 라벨 없음 (b) Enabled=false (c) 매핑 없음 — 각각 run 0건
}
```

- [ ] **Step 2: 통과할 때까지 수정**

Run: `go test ./pkg/ruler/coderca/ -run TestUJ5 -count=1 -v`
Expected: PASS — **이 게이트 통과 전 Phase B 착수 금지**

- [ ] **Step 3: 전체 회귀**

Run: `go test ./pkg/ruler/... ./pkg/signoz/ ./pkg/sqlmigration/ ./pkg/query-service/rules/ -count=1`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add pkg/ruler/coderca/integration_test.go
git commit -m "[Test] CF-11 A 게이트 — UJ-5 e2e(트리거→워커→핸드오프) + fail-open/closed"
```

---

## Phase B — HTTP API

### Task 12: 스토어 Delete 메서드 (repo·service map)

**Files:**
- Modify: `pkg/types/ruletypes/codebase_config_store.go` (+Delete), `pkg/types/ruletypes/codebase_service_map_store.go` (+Delete)
- Modify: `pkg/ruler/coderca/codebaseconfigstore/sqlcodebaseconfigstore/sqlcodebaseconfigstore.go`, `.../sqlcodebaseservicemapstore/sqlcodebaseservicemapstore.go`
- Test: 각 기존 `_test.go` 보강

- [ ] **Step 1: 실패하는 테스트** — Upsert→Delete→Get이 NotFound가 되는 왕복 + 타 org 행 보존(테넌트 격리) 단언을 두 스토어 테스트에 추가.
- [ ] **Step 2: 실패 확인** — `go test ./pkg/ruler/coderca/codebaseconfigstore/... -count=1` → FAIL.
- [ ] **Step 3: 구현** — 인터페이스에 `Delete(ctx context.Context, orgID, repoID string) error` / `Delete(ctx context.Context, orgID, serviceName string) error` 추가, SQL 구현(`DELETE FROM ... WHERE org_id=? AND ...`; 0행이어도 nil — 멱등).
- [ ] **Step 4: 통과 확인** — 동일 명령 PASS + `go build ./...`.
- [ ] **Step 5: Commit** — `git commit -m "[Feat] codebase repo·service map 스토어 Delete 추가 (멱등)"`

---

### Task 13: coderca_handler.go — HTTP 핸들러

**Files:**
- Create: `pkg/ruler/signozruler/coderca_handler.go`
- Modify: `pkg/ruler/signozruler/handler.go` (필드 4개 + NewHandler 파라미터)
- Test: `pkg/ruler/signozruler/coderca_handler_test.go`

API 표면 (모두 org 스코프 = claims에서, `ai_config_handler.go` 패턴 미러):

| Method | Path | Handler | 권한(라우트 등록 시) |
|---|---|---|---|
| GET | `/api/v2/ds/coderca/config` | GetCodebaseRCAConfig (미존재 → 기본값 반환) | View |
| PUT | `/api/v2/ds/coderca/config` | UpdateCodebaseRCAConfig | Edit |
| GET | `/api/v2/ds/coderca/repos` | ListCodebaseRepos (credential 스크럽: 있으면 `<unchanged>`) | View |
| PUT | `/api/v2/ds/coderca/repos` | UpsertCodebaseRepo (`<unchanged>` 센티널 = 기존 자격증명 유지) | Edit |
| DELETE | `/api/v2/ds/coderca/repos/{repoId}` | DeleteCodebaseRepo | Edit |
| GET | `/api/v2/ds/coderca/service-maps` | ListCodebaseServiceMaps | View |
| PUT | `/api/v2/ds/coderca/service-maps` | UpsertCodebaseServiceMap | Edit |
| DELETE | `/api/v2/ds/coderca/service-maps/{serviceName}` | DeleteCodebaseServiceMap | Edit |
| GET | `/api/v2/ds/coderca/runs?status=&service=&limit=&offset=` | ListCodeRCARuns | View |
| GET | `/api/v2/ds/coderca/runs/{runId}` | GetCodeRCARun (404: ErrRunNotFound) | View |

- [ ] **Step 1: 실패하는 테스트 작성** — `handler_ai_test.go`의 핸들러 테스트 하니스(요청 빌드·claims 주입·sqlite 스토어) 패턴을 Read 후 미러. 필수 케이스:

```go
// TestUpsertCodebaseRepoStoresCiphertextAndScrubsResponse:
//   PUT repos {credential:"tok"} → 200; DB의 credential_ciphertext != "tok"(암호문);
//   GET repos 응답의 credential == "<unchanged>" (평문 비노출 — 설계 AC).
// TestUpsertCodebaseRepoUnchangedSentinelPreservesCredential:
//   PUT {credential:"<unchanged>"} → 기존 자격증명 유지 (Get으로 검증).
// TestUpsertCodebaseRepoFailClosedWithoutEncryption:
//   encryptionAvailable=false 하니스에서 credential 있는 PUT → 400
//   (ValidateCodebaseRepo가 거부).
// TestRCAConfigGetDefaultsAndPutRoundTrip:
//   GET → 기본값(Enabled=false); PUT 유효 페이로드 → 204; GET 반영.
//   PUT MinSeverity:"nonsense" → 400.
// TestServiceMapCRUD: PUT→GET 목록→DELETE→빈 목록.
// TestRunsListAndDetail: runstore에 시드 후 GET runs(필터) / GET runs/{id} /
//   타 org runID → 404.
```

- [ ] **Step 2: 실패 확인** — `go test ./pkg/ruler/signozruler/ -run "Codebase|RCAConfig|CodeRCA" -count=1` → FAIL.

- [ ] **Step 3: 구현** — `handler` 구조체에 추가:

```go
	// CF-11 code RCA settings + run history (coderca_handler.go).
	codebaseRepoStore ruletypes.CodebaseRepoStore
	codebaseMapStore  ruletypes.CodebaseServiceMapStore
	codercaCfgStore   ruletypes.CodebaseRCAConfigStore
	codercaRunStore   *codercarunstore.Store
```

`NewHandler` 파라미터 4개 추가(기존 순서 뒤에). 각 핸들러는 `ai_config_handler.go`와 동일 골격: `extractOrgID` → `binding.JSON.BindBody` → `Validate*`(orgID는 claims로 강제) → store 호출 → `render.Success`/`render.Error`. 자격증명 처리:

```go
// Upsert에서:
	encryptionAvailable := !handler.aiCipherInsecure // 아래 참고
	if incoming.Credential == APIKeyPlaceholder {
		existing, getErr := handler.codebaseRepoStore.Get(req.Context(), orgID, incoming.RepoID, handler.aiCipher.DecryptFunc())
		if getErr == nil {
			incoming.Credential = existing.Credential
		} else if errors.Is(getErr, ruletypes.ErrCodebaseRepoNotFound) {
			incoming.Credential = ""
		} else { render.Error(rw, ...); return }
	}
	if err := ruletypes.ValidateCodebaseRepo(incoming, encryptionAvailable); err != nil { ... 400 ... }
	if err := handler.codebaseRepoStore.Upsert(req.Context(), incoming, handler.aiCipher.EncryptFunc()); err != nil { ... }
```

> `encryptionAvailable`: `secretbox.FromEnv()`의 `insecure` 플래그가 핸들러까지 전달돼야 한다. `handler`에 `aiCipherInsecure bool` 필드를 추가하고 NewHandler 파라미터로 받는다(signoz.go의 `insecure` 변수를 전달 — Task 14). 목록/단건 응답 전 `repo.Credential = scrubAPIKey(repo.Credential)` 재사용.

run 핸들러:

```go
func (handler *handler) ListCodeRCARuns(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	// ... ai_config과 동일한 org 가드 ...
	q := req.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	runs, err := handler.codercaRunStore.ListRuns(req.Context(), orgID, codercarunstore.ListRunsParams{
		Status: q.Get("status"), Service: q.Get("service"), Limit: limit, Offset: offset,
	})
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "list coderca runs"))
		return
	}
	render.Success(rw, http.StatusOK, runs)
}

func (handler *handler) GetCodeRCARun(rw http.ResponseWriter, req *http.Request) {
	// mux.Vars(req)["runId"]; ErrRunNotFound →
	// render.Error(rw, errors.Newf(errors.TypeNotFound, errors.CodeNotFound, "run not found"))
}
```

- [ ] **Step 4: 통과 확인** — `go test ./pkg/ruler/signozruler/ -count=1` PASS (기존 포함). `go build ./...`는 NewHandler 호출부(signoz/handler.go) 미수정으로 실패할 수 있음 — Task 14에서 해소하므로 여기서는 패키지 단위 테스트만.

- [ ] **Step 5: Commit**

```bash
git add pkg/ruler/signozruler/
git commit -m "[Feat] coderca HTTP 핸들러 — 설정·저장소·매핑 CRUD + run 이력 조회"
```

---

### Task 14: 라우트 등록 + NewHandlers 배선

**Files:**
- Modify: `pkg/apiserver/signozapiserver/ruler.go` (ai/config 블록 314행 인접에 coderca 블록 추가)
- Modify: `pkg/signoz/handler.go` (`NewHandlers` 파라미터·`signozruler.NewHandler` 호출 132행)
- Modify: `pkg/signoz/signoz.go` (577행 NewHandlers 호출에 스토어 전달)
- Modify: `pkg/signoz/handler_test.go` (59행 호출 시그니처)

- [ ] **Step 1: 라우트 등록** — `ruler.go`의 `GetAIConfig` 블록(314행) 패턴 그대로 10개 라우트 추가. 대표 예 (나머지는 표의 Method/Path/권한으로 동일 패턴 반복):

```go
	if err := router.Handle("/api/v2/ds/coderca/config", handler.New(provider.authZ.ViewAccess(provider.rulerHandler.GetCodebaseRCAConfig), handler.OpenAPIDef{
		ID:          "GetCodebaseRCAConfig",
		Tags:        []string{"coderca"},
		Summary:     "Get code RCA config",
		Description: "Returns the org's CF-11 code-RCA feature toggle and cost thresholds (defaults when unset).",
		Response:    new(ruletypes.CodebaseRCAConfig),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}
```

> `OpenAPIDef` 필드 구성(Tags/Summary 유무)은 기존 GetAIConfig 블록을 Read해 그대로 미러.

- [ ] **Step 2: NewHandlers/NewHandler 배선** — `pkg/signoz/handler.go`의 `NewHandlers`에 파라미터 추가: `codebaseRepoStore ruletypes.CodebaseRepoStore, codebaseMapStore ruletypes.CodebaseServiceMapStore, codercaCfgStore ruletypes.CodebaseRCAConfigStore, codercaRunStore *codercarunstore.Store, aiCipherInsecure bool` — `signozruler.NewHandler(...)` 호출에 전달. `signoz.go:577` 호출에 Task 10에서 만든 스토어 변수 + `insecure` 전달. `handler_test.go:59` 호출엔 `nil, nil, nil, nil, false` 추가.

- [ ] **Step 3: 빌드 + 전체 테스트**

Run: `go build ./... && go test ./pkg/signoz/ ./pkg/apiserver/... ./pkg/ruler/signozruler/ -count=1`
Expected: PASS — **B 게이트**

- [ ] **Step 4: Commit**

```bash
git add pkg/apiserver/signozapiserver/ruler.go pkg/signoz/
git commit -m "[Feat] coderca 라우트 등록 + NewHandlers 배선 (B 게이트)"
```

---

## Phase C — FE 설정 + 이력

### Task 15: API 클라이언트 + 타입

**Files:**
- Create: `frontend/src/api/codeRca/types.ts`, `getConfig.ts`, `updateConfig.ts`, `listRepos.ts`, `upsertRepo.ts`, `deleteRepo.ts`, `listServiceMaps.ts`, `upsertServiceMap.ts`, `deleteServiceMap.ts`, `listRuns.ts`, `getRun.ts`

- [ ] **Step 1: 타입 + 클라이언트 작성** — `api/aiModule/getAIConfig.ts` 패턴(`ApiV2Instance`) 미러. `types.ts`:

```typescript
export interface CodeRcaConfig {
	contractVersion: string;
	orgId: string;
	enabled: boolean;
	minSeverity: 'critical' | 'high' | 'error' | 'warning' | 'info';
	cooldownWindowSecs: number;
	maxRunsPerDay: number;
	maxQueueDepth: number;
	maxConcurrentRuns: number;
	allowUnboundWithoutAnomaly: boolean;
	updatedAt: string;
}

export interface CodebaseRepo {
	contractVersion: string;
	orgId: string;
	repoId: string;
	gitUrl: string;
	defaultBranch: string;
	credential: string; // '<unchanged>' 센티널 = 저장된 자격증명 유지
	enabled: boolean;
	branchName: string;
	fetched: boolean;
	baselineCommit: string;
	lastSyncAt: string;
	lastSyncStatus: string;
}

export interface CodebaseServiceMap {
	orgId: string;
	serviceName: string;
	repoId: string;
	subpath: string;
}

export type CodeRcaRunStatus =
	| 'queued' | 'running' | 'done' | 'failed' | 'timeout' | 'unparseable';

export interface CodeRcaRunSummary {
	runId: string;
	orgId: string;
	service: string;
	status: CodeRcaRunStatus;
	baselineCommit: string;
	createdAt: number;
	finishedAt: number;
	attempts: number;
	resultRef: string;
}

export interface CodeRcaRunDetail extends CodeRcaRunSummary {
	rootCause: string;
	proposedFix: string;
	confidence: string;
	limitations: string;
}

export const CREDENTIAL_UNCHANGED = '<unchanged>';
```

클라이언트 예 (`listRuns.ts` — 나머지도 동일 골격, 경로/메서드만 변경):

```typescript
import { ApiV2Instance } from 'api';
import { AxiosResponse } from 'axios';

import { CodeRcaRunSummary } from './types';

export interface ListRunsParams {
	status?: string;
	service?: string;
	limit?: number;
	offset?: number;
}

const listRuns = (
	params: ListRunsParams,
): Promise<AxiosResponse<CodeRcaRunSummary[]>> =>
	ApiV2Instance.get<CodeRcaRunSummary[]>('/ds/coderca/runs', { params });

export default listRuns;
```

> DELETE 클라이언트는 `ApiV2Instance.delete(\`/ds/coderca/repos/${encodeURIComponent(repoId)}\`)`. 응답이 `render.Success` 래핑(`{status, data}`)인지 평문인지는 `getAIConfig` 사용부(AIModuleSettings의 `res.data` 접근)를 확인해 동일하게 처리.

- [ ] **Step 2: 타입 체크**

Run: `cd frontend && node node_modules/typescript/bin/tsc --noEmit -p tsconfig.json 2>&1 | head -20`
Expected: 신규 에러 0건

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api/codeRca/
git commit -m "[Feat] coderca FE API 클라이언트·타입 (수기, aiModule 패턴)"
```

---

### Task 16: CodeRcaSettings 컨테이너 (설정 탭 + 이력 탭)

**Files:**
- Create: `frontend/src/container/CodeRcaSettings/CodeRcaSettings.tsx`
- Create: `frontend/src/container/CodeRcaSettings/ConfigTab.tsx`
- Create: `frontend/src/container/CodeRcaSettings/RunsTab.tsx`
- Create: `frontend/src/container/CodeRcaSettings/CodeRcaSettings.styles.scss`
- Create: `frontend/public/locales/en/codeRca.json`, `frontend/public/locales/ko/codeRca.json`
- Test: `frontend/src/container/CodeRcaSettings/CodeRcaSettings.test.tsx`

- [ ] **Step 1: 실패하는 테스트 작성** — `AIModuleSettings.test.tsx`의 렌더·mock 패턴을 Read 후 미러. **i18n 규칙**: 전역 test-utils mock이 t()를 키로 반환하므로 assertion은 키 문자열 기준. api/codeRca/*는 `jest.mock`으로 대체.

```tsx
// 케이스:
// 1) 로드 시 getConfig/listRepos/listServiceMaps 호출, 'tab_config' 탭 렌더
// 2) enabled 토글 + 저장 → updateConfig 호출 payload 단언
// 3) repo 추가 폼 제출 → upsertRepo 호출 (credential 그대로),
//    기존 행 수정 시 credential 미입력이면 '<unchanged>' 전송
// 4) 'tab_runs' 탭 클릭 → listRuns 호출, 상태 태그 렌더
// 5) run 행 클릭 → getRun 호출, drawer에 rootCause/proposedFix 텍스트 렌더
```

- [ ] **Step 2: 실패 확인**

Run: `cd frontend && node node_modules/jest/bin/jest.js --silent "CodeRcaSettings"`
Expected: FAIL

- [ ] **Step 3: 구현** — 골격 (antd `Tabs`·`Table`·`Drawer`·`Form`, AIModuleSettings의 카드/필드 클래스 구조와 스타일 미러):

`CodeRcaSettings.tsx`:

```tsx
import { useTranslation } from 'react-i18next';
import { Tabs } from 'antd';
import { useAppContext } from 'providers/App/App';
import { USER_ROLES } from 'types/roles';

import ConfigTab from './ConfigTab';
import RunsTab from './RunsTab';

import './CodeRcaSettings.styles.scss';

function CodeRcaSettings(): JSX.Element {
	const { t } = useTranslation(['codeRca']);
	const { user } = useAppContext();
	const isAdmin = user.role === USER_ROLES.ADMIN;

	return (
		<div className="code-rca-settings" data-testid="code-rca-settings">
			<header className="code-rca-settings__header">
				<h1 className="code-rca-settings__header-title">{t('header_title')}</h1>
				<p className="code-rca-settings__header-subtitle">{t('header_subtitle')}</p>
			</header>
			<Tabs
				defaultActiveKey="config"
				items={[
					{ key: 'config', label: t('tab_config'), children: <ConfigTab isAdmin={isAdmin} /> },
					{ key: 'runs', label: t('tab_runs'), children: <RunsTab /> },
				]}
			/>
		</div>
	);
}

export default CodeRcaSettings;
```

`ConfigTab.tsx` — 3개 카드:
1. **기능·임계값 카드**: `enabled` Switch, `minSeverity` Select, `cooldownWindowSecs`/`maxRunsPerDay`/`maxQueueDepth`/`maxConcurrentRuns` InputNumber, `allowUnboundWithoutAnomaly` Switch(경고 문구 노출). 저장 버튼 → `updateConfig` (성공/실패 toast — AIModuleSettings의 `@signozhq/ui` toast 패턴).
2. **저장소 카드**: `listRepos` 데이터를 antd Table(repoId, gitUrl, defaultBranch, enabled, lastSyncStatus, baselineCommit 칼럼) + 추가/수정 모달 Form(repoId·gitUrl·defaultBranch·credential[Password, 기존 행이면 placeholder `t('credential_unchanged_hint')`]·enabled). 수정 시 credential 빈 입력 → `CREDENTIAL_UNCHANGED` 전송. 삭제 버튼 → `deleteRepo` + confirm.
3. **서비스 매핑 카드**: Table(serviceName, repoId, subpath) + 추가 폼 + 삭제. `contractVersion`은 `'ds.codebase_repo.v1'` 상수로 페이로드에 포함.

`RunsTab.tsx`:

```tsx
// 상태 필터 Select(전체/queued/running/done/failed/timeout/unparseable) +
// listRuns Table: columns = [createdAt(시각 포맷), service, status(Tag 색:
//   done=green, failed/timeout=red, running=blue, queued=default,
//   unparseable=orange), baselineCommit(앞 8자 <code>), attempts]
// 행 클릭 → getRun(runId) → Drawer:
//   <h3>{t('run_root_cause')}</h3><pre className="code-rca-settings__report">{detail.rootCause}</pre>
//   <h3>{t('run_proposed_fix')}</h3><pre ...>{detail.proposedFix}</pre>
//   confidence Tag + limitations + baselineCommit 전체 + t('run_hitl_notice') Alert(수정은 자동 적용되지 않음)
// 새로고침 버튼(폴링 없음 — YAGNI).
```

`ko/codeRca.json` (en은 영어 대응값):

```json
{
	"header_title": "AI 코드 RCA",
	"header_subtitle": "SOP 미연계 이상 알람의 코드 근본원인을 AI가 분석합니다. 수정 제안은 검토용이며 자동 적용되지 않습니다.",
	"tab_config": "설정",
	"tab_runs": "분석 이력",
	"field_enabled": "기능 사용",
	"field_min_severity": "최소 심각도",
	"field_cooldown": "중복 억제 시간(초)",
	"field_max_runs_per_day": "일일 최대 분석 수",
	"field_max_queue_depth": "최대 대기 큐",
	"field_max_concurrent": "동시 분석 수",
	"field_allow_unbound": "이상 신호 없이 허용(비권장)",
	"allow_unbound_warning": "이상 탐지 신호 없이 SOP 미연계 알람만으로 분석이 발화됩니다. 비용이 증가할 수 있습니다.",
	"save": "저장",
	"saved": "저장되었습니다",
	"save_failed": "저장에 실패했습니다",
	"repos_title": "분석 대상 저장소",
	"repo_add": "저장소 추가",
	"repo_id": "저장소 ID",
	"repo_git_url": "Git URL",
	"repo_default_branch": "기본 브랜치",
	"repo_credential": "읽기 자격증명(PAT)",
	"credential_unchanged_hint": "비워두면 기존 자격증명이 유지됩니다",
	"repo_enabled": "사용",
	"repo_last_sync": "마지막 동기화",
	"repo_baseline": "기준 커밋",
	"repo_delete_confirm": "이 저장소 연결을 삭제할까요?",
	"maps_title": "서비스 → 저장소 매핑",
	"map_add": "매핑 추가",
	"map_service": "서비스명",
	"map_repo": "저장소 ID",
	"map_subpath": "하위 경로(모노레포)",
	"map_delete_confirm": "이 매핑을 삭제할까요?",
	"runs_filter_status": "상태",
	"runs_refresh": "새로고침",
	"run_created": "생성 시각",
	"run_service": "서비스",
	"run_status": "상태",
	"run_baseline": "기준 커밋",
	"run_attempts": "시도",
	"run_root_cause": "근본 원인",
	"run_proposed_fix": "수정 제안 (미적용)",
	"run_confidence": "신뢰도",
	"run_limitations": "한계",
	"run_hitl_notice": "AI가 생성한 제안입니다. 적용 전 반드시 사람이 검토해야 합니다.",
	"run_empty": "분석 이력이 없습니다"
}
```

- [ ] **Step 4: 통과 확인**

Run: `cd frontend && node node_modules/jest/bin/jest.js --silent "CodeRcaSettings"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/container/CodeRcaSettings/ frontend/public/locales/en/codeRca.json frontend/public/locales/ko/codeRca.json
git commit -m "[Feat] CodeRcaSettings 컨테이너 — 설정·저장소·매핑 + 분석 이력 탭 (en/ko)"
```

---

### Task 17: 라우팅·메뉴·권한 등록 (FE seam)

**Files:**
- Modify: `frontend/src/constants/routes.ts` (92행 인접)
- Modify: `frontend/src/utils/permission/index.ts` (137행 인접)
- Modify: `frontend/src/pages/Settings/Settings.tsx` (navItemKeyMap 34행 + isEnabled 블록 4곳: 110·124·145·158·175행)
- Modify: `frontend/src/pages/Settings/config.tsx` (239행 aiModuleSettings 인접)
- Modify: `frontend/src/pages/Settings/utils.ts` (87행 인접)
- Modify: `frontend/src/container/SideNav/menuItems.tsx` (AI_MODULE_SETTINGS 항목 358행 인접)
- Modify: `frontend/src/container/TopNav/DateTimeSelectionV2/constants.ts` (196행 인접)
- Modify: `frontend/public/locales/en/routes.json`, `frontend/public/locales/ko/routes.json` (`ai_module` 키 인접)

- [ ] **Step 1: 각 파일에 ai-module 항목을 미러해 1줄/1블록씩 추가**

```typescript
// routes.ts
	CODE_RCA_SETTINGS: '/settings/code-rca',
// permission/index.ts
	CODE_RCA_SETTINGS: ['ADMIN', 'EDITOR', 'VIEWER'], // 페이지 진입 허용; 쓰기는 백엔드 EditAccess가 차단
// Settings.tsx navItemKeyMap
	'code-rca': 'routes:code_rca',
// Settings.tsx — AI_MODULE_SETTINGS가 나오는 isEnabled 블록 4곳 모두에
	item.key === ROUTES.CODE_RCA_SETTINGS ||  // (AI_MODULE_SETTINGS 줄 바로 옆)
// config.tsx — aiModuleSettings 미러 (아이콘 Sparkles 재사용)
export const codeRcaSettings = (t: TFunction): RouteTabProps['routes'] => [
	{
		Component: CodeRcaSettings,
		name: (
			<div className="periscope-tab">
				<Sparkles size={16} /> {t('routes:code_rca').toString()}
			</div>
		),
		route: ROUTES.CODE_RCA_SETTINGS,
		key: ROUTES.CODE_RCA_SETTINGS,
	},
];
// utils.ts 87행 — settings.push(..., aiModuleSettings(t), codeRcaSettings(t));
// menuItems.tsx — AI_MODULE_SETTINGS 항목(358행) 미러로 settings 섹션에 추가
//   (해당 항목의 itemKey/label 구조를 Read 후 동일 형태로 'code-rca' 추가)
// DateTimeSelectionV2/constants.ts — ROUTES.CODE_RCA_SETTINGS 추가
// locales routes.json — en: "code_rca": "Code RCA" / ko: "code_rca": "코드 RCA"
```

> `Settings.tsx`의 isEnabled 블록은 역할군별 노출 제어다 — AI_MODULE_SETTINGS와 동일한 노출 정책을 쓴다. VIEWER 노출이 과하면 ADMIN/EDITOR만으로 줄여도 되지만, 이력 조회는 운영자(viewer) 가치이므로 진입은 열고 변경은 서버 권한으로 막는 위 기본을 권장.

- [ ] **Step 2: 타입 체크 + 전체 FE 테스트 스모크**

Run: `cd frontend && node node_modules/typescript/bin/tsc --noEmit -p tsconfig.json 2>&1 | head -20`
Run: `cd frontend && node node_modules/jest/bin/jest.js --silent "Settings|CodeRca" 2>&1 | tail -10`
Expected: 에러 0건 / PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/constants/routes.ts frontend/src/utils/permission/index.ts frontend/src/pages/Settings/ frontend/src/container/SideNav/menuItems.tsx frontend/src/container/TopNav/DateTimeSelectionV2/constants.ts frontend/public/locales/en/routes.json frontend/public/locales/ko/routes.json
git commit -m "[Feat] /settings/code-rca 라우팅·메뉴·권한·i18n 등록 (FE seam)"
```

---

### Task 18: C 게이트 — 수동 확인 + 전체 회귀

- [ ] **Step 1: 백엔드 전체 테스트**

Run: `go build ./... && go test ./pkg/... -count=1 2>&1 | tail -20`
Expected: PASS

- [ ] **Step 2: FE 전체 게이트**

Run: `cd frontend && node node_modules/typescript/bin/tsc --noEmit -p tsconfig.json 2>&1 | head -40`
Expected: 에러 0건

- [ ] **Step 3: 수동 스모크 (가능한 환경에서)** — 서버 기동 후:
  1. `/settings/code-rca` 진입 → 설정 저장 → 새로고침 후 유지 확인.
  2. 저장소 등록(공개 repo, 자격증명 없이) + 서비스 매핑 등록.
  3. anomaly 룰 발화 또는 수동 `anomaly=true` 라벨 알람으로 UJ-5 트리거 → 이력 탭에서 run 상태 전이(queued→running→done/failed) 확인.
  4. 알람 채널에 "Code RCA suggestion" 메타-알람 수신 확인.
  - 환경이 없으면 이 단계는 보류로 표시하고 사용자에게 보고.

- [ ] **Step 4: Commit (잔여 수정이 있었다면)**

---

### Task 19: BMAD 산출물 정합 (PROCESS.md §10 절차)

**Files:**
- Modify: `docs/spec/04-stories/11.1.story.md`, `11.2.story.md`, `11.6.story.md` (status → done, seam 문구 제거)
- Modify: `docs/spec/01-prd/features/CF-11-code-rca.md` (frontmatter `status: implemented`, `caveats`/`open_items` 갱신, §통합 seam 절 정리)
- Modify: `docs/spec/_shared/traceability.md` (§6.1 상태, §8 open 2건 해소, §5 모듈 표)
- Modify: `docs/spec/05-wbs/index.md` (M-7 → 완료, WBS-1.7/스토리 상태·종료일)
- Modify: `docs/spec/03-epics/index.md` + `epic-11-code-rca.md` (status)
- Modify: `docs/spec/_shared/component-source-map.md` (F10 as-built: trigger/worker/handler/FE 경로 추가)
- Modify: `docs/spec/01-prd/index.md` (§6 CF-11 ◑→★, §9.3 해당 항목 제거)

- [ ] **Step 1: 전 문서 일괄 갱신** — frontmatter `updated:` 오늘 날짜, traceability §7 체크리스트로 상호 정합 검증 (CF frontmatter ↔ §1/§2/§6, 스토리 ↔ §6.1, 에픽 stories 목록).
- [ ] **Step 2: TODO 마커 검사** — `grep -rn "^TODO\|: TODO\|- TODO" docs/spec/` 잔존 없는지.
- [ ] **Step 3: Commit**

```bash
git add docs/spec/
git commit -m "[Docs] CF-11 implemented 정합 — 스토리·traceability·WBS·PRD 상태 갱신"
```

---

## 검증 체크리스트 (스펙 ↔ 태스크)

| 스펙 요구 | 태스크 |
|---|---|
| §3.1 Trigger 파사드 (fail-open, 즉시 반환) | 5, 9, 11 |
| §3.1 워커 루프 (lifecycle) | 6, 10 |
| §3.1 Deliverer/Auditor 구체화 (CF-3/CF-6 재사용) | 8 |
| §3.2 게이트 체인 + anomaly fail-closed (§10) | 5, 7 |
| §3.3 seam 1줄 배선 (hook·server) | 9, 10 |
| §3.4 A 게이트 (e2e + fail-open + 비발화) | 11 |
| §4 HTTP API (CRUD + secrets 비노출 + run 이력 + baseline 명시) | 1, 2, 3, 12, 13, 14 |
| §5 FE 설정 + 이력 (ko namespace, jest mock 규칙) | 15, 16, 17, 18 |
| §7 BMAD 문서 정합 + dispatchhook 주석 정정 | 9(주석), 19 |
| §9 미확정 3건 | anomaly 신호=Task 5·7 / 보고서 저장=Task 2·3·4 / 이력 배치=Task 16(탭) |
