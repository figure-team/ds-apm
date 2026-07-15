package signozapiserver

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestRulerRouteAuthzContract locks the authz wrapper and the OpenAPI
// SecuritySchemes role declared for each sensitive ds route in ruler.go
// (RBAC hardening spec D-2/D-3). Booting the real provider needs heavy
// dependencies, so — in the same lightweight-contract spirit as
// TestRemediationTargetRouteOrdering (a mux mirror-registration test) — this
// asserts the contract at the registration-source level: every route is a
// stable one-line `router.Handle("...", handler.New(provider.authZ.XxxAccess(...`
// call, so a source scan catches a wrong wrapper or a stale SecuritySchemes
// role — the one regression `go test` compilation cannot see.
func TestRulerRouteAuthzContract(t *testing.T) {
	src, err := os.ReadFile("ruler.go")
	if err != nil {
		t.Fatalf("read ruler.go: %v", err)
	}
	blocks := strings.Split(string(src), "router.Handle(")

	findBlock := func(id string) (string, bool) {
		re := regexp.MustCompile(`ID:\s*"` + regexp.QuoteMeta(id) + `"`)
		for _, b := range blocks {
			if re.MatchString(b) {
				return b, true
			}
		}
		return "", false
	}

	cases := []struct {
		id      string
		wrapper string // provider.authZ.<wrapper>(
		role    string // newSecuritySchemes(types.<role>)
	}{
		// RBAC 보완 대상 (spec FR-2/FR-3).
		{"GetRemediation", "AdminAccess", "RoleAdmin"},
		{"ListRemediations", "AdminAccess", "RoleAdmin"},
		{"PreviewAIStrategy", "EditAccess", "RoleEditor"},
		{"GenerateIncidentReport", "EditAccess", "RoleEditor"},
		// 회귀 가드: 아래 라우트는 이번 변경에서 그대로여야 한다.
		{"GetLatestAIStrategyHistory", "ViewAccess", "RoleViewer"},
		{"ApproveRemediation", "AdminAccess", "RoleAdmin"},
		{"RejectRemediation", "AdminAccess", "RoleAdmin"},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			block, ok := findBlock(tc.id)
			if !ok {
				t.Fatalf("route with ID %q not found in ruler.go", tc.id)
			}
			if !strings.Contains(block, "authZ."+tc.wrapper+"(") {
				t.Errorf("route %s: want authz wrapper %s, registration block:\n%.200s",
					tc.id, tc.wrapper, block)
			}
			if !strings.Contains(block, "newSecuritySchemes(types."+tc.role+")") {
				t.Errorf("route %s: want SecuritySchemes types.%s, registration block:\n%.200s",
					tc.id, tc.role, block)
			}
		})
	}
}
