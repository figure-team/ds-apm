package ruletypes

import (
	"fmt"
	"strings"

	"github.com/uptrace/bun"
)

// StorableCodebaseRepo is the at-rest form of CodebaseRepo. The read
// credential is stored as ciphertext (secretbox); plaintext never touches the
// database. PK is (org_id, repo_id) for tenant isolation.
type StorableCodebaseRepo struct {
	bun.BaseModel `bun:"table:ds_codebase_repo"`

	OrgID                string `bun:"org_id,pk,notnull,type:text"`
	RepoID               string `bun:"repo_id,pk,notnull,type:text"`
	GitURL               string `bun:"git_url,notnull,type:text"`
	DefaultBranch        string `bun:"default_branch,notnull,default:'',type:text"`
	CredentialCiphertext string `bun:"credential_ciphertext,notnull,default:'',type:text"`
	Enabled              bool   `bun:"enabled,notnull,default:false,type:boolean"`
	BranchName           string `bun:"branch_name,notnull,default:'',type:text"`
	Fetched              bool   `bun:"fetched,notnull,default:false,type:boolean"`
	BaselineCommit       string `bun:"baseline_commit,notnull,default:'',type:text"`
	LastSyncAt           string `bun:"last_sync_at,notnull,default:'',type:text"`
	LastSyncStatus       string `bun:"last_sync_status,notnull,default:'',type:text"`
}

// FromDomainCodebaseRepo encrypts the credential via the provided encryptor
// and returns the storable form. orgID and repoID are required.
func FromDomainCodebaseRepo(repo CodebaseRepo, encrypt func(string) (string, error)) (*StorableCodebaseRepo, error) {
	if strings.TrimSpace(repo.OrgID) == "" {
		return nil, fmt.Errorf("storable codebase repo: orgID must not be empty")
	}
	if strings.TrimSpace(repo.RepoID) == "" {
		return nil, fmt.Errorf("storable codebase repo: repoID must not be empty")
	}

	credCT, err := encrypt(repo.Credential)
	if err != nil {
		return nil, fmt.Errorf("storable codebase repo: encrypt credential: %w", err)
	}

	return &StorableCodebaseRepo{
		OrgID:                repo.OrgID,
		RepoID:               repo.RepoID,
		GitURL:               repo.GitURL,
		DefaultBranch:        repo.DefaultBranch,
		CredentialCiphertext: credCT,
		Enabled:              repo.Enabled,
		BranchName:           repo.BranchName,
		Fetched:              repo.Fetched,
		BaselineCommit:       repo.BaselineCommit,
		LastSyncAt:           repo.LastSyncAt,
		LastSyncStatus:       repo.LastSyncStatus,
	}, nil
}

// ToDomain decrypts the credential via the provided decryptor. The returned
// CodebaseRepo carries the plaintext credential; callers must scrub it before
// returning over the network.
func (s *StorableCodebaseRepo) ToDomain(decrypt func(string) (string, error)) (CodebaseRepo, error) {
	cred, err := decrypt(s.CredentialCiphertext)
	if err != nil {
		return CodebaseRepo{}, fmt.Errorf("storable codebase repo: decrypt credential: %w", err)
	}

	return CodebaseRepo{
		ContractVersion: CodebaseRepoContractVersion,
		OrgID:           s.OrgID,
		RepoID:          s.RepoID,
		GitURL:          s.GitURL,
		DefaultBranch:   s.DefaultBranch,
		Credential:      cred,
		Enabled:         s.Enabled,
		BranchName:      s.BranchName,
		Fetched:         s.Fetched,
		BaselineCommit:  s.BaselineCommit,
		LastSyncAt:      s.LastSyncAt,
		LastSyncStatus:  s.LastSyncStatus,
	}, nil
}
