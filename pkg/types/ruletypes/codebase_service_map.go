package ruletypes

// CodebaseServiceMap maps a service to the repository CF-11 should explore for
// it (design §8, table ds_codebase_service_map). The mapping is explicit:
// unmapped services are skipped (`no_repo_mapping`) rather than guessed, which
// keeps the cost gates honest. Subpath narrows exploration inside a monorepo.
//
// This is the domain shape consumed by the pure service→repo resolver. Its
// Storable form + store interface are added with the config-store seam; only
// the resolver (M2) needs this type today.
type CodebaseServiceMap struct {
	OrgID       string `json:"orgId"`
	ServiceName string `json:"serviceName"`
	RepoID      string `json:"repoId"`
	Subpath     string `json:"subpath"`
}
