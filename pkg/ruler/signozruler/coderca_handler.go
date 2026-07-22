package signozruler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/exportmd"
	codercarunstore "github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/gorilla/mux"
)

// GetCodebaseRCAConfig handles GET /api/v2/ds/coderca/config.
// Returns the per-org CF-11 config; when no row exists, the fail-closed
// defaults are returned with 200 so the UI always has a well-typed object.
// The config carries no secrets, so no scrubbing is needed.
func (handler *handler) GetCodebaseRCAConfig(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	cfg, err := handler.codercaCfgStore.Get(req.Context(), orgID)
	if errors.Is(err, ruletypes.ErrCodebaseRCAConfigNotFound) {
		render.Success(rw, http.StatusOK, ruletypes.DefaultCodebaseRCAConfig(orgID))
		return
	}
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "fetch codebase RCA config"))
		return
	}
	render.Success(rw, http.StatusOK, cfg)
}

// UpdateCodebaseRCAConfig handles PUT /api/v2/ds/coderca/config.
// Validates and upserts the per-org config. OrgID and contractVersion are
// forced from the server side, not trusted from the body.
func (handler *handler) UpdateCodebaseRCAConfig(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	var incoming ruletypes.CodebaseRCAConfig
	if err := binding.JSON.BindBody(req.Body, &incoming); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	// Enforce the org from claims, not from the body.
	incoming.OrgID = orgID
	incoming.ContractVersion = ruletypes.CodebaseRCAConfigContractVersion
	if incoming.UpdatedAt == "" {
		incoming.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	if err := ruletypes.ValidateCodebaseRCAConfig(incoming); err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "codebase RCA config validation failed"))
		return
	}

	if err := handler.codercaCfgStore.Upsert(req.Context(), incoming); err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "upsert codebase RCA config"))
		return
	}

	render.Success(rw, http.StatusNoContent, nil)
}

// ListCodebaseRepos handles GET /api/v2/ds/coderca/repos.
// Returns the org's registered repos with credentials scrubbed: a non-empty
// credential is replaced with the APIKeyPlaceholder sentinel so plaintext is
// never serialized to clients (design AC).
func (handler *handler) ListCodebaseRepos(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	repos, err := handler.codebaseRepoStore.List(req.Context(), orgID, handler.aiCipher.DecryptFunc())
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "list codebase repos"))
		return
	}
	for i := range repos {
		repos[i].Credential = scrubAPIKey(repos[i].Credential)
	}
	render.Success(rw, http.StatusOK, repos)
}

// UpsertCodebaseRepo handles PUT /api/v2/ds/coderca/repos.
// Validates and upserts a repo registration. If the incoming credential is the
// APIKeyPlaceholder sentinel, the existing stored credential is preserved.
// When encryption is unavailable, a non-empty credential is rejected
// fail-closed by ValidateCodebaseRepo.
func (handler *handler) UpsertCodebaseRepo(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	var incoming ruletypes.CodebaseRepo
	if err := binding.JSON.BindBody(req.Body, &incoming); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	// Enforce the org from claims, not from the body.
	incoming.OrgID = orgID
	incoming.ContractVersion = ruletypes.CodebaseRepoContractVersion

	// APIKeyPlaceholder sentinel: preserve the existing stored credential.
	if incoming.Credential == APIKeyPlaceholder {
		existing, getErr := handler.codebaseRepoStore.Get(req.Context(), orgID, incoming.RepoID, handler.aiCipher.DecryptFunc())
		if getErr == nil {
			incoming.Credential = existing.Credential
		} else if errors.Is(getErr, ruletypes.ErrCodebaseRepoNotFound) {
			incoming.Credential = ""
		} else {
			render.Error(rw, errors.WrapInternalf(getErr, errors.CodeInternal, "fetch existing codebase repo for credential preservation"))
			return
		}
	}

	encryptionAvailable := !handler.aiCipherInsecure
	if err := ruletypes.ValidateCodebaseRepo(incoming, encryptionAvailable); err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "codebase repo validation failed"))
		return
	}

	if err := handler.codebaseRepoStore.Upsert(req.Context(), incoming, handler.aiCipher.EncryptFunc()); err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "upsert codebase repo"))
		return
	}

	render.Success(rw, http.StatusNoContent, nil)
}

// DeleteCodebaseRepo handles DELETE /api/v2/ds/coderca/repos/{repoId}.
// Idempotent: deleting a non-existent repo returns 204.
func (handler *handler) DeleteCodebaseRepo(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	repoID := strings.TrimSpace(mux.Vars(req)["repoId"])
	if repoID == "" {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := handler.codebaseRepoStore.Delete(req.Context(), orgID, repoID); err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "delete codebase repo"))
		return
	}
	render.Success(rw, http.StatusNoContent, nil)
}

// ListCodebaseServiceMaps handles GET /api/v2/ds/coderca/service-maps.
// Mappings carry no secrets, so the org's list is returned as-is.
func (handler *handler) ListCodebaseServiceMaps(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	maps, err := handler.codebaseMapStore.List(req.Context(), orgID)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "list codebase service maps"))
		return
	}
	render.Success(rw, http.StatusOK, maps)
}

// UpsertCodebaseServiceMap handles PUT /api/v2/ds/coderca/service-maps.
// Validates required fields and upserts the service→repo mapping. OrgID is
// forced from claims. CodebaseServiceMap has no Validate func, so basic
// non-empty checks on serviceName/repoId are enforced here.
func (handler *handler) UpsertCodebaseServiceMap(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	var incoming ruletypes.CodebaseServiceMap
	if err := binding.JSON.BindBody(req.Body, &incoming); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	// Enforce the org from claims, not from the body.
	incoming.OrgID = orgID

	if strings.TrimSpace(incoming.ServiceName) == "" {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "serviceName: must not be empty"))
		return
	}
	if strings.TrimSpace(incoming.RepoID) == "" {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "repoId: must not be empty"))
		return
	}

	if err := handler.codebaseMapStore.Upsert(req.Context(), incoming); err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "upsert codebase service map"))
		return
	}
	render.Success(rw, http.StatusNoContent, nil)
}

// DeleteCodebaseServiceMap handles DELETE /api/v2/ds/coderca/service-maps/{serviceName}.
// Idempotent: deleting a non-existent mapping returns 204.
func (handler *handler) DeleteCodebaseServiceMap(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	serviceName := strings.TrimSpace(mux.Vars(req)["serviceName"])
	if serviceName == "" {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := handler.codebaseMapStore.Delete(req.Context(), orgID, serviceName); err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "delete codebase service map"))
		return
	}
	render.Success(rw, http.StatusNoContent, nil)
}

// ListCodeRCARuns handles GET /api/v2/ds/coderca/runs?status=&service=&limit=&offset=.
// Returns the org's run history (newest first), filtered by the query params.
func (handler *handler) ListCodeRCARuns(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	q := req.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	runs, err := handler.codercaRunStore.ListRuns(req.Context(), orgID, codercarunstore.ListRunsParams{
		Status:  q.Get("status"),
		Service: q.Get("service"),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "list coderca runs"))
		return
	}
	render.Success(rw, http.StatusOK, runs)
}

// GetCodeRCARun handles GET /api/v2/ds/coderca/runs/{runId}.
// Returns one run with its persisted RCA report. Tenant-isolated: a run
// belonging to another org returns a typed 404 (existence is not leaked).
func (handler *handler) GetCodeRCARun(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	runID := strings.TrimSpace(mux.Vars(req)["runId"])
	if runID == "" {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	run, err := handler.codercaRunStore.GetRun(req.Context(), orgID, runID)
	if errors.Is(err, codercarunstore.ErrRunNotFound) {
		render.Error(rw, errors.NewNotFoundf(errors.CodeNotFound, "coderca run was not found"))
		return
	}
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "fetch coderca run"))
		return
	}
	render.Success(rw, http.StatusOK, run)
}

// EnqueueCodeRCARun handles POST /api/v2/ds/coderca/runs — an on-demand "test
// run" surface. Code-RCA runs are normally admitted by anomaly-alarm dispatch;
// this lets a user trigger one for a service from the UI. The run is queued via
// the same admission path (cooldown/quota/queue-depth still apply) and picked up
// by the worker; the client polls GET /coderca/runs/{runId} for the result.
func (handler *handler) EnqueueCodeRCARun(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	var body struct {
		Service string `json:"service"`
	}
	if err := binding.JSON.BindBody(req.Body, &body); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck
	service := strings.TrimSpace(body.Service)
	if service == "" {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "service is required"))
		return
	}

	cfg, cErr := handler.codercaCfgStore.Get(req.Context(), orgID)
	if errors.Is(cErr, ruletypes.ErrCodebaseRCAConfigNotFound) {
		cfg = ruletypes.DefaultCodebaseRCAConfig(orgID)
	} else if cErr != nil {
		render.Error(rw, errors.WrapInternalf(cErr, errors.CodeInternal, "fetch codebase RCA config"))
		return
	}

	now := time.Now()
	res, err := handler.codercaRunStore.Admit(req.Context(), codercarunstore.AdmitParams{
		OrgID:   orgID,
		Service: service,
		// Unique dedup key so a manual test is never deduped against a prior run.
		DedupKey:       "manual-" + service + "-" + strconv.FormatInt(now.UnixNano(), 10),
		Now:            now,
		CooldownWindow: time.Duration(cfg.CooldownWindowSecs) * time.Second,
		MaxRunsPerDay:  cfg.MaxRunsPerDay,
		MaxQueueDepth:  cfg.MaxQueueDepth,
	})
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "enqueue code RCA run"))
		return
	}
	render.Success(rw, http.StatusOK, map[string]any{
		"admitted": res.Admitted,
		"runId":    res.RunID,
		"reason":   string(res.Reason),
	})
}

// ExportCodeRCARun handles POST /api/v2/ds/coderca/runs/{runId}/export.
// Renders a done run as a markdown artifact and writes it under the mapped
// repo's artifactPath (<root>/ds-hub/) for hand-off to ds-navi.
func (handler *handler) ExportCodeRCARun(rw http.ResponseWriter, req *http.Request) {
	orgID, err := extractOrgID(req.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}
	if orgID == "" {
		render.Error(rw, errors.Newf(errors.TypeUnauthenticated, errors.CodeUnauthenticated, "missing org id in claims"))
		return
	}

	runID := strings.TrimSpace(mux.Vars(req)["runId"])
	if runID == "" {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	run, err := handler.codercaRunStore.GetRun(req.Context(), orgID, runID)
	if errors.Is(err, codercarunstore.ErrRunNotFound) {
		render.Error(rw, errors.NewNotFoundf(errors.CodeNotFound, "coderca run was not found"))
		return
	}
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "fetch coderca run"))
		return
	}
	if run.Status != coderca.RunStatusDone {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput,
			"완료(done)된 실행만 내보낼 수 있습니다 (현재 상태: %s)", run.Status))
		return
	}

	mappings, err := handler.codebaseMapStore.List(req.Context(), orgID)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "list codebase service maps"))
		return
	}
	m, ok := coderca.ResolveServiceRepo(mappings, orgID, run.Service)
	if !ok {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput,
			"서비스에 매핑된 저장소가 없습니다: %s", run.Service))
		return
	}
	repo, err := handler.codebaseRepoStore.Get(req.Context(), orgID, m.RepoID, handler.aiCipher.DecryptFunc())
	if errors.Is(err, ruletypes.ErrCodebaseRepoNotFound) {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput,
			"서비스에 매핑된 저장소가 없습니다: %s", run.Service))
		return
	}
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "fetch codebase repo"))
		return
	}
	if !repo.Enabled {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput,
			"서비스에 매핑된 저장소가 비활성화되어 있습니다: %s", m.RepoID))
		return
	}
	if strings.TrimSpace(repo.ArtifactPath) == "" {
		render.Error(rw, errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput,
			"저장소에 산출물 경로(artifactPath)가 설정되지 않았습니다: %s", m.RepoID))
		return
	}

	path, err := exportmd.Write(strings.TrimSpace(repo.ArtifactPath), run)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "write RCA export artifact"))
		return
	}
	render.Success(rw, http.StatusOK, map[string]string{"path": path})
}
