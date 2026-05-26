package ruletypes

import (
	"encoding/json"
	"fmt"
	"strings"
)

const AIConfigContractVersion = "ds.ai_config.v1"

// MaxSecretLen caps the byte length of apiKey and oauthToken fields. Bounds
// memory usage, ciphertext column size, and guards against pathological
// pastes. Real Anthropic/OpenAI credentials are < 200 bytes; 8 KiB is a
// generous ceiling that still rejects accidental JSON-blob pastes.
const MaxSecretLen = 8192

// AIConfig is the per-tenant AI module configuration. Stored encrypted at
// rest (api_key, oauth_token); plaintext is only handled in-process.
type AIConfig struct {
	ContractVersion string `json:"contractVersion"`
	OrgID           string `json:"orgId"`
	Provider        string `json:"provider"`       // "" | "local" | "mock" | "llm"
	LLMProvider     string `json:"llmProvider"`    // "" | "claude" | "codex" (llm only)
	Transport       string `json:"transport"`      // "" | "api" | "cli" (llm only)
	Model           string `json:"model"`          // optional override
	APIKey          string `json:"apiKey"`         // plaintext — DO NOT serialize to clients except via dedicated endpoint
	OAuthToken      string `json:"oauthToken"`     // plaintext — used only for transport=cli; same scrubbing rules as APIKey
	BinaryPath      string `json:"binaryPath"`     // optional cli binary path
	TimeoutSeconds  int    `json:"timeoutSeconds"` // 0 = default
	UpdatedAt       string `json:"updatedAt"`      // RFC3339
}

var allowedProviders = map[string]struct{}{
	"":      {},
	"local": {},
	"mock":  {},
	"llm":   {},
}

var allowedTransports = map[string]struct{}{
	"":    {},
	"api": {},
	"cli": {},
}

var allowedLLMProviders = map[string]struct{}{
	"claude": {},
	"codex":  {},
}

// ValidateAIConfig validates a per-tenant AI configuration.
func ValidateAIConfig(cfg AIConfig) error {
	var errs []string

	if strings.TrimSpace(cfg.ContractVersion) != AIConfigContractVersion {
		errs = append(errs, fmt.Sprintf("contractVersion: must be %q, got %q", AIConfigContractVersion, cfg.ContractVersion))
	}
	if strings.TrimSpace(cfg.OrgID) == "" {
		errs = append(errs, "orgId: must not be empty")
	}
	if _, ok := allowedProviders[cfg.Provider]; !ok {
		errs = append(errs, fmt.Sprintf("provider: %q is not allowed (allowed: \"\", \"local\", \"mock\", \"llm\")", cfg.Provider))
	}
	if cfg.Provider == "llm" {
		if _, ok := allowedLLMProviders[cfg.LLMProvider]; !ok {
			errs = append(errs, fmt.Sprintf("llmProvider: %q is not allowed for provider=llm (allowed: \"claude\", \"codex\")", cfg.LLMProvider))
		}
		if _, ok := allowedTransports[cfg.Transport]; !ok {
			errs = append(errs, fmt.Sprintf("transport: %q is not allowed for provider=llm (allowed: \"api\", \"cli\")", cfg.Transport))
		}
		if cfg.Transport == "" {
			errs = append(errs, "transport: must be \"api\" or \"cli\" when provider=llm")
		}
	}

	if cfg.OAuthToken != "" && (cfg.Provider != "llm" || cfg.Transport != "cli") {
		errs = append(errs, "oauthToken: only allowed when provider=\"llm\" and transport=\"cli\"")
	}

	if err := validateAPIKey(cfg.APIKey); err != "" {
		errs = append(errs, err)
	}
	if err := validateOAuthToken(cfg.OAuthToken); err != "" {
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		return nil
	}
	combined := strings.Join(errs, "; ")
	return fmt.Errorf("ai config validation: %s", combined)
}

// validateAPIKey checks an apiKey value. APIKey is always injected as an env
// var (ANTHROPIC_API_KEY / OPENAI_API_KEY) so CR/LF must be rejected.
func validateAPIKey(value string) string {
	if len(value) > MaxSecretLen {
		return fmt.Sprintf("apiKey: exceeds %d-byte limit (got %d)", MaxSecretLen, len(value))
	}
	if strings.ContainsAny(value, "\r\n") {
		return "apiKey: must not contain CR or LF"
	}
	return ""
}

// validateOAuthToken checks an oauthToken value. The token has two shapes:
//   - Raw OAuth/API key string: same constraints as APIKey (no CR/LF, since
//     it's env-injected).
//   - JSON object (Codex ChatGPT subscription's auth.json): newlines are
//     permitted because the content is written to a file, not env-injected.
//     Must parse as valid JSON to avoid materializing a corrupt auth.json.
func validateOAuthToken(value string) string {
	if len(value) > MaxSecretLen {
		return fmt.Sprintf("oauthToken: exceeds %d-byte limit (got %d)", MaxSecretLen, len(value))
	}
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "{") {
		var probe map[string]any
		if err := json.Unmarshal([]byte(trimmed), &probe); err != nil {
			return fmt.Sprintf("oauthToken: looked like JSON but failed to parse: %v", err)
		}
		return ""
	}
	if strings.ContainsAny(value, "\r\n") {
		return "oauthToken: must not contain CR or LF"
	}
	return ""
}
