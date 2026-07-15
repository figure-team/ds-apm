package app

import (
	"os"
	"regexp"
	"testing"
)

// TestIntegrationRoutesAuthzContract locks the authz wrapper used by the
// integrations subrouter registrations in http_handler.go (RBAC hardening
// spec D-1): install/uninstall must be editor-gated while the read routes
// stay viewer-accessible. Source-level assertion for the same reason as
// TestRulerRouteAuthzContract — a wrong wrapper still compiles.
func TestIntegrationRoutesAuthzContract(t *testing.T) {
	src, err := os.ReadFile("http_handler.go")
	if err != nil {
		t.Fatalf("read http_handler.go: %v", err)
	}

	cases := []struct {
		name    string
		pattern string // capture group 1 = wrapper name
		want    string
	}{
		{"InstallIntegration", `"/install", am\.(\w+)\(aH\.InstallIntegration\)`, "EditAccess"},
		{"UninstallIntegration", `"/uninstall", am\.(\w+)\(aH\.UninstallIntegration\)`, "EditAccess"},
		// 회귀 가드: 조회 계열은 뷰어 접근을 유지해야 한다.
		{"ListIntegrations", `"", am\.(\w+)\(aH\.ListIntegrations\)`, "ViewAccess"},
		{"GetIntegration", `"/\{integrationId\}", am\.(\w+)\(aH\.GetIntegration\)`, "ViewAccess"},
		{"GetIntegrationConnectionStatus", `"/\{integrationId\}/connection_status", am\.(\w+)\(aH\.GetIntegrationConnectionStatus\)`, "ViewAccess"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := regexp.MustCompile(tc.pattern).FindStringSubmatch(string(src))
			if m == nil {
				t.Fatalf("registration for %s not found in http_handler.go", tc.name)
			}
			if m[1] != tc.want {
				t.Errorf("%s: registered with am.%s, want am.%s", tc.name, m[1], tc.want)
			}
		})
	}
}
