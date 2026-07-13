package signozruler

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/ssh"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/http/binding"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/ruler/remediation"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// errRemediationEncryptionNotReady is the fail-closed error returned when the
// encryption master key (DS_APM_AI_CONFIG_ENCRYPTION_KEY) is not configured and a
// request would otherwise seal, unseal, or store a private key (design §3.3).
var errRemediationEncryptionNotReady = errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput,
	"암호화 마스터키(DS_APM_AI_CONFIG_ENCRYPTION_KEY) 미구성 — 원격 타겟을 등록할 수 없습니다")

// --- wire types (design §3.1/§3.2; the frontend api module consumes this exact contract) ---

// remediationTargetWire is the read-response shape. SealedCredential is NEVER
// carried here (design §3.5); HasCredential is the derived "keeps existing key"
// hint for the edit form.
type remediationTargetWire struct {
	ID                 string                       `json:"id"`
	OrgID              string                       `json:"orgId"`
	Name               string                       `json:"name"`
	Host               string                       `json:"host"`
	Port               int                          `json:"port"`
	User               string                       `json:"user"`
	CredentialKind     string                       `json:"credentialKind"`
	HostKeyFingerprint string                       `json:"hostKeyFingerprint"`
	ServiceSelectors   []string                     `json:"serviceSelectors"`
	HasCredential      bool                         `json:"hasCredential"`
	CreatedAt          string                       `json:"createdAt"`
	UpdatedAt          string                       `json:"updatedAt"`
	Health             *remediationTargetHealthWire `json:"health,omitempty"`
}

// remediationTargetHealthWire is the badge payload (spec §3). Only the list
// response carries it; create/update responses omit it (FE reloads the list).
type remediationTargetHealthWire struct {
	Status    string `json:"status"`
	CheckedAt string `json:"checkedAt,omitempty"`
	Error     string `json:"error,omitempty"`
}

// healthWireFor maps a checker snapshot entry to wire form. Missing entry or
// unknown → bare unknown (checkedAt 생략, spec §3).
func healthWireFor(snap map[string]remediation.TargetHealth, id string) *remediationTargetHealthWire {
	hlt, ok := snap[id]
	if !ok || hlt.Status == remediation.TargetHealthUnknown {
		return &remediationTargetHealthWire{Status: string(remediation.TargetHealthUnknown)}
	}
	w := &remediationTargetHealthWire{Status: string(hlt.Status), Error: hlt.Error}
	if !hlt.CheckedAt.IsZero() {
		w.CheckedAt = hlt.CheckedAt.UTC().Format(time.RFC3339)
	}
	return w
}

type remediationTargetListResponse struct {
	Targets         []remediationTargetWire `json:"targets"`
	EncryptionReady bool                    `json:"encryptionReady"`
}

type remediationCredentialRequest struct {
	Kind             string `json:"kind"`
	PrivateKeyPEM    string `json:"privateKeyPEM"`
	SealedPrivateKey string `json:"sealedPrivateKey"`
}

type remediationTargetUpsertRequest struct {
	Name               string                        `json:"name"`
	Host               string                        `json:"host"`
	Port               int                           `json:"port"`
	User               string                        `json:"user"`
	ServiceSelectors   []string                      `json:"serviceSelectors"`
	HostKeyFingerprint string                        `json:"hostKeyFingerprint"`
	Credential         *remediationCredentialRequest `json:"credential"`
}

type remediationKeygenResponse struct {
	PublicKeyOpenSSH string `json:"publicKeyOpenSSH"`
	SealedPrivateKey string `json:"sealedPrivateKey"`
}

type remediationFingerprintRequest struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type remediationFingerprintResponse struct {
	Fingerprint string `json:"fingerprint"`
	KeyType     string `json:"keyType"`
}

type remediationConnTestRequest struct {
	TargetID           string                        `json:"targetId"`
	Host               string                        `json:"host"`
	Port               int                           `json:"port"`
	User               string                        `json:"user"`
	HostKeyFingerprint string                        `json:"hostKeyFingerprint"`
	Credential         *remediationCredentialRequest `json:"credential"`
}

type remediationConnTestResponse struct {
	OK       bool   `json:"ok"`
	ExitCode int    `json:"exitCode"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
}

// invalidInput is a terse constructor for the 400s this handler returns.
func invalidInput(format string, args ...any) error {
	return errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, format, args...)
}

func toRemediationTargetWire(t ruletypes.RemediationTarget) remediationTargetWire {
	return remediationTargetWire{
		ID:                 t.ID,
		OrgID:              t.OrgID,
		Name:               t.Name,
		Host:               t.Host,
		Port:               t.Port,
		User:               t.User,
		CredentialKind:     t.CredentialKind,
		HostKeyFingerprint: t.HostKeyFingerprint,
		ServiceSelectors:   t.ServiceSelectors,
		HasCredential:      strings.TrimSpace(t.SealedCredential) != "",
		CreatedAt:          t.CreatedAt,
		UpdatedAt:          t.UpdatedAt,
	}
}

// resolveCredential turns a credential request into a sealed blob for storage
// (design §3.2). Exactly one of privateKeyPEM / sealedPrivateKey must be set.
//   - privateKeyPEM: parse-validated then Encrypt (plaintext lives only in this
//     function's scope).
//   - sealedPrivateKey: Decrypt→parse-validated, then the original blob is
//     returned unchanged (guards against tampering / a different master key).
//
// Both-set, neither-set, and (when a key must be sealed/unsealed) the insecure
// cipher are rejected — the last one is the fail-closed gate (§3.3).
func (h *handler) resolveCredential(c *remediationCredentialRequest) (string, error) {
	if c == nil || (c.PrivateKeyPEM == "" && c.SealedPrivateKey == "") {
		return "", invalidInput("credential: privateKeyPEM 또는 sealedPrivateKey 중 하나가 필요합니다")
	}
	if c.PrivateKeyPEM != "" && c.SealedPrivateKey != "" {
		return "", invalidInput("credential: privateKeyPEM과 sealedPrivateKey는 동시에 줄 수 없습니다")
	}
	if h.aiCipherInsecure {
		return "", errRemediationEncryptionNotReady
	}
	if c.PrivateKeyPEM != "" {
		if _, err := ssh.ParsePrivateKey([]byte(c.PrivateKeyPEM)); err != nil {
			return "", invalidInput("credential: 개인키 PEM 파싱 실패")
		}
		return h.aiCipher.Encrypt(c.PrivateKeyPEM)
	}
	plain, err := h.aiCipher.Decrypt(c.SealedPrivateKey)
	if err != nil {
		return "", invalidInput("credential: 봉인 blob 복호 실패")
	}
	if _, err := ssh.ParsePrivateKey([]byte(plain)); err != nil {
		return "", invalidInput("credential: 봉인된 개인키 파싱 실패")
	}
	return c.SealedPrivateKey, nil
}

// resolveTestPlaintextKey returns the PLAINTEXT PEM used to dial a connection
// test. Unlike resolveCredential it never seals — the key is used in memory and
// discarded. A draft privateKeyPEM needs no cipher (allowed even when insecure,
// §3.3); a sealedPrivateKey draft requires the cipher (fail-closed gate).
func (h *handler) resolveTestPlaintextKey(c *remediationCredentialRequest) (string, error) {
	if c == nil || (c.PrivateKeyPEM == "" && c.SealedPrivateKey == "") {
		return "", invalidInput("credential: privateKeyPEM 또는 sealedPrivateKey 중 하나가 필요합니다")
	}
	if c.PrivateKeyPEM != "" && c.SealedPrivateKey != "" {
		return "", invalidInput("credential: privateKeyPEM과 sealedPrivateKey는 동시에 줄 수 없습니다")
	}
	if c.PrivateKeyPEM != "" {
		return c.PrivateKeyPEM, nil
	}
	if h.aiCipherInsecure {
		return "", errRemediationEncryptionNotReady
	}
	plain, err := h.aiCipher.Decrypt(c.SealedPrivateKey)
	if err != nil {
		return "", invalidInput("credential: 봉인 blob 복호 실패")
	}
	return plain, nil
}

// ListRemediationTargets handles GET /api/v2/ds/remediation/targets. Admin-only.
// Returns targets without any credential material plus the encryptionReady flag
// that drives the UI's fail-closed banner (design §3.1/§3.3).
func (h *handler) ListRemediationTargets(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
		return
	}
	targets, err := h.remediationTargetStore.List(req.Context(), orgID)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "list remediation targets"))
		return
	}
	snap := h.remediationHealth.Snapshot() // nil 체커 → nil 맵 → 전부 unknown (fail-open)
	wire := make([]remediationTargetWire, 0, len(targets))
	for _, t := range targets {
		w := toRemediationTargetWire(t)
		w.Health = healthWireFor(snap, t.ID)
		wire = append(wire, w)
	}
	render.Success(rw, http.StatusOK, remediationTargetListResponse{
		Targets:         wire,
		EncryptionReady: !h.aiCipherInsecure,
	})
}

// CreateRemediationTarget handles POST /api/v2/ds/remediation/targets. Admin-only.
func (h *handler) CreateRemediationTarget(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
		return
	}
	// Fail-closed gate (§3.3): a create always seals a key.
	if h.aiCipherInsecure {
		render.Error(rw, errRemediationEncryptionNotReady)
		return
	}
	var in remediationTargetUpsertRequest
	if err := binding.JSON.BindBody(req.Body, &in); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	sealed, err := h.resolveCredential(in.Credential)
	if err != nil {
		render.Error(rw, err)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	target := ruletypes.RemediationTarget{
		ID:                 uuid.NewString(),
		OrgID:              orgID,
		Name:               in.Name,
		Host:               in.Host,
		Port:               in.Port,
		User:               in.User,
		SealedCredential:   sealed,
		CredentialKind:     ruletypes.RemediationCredentialKindPrivateKey,
		HostKeyFingerprint: in.HostKeyFingerprint,
		ServiceSelectors:   in.ServiceSelectors,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := h.remediationTargetStore.Create(req.Context(), orgID, target); err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "create remediation target"))
		return
	}
	h.remediationHealth.Poke(target) // 신규 타겟 첫 배지를 즉시 채운다 (fire-and-forget, spec §2.2)
	render.Success(rw, http.StatusCreated, toRemediationTargetWire(target))
}

// UpdateRemediationTarget handles PUT /api/v2/ds/remediation/targets/{targetId}.
// Admin-only. When credential is omitted the existing sealed key is preserved by
// copying it from the stored row BEFORE Update runs (design §3.2 trap: the store
// re-validates and a blank sealedCredential is always a 400).
func (h *handler) UpdateRemediationTarget(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
		return
	}
	targetID := strings.TrimSpace(mux.Vars(req)["targetId"])
	if targetID == "" {
		render.Error(rw, invalidInput("targetId: 필수"))
		return
	}
	var in remediationTargetUpsertRequest
	if err := binding.JSON.BindBody(req.Body, &in); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	existing, err := h.remediationTargetStore.Get(req.Context(), orgID, targetID)
	if err != nil {
		render.Error(rw, errors.Newf(errors.TypeNotFound, errors.CodeNotFound, "remediation target not found"))
		return
	}

	target := ruletypes.RemediationTarget{
		ID:                 targetID,
		OrgID:              orgID,
		Name:               in.Name,
		Host:               in.Host,
		Port:               in.Port,
		User:               in.User,
		HostKeyFingerprint: in.HostKeyFingerprint,
		ServiceSelectors:   in.ServiceSelectors,
		CredentialKind:     ruletypes.RemediationCredentialKindPrivateKey,
		CreatedAt:          existing.CreatedAt,
		UpdatedAt:          time.Now().UTC().Format(time.RFC3339),
	}

	if in.Credential == nil {
		// Keep the existing key (§3.2): copy the sealed blob + kind forward, else
		// ValidateRemediationTarget rejects the blank sealedCredential.
		target.SealedCredential = existing.SealedCredential
		target.CredentialKind = existing.CredentialKind
	} else {
		// A new key is being set → fail-closed gate applies.
		if h.aiCipherInsecure {
			render.Error(rw, errRemediationEncryptionNotReady)
			return
		}
		sealed, err := h.resolveCredential(in.Credential)
		if err != nil {
			render.Error(rw, err)
			return
		}
		target.SealedCredential = sealed
	}

	if err := h.remediationTargetStore.Update(req.Context(), orgID, target); err != nil {
		render.Error(rw, errors.WrapInvalidInputf(err, errors.CodeInvalidInput, "update remediation target"))
		return
	}
	h.remediationHealth.Poke(target) // 수정된 host/port/지문 기준 즉시 재프로브
	render.Success(rw, http.StatusOK, toRemediationTargetWire(target))
}

// DeleteRemediationTarget handles DELETE /api/v2/ds/remediation/targets/{targetId}.
// Admin-only. The confirm prompt is a frontend concern (design §3.1).
func (h *handler) DeleteRemediationTarget(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
		return
	}
	targetID := strings.TrimSpace(mux.Vars(req)["targetId"])
	if targetID == "" {
		render.Error(rw, invalidInput("targetId: 필수"))
		return
	}
	if err := h.remediationTargetStore.Delete(req.Context(), orgID, targetID); err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "delete remediation target"))
		return
	}
	render.Success(rw, http.StatusOK, nil)
}

// KeygenRemediationTarget handles POST /api/v2/ds/remediation/targets/keygen.
// Admin-only. Generates an ed25519 keypair server-side: the OpenSSH public key is
// returned for the operator to install in authorized_keys, and the private key is
// sealed and returned as an opaque blob (the plaintext never reaches the browser,
// design §3.5). Stateless round-trip: the client returns sealedPrivateKey verbatim
// at save time.
func (h *handler) KeygenRemediationTarget(rw http.ResponseWriter, req *http.Request) {
	if _, err := requireOrg(req); err != nil {
		render.Error(rw, err)
		return
	}
	// Fail-closed gate (§3.3): keygen seals the private key.
	if h.aiCipherInsecure {
		render.Error(rw, errRemediationEncryptionNotReady)
		return
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "generate keypair"))
		return
	}
	pemBlock, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "marshal private key"))
		return
	}
	pemBytes := pem.EncodeToMemory(pemBlock)
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "marshal public key"))
		return
	}
	authorized := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
	sealed, err := h.aiCipher.Encrypt(string(pemBytes))
	if err != nil {
		render.Error(rw, errors.WrapInternalf(err, errors.CodeInternal, "seal private key"))
		return
	}
	render.Success(rw, http.StatusOK, remediationKeygenResponse{
		PublicKeyOpenSSH: authorized + " ds-apm-remediation",
		SealedPrivateKey: sealed,
	})
}

// FingerprintRemediationTarget handles POST /api/v2/ds/remediation/targets/fingerprint.
// Admin-only. Dials {host, port} and returns the server host-key SHA256
// fingerprint for TOFU + manual confirmation (design §3.4). No credential and no
// cipher are involved (this runs before authentication).
func (h *handler) FingerprintRemediationTarget(rw http.ResponseWriter, req *http.Request) {
	if _, err := requireOrg(req); err != nil {
		render.Error(rw, err)
		return
	}
	var in remediationFingerprintRequest
	if err := binding.JSON.BindBody(req.Body, &in); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	if strings.TrimSpace(in.Host) == "" {
		render.Error(rw, invalidInput("host: 필수"))
		return
	}
	if in.Port < 1 || in.Port > 65535 {
		render.Error(rw, invalidInput("port: 1..65535 범위여야 합니다 (got %d)", in.Port))
		return
	}

	fingerprint, keyType, err := remediation.FetchHostKeyFingerprint(req.Context(), in.Host, in.Port, 5*time.Second)
	if err != nil {
		render.Error(rw, invalidInput("지문 수집 실패: %s", strings.TrimSpace(err.Error())))
		return
	}
	render.Success(rw, http.StatusOK, remediationFingerprintResponse{
		Fingerprint: fingerprint,
		KeyType:     keyType,
	})
}

// TestRemediationTarget handles POST /api/v2/ds/remediation/targets/test.
// Admin-only. Runs `echo ok` against either a stored target (targetId) or a draft
// (inline connection params + credential). A transport/connection failure is
// reported as 200 + ok:false so the UI shows it inline (design §3.1/§3.6).
func (h *handler) TestRemediationTarget(rw http.ResponseWriter, req *http.Request) {
	orgID, err := requireOrg(req)
	if err != nil {
		render.Error(rw, err)
		return
	}
	var in remediationConnTestRequest
	if err := binding.JSON.BindBody(req.Body, &in); err != nil {
		render.Error(rw, err)
		return
	}
	defer req.Body.Close() //nolint:errcheck

	var (
		target ruletypes.RemediationTarget
		keyPEM string
	)
	if strings.TrimSpace(in.TargetID) != "" {
		// Stored-target path: decrypt requires the cipher → fail-closed gate.
		if h.aiCipherInsecure {
			render.Error(rw, errRemediationEncryptionNotReady)
			return
		}
		stored, err := h.remediationTargetStore.Get(req.Context(), orgID, strings.TrimSpace(in.TargetID))
		if err != nil {
			render.Error(rw, errors.Newf(errors.TypeNotFound, errors.CodeNotFound, "remediation target not found"))
			return
		}
		plain, err := h.aiCipher.Decrypt(stored.SealedCredential)
		if err != nil || strings.TrimSpace(plain) == "" {
			render.Error(rw, invalidInput("저장된 자격증명 복호 실패"))
			return
		}
		target = ruletypes.RemediationTarget{
			Host:               stored.Host,
			Port:               stored.Port,
			User:               stored.User,
			HostKeyFingerprint: stored.HostKeyFingerprint,
		}
		keyPEM = plain
	} else {
		// Draft path: an empty fingerprint would collide with the pinned host-key
		// callback and produce a confusing "host key mismatch: got ... want "
		// error — reject early (§3.6, paired with the frontend button disable).
		if strings.TrimSpace(in.HostKeyFingerprint) == "" {
			render.Error(rw, invalidInput("호스트키 지문이 필요합니다 — 지문 가져오기를 먼저 실행하세요"))
			return
		}
		plain, err := h.resolveTestPlaintextKey(in.Credential)
		if err != nil {
			render.Error(rw, err)
			return
		}
		target = ruletypes.RemediationTarget{
			Host:               in.Host,
			Port:               in.Port,
			User:               in.User,
			HostKeyFingerprint: in.HostKeyFingerprint,
		}
		keyPEM = plain
	}

	out, exitCode, testErr := remediation.TestConnection(req.Context(), target, keyPEM)
	resp := remediationConnTestResponse{
		OK:       testErr == nil && exitCode == 0,
		ExitCode: exitCode,
		Output:   truncateRemediationSnippet(out),
	}
	if testErr != nil {
		resp.Error = strings.TrimSpace(testErr.Error())
	}
	// Transport failure is a result, not a server error: 200 + ok:false.
	render.Success(rw, http.StatusOK, resp)
}
