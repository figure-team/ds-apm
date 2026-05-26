package ruletypes

import (
	"strings"
	"testing"
)

func validBase() AIConfig {
	return AIConfig{
		ContractVersion: AIConfigContractVersion,
		OrgID:           "org-1",
		Provider:        "llm",
		LLMProvider:     "claude",
		Transport:       "api",
		APIKey:          "k",
	}
}

func TestValidateAIConfig_AcceptsOAuthTokenForCLI(t *testing.T) {
	cfg := validBase()
	cfg.Transport = "cli"
	cfg.APIKey = ""
	cfg.OAuthToken = "tok"
	if err := ValidateAIConfig(cfg); err != nil {
		t.Fatalf("expected valid; got %v", err)
	}
}

func TestValidateAIConfig_OAuthTokenAllowedEmpty(t *testing.T) {
	// Empty token is allowed at validation time; runtime decides if
	// missing-token is fatal (so user can save partial config and login later).
	cfg := validBase()
	cfg.Transport = "cli"
	cfg.APIKey = ""
	cfg.OAuthToken = ""
	if err := ValidateAIConfig(cfg); err != nil {
		t.Fatalf("expected valid; got %v", err)
	}
}

func TestValidateAIConfig_RejectsOAuthTokenOnAPITransport(t *testing.T) {
	cfg := validBase()
	cfg.Transport = "api"
	cfg.OAuthToken = "tok-but-on-api"
	err := ValidateAIConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "oauthToken") {
		t.Fatalf("expected oauthToken validation error; got %v", err)
	}
}

func TestValidateAIConfig_RejectsOAuthTokenOnEmptyTransport(t *testing.T) {
	cfg := validBase()
	cfg.Transport = ""
	cfg.OAuthToken = "tok"
	// Provider=llm with empty transport already errors on transport itself;
	// the test asserts the oauthToken rule also flags the case.
	err := ValidateAIConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "oauthToken") {
		t.Fatalf("expected oauthToken validation error; got %v", err)
	}
}

func TestValidateAIConfig_RejectsOAuthTokenWhenProviderNotLLM(t *testing.T) {
	cfg := validBase()
	cfg.Provider = "local"
	cfg.LLMProvider = ""
	cfg.Transport = ""
	cfg.APIKey = ""
	cfg.OAuthToken = "tok"
	err := ValidateAIConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "oauthToken") {
		t.Fatalf("expected oauthToken validation error; got %v", err)
	}
}

func TestValidateAIConfig_RejectsAPIKeyOverMaxLen(t *testing.T) {
	cfg := validBase()
	cfg.APIKey = strings.Repeat("k", MaxSecretLen+1)
	err := ValidateAIConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "apiKey") {
		t.Fatalf("expected apiKey length error; got %v", err)
	}
}

func TestValidateAIConfig_AcceptsAPIKeyAtMaxLen(t *testing.T) {
	cfg := validBase()
	cfg.APIKey = strings.Repeat("k", MaxSecretLen)
	if err := ValidateAIConfig(cfg); err != nil {
		t.Fatalf("expected valid at exact max len; got %v", err)
	}
}

func TestValidateAIConfig_RejectsOAuthTokenOverMaxLen(t *testing.T) {
	cfg := validBase()
	cfg.Transport = "cli"
	cfg.APIKey = ""
	cfg.OAuthToken = strings.Repeat("t", MaxSecretLen+1)
	err := ValidateAIConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "oauthToken") {
		t.Fatalf("expected oauthToken length error; got %v", err)
	}
}

func TestValidateAIConfig_RejectsAPIKeyWithNewline(t *testing.T) {
	cfg := validBase()
	cfg.APIKey = "sk-ant-\n-leaked"
	err := ValidateAIConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "apiKey") {
		t.Fatalf("expected apiKey newline error; got %v", err)
	}
}

func TestValidateAIConfig_RejectsOAuthTokenWithCR(t *testing.T) {
	cfg := validBase()
	cfg.Transport = "cli"
	cfg.APIKey = ""
	cfg.OAuthToken = "sk-ant-oat01-x\r-y"
	err := ValidateAIConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "oauthToken") {
		t.Fatalf("expected oauthToken CR error; got %v", err)
	}
}

func TestValidateAIConfig_AcceptsJSONOAuthTokenForCodexCLI(t *testing.T) {
	// Codex ChatGPT-subscription users paste the full ~/.codex/auth.json
	// content. Newlines must be permitted because the content is written to
	// a file at runtime, not env-injected.
	cfg := validBase()
	cfg.LLMProvider = "codex"
	cfg.Transport = "cli"
	cfg.APIKey = ""
	cfg.OAuthToken = "{\n  \"auth_mode\": \"chatgpt\",\n  \"tokens\": {\"access_token\": \"a\"}\n}"
	if err := ValidateAIConfig(cfg); err != nil {
		t.Fatalf("expected valid JSON paste; got %v", err)
	}
}

func TestValidateAIConfig_RejectsMalformedJSONOAuthToken(t *testing.T) {
	cfg := validBase()
	cfg.LLMProvider = "codex"
	cfg.Transport = "cli"
	cfg.APIKey = ""
	cfg.OAuthToken = `{"unterminated`
	err := ValidateAIConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "failed to parse") {
		t.Fatalf("expected JSON parse error; got %v", err)
	}
}

func TestValidateAIConfig_AcceptsSentinelValue(t *testing.T) {
	// The handler validates BEFORE substituting <unchanged>, so the sentinel
	// itself must pass length + control-char checks.
	cfg := validBase()
	cfg.APIKey = "<unchanged>"
	if err := ValidateAIConfig(cfg); err != nil {
		t.Fatalf("sentinel must not trip apiKey length/CTRL rules: %v", err)
	}

	cfg = validBase()
	cfg.Transport = "cli"
	cfg.APIKey = ""
	cfg.OAuthToken = "<unchanged>"
	if err := ValidateAIConfig(cfg); err != nil {
		t.Fatalf("sentinel must not trip oauthToken length/CTRL rules: %v", err)
	}
}
