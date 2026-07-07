package remediation

import (
	"os/exec"
	"os/user"
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

func fakeProbe(tools map[string]bool, users map[string]bool, env map[string]string) toolProbe {
	return toolProbe{
		lookPath: func(name string) (string, error) {
			if tools[name] {
				return "/usr/bin/" + name, nil
			}
			return "", exec.ErrNotFound
		},
		lookupUser: func(name string) (*user.User, error) {
			if users[name] {
				return &user.User{Username: name}, nil
			}
			return nil, user.UnknownUserError(name)
		},
		env: func(k string) string { return env[k] },
	}
}

func TestResolveLocalSandbox_RunbookWithPrlimit(t *testing.T) {
	p := fakeProbe(map[string]bool{"prlimit": true, "setpriv": true}, map[string]bool{"remed": true}, nil)
	prefix, profile, err := resolveLocalSandbox(ruletypes.RemediationSourceRunbook, p)
	if err != nil || profile != sandboxProfilePrlimit {
		t.Fatalf("want prlimit profile, got profile=%q err=%v", profile, err)
	}
	if prefix[0] != "/usr/bin/prlimit" {
		t.Fatalf("prefix must start with prlimit: %v", prefix)
	}
}

func TestResolveLocalSandbox_RunbookWithoutTools_BestEffortNone(t *testing.T) {
	p := fakeProbe(nil, nil, nil)
	prefix, profile, err := resolveLocalSandbox(ruletypes.RemediationSourceRunbook, p)
	if err != nil || profile != sandboxProfileNone || prefix != nil {
		t.Fatalf("runbook은 도구 부재 시 무샌드박스 best-effort: prefix=%v profile=%q err=%v", prefix, profile, err)
	}
}

func TestResolveLocalSandbox_LLMRestricted(t *testing.T) {
	p := fakeProbe(map[string]bool{"prlimit": true, "setpriv": true}, map[string]bool{"remed": true}, nil)
	prefix, profile, err := resolveLocalSandbox(ruletypes.RemediationSourceLLMGenerated, p)
	if err != nil || profile != sandboxProfileRestricted {
		t.Fatalf("want restricted profile, got profile=%q err=%v", profile, err)
	}
	joined := strings.Join(prefix, " ")
	for _, want := range []string{"/usr/bin/setpriv", "--reuid=remed", "--no-new-privs", "/usr/bin/prlimit"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("prefix missing %q: %v", want, prefix)
		}
	}
}

func TestResolveLocalSandbox_LLMFailClosed(t *testing.T) {
	cases := []struct {
		name  string
		tools map[string]bool
		users map[string]bool
	}{
		{"setpriv 없음", map[string]bool{"prlimit": true}, map[string]bool{"remed": true}},
		{"prlimit 없음", map[string]bool{"setpriv": true}, map[string]bool{"remed": true}},
		{"remed 계정 없음", map[string]bool{"setpriv": true, "prlimit": true}, nil},
	}
	for _, c := range cases {
		p := fakeProbe(c.tools, c.users, nil)
		if _, _, err := resolveLocalSandbox(ruletypes.RemediationSourceLLMGenerated, p); err == nil {
			t.Errorf("%s: llm 소스는 fail-closed여야 함", c.name)
		}
	}
}

func TestResolveLocalSandbox_EnvOff(t *testing.T) {
	p := fakeProbe(nil, nil, map[string]string{"DS_APM_REMED_SANDBOX": "off"})
	prefix, profile, err := resolveLocalSandbox(ruletypes.RemediationSourceLLMGenerated, p)
	if err != nil || profile != sandboxProfileOff || prefix != nil {
		t.Fatalf("명시적 off는 llm 포함 전체 해제: prefix=%v profile=%q err=%v", prefix, profile, err)
	}
}
