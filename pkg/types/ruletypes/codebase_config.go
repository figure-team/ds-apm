package ruletypes

import (
	"fmt"
	"strings"
)

// CodebaseRepoContractVersion versions the CF-11 repo-registration payload.
const CodebaseRepoContractVersion = "ds.codebase_repo.v1"

// CodebaseRepo is a registered source repository for CF-11 code RCA, plus its
// tracked source state. The read credential is stored encrypted at rest
// (secretbox); plaintext is handled only in-process and must be scrubbed
// before returning over the network.
type CodebaseRepo struct {
	ContractVersion string `json:"contractVersion"`
	OrgID           string `json:"orgId"`
	RepoID          string `json:"repoId"`
	GitURL          string `json:"gitUrl"`
	DefaultBranch   string `json:"defaultBranch"`
	// Credential is a read-only git credential (PAT / token). Plaintext
	// in-process only; encrypted at rest; never serialized to clients.
	Credential string `json:"credential"`
	Enabled    bool   `json:"enabled"`

	// Source state — managed by the source-state manager, surfaced read-only
	// in the UI (design §8).
	BranchName     string `json:"branchName"`
	Fetched        bool   `json:"fetched"`
	BaselineCommit string `json:"baselineCommit"`
	LastSyncAt     string `json:"lastSyncAt"` // RFC3339
	LastSyncStatus string `json:"lastSyncStatus"`
}

// ValidateCodebaseRepo validates a repo registration.
//
// encryptionAvailable reflects whether a real secretbox key is configured
// (i.e. secretbox.FromEnv did NOT fall back to plaintext). When false, a
// non-empty Credential is rejected fail-closed (design §9 / Codex r2 #6):
// we never silently persist a git credential in plaintext. Public /
// credential-less repos remain allowed in that mode.
func ValidateCodebaseRepo(repo CodebaseRepo, encryptionAvailable bool) error {
	var errs []string

	if strings.TrimSpace(repo.ContractVersion) != CodebaseRepoContractVersion {
		errs = append(errs, fmt.Sprintf("contractVersion: must be %q, got %q", CodebaseRepoContractVersion, repo.ContractVersion))
	}
	if strings.TrimSpace(repo.OrgID) == "" {
		errs = append(errs, "orgId: must not be empty")
	}
	if strings.TrimSpace(repo.RepoID) == "" {
		errs = append(errs, "repoId: must not be empty")
	}
	if strings.TrimSpace(repo.GitURL) == "" {
		errs = append(errs, "gitUrl: must not be empty")
	} else if !looksLikeGitURL(repo.GitURL) {
		errs = append(errs, fmt.Sprintf("gitUrl: %q is not a valid git remote (need scheme:// or user@host:path)", repo.GitURL))
	}

	if len(repo.Credential) > MaxSecretLen {
		errs = append(errs, fmt.Sprintf("credential: exceeds %d-byte limit (got %d)", MaxSecretLen, len(repo.Credential)))
	}
	// Credential is delivered to git via GIT_ASKPASS env, so CR/LF must be rejected.
	if strings.ContainsAny(repo.Credential, "\r\n") {
		errs = append(errs, "credential: must not contain CR or LF")
	}
	// Fail-closed: refuse to persist a credential when encryption is unavailable.
	if repo.Credential != "" && !encryptionAvailable {
		errs = append(errs, "credential: encryption is not configured (DS_APM_AI_CONFIG_ENCRYPTION_KEY); refusing to store a git credential in plaintext")
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("codebase repo validation: %s", strings.Join(errs, "; "))
}

// looksLikeGitURL is a permissive sanity check: a git remote is either a URL
// with a scheme (https://, ssh://, git://) or scp-like (git@host:path).
func looksLikeGitURL(s string) bool {
	return strings.Contains(s, "://") || strings.Contains(s, "@")
}
