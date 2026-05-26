package ruletypes

import "testing"

// reverseEncrypt encrypts by reversing the string, decrypts by reversing back.
// Lets us verify the encrypt function is actually invoked on each field.
func reverseEncrypt(s string) (string, error) {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r), nil
}

func TestStorableAIConfig_RoundTrip_BothSecrets(t *testing.T) {
	cfg := AIConfig{
		ContractVersion: AIConfigContractVersion,
		OrgID:           "org-1",
		Provider:        "llm",
		LLMProvider:     "claude",
		Transport:       "cli",
		Model:           "claude-sonnet-4-6",
		APIKey:          "api-secret",
		OAuthToken:      "oauth-secret",
		BinaryPath:      "/usr/local/bin/claude",
		TimeoutSeconds:  30,
		UpdatedAt:       "2026-05-21T00:00:00Z",
	}

	storable, err := FromDomainAIConfig(cfg, reverseEncrypt)
	if err != nil {
		t.Fatalf("FromDomainAIConfig: %v", err)
	}
	if storable.APIKeyCiphertext != "terces-ipa" {
		t.Fatalf("APIKeyCiphertext not encrypted: %q", storable.APIKeyCiphertext)
	}
	if storable.OAuthTokenCiphertext != "terces-htuao" {
		t.Fatalf("OAuthTokenCiphertext not encrypted: %q", storable.OAuthTokenCiphertext)
	}

	restored, err := storable.ToDomain(reverseEncrypt) // reverse-of-reverse = identity
	if err != nil {
		t.Fatalf("ToDomain: %v", err)
	}
	if restored.APIKey != "api-secret" {
		t.Fatalf("APIKey not restored: %q", restored.APIKey)
	}
	if restored.OAuthToken != "oauth-secret" {
		t.Fatalf("OAuthToken not restored: %q", restored.OAuthToken)
	}
}
