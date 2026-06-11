package ruletypes

import (
	"errors"
	"testing"
)

func cbIdentityEncrypt(s string) (string, error) { return s, nil }
func cbIdentityDecrypt(s string) (string, error) { return s, nil }

func domainRepo() CodebaseRepo {
	return CodebaseRepo{
		ContractVersion: CodebaseRepoContractVersion,
		OrgID:           "org1",
		RepoID:          "payments-api",
		GitURL:          "https://github.com/acme/payments.git",
		DefaultBranch:   "main",
		Credential:      "ghp_secret",
		Enabled:         true,
		BranchName:      "main",
		Fetched:         true,
		BaselineCommit:  "abc123",
		LastSyncAt:      "2026-06-11T00:00:00Z",
		LastSyncStatus:  "ok",
	}
}

func TestCodebaseRepoRoundTrip(t *testing.T) {
	repo := domainRepo()

	storable, err := FromDomainCodebaseRepo(repo, cbIdentityEncrypt)
	if err != nil {
		t.Fatalf("FromDomainCodebaseRepo: %v", err)
	}
	if storable.CredentialCiphertext != "ghp_secret" {
		t.Errorf("credential not routed through encryptor: %q", storable.CredentialCiphertext)
	}

	got, err := storable.ToDomain(cbIdentityDecrypt)
	if err != nil {
		t.Fatalf("ToDomain: %v", err)
	}
	if got.ContractVersion != CodebaseRepoContractVersion {
		t.Errorf("ContractVersion = %q", got.ContractVersion)
	}
	for _, c := range []struct {
		field    string
		a, b     interface{}
	}{
		{"OrgID", repo.OrgID, got.OrgID},
		{"RepoID", repo.RepoID, got.RepoID},
		{"GitURL", repo.GitURL, got.GitURL},
		{"DefaultBranch", repo.DefaultBranch, got.DefaultBranch},
		{"Credential", repo.Credential, got.Credential},
		{"Enabled", repo.Enabled, got.Enabled},
		{"BranchName", repo.BranchName, got.BranchName},
		{"Fetched", repo.Fetched, got.Fetched},
		{"BaselineCommit", repo.BaselineCommit, got.BaselineCommit},
		{"LastSyncAt", repo.LastSyncAt, got.LastSyncAt},
		{"LastSyncStatus", repo.LastSyncStatus, got.LastSyncStatus},
	} {
		if c.a != c.b {
			t.Errorf("%s: roundtrip %v -> %v", c.field, c.a, c.b)
		}
	}
}

func TestFromDomainCodebaseRepoRequiresKeys(t *testing.T) {
	r := domainRepo()
	r.OrgID = ""
	if _, err := FromDomainCodebaseRepo(r, cbIdentityEncrypt); err == nil {
		t.Error("empty OrgID must error")
	}

	r = domainRepo()
	r.RepoID = ""
	if _, err := FromDomainCodebaseRepo(r, cbIdentityEncrypt); err == nil {
		t.Error("empty RepoID must error")
	}
}

func TestFromDomainCodebaseRepoEncryptError(t *testing.T) {
	boom := errors.New("boom")
	_, err := FromDomainCodebaseRepo(domainRepo(), func(string) (string, error) { return "", boom })
	if !errors.Is(err, boom) {
		t.Errorf("encrypt error must propagate, got %v", err)
	}
}
