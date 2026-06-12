package ruletypes

import "testing"

func TestCodebaseServiceMapRoundTrip(t *testing.T) {
	m := CodebaseServiceMap{OrgID: "o1", ServiceName: "payments", RepoID: "r1", Subpath: "services/pay"}

	s, err := FromDomainCodebaseServiceMap(m)
	if err != nil {
		t.Fatalf("FromDomainCodebaseServiceMap: %v", err)
	}
	got := s.ToDomain()
	if got != m {
		t.Errorf("round trip = %+v, want %+v", got, m)
	}
}

func TestFromDomainCodebaseServiceMapValidates(t *testing.T) {
	tests := []struct {
		name string
		m    CodebaseServiceMap
	}{
		{"missing org", CodebaseServiceMap{ServiceName: "s", RepoID: "r"}},
		{"missing service", CodebaseServiceMap{OrgID: "o", RepoID: "r"}},
		{"missing repo", CodebaseServiceMap{OrgID: "o", ServiceName: "s"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := FromDomainCodebaseServiceMap(tc.m); err == nil {
				t.Errorf("expected validation error for %+v", tc.m)
			}
		})
	}
}

func TestFromDomainCodebaseServiceMapValidInput(t *testing.T) {
	if _, err := FromDomainCodebaseServiceMap(CodebaseServiceMap{OrgID: "o", ServiceName: "s", RepoID: "r"}); err != nil {
		t.Errorf("valid mapping rejected: %v", err)
	}
}
