package claudecli

import (
	"strings"
	"testing"
)

func TestProvider_EnvContainsOAuthTokenWhenSet(t *testing.T) {
	p := New(
		WithBinary("/bin/true"),
		WithOAuthToken("tok-abc"),
	)
	env := p.commandEnv([]string{"PATH=/usr/bin"})
	found := false
	for _, e := range env {
		if e == "CLAUDE_CODE_OAUTH_TOKEN=tok-abc" {
			found = true
		}
	}
	if !found {
		t.Fatalf("CLAUDE_CODE_OAUTH_TOKEN not injected: %v", env)
	}
}

func TestProvider_NoOAuthEnvWhenEmpty(t *testing.T) {
	p := New(WithBinary("/bin/true"))
	for _, e := range p.commandEnv([]string{"PATH=/usr/bin"}) {
		if strings.HasPrefix(e, "CLAUDE_CODE_OAUTH_TOKEN=") {
			t.Fatalf("unexpected env: %s", e)
		}
	}
}

func TestProvider_OAuthTokenOverridesInheritedEnv(t *testing.T) {
	p := New(WithOAuthToken("override"))
	count := 0
	for _, e := range p.commandEnv([]string{"CLAUDE_CODE_OAUTH_TOKEN=stale", "PATH=/usr/bin"}) {
		if strings.HasPrefix(e, "CLAUDE_CODE_OAUTH_TOKEN=") {
			count++
			if e != "CLAUDE_CODE_OAUTH_TOKEN=override" {
				t.Fatalf("wrong value: %s", e)
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one CLAUDE_CODE_OAUTH_TOKEN entry, got %d", count)
	}
}
