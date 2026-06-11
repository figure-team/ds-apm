package ruletypes

import (
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
	// STUB — replaced in GREEN.
	return nil
}

// looksLikeGitURL is a permissive sanity check: a git remote is either a URL
// with a scheme (https://, ssh://, git://) or scp-like (git@host:path).
func looksLikeGitURL(s string) bool {
	return strings.Contains(s, "://") || strings.Contains(s, "@")
}
