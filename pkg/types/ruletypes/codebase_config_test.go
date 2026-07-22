package ruletypes

import (
	"path/filepath"
	"testing"
)

func validRepo() CodebaseRepo {
	return CodebaseRepo{
		ContractVersion: CodebaseRepoContractVersion,
		OrgID:           "org1",
		RepoID:          "payments-api",
		GitURL:          "https://github.com/acme/payments.git",
		DefaultBranch:   "main",
	}
}

func TestValidateCodebaseRepo(t *testing.T) {
	tests := []struct {
		name                string
		mutate              func(*CodebaseRepo)
		encryptionAvailable bool
		wantErr             bool
	}{
		{name: "valid, no credential, encryption off", mutate: nil, encryptionAvailable: false, wantErr: false},
		{name: "valid, no credential, encryption on", mutate: nil, encryptionAvailable: true, wantErr: false},
		{
			name:                "credential with encryption on is allowed",
			mutate:              func(r *CodebaseRepo) { r.Credential = "ghp_token" },
			encryptionAvailable: true,
			wantErr:             false,
		},
		{
			// The fail-closed rule (Codex r2 #6): never persist a credential in plaintext.
			name:                "FAIL-CLOSED: credential with encryption off is rejected",
			mutate:              func(r *CodebaseRepo) { r.Credential = "ghp_token" },
			encryptionAvailable: false,
			wantErr:             true,
		},
		{
			name:                "wrong contract version",
			mutate:              func(r *CodebaseRepo) { r.ContractVersion = "ds.codebase_repo.v0" },
			encryptionAvailable: true,
			wantErr:             true,
		},
		{
			name:                "empty orgId",
			mutate:              func(r *CodebaseRepo) { r.OrgID = "" },
			encryptionAvailable: true,
			wantErr:             true,
		},
		{
			name:                "empty repoId",
			mutate:              func(r *CodebaseRepo) { r.RepoID = "  " },
			encryptionAvailable: true,
			wantErr:             true,
		},
		{
			name:                "empty gitUrl",
			mutate:              func(r *CodebaseRepo) { r.GitURL = "" },
			encryptionAvailable: true,
			wantErr:             true,
		},
		{
			name:                "garbage gitUrl",
			mutate:              func(r *CodebaseRepo) { r.GitURL = "not-a-url" },
			encryptionAvailable: true,
			wantErr:             true,
		},
		{
			name:                "ssh scp-like gitUrl is accepted",
			mutate:              func(r *CodebaseRepo) { r.GitURL = "git@github.com:acme/payments.git" },
			encryptionAvailable: true,
			wantErr:             false,
		},
		{
			name:                "credential with CRLF rejected (askpass env-injected)",
			mutate:              func(r *CodebaseRepo) { r.Credential = "tok\nen" },
			encryptionAvailable: true,
			wantErr:             true,
		},
		{
			name:                "credential over MaxSecretLen rejected",
			mutate:              func(r *CodebaseRepo) { r.Credential = string(make([]byte, MaxSecretLen+1)) },
			encryptionAvailable: true,
			wantErr:             true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := validRepo()
			if tc.mutate != nil {
				tc.mutate(&repo)
			}
			err := ValidateCodebaseRepo(repo, tc.encryptionAvailable)
			if tc.wantErr && err == nil {
				t.Errorf("ValidateCodebaseRepo() = nil, want error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("ValidateCodebaseRepo() = %v, want nil", err)
			}
		})
	}
}

func TestValidateCodebaseRepoArtifactPath(t *testing.T) {
	base := validRepo()

	base.ArtifactPath = "relative/path"
	if err := ValidateCodebaseRepo(base, true); err == nil {
		t.Fatal("relative artifactPath must be rejected")
	}

	base.ArtifactPath = ""
	if err := ValidateCodebaseRepo(base, true); err != nil {
		t.Fatalf("empty artifactPath must be allowed: %v", err)
	}

	abs, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	base.ArtifactPath = abs
	if err := ValidateCodebaseRepo(base, true); err != nil {
		t.Fatalf("absolute artifactPath must pass: %v", err)
	}
}
