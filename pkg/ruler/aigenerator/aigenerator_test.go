package aigenerator

import (
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

func TestNew_DefaultsToLocal(t *testing.T) {
	gen, err := New(Config{Provider: ""})
	require.NoError(t, err)
	var _ ruletypes.AIStrategyGenerator = gen
}

func TestNew_MockRequiresDir(t *testing.T) {
	_, err := New(Config{Provider: "mock", MockFixtureDir: ""})
	require.Error(t, err)
}

func TestNew_MockWithEmptyDir(t *testing.T) {
	gen, err := New(Config{Provider: "mock", MockFixtureDir: t.TempDir()})
	require.NoError(t, err)
	require.NotNil(t, gen)
}

func TestNew_UnknownProviderErrors(t *testing.T) {
	_, err := New(Config{Provider: "bogus"})
	require.Error(t, err)
}

func TestNew_LLMClaudeAPIRequiresKey(t *testing.T) {
	_, err := New(Config{Provider: "llm", LLMProvider: "claude", LLMTransport: "api", LLMAPIKey: ""})
	require.Error(t, err)
	require.Contains(t, err.Error(), "apiKey")
}

func TestNew_LLMClaudeAPIHappy(t *testing.T) {
	gen, err := New(Config{Provider: "llm", LLMProvider: "claude", LLMTransport: "api", LLMAPIKey: "k", LLMTimeoutSeconds: 5})
	require.NoError(t, err)
	require.NotNil(t, gen)
}

func TestNew_LLMClaudeCLIHappy(t *testing.T) {
	gen, err := New(Config{Provider: "llm", LLMProvider: "claude", LLMTransport: "cli"})
	require.NoError(t, err)
	require.NotNil(t, gen)
}

func TestNew_LLMCodexAPIRequiresKey(t *testing.T) {
	_, err := New(Config{Provider: "llm", LLMProvider: "codex", LLMTransport: "api"})
	require.Error(t, err)
}

func TestNew_LLMCodexAPIHappy(t *testing.T) {
	gen, err := New(Config{Provider: "llm", LLMProvider: "codex", LLMTransport: "api", LLMAPIKey: "k"})
	require.NoError(t, err)
	require.NotNil(t, gen)
}

func TestNew_LLMCodexCLIHappy(t *testing.T) {
	gen, err := New(Config{Provider: "llm", LLMProvider: "codex", LLMTransport: "cli"})
	require.NoError(t, err)
	require.NotNil(t, gen)
}

func TestNew_LLMUnknownProvider(t *testing.T) {
	_, err := New(Config{Provider: "llm", LLMProvider: "bogus", LLMTransport: "api"})
	require.Error(t, err)
}

func TestNew_LLMMissingTransport(t *testing.T) {
	_, err := New(Config{Provider: "llm", LLMProvider: "claude"})
	require.Error(t, err)
}

func TestNew_LLMUnknownTransport(t *testing.T) {
	_, err := New(Config{Provider: "llm", LLMProvider: "claude", LLMTransport: "smoke"})
	require.Error(t, err)
}

func TestNew_ClaudeCLIWithOAuthToken(t *testing.T) {
	gen, err := New(Config{
		Provider:      "llm",
		LLMProvider:   "claude",
		LLMTransport:  "cli",
		LLMOAuthToken: "tok-1",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if gen == nil {
		t.Fatalf("nil generator")
	}
}
