package coderca

import (
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// ResolveServiceRepo finds the repository mapping for (orgID, service) from an
// explicit mapping set. Resolution is exact (trimmed) on both org and service —
// no fuzzy fallback — so an unmapped service returns ok=false and the caller
// skips with reason `no_repo_mapping` (design §6.1/§14). Tenant isolation: a
// service mapped under another org never matches.
//
// M2-1 STUB: always unresolved so the "found" assertions fail (RED).
func ResolveServiceRepo(maps []ruletypes.CodebaseServiceMap, orgID, service string) (ruletypes.CodebaseServiceMap, bool) {
	return ruletypes.CodebaseServiceMap{}, false
}
