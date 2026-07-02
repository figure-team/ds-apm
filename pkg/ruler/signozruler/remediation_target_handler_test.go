package signozruler

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// genEd25519PEM returns a valid OpenSSH ed25519 private key PEM for the credential
// paths (ssh.ParsePrivateKey must accept it).
func genEd25519PEM(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	return string(pem.EncodeToMemory(block))
}

// authedTargetReq builds an org-scoped request with a JSON body and mux vars.
func authedTargetReq(method, target, body string, vars map[string]string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req = withSOPTestClaims(req)
	for k, v := range vars {
		req = muxSetVar(req, k, v)
	}
	return req
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

// firstTarget returns the single target held by the fake store (fails otherwise).
func firstTarget(t *testing.T, ts *fakeRemediationTargetStore) ruletypes.RemediationTarget {
	t.Helper()
	all, _ := ts.List(context.Background(), testOrgID)
	if len(all) != 1 {
		t.Fatalf("want exactly 1 stored target, got %d", len(all))
	}
	return all[0]
}

func validUpsertBody(t *testing.T, cred *remediationCredentialRequest) string {
	return mustJSON(t, remediationTargetUpsertRequest{
		Name:               "site-a",
		Host:               "10.0.0.5",
		Port:               22,
		User:               "svc",
		ServiceSelectors:   []string{"svc-a"},
		HostKeyFingerprint: "SHA256:abc",
		Credential:         cred,
	})
}

func TestRemediationTarget_Create_PEMPath_Seals(t *testing.T) {
	h, _, targetStore, _ := newRemediationHandlerWithTargetStore(t)
	keyPEM := genEd25519PEM(t)
	body := validUpsertBody(t, &remediationCredentialRequest{Kind: "private_key", PrivateKeyPEM: keyPEM})

	rw := httptest.NewRecorder()
	h.CreateRemediationTarget(rw, authedTargetReq(http.MethodPost, "/api/v2/ds/remediation/targets", body, nil))

	if rw.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d (body=%s)", rw.Code, rw.Body.String())
	}
	// Plaintext cipher: sealed == plaintext, so the stored blob is the PEM itself.
	got := firstTarget(t, targetStore)
	if got.SealedCredential != keyPEM {
		t.Fatalf("stored SealedCredential mismatch:\n got %q\nwant %q", got.SealedCredential, keyPEM)
	}
	if got.CredentialKind != ruletypes.RemediationCredentialKindPrivateKey {
		t.Fatalf("credentialKind: got %q", got.CredentialKind)
	}
	// Response must never carry the sealed blob.
	if strings.Contains(rw.Body.String(), keyPEM) || strings.Contains(rw.Body.String(), "sealedCredential") {
		t.Fatalf("response leaked credential material: %s", rw.Body.String())
	}
}

func TestRemediationTarget_Create_ValidationFail_400(t *testing.T) {
	h, _, _, _ := newRemediationHandlerWithTargetStore(t)
	// Missing serviceSelectors → ValidateRemediationTarget (in the store) rejects.
	body := mustJSON(t, remediationTargetUpsertRequest{
		Name: "site-a", Host: "10.0.0.5", Port: 22, User: "svc",
		HostKeyFingerprint: "SHA256:abc",
		Credential:         &remediationCredentialRequest{Kind: "private_key", PrivateKeyPEM: genEd25519PEM(t)},
	})

	rw := httptest.NewRecorder()
	h.CreateRemediationTarget(rw, authedTargetReq(http.MethodPost, "/api/v2/ds/remediation/targets", body, nil))

	if rw.Code != http.StatusBadRequest {
		t.Fatalf("want 400 on validation failure, got %d (body=%s)", rw.Code, rw.Body.String())
	}
}

func TestRemediationTarget_Create_InsecureCipher_400(t *testing.T) {
	h, _, _, _ := newRemediationHandlerWithTargetStore(t)
	h.aiCipherInsecure = true
	body := validUpsertBody(t, &remediationCredentialRequest{Kind: "private_key", PrivateKeyPEM: genEd25519PEM(t)})

	rw := httptest.NewRecorder()
	h.CreateRemediationTarget(rw, authedTargetReq(http.MethodPost, "/api/v2/ds/remediation/targets", body, nil))

	if rw.Code != http.StatusBadRequest {
		t.Fatalf("want 400 when master key missing, got %d (body=%s)", rw.Code, rw.Body.String())
	}
	if !strings.Contains(rw.Body.String(), "DS_APM_AI_CONFIG_ENCRYPTION_KEY") {
		t.Fatalf("expected fail-closed message, got %s", rw.Body.String())
	}
}

func TestRemediationTarget_Create_SealedPath_OK(t *testing.T) {
	h, _, targetStore, _ := newRemediationHandlerWithTargetStore(t)
	// Plaintext cipher: a valid PEM handed in as sealedPrivateKey decrypts to
	// itself and parses, so it is stored verbatim.
	keyPEM := genEd25519PEM(t)
	body := validUpsertBody(t, &remediationCredentialRequest{Kind: "private_key", SealedPrivateKey: keyPEM})

	rw := httptest.NewRecorder()
	h.CreateRemediationTarget(rw, authedTargetReq(http.MethodPost, "/api/v2/ds/remediation/targets", body, nil))

	if rw.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d (body=%s)", rw.Code, rw.Body.String())
	}
	if got := firstTarget(t, targetStore); got.SealedCredential != keyPEM {
		t.Fatalf("sealed blob not stored verbatim: got %q", got.SealedCredential)
	}
}

func TestRemediationTarget_Create_SealedPath_Tampered_400(t *testing.T) {
	h, _, _, _ := newRemediationHandlerWithTargetStore(t)
	// Garbage sealed blob: plaintext-decrypts to garbage → ParsePrivateKey fails.
	body := validUpsertBody(t, &remediationCredentialRequest{Kind: "private_key", SealedPrivateKey: "not-a-real-sealed-key"})

	rw := httptest.NewRecorder()
	h.CreateRemediationTarget(rw, authedTargetReq(http.MethodPost, "/api/v2/ds/remediation/targets", body, nil))

	if rw.Code != http.StatusBadRequest {
		t.Fatalf("want 400 on tampered sealed blob, got %d (body=%s)", rw.Code, rw.Body.String())
	}
}

func TestRemediationTarget_Create_BothCredentials_400(t *testing.T) {
	h, _, _, _ := newRemediationHandlerWithTargetStore(t)
	body := validUpsertBody(t, &remediationCredentialRequest{
		Kind: "private_key", PrivateKeyPEM: genEd25519PEM(t), SealedPrivateKey: "x",
	})

	rw := httptest.NewRecorder()
	h.CreateRemediationTarget(rw, authedTargetReq(http.MethodPost, "/api/v2/ds/remediation/targets", body, nil))

	if rw.Code != http.StatusBadRequest {
		t.Fatalf("want 400 when both credentials set, got %d (body=%s)", rw.Code, rw.Body.String())
	}
}

func TestRemediationTarget_Update_OmitCredential_KeepsExisting(t *testing.T) {
	h, _, targetStore, _ := newRemediationHandlerWithTargetStore(t)
	const id = "11111111-1111-4111-8111-111111111111"
	targetStore.seed(ruletypes.RemediationTarget{
		ID: id, OrgID: testOrgID, Name: "old-name", Host: "10.0.0.5", Port: 22, User: "svc",
		SealedCredential: "original-sealed-blob", CredentialKind: ruletypes.RemediationCredentialKindPrivateKey,
		HostKeyFingerprint: "SHA256:abc", ServiceSelectors: []string{"svc-a"}, CreatedAt: "2026-07-01T00:00:00Z",
	})
	// PUT without credential; change the name.
	body := mustJSON(t, remediationTargetUpsertRequest{
		Name: "new-name", Host: "10.0.0.5", Port: 22, User: "svc",
		ServiceSelectors: []string{"svc-a"}, HostKeyFingerprint: "SHA256:abc", Credential: nil,
	})

	rw := httptest.NewRecorder()
	h.UpdateRemediationTarget(rw, authedTargetReq(http.MethodPut,
		"/api/v2/ds/remediation/targets/"+id, body, map[string]string{"targetId": id}))

	if rw.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (body=%s)", rw.Code, rw.Body.String())
	}
	got := firstTarget(t, targetStore)
	if got.SealedCredential != "original-sealed-blob" {
		t.Fatalf("credential must be preserved, got %q", got.SealedCredential)
	}
	if got.Name != "new-name" {
		t.Fatalf("name must update, got %q", got.Name)
	}
	if got.CreatedAt != "2026-07-01T00:00:00Z" {
		t.Fatalf("createdAt must be preserved, got %q", got.CreatedAt)
	}
}

func TestRemediationTarget_List_NoCredentialLeak_And_EncryptionReady(t *testing.T) {
	h, _, targetStore, _ := newRemediationHandlerWithTargetStore(t)
	targetStore.seed(ruletypes.RemediationTarget{
		ID: "22222222-2222-4222-8222-222222222222", OrgID: testOrgID, Name: "site-a",
		Host: "10.0.0.6", Port: 22, User: "svc", SealedCredential: "super-secret-blob",
		CredentialKind: ruletypes.RemediationCredentialKindPrivateKey, HostKeyFingerprint: "SHA256:def",
		ServiceSelectors: []string{"svc-a"},
	})

	rw := httptest.NewRecorder()
	h.ListRemediationTargets(rw, authedTargetReq(http.MethodGet, "/api/v2/ds/remediation/targets", "", nil))

	if rw.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (body=%s)", rw.Code, rw.Body.String())
	}
	body := rw.Body.String()
	if strings.Contains(body, "super-secret-blob") || strings.Contains(body, "sealedCredential") {
		t.Fatalf("list leaked credential material: %s", body)
	}
	if !strings.Contains(body, `"encryptionReady":true`) {
		t.Fatalf("expected encryptionReady:true, got %s", body)
	}
	if !strings.Contains(body, `"hasCredential":true`) {
		t.Fatalf("expected hasCredential:true, got %s", body)
	}
}

func TestRemediationTarget_List_EncryptionNotReady(t *testing.T) {
	h, _, _, _ := newRemediationHandlerWithTargetStore(t)
	h.aiCipherInsecure = true

	rw := httptest.NewRecorder()
	h.ListRemediationTargets(rw, authedTargetReq(http.MethodGet, "/api/v2/ds/remediation/targets", "", nil))

	if rw.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (body=%s)", rw.Code, rw.Body.String())
	}
	if !strings.Contains(rw.Body.String(), `"encryptionReady":false`) {
		t.Fatalf("expected encryptionReady:false, got %s", rw.Body.String())
	}
}

func TestRemediationTarget_Keygen_OK(t *testing.T) {
	h, _, _, _ := newRemediationHandlerWithTargetStore(t)

	rw := httptest.NewRecorder()
	h.KeygenRemediationTarget(rw, authedTargetReq(http.MethodPost, "/api/v2/ds/remediation/targets/keygen", "", nil))

	if rw.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (body=%s)", rw.Code, rw.Body.String())
	}
	// render.Success wraps the payload under "data".
	var env struct {
		Data remediationKeygenResponse `json:"data"`
	}
	if err := json.Unmarshal(rw.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v (body=%s)", err, rw.Body.String())
	}
	resp := env.Data
	if !strings.HasPrefix(resp.PublicKeyOpenSSH, "ssh-ed25519 ") {
		t.Fatalf("public key must be OpenSSH ed25519, got %q", resp.PublicKeyOpenSSH)
	}
	if resp.SealedPrivateKey == "" {
		t.Fatal("sealedPrivateKey must be present")
	}
	// Plaintext cipher: sealed == PEM, so it must parse as a private key.
	if _, err := ssh.ParsePrivateKey([]byte(resp.SealedPrivateKey)); err != nil {
		t.Fatalf("sealed blob must decrypt+parse to a private key: %v", err)
	}
}

func TestRemediationTarget_Keygen_InsecureCipher_400(t *testing.T) {
	h, _, _, _ := newRemediationHandlerWithTargetStore(t)
	h.aiCipherInsecure = true

	rw := httptest.NewRecorder()
	h.KeygenRemediationTarget(rw, authedTargetReq(http.MethodPost, "/api/v2/ds/remediation/targets/keygen", "", nil))

	if rw.Code != http.StatusBadRequest {
		t.Fatalf("want 400 when master key missing, got %d (body=%s)", rw.Code, rw.Body.String())
	}
}

func TestRemediationTarget_Test_DraftMissingFingerprint_400(t *testing.T) {
	h, _, _, _ := newRemediationHandlerWithTargetStore(t)
	body := mustJSON(t, remediationConnTestRequest{
		Host: "10.0.0.5", Port: 22, User: "svc", HostKeyFingerprint: "",
		Credential: &remediationCredentialRequest{Kind: "private_key", PrivateKeyPEM: genEd25519PEM(t)},
	})

	rw := httptest.NewRecorder()
	h.TestRemediationTarget(rw, authedTargetReq(http.MethodPost, "/api/v2/ds/remediation/targets/test", body, nil))

	if rw.Code != http.StatusBadRequest {
		t.Fatalf("want 400 on missing fingerprint, got %d (body=%s)", rw.Code, rw.Body.String())
	}
	if !strings.Contains(rw.Body.String(), "호스트키 지문") {
		t.Fatalf("expected fingerprint-required message, got %s", rw.Body.String())
	}
}

func TestRemediationTarget_Handlers_401WithoutClaims(t *testing.T) {
	h, _, _, _ := newRemediationHandlerWithTargetStore(t)
	cases := []struct {
		name   string
		invoke func(http.ResponseWriter, *http.Request)
	}{
		{"List", h.ListRemediationTargets},
		{"Create", h.CreateRemediationTarget},
		{"Keygen", h.KeygenRemediationTarget},
		{"Fingerprint", h.FingerprintRemediationTarget},
		{"Test", h.TestRemediationTarget},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v2/ds/remediation/targets", http.NoBody)
			rw := httptest.NewRecorder()
			tc.invoke(rw, req)
			if rw.Code != http.StatusUnauthorized {
				t.Fatalf("%s must 401 without claims, got %d", tc.name, rw.Code)
			}
		})
	}
}
