package signozruler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/secretbox"
	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	codercacfgstore "github.com/SigNoz/signoz/pkg/ruler/coderca/codebaseconfigstore/sqlcodebasercaconfigstore"
	coderepostore "github.com/SigNoz/signoz/pkg/ruler/coderca/codebaseconfigstore/sqlcodebaseconfigstore"
	codemapstore "github.com/SigNoz/signoz/pkg/ruler/coderca/codebaseconfigstore/sqlcodebaseservicemapstore"
	codercarunstore "github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstoretest"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

const codercaTestOrgID = "00000000-0000-0000-0000-000000000001"

// applyCodercaDDL applies the DDL for all CF-11 tables exercised by the
// coderca handler tests (mirrors migrations 081 + 083). Production registers
// these via a migration seam; tests apply DDL directly, per the ai_config
// store-test pattern.
func applyCodercaDDL(ctx context.Context, ss sqlstore.SQLStore) error {
	stmts := []string{
		`CREATE TABLE ds_codebase_repo (
			org_id                 TEXT      NOT NULL,
			repo_id                TEXT      NOT NULL,
			git_url                TEXT      NOT NULL,
			default_branch         TEXT      NOT NULL DEFAULT '',
			credential_ciphertext  TEXT      NOT NULL DEFAULT '',
			enabled                BOOLEAN   NOT NULL DEFAULT FALSE,
			branch_name            TEXT      NOT NULL DEFAULT '',
			fetched                BOOLEAN   NOT NULL DEFAULT FALSE,
			baseline_commit        TEXT      NOT NULL DEFAULT '',
			last_sync_at           TEXT      NOT NULL DEFAULT '',
			last_sync_status       TEXT      NOT NULL DEFAULT '',
			artifact_path          TEXT      NOT NULL DEFAULT '',
			PRIMARY KEY (org_id, repo_id)
		)`,
		`CREATE TABLE ds_codebase_service_map (
			org_id        TEXT NOT NULL,
			service_name  TEXT NOT NULL,
			repo_id       TEXT NOT NULL,
			subpath       TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (org_id, service_name)
		)`,
		`CREATE TABLE ds_codebase_config (
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
		`CREATE TABLE coderca_run (
			run_id          TEXT    NOT NULL PRIMARY KEY,
			org_id          TEXT    NOT NULL,
			service         TEXT    NOT NULL DEFAULT '',
			dedup_key       TEXT    NOT NULL,
			status          TEXT    NOT NULL,
			baseline_commit TEXT    NOT NULL DEFAULT '',
			created_at      INTEGER NOT NULL,
			claimed_by      TEXT    NOT NULL DEFAULT '',
			lease_token     TEXT    NOT NULL DEFAULT '',
			lease_until     INTEGER NOT NULL DEFAULT 0,
			heartbeat_at    INTEGER NOT NULL DEFAULT 0,
			attempts        INTEGER NOT NULL DEFAULT 0,
			finished_at     INTEGER NOT NULL DEFAULT 0,
			result_ref      TEXT    NOT NULL DEFAULT '',
			root_cause      TEXT    NOT NULL DEFAULT '',
			proposed_fix    TEXT    NOT NULL DEFAULT '',
			confidence      TEXT    NOT NULL DEFAULT '',
			limitations     TEXT    NOT NULL DEFAULT '',
			failure_reason  TEXT    NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE coderca_admission (
			org_id           TEXT    NOT NULL,
			dedup_key        TEXT    NOT NULL,
			last_admitted_at INTEGER NOT NULL,
			hit_count        INTEGER NOT NULL DEFAULT 0,
			last_run_ref     TEXT    NOT NULL DEFAULT '',
			PRIMARY KEY (org_id, dedup_key)
		)`,
		`CREATE TABLE coderca_budget (
			org_id TEXT    NOT NULL,
			day    TEXT    NOT NULL,
			used   INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (org_id, day)
		)`,
		`CREATE TABLE coderca_capacity (
			scope               TEXT    NOT NULL PRIMARY KEY,
			running             INTEGER NOT NULL DEFAULT 0,
			max_concurrent_runs INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE coderca_skip_stat (
			org_id TEXT    NOT NULL,
			reason TEXT    NOT NULL,
			day    TEXT    NOT NULL,
			count  INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (org_id, reason, day)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := ss.BunDB().ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// newCodercaTestHandler builds a handler wired with real sqlite-backed CF-11
// stores and a real-key cipher (encryption available). The sqlstore is
// returned so tests can inspect ciphertext directly.
func newCodercaTestHandler(t *testing.T) (*handler, sqlstore.SQLStore) {
	t.Helper()
	ss := sqlitesqlstoretest.New(t)
	require.NoError(t, applyCodercaDDL(context.Background(), ss))

	cipher := newRealKeyCipher(t)
	h := &handler{
		aiCipher:          cipher,
		aiCipherInsecure:  false,
		codebaseRepoStore: coderepostore.New(ss),
		codebaseMapStore:  codemapstore.New(ss),
		codercaCfgStore:   codercacfgstore.New(ss),
		codercaRunStore:   codercarunstore.New(ss),
	}
	return h, ss
}

// newRealKeyCipher constructs a secretbox cipher backed by a real 32-byte key
// (base64-encoded) so that encryption is "available" (ciphertext != plaintext).
func newRealKeyCipher(t *testing.T) *secretbox.Cipher {
	t.Helper()
	// base64 of exactly 32 bytes "0123456789abcdef0123456789abcdef".
	keyB64 := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	cipher, err := secretbox.New(keyB64)
	require.NoError(t, err)
	return cipher
}

func codercaReq(t *testing.T, method, target string, body string) *http.Request {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
	}
	return withSOPTestClaims(r)
}

// TestUpsertCodebaseRepoStoresCiphertextAndScrubsResponse verifies the
// design AC: a plaintext credential is stored encrypted (DB column !=
// plaintext) and never returned plaintext over the wire (List scrubs it to
// the APIKeyPlaceholder sentinel).
func TestUpsertCodebaseRepoStoresCiphertextAndScrubsResponse(t *testing.T) {
	h, ss := newCodercaTestHandler(t)

	body := `{
		"contractVersion":"ds.codebase_repo.v1",
		"repoId":"payments",
		"gitUrl":"https://github.com/acme/payments.git",
		"defaultBranch":"main",
		"credential":"tok-secret-123",
		"enabled":true
	}`
	rw := httptest.NewRecorder()
	h.UpsertCodebaseRepo(rw, codercaReq(t, http.MethodPut, "/api/v2/ds/coderca/repos", body))
	require.Equal(t, http.StatusNoContent, rw.Code, "body=%s", rw.Body.String())

	// The persisted credential ciphertext must not equal the plaintext.
	var ciphertext string
	require.NoError(t, ss.BunDB().QueryRowContext(context.Background(),
		"SELECT credential_ciphertext FROM ds_codebase_repo WHERE org_id = ? AND repo_id = ?",
		codercaTestOrgID, "payments").Scan(&ciphertext))
	require.NotEmpty(t, ciphertext)
	require.NotEqual(t, "tok-secret-123", ciphertext, "credential must be stored encrypted, not plaintext")

	// List must scrub the credential to the sentinel, never returning plaintext.
	listRW := httptest.NewRecorder()
	h.ListCodebaseRepos(listRW, codercaReq(t, http.MethodGet, "/api/v2/ds/coderca/repos", ""))
	require.Equal(t, http.StatusOK, listRW.Code, "body=%s", listRW.Body.String())
	require.NotContains(t, listRW.Body.String(), "tok-secret-123", "plaintext credential leaked in response")

	var listed struct {
		Data []ruletypes.CodebaseRepo `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listRW.Body.Bytes(), &listed))
	require.Len(t, listed.Data, 1)
	require.Equal(t, APIKeyPlaceholder, listed.Data[0].Credential)
}

// TestUpsertCodebaseRepoUnchangedSentinelPreservesCredential verifies that a
// PUT carrying the APIKeyPlaceholder sentinel preserves the previously stored
// credential rather than wiping it.
func TestUpsertCodebaseRepoUnchangedSentinelPreservesCredential(t *testing.T) {
	h, _ := newCodercaTestHandler(t)

	// Seed a repo with a real credential.
	seed := `{
		"contractVersion":"ds.codebase_repo.v1",
		"repoId":"payments",
		"gitUrl":"https://github.com/acme/payments.git",
		"defaultBranch":"main",
		"credential":"original-token",
		"enabled":true
	}`
	rw := httptest.NewRecorder()
	h.UpsertCodebaseRepo(rw, codercaReq(t, http.MethodPut, "/api/v2/ds/coderca/repos", seed))
	require.Equal(t, http.StatusNoContent, rw.Code, "body=%s", rw.Body.String())

	// PUT again with the sentinel: credential must be preserved.
	update := `{
		"contractVersion":"ds.codebase_repo.v1",
		"repoId":"payments",
		"gitUrl":"https://github.com/acme/payments.git",
		"defaultBranch":"develop",
		"credential":"<unchanged>",
		"enabled":true
	}`
	rw2 := httptest.NewRecorder()
	h.UpsertCodebaseRepo(rw2, codercaReq(t, http.MethodPut, "/api/v2/ds/coderca/repos", update))
	require.Equal(t, http.StatusNoContent, rw2.Code, "body=%s", rw2.Body.String())

	// Decrypt + verify the credential survived unchanged, and the branch updated.
	got, err := h.codebaseRepoStore.Get(context.Background(), codercaTestOrgID, "payments", h.aiCipher.DecryptFunc())
	require.NoError(t, err)
	require.Equal(t, "original-token", got.Credential, "sentinel must preserve the existing credential")
	require.Equal(t, "develop", got.DefaultBranch, "non-secret fields must still update")
}

// TestUpsertCodebaseRepoFailClosedWithoutEncryption verifies that with
// encryption unavailable (aiCipherInsecure=true), a PUT carrying a non-empty
// credential is rejected fail-closed with 400 (ValidateCodebaseRepo).
func TestUpsertCodebaseRepoFailClosedWithoutEncryption(t *testing.T) {
	ss := sqlitesqlstoretest.New(t)
	require.NoError(t, applyCodercaDDL(context.Background(), ss))

	h := &handler{
		aiCipher:          secretbox.PlaintextCipher(),
		aiCipherInsecure:  true, // encryption NOT available
		codebaseRepoStore: coderepostore.New(ss),
	}

	body := `{
		"contractVersion":"ds.codebase_repo.v1",
		"repoId":"payments",
		"gitUrl":"https://github.com/acme/payments.git",
		"defaultBranch":"main",
		"credential":"tok-should-be-rejected",
		"enabled":true
	}`
	rw := httptest.NewRecorder()
	h.UpsertCodebaseRepo(rw, codercaReq(t, http.MethodPut, "/api/v2/ds/coderca/repos", body))
	require.Equal(t, http.StatusBadRequest, rw.Code, "body=%s", rw.Body.String())
}

// TestRCAConfigGetDefaultsAndPutRoundTrip verifies: GET on a fresh org returns
// the fail-closed defaults (Enabled=false); a valid PUT persists (204) and is
// reflected on GET; an invalid MinSeverity PUT is rejected (400).
func TestRCAConfigGetDefaultsAndPutRoundTrip(t *testing.T) {
	h, _ := newCodercaTestHandler(t)

	// GET fresh → defaults.
	getRW := httptest.NewRecorder()
	h.GetCodebaseRCAConfig(getRW, codercaReq(t, http.MethodGet, "/api/v2/ds/coderca/config", ""))
	require.Equal(t, http.StatusOK, getRW.Code, "body=%s", getRW.Body.String())
	var def struct {
		Data ruletypes.CodebaseRCAConfig `json:"data"`
	}
	require.NoError(t, json.Unmarshal(getRW.Body.Bytes(), &def))
	require.False(t, def.Data.Enabled, "default config must be disabled (fail-closed)")
	require.Equal(t, "high", def.Data.MinSeverity)

	// Valid PUT → 204.
	put := `{
		"contractVersion":"ds.codebase_rca_config.v1",
		"enabled":true,
		"minSeverity":"critical",
		"cooldownWindowSecs":3600,
		"maxRunsPerDay":10,
		"maxQueueDepth":25,
		"maxConcurrentRuns":1
	}`
	putRW := httptest.NewRecorder()
	h.UpdateCodebaseRCAConfig(putRW, codercaReq(t, http.MethodPut, "/api/v2/ds/coderca/config", put))
	require.Equal(t, http.StatusNoContent, putRW.Code, "body=%s", putRW.Body.String())

	// GET reflects the update.
	getRW2 := httptest.NewRecorder()
	h.GetCodebaseRCAConfig(getRW2, codercaReq(t, http.MethodGet, "/api/v2/ds/coderca/config", ""))
	require.Equal(t, http.StatusOK, getRW2.Code)
	var got struct {
		Data ruletypes.CodebaseRCAConfig `json:"data"`
	}
	require.NoError(t, json.Unmarshal(getRW2.Body.Bytes(), &got))
	require.True(t, got.Data.Enabled)
	require.Equal(t, "critical", got.Data.MinSeverity)
	require.Equal(t, 10, got.Data.MaxRunsPerDay)
	require.Equal(t, codercaTestOrgID, got.Data.OrgID, "org must be forced from claims")

	// Invalid MinSeverity → 400.
	bad := `{
		"contractVersion":"ds.codebase_rca_config.v1",
		"enabled":true,
		"minSeverity":"nonsense",
		"cooldownWindowSecs":3600,
		"maxRunsPerDay":10,
		"maxQueueDepth":25,
		"maxConcurrentRuns":1
	}`
	badRW := httptest.NewRecorder()
	h.UpdateCodebaseRCAConfig(badRW, codercaReq(t, http.MethodPut, "/api/v2/ds/coderca/config", bad))
	require.Equal(t, http.StatusBadRequest, badRW.Code, "body=%s", badRW.Body.String())
}

// TestServiceMapCRUD verifies the service-map lifecycle: PUT → GET list (1) →
// DELETE → GET list (0).
func TestServiceMapCRUD(t *testing.T) {
	h, _ := newCodercaTestHandler(t)

	put := `{"serviceName":"payments","repoId":"repo-pay","subpath":"services/pay"}`
	putRW := httptest.NewRecorder()
	h.UpsertCodebaseServiceMap(putRW, codercaReq(t, http.MethodPut, "/api/v2/ds/coderca/service-maps", put))
	require.Equal(t, http.StatusNoContent, putRW.Code, "body=%s", putRW.Body.String())

	// GET list → 1 entry.
	listRW := httptest.NewRecorder()
	h.ListCodebaseServiceMaps(listRW, codercaReq(t, http.MethodGet, "/api/v2/ds/coderca/service-maps", ""))
	require.Equal(t, http.StatusOK, listRW.Code)
	var listed struct {
		Data []ruletypes.CodebaseServiceMap `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listRW.Body.Bytes(), &listed))
	require.Len(t, listed.Data, 1)
	require.Equal(t, "payments", listed.Data[0].ServiceName)
	require.Equal(t, "repo-pay", listed.Data[0].RepoID)
	require.Equal(t, codercaTestOrgID, listed.Data[0].OrgID, "org must be forced from claims")

	// DELETE.
	delReq := codercaReq(t, http.MethodDelete, "/api/v2/ds/coderca/service-maps/payments", "")
	delReq = muxSetVar(delReq, "serviceName", "payments")
	delRW := httptest.NewRecorder()
	h.DeleteCodebaseServiceMap(delRW, delReq)
	require.Equal(t, http.StatusNoContent, delRW.Code, "body=%s", delRW.Body.String())

	// GET list → 0 entries.
	listRW2 := httptest.NewRecorder()
	h.ListCodebaseServiceMaps(listRW2, codercaReq(t, http.MethodGet, "/api/v2/ds/coderca/service-maps", ""))
	require.Equal(t, http.StatusOK, listRW2.Code)
	var listed2 struct {
		Data []ruletypes.CodebaseServiceMap `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listRW2.Body.Bytes(), &listed2))
	require.Empty(t, listed2.Data)
}

// TestRunsListAndDetail seeds runs via the runstore (Admit → ClaimNext →
// Finalize) for the test org and verifies: list (with status/service filter),
// single detail fetch, and that a run belonging to another org returns 404
// (tenant isolation — existence not leaked).
func TestRunsListAndDetail(t *testing.T) {
	h, ss := newCodercaTestHandler(t)
	ctx := context.Background()
	store := codercarunstore.New(ss)
	now := time.Unix(1_700_000_000, 0)

	// Seed a finalized run for the test org.
	admit, err := store.Admit(ctx, codercarunstore.AdmitParams{
		OrgID:          codercaTestOrgID,
		Service:        "payments",
		DedupKey:       "k1",
		Now:            now,
		CooldownWindow: 6 * time.Hour,
		MaxRunsPerDay:  100,
		MaxQueueDepth:  100,
	})
	require.NoError(t, err)
	require.True(t, admit.Admitted)

	claim, err := store.ClaimNext(ctx, codercarunstore.ClaimParams{
		Scope:         "global",
		ClaimedBy:     "worker-1",
		Now:           now,
		LeaseTTL:      time.Hour,
		MaxConcurrent: 1,
	})
	require.NoError(t, err)
	require.True(t, claim.Claimed)
	require.Equal(t, admit.RunID, claim.RunID)

	finalized, err := store.Finalize(ctx, codercarunstore.FinalizeParams{
		Scope:          "global",
		RunID:          claim.RunID,
		LeaseToken:     claim.LeaseToken,
		Status:         "done",
		ResultRef:      "s3://reports/r1",
		Now:            now.Add(time.Minute),
		BaselineCommit: "deadbeef",
		RootCause:      "null pointer in payment handler",
		ProposedFix:    "add nil check",
		Confidence:     "high",
		Limitations:    "limited trace window",
	})
	require.NoError(t, err)
	require.True(t, finalized)

	// Seed a run for a DIFFERENT org (cross-tenant isolation check).
	otherAdmit, err := store.Admit(ctx, codercarunstore.AdmitParams{
		OrgID:          "other-org",
		Service:        "billing",
		DedupKey:       "k-other",
		Now:            now,
		CooldownWindow: 6 * time.Hour,
		MaxRunsPerDay:  100,
		MaxQueueDepth:  100,
	})
	require.NoError(t, err)
	require.True(t, otherAdmit.Admitted)

	// GET list (no filter) → 1 run for the test org.
	listRW := httptest.NewRecorder()
	h.ListCodeRCARuns(listRW, codercaReq(t, http.MethodGet, "/api/v2/ds/coderca/runs", ""))
	require.Equal(t, http.StatusOK, listRW.Code, "body=%s", listRW.Body.String())
	var listed struct {
		Data []codercarunstore.RunSummary `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listRW.Body.Bytes(), &listed))
	require.Len(t, listed.Data, 1, "list must be org-scoped")
	require.Equal(t, admit.RunID, listed.Data[0].RunID)
	require.Equal(t, "payments", listed.Data[0].Service)

	// GET list with status filter (done) → 1; with status filter (queued) → 0.
	doneRW := httptest.NewRecorder()
	h.ListCodeRCARuns(doneRW, codercaReq(t, http.MethodGet, "/api/v2/ds/coderca/runs?status=done", ""))
	require.Equal(t, http.StatusOK, doneRW.Code)
	var doneList struct {
		Data []codercarunstore.RunSummary `json:"data"`
	}
	require.NoError(t, json.Unmarshal(doneRW.Body.Bytes(), &doneList))
	require.Len(t, doneList.Data, 1)

	queuedRW := httptest.NewRecorder()
	h.ListCodeRCARuns(queuedRW, codercaReq(t, http.MethodGet, "/api/v2/ds/coderca/runs?status=queued", ""))
	require.Equal(t, http.StatusOK, queuedRW.Code)
	var queuedList struct {
		Data []codercarunstore.RunSummary `json:"data"`
	}
	require.NoError(t, json.Unmarshal(queuedRW.Body.Bytes(), &queuedList))
	require.Empty(t, queuedList.Data)

	// Service filter (billing belongs to other-org) → 0 for the test org.
	svcRW := httptest.NewRecorder()
	h.ListCodeRCARuns(svcRW, codercaReq(t, http.MethodGet, "/api/v2/ds/coderca/runs?service=billing", ""))
	require.Equal(t, http.StatusOK, svcRW.Code)
	var svcList struct {
		Data []codercarunstore.RunSummary `json:"data"`
	}
	require.NoError(t, json.Unmarshal(svcRW.Body.Bytes(), &svcList))
	require.Empty(t, svcList.Data)

	// GET detail for the test org's run → 200 with report.
	detReq := codercaReq(t, http.MethodGet, "/api/v2/ds/coderca/runs/"+admit.RunID, "")
	detReq = muxSetVar(detReq, "runId", admit.RunID)
	detRW := httptest.NewRecorder()
	h.GetCodeRCARun(detRW, detReq)
	require.Equal(t, http.StatusOK, detRW.Code, "body=%s", detRW.Body.String())
	var detail struct {
		Data codercarunstore.RunDetail `json:"data"`
	}
	require.NoError(t, json.Unmarshal(detRW.Body.Bytes(), &detail))
	require.Equal(t, admit.RunID, detail.Data.RunID)
	require.Equal(t, "null pointer in payment handler", detail.Data.RootCause)

	// GET detail for ANOTHER org's run → 404 (tenant isolation).
	crossReq := codercaReq(t, http.MethodGet, "/api/v2/ds/coderca/runs/"+otherAdmit.RunID, "")
	crossReq = muxSetVar(crossReq, "runId", otherAdmit.RunID)
	crossRW := httptest.NewRecorder()
	h.GetCodeRCARun(crossRW, crossReq)
	require.Equal(t, http.StatusNotFound, crossRW.Code, "cross-org run must return 404, not leak existence")
}

// seedCodercaRun admits, claims and finalizes one run for the test org and
// returns its runID. status is the terminal status ("done", "failed", ...).
func seedCodercaRun(t *testing.T, store *codercarunstore.Store, service, dedupKey string, status coderca.RunStatus) string {
	t.Helper()
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0)

	admit, err := store.Admit(ctx, codercarunstore.AdmitParams{
		OrgID:          codercaTestOrgID,
		Service:        service,
		DedupKey:       dedupKey,
		Now:            now,
		CooldownWindow: 6 * time.Hour,
		MaxRunsPerDay:  100,
		MaxQueueDepth:  100,
	})
	require.NoError(t, err)
	require.True(t, admit.Admitted)

	claim, err := store.ClaimNext(ctx, codercarunstore.ClaimParams{
		Scope:         "global",
		ClaimedBy:     "worker-1",
		Now:           now,
		LeaseTTL:      time.Hour,
		MaxConcurrent: 1,
	})
	require.NoError(t, err)
	require.True(t, claim.Claimed)
	require.Equal(t, admit.RunID, claim.RunID)

	finalized, err := store.Finalize(ctx, codercarunstore.FinalizeParams{
		Scope:          "global",
		RunID:          claim.RunID,
		LeaseToken:     claim.LeaseToken,
		Status:         status,
		Now:            now.Add(time.Minute),
		BaselineCommit: "deadbeef",
		RootCause:      "null pointer in payment handler",
		ProposedFix:    "add nil check",
		Confidence:     "high",
		Limitations:    "limited trace window",
	})
	require.NoError(t, err)
	require.True(t, finalized)
	return admit.RunID
}

// exportReq builds the POST …/runs/{runId}/export request with claims + mux var.
func exportReq(t *testing.T, runID string) *http.Request {
	t.Helper()
	req := codercaReq(t, http.MethodPost, "/api/v2/ds/coderca/runs/"+runID+"/export", "")
	return muxSetVar(req, "runId", runID)
}

// TestExportCodeRCARun verifies the ds-hub export endpoint: a done run whose
// service maps to an enabled repo with an artifactPath is rendered to
// <artifactPath>/ds-hub/<date>_<time>_rca_<service>.md; non-done runs, unmapped
// services, disabled repos and missing artifactPath are rejected with 400.
func TestExportCodeRCARun(t *testing.T) {
	h, ss := newCodercaTestHandler(t)
	ctx := context.Background()
	store := codercarunstore.New(ss)

	artifactRoot := t.TempDir()
	require.NoError(t, h.codebaseRepoStore.Upsert(ctx, ruletypes.CodebaseRepo{
		OrgID:        codercaTestOrgID,
		RepoID:       "repo-pay",
		GitURL:       "https://github.com/acme/payments.git",
		Enabled:      true,
		ArtifactPath: artifactRoot,
	}, h.aiCipher.EncryptFunc()))
	require.NoError(t, h.codebaseMapStore.Upsert(ctx, ruletypes.CodebaseServiceMap{
		OrgID:       codercaTestOrgID,
		ServiceName: "payments",
		RepoID:      "repo-pay",
	}))

	t.Run("done run exports the markdown artifact", func(t *testing.T) {
		runID := seedCodercaRun(t, store, "payments", "k-exp-1", "done")

		rw := httptest.NewRecorder()
		h.ExportCodeRCARun(rw, exportReq(t, runID))
		require.Equal(t, http.StatusOK, rw.Code, "body=%s", rw.Body.String())

		var resp struct {
			Data struct {
				Path string `json:"path"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &resp))
		require.NotEmpty(t, resp.Data.Path)

		content, err := os.ReadFile(resp.Data.Path)
		require.NoError(t, err)
		require.Contains(t, string(content), "## 근본 원인")
		require.Contains(t, string(content), "null pointer in payment handler")
		require.Contains(t, string(content), "runId: "+runID)
	})

	t.Run("non-done run is rejected", func(t *testing.T) {
		runID := seedCodercaRun(t, store, "payments", "k-exp-2", "failed")

		rw := httptest.NewRecorder()
		h.ExportCodeRCARun(rw, exportReq(t, runID))
		require.Equal(t, http.StatusBadRequest, rw.Code, "body=%s", rw.Body.String())
	})

	t.Run("unmapped service is rejected", func(t *testing.T) {
		runID := seedCodercaRun(t, store, "orphan", "k-exp-3", "done")

		rw := httptest.NewRecorder()
		h.ExportCodeRCARun(rw, exportReq(t, runID))
		require.Equal(t, http.StatusBadRequest, rw.Code, "body=%s", rw.Body.String())
	})

	t.Run("repo without artifactPath is rejected", func(t *testing.T) {
		require.NoError(t, h.codebaseRepoStore.Upsert(ctx, ruletypes.CodebaseRepo{
			OrgID:   codercaTestOrgID,
			RepoID:  "repo-nopath",
			GitURL:  "https://github.com/acme/nopath.git",
			Enabled: true,
		}, h.aiCipher.EncryptFunc()))
		require.NoError(t, h.codebaseMapStore.Upsert(ctx, ruletypes.CodebaseServiceMap{
			OrgID:       codercaTestOrgID,
			ServiceName: "nopath-svc",
			RepoID:      "repo-nopath",
		}))
		runID := seedCodercaRun(t, store, "nopath-svc", "k-exp-4", "done")

		rw := httptest.NewRecorder()
		h.ExportCodeRCARun(rw, exportReq(t, runID))
		require.Equal(t, http.StatusBadRequest, rw.Code, "body=%s", rw.Body.String())
	})

	t.Run("disabled repo is rejected", func(t *testing.T) {
		require.NoError(t, h.codebaseRepoStore.Upsert(ctx, ruletypes.CodebaseRepo{
			OrgID:        codercaTestOrgID,
			RepoID:       "repo-off",
			GitURL:       "https://github.com/acme/off.git",
			Enabled:      false,
			ArtifactPath: artifactRoot,
		}, h.aiCipher.EncryptFunc()))
		require.NoError(t, h.codebaseMapStore.Upsert(ctx, ruletypes.CodebaseServiceMap{
			OrgID:       codercaTestOrgID,
			ServiceName: "off-svc",
			RepoID:      "repo-off",
		}))
		runID := seedCodercaRun(t, store, "off-svc", "k-exp-5", "done")

		rw := httptest.NewRecorder()
		h.ExportCodeRCARun(rw, exportReq(t, runID))
		require.Equal(t, http.StatusBadRequest, rw.Code, "body=%s", rw.Body.String())
	})
}
