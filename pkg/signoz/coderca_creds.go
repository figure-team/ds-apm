package signoz

import (
	"context"
	"strings"

	"github.com/SigNoz/signoz/pkg/ruler/aiconfigstore/secretbox"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/clirunner"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// codercaCredsResolver implements codercaengine.CredsResolver: it resolves a
// run's agent CLI credentials from the SAME per-org AI config the AI module
// uses, so an on-demand Code RCA reuses the token configured in the UI instead
// of a separate DS_APM_CODERCA_AUTH_TOKEN env var.
//
// It returns ok=false (engine falls back to its env-derived Config) unless the
// stored config is a CLI-transport LLM credential — the only shape the coderca
// agent CLI can consume.
type codercaCredsResolver struct {
	store  ruletypes.AIConfigStore
	cipher *secretbox.Cipher
}

func newCodercaCredsResolver(store ruletypes.AIConfigStore, cipher *secretbox.Cipher) codercaCredsResolver {
	return codercaCredsResolver{store: store, cipher: cipher}
}

func (r codercaCredsResolver) Resolve(ctx context.Context, orgID string) (clirunner.Agent, string, string, bool) {
	cfg, err := r.store.Get(ctx, orgID, r.cipher.DecryptFunc())
	if err != nil {
		return "", "", "", false
	}
	// The coderca agent shells out to a CLI, so only a CLI-transport LLM
	// credential with a token is usable here.
	if cfg.Provider != "llm" || cfg.Transport != "cli" || strings.TrimSpace(cfg.OAuthToken) == "" {
		return "", "", "", false
	}
	switch cfg.LLMProvider {
	case string(clirunner.AgentClaude), string(clirunner.AgentCodex):
		return clirunner.Agent(cfg.LLMProvider), cfg.Model, cfg.OAuthToken, true
	default:
		return "", "", "", false
	}
}
