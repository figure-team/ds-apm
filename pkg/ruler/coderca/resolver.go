package coderca

import (
	"strings"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// ResolveServiceRepo finds the repository mapping for (orgID, service) from an
// explicit mapping set. Resolution is exact (trimmed) on both org and service —
// no fuzzy fallback — so an unmapped service returns ok=false and the caller
// skips with reason `no_repo_mapping` (design §6.1/§14). Tenant isolation: a
// service mapped under another org never matches. An empty service never
// resolves.
func ResolveServiceRepo(maps []ruletypes.CodebaseServiceMap, orgID, service string) (ruletypes.CodebaseServiceMap, bool) {
	org := strings.TrimSpace(orgID)
	svc := strings.TrimSpace(service)
	if svc == "" {
		return ruletypes.CodebaseServiceMap{}, false
	}
	for _, m := range maps {
		if strings.TrimSpace(m.OrgID) == org && strings.TrimSpace(m.ServiceName) == svc {
			return m, true
		}
	}
	return ruletypes.CodebaseServiceMap{}, false
}
