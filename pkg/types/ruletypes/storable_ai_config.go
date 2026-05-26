package ruletypes

import (
	"fmt"
	"strings"

	"github.com/uptrace/bun"
)

type StorableAIConfig struct {
	bun.BaseModel `bun:"table:ds_ai_config"`

	OrgID            string `bun:"org_id,pk,notnull,type:text"`
	Provider         string `bun:"provider,notnull,type:text"`
	LLMProvider      string `bun:"llm_provider,notnull,default:'',type:text"`
	Transport        string `bun:"transport,notnull,default:'',type:text"`
	Model            string `bun:"model,notnull,default:'',type:text"`
	APIKeyCiphertext     string `bun:"api_key_ciphertext,notnull,default:'',type:text"`
	OAuthTokenCiphertext string `bun:"oauth_token_ciphertext,notnull,default:'',type:text"`
	BinaryPath           string `bun:"binary_path,notnull,default:'',type:text"`
	TimeoutSeconds   int    `bun:"timeout_seconds,notnull,default:0,type:integer"`
	UpdatedAt        string `bun:"updated_at,notnull,type:text"`
}

// FromDomainAIConfig serializes the API-key plaintext via the provided
// encryptor and returns the storable form. orgID is required.
func FromDomainAIConfig(cfg AIConfig, encrypt func(string) (string, error)) (*StorableAIConfig, error) {
	if strings.TrimSpace(cfg.OrgID) == "" {
		return nil, fmt.Errorf("storable ai config: orgID must not be empty")
	}

	apiCT, err := encrypt(cfg.APIKey)
	if err != nil {
		return nil, fmt.Errorf("storable ai config: encrypt api key: %w", err)
	}

	oauthCT, err := encrypt(cfg.OAuthToken)
	if err != nil {
		return nil, fmt.Errorf("storable ai config: encrypt oauth token: %w", err)
	}

	return &StorableAIConfig{
		OrgID:                cfg.OrgID,
		Provider:             cfg.Provider,
		LLMProvider:          cfg.LLMProvider,
		Transport:            cfg.Transport,
		Model:                cfg.Model,
		APIKeyCiphertext:     apiCT,
		OAuthTokenCiphertext: oauthCT,
		BinaryPath:           cfg.BinaryPath,
		TimeoutSeconds:       cfg.TimeoutSeconds,
		UpdatedAt:            cfg.UpdatedAt,
	}, nil
}

// ToDomain decrypts api_key_ciphertext via the provided decryptor. The
// returned AIConfig contains the plaintext APIKey; callers must scrub
// it before returning over the network.
func (s *StorableAIConfig) ToDomain(decrypt func(string) (string, error)) (AIConfig, error) {
	apiKey, err := decrypt(s.APIKeyCiphertext)
	if err != nil {
		return AIConfig{}, fmt.Errorf("storable ai config: decrypt api key: %w", err)
	}

	oauth, err := decrypt(s.OAuthTokenCiphertext)
	if err != nil {
		return AIConfig{}, fmt.Errorf("storable ai config: decrypt oauth token: %w", err)
	}

	return AIConfig{
		ContractVersion: AIConfigContractVersion,
		OrgID:           s.OrgID,
		Provider:        s.Provider,
		LLMProvider:     s.LLMProvider,
		Transport:       s.Transport,
		Model:           s.Model,
		APIKey:          apiKey,
		OAuthToken:      oauth,
		BinaryPath:      s.BinaryPath,
		TimeoutSeconds:  s.TimeoutSeconds,
		UpdatedAt:       s.UpdatedAt,
	}, nil
}
