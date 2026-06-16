// Package aigenerator exposes the AIStrategyGenerator factory. Production
// code injects whatever implementation the factory returns; tests can also
// construct generators directly from the sub-packages.
package aigenerator

import (
	"fmt"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/llmaigenerator"
	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/llmaigenerator/claudeapi"
	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/llmaigenerator/claudecli"
	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/llmaigenerator/codexapi"
	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/llmaigenerator/codexcli"
	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/localaigenerator"
	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/mockaigenerator"
	"github.com/SigNoz/signoz/pkg/ruler/runbookdrafter/llmrunbookdrafter"
	"github.com/SigNoz/signoz/pkg/ruler/runbookdrafter/mockrunbookdrafter"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// Config selects the generator implementation.
//
// Provider:
//
//	""        → local (default)
//	"local"   → local
//	"mock"    → mockaigenerator (requires MockFixtureDir)
//	"llm"     → llmaigenerator (requires LLMProvider + LLMTransport + at
//	            minimum the relevant API key for api transports)
//
// LLMProvider (only honored when Provider="llm"):
//
//	"claude"  → use claudeapi or claudecli
//	"codex"   → use codexapi or codexcli
//
// LLMTransport (only honored when Provider="llm"):
//
//	"api"  → HTTP API client (requires API key env var)
//	"cli"  → shell out to local CLI binary (must be authenticated already)
//
// LLMModel: optional, defaults to the provider+transport package default.
// LLMTimeoutSeconds: optional, defaults to 15.
// LLMAPIKey: required when LLMTransport="api". Caller is expected to
//
//	forward ANTHROPIC_API_KEY (claude) or OPENAI_API_KEY (codex) into this
//	field. Empty values when transport="api" cause New to error.
//
// LLMOAuthToken: optional, only honored when LLMTransport="cli". Passed to
//
//	the cli Provider via WithOAuthToken; the provider injects it into the
//	child process env (CLAUDE_CODE_OAUTH_TOKEN for claude, OPENAI_API_KEY for
//	codex). Empty values are ignored — the child inherits whatever env the
//	server has.
//
// LLMBinary: optional, defaults to "claude" or "codex".
type Config struct {
	Provider          string
	MockFixtureDir    string
	LLMProvider       string
	LLMTransport      string
	LLMModel          string
	LLMTimeoutSeconds int
	LLMAPIKey         string
	LLMOAuthToken     string
	LLMBinary         string
	// LLMEndpoint optionally overrides the provider's default HTTP endpoint
	// (only honored for LLMTransport="api"). Used to point the api transports
	// at a mock server (e.g. wiremock) in tests; empty keeps the provider
	// default (api.anthropic.com / api.openai.com).
	LLMEndpoint string
}

// New constructs the AIStrategyGenerator selected by cfg.Provider.
func New(cfg Config) (ruletypes.AIStrategyGenerator, error) {
	switch cfg.Provider {
	case "", "local":
		return localaigenerator.New(), nil
	case "mock":
		if cfg.MockFixtureDir == "" {
			return nil, fmt.Errorf("aigenerator: provider=mock requires MockFixtureDir")
		}
		return mockaigenerator.New(cfg.MockFixtureDir, localaigenerator.New())
	case "llm":
		return newLLM(cfg)
	default:
		return nil, fmt.Errorf("aigenerator: unknown provider %q", cfg.Provider)
	}
}

// DefaultCLITimeout is the per-call deadline used for LLMTransport="cli" when
// the config leaves LLMTimeoutSeconds unset (<= 0). CLI transports boot node and
// initialize the agent before the model call even starts, which routinely
// exceeds the 15s api default and gets the child process killed mid-run; give
// them a roomier default. API transports keep llmaigenerator.DefaultTimeout.
const DefaultCLITimeout = 120 * time.Second

func newLLM(cfg Config) (ruletypes.AIStrategyGenerator, error) {
	timeout := time.Duration(cfg.LLMTimeoutSeconds) * time.Second
	// llmaigenerator.New applies its own (api) default when timeout <= 0; CLI
	// needs a longer floor, applied here before that default kicks in.
	if cfg.LLMTimeoutSeconds <= 0 && cfg.LLMTransport == "cli" {
		timeout = DefaultCLITimeout
	}

	provider, err := buildLLMProvider(cfg)
	if err != nil {
		return nil, err
	}
	model := cfg.LLMModel
	if model == "" {
		model = defaultModelFor(cfg.LLMProvider)
	}
	return llmaigenerator.New(provider, model, timeout), nil
}

func buildLLMProvider(cfg Config) (llmaigenerator.Provider, error) {
	switch cfg.LLMProvider {
	case "claude":
		switch cfg.LLMTransport {
		case "api":
			opts := []claudeapi.Option{}
			if cfg.LLMModel != "" {
				opts = append(opts, claudeapi.WithModel(cfg.LLMModel))
			}
			if cfg.LLMEndpoint != "" {
				opts = append(opts, claudeapi.WithEndpoint(cfg.LLMEndpoint))
			}
			return claudeapi.New(cfg.LLMAPIKey, opts...)
		case "cli":
			opts := []claudecli.Option{}
			if cfg.LLMModel != "" {
				opts = append(opts, claudecli.WithModel(cfg.LLMModel))
			}
			if cfg.LLMBinary != "" {
				opts = append(opts, claudecli.WithBinary(cfg.LLMBinary))
			}
			if cfg.LLMOAuthToken != "" {
				opts = append(opts, claudecli.WithOAuthToken(cfg.LLMOAuthToken))
			}
			return claudecli.New(opts...), nil
		default:
			return nil, fmt.Errorf("aigenerator: provider=llm/claude requires LLMTransport (api|cli); got %q", cfg.LLMTransport)
		}
	case "codex":
		switch cfg.LLMTransport {
		case "api":
			opts := []codexapi.Option{}
			if cfg.LLMModel != "" {
				opts = append(opts, codexapi.WithModel(cfg.LLMModel))
			}
			if cfg.LLMEndpoint != "" {
				opts = append(opts, codexapi.WithEndpoint(cfg.LLMEndpoint))
			}
			return codexapi.New(cfg.LLMAPIKey, opts...)
		case "cli":
			opts := []codexcli.Option{}
			if cfg.LLMModel != "" {
				opts = append(opts, codexcli.WithModel(cfg.LLMModel))
			}
			if cfg.LLMBinary != "" {
				opts = append(opts, codexcli.WithBinary(cfg.LLMBinary))
			}
			if cfg.LLMOAuthToken != "" {
				opts = append(opts, codexcli.WithOAuthToken(cfg.LLMOAuthToken))
			}
			return codexcli.New(opts...), nil
		default:
			return nil, fmt.Errorf("aigenerator: provider=llm/codex requires LLMTransport (api|cli); got %q", cfg.LLMTransport)
		}
	default:
		return nil, fmt.Errorf("aigenerator: provider=llm requires LLMProvider (claude|codex); got %q", cfg.LLMProvider)
	}
}

func defaultModelFor(llmProvider string) string {
	switch llmProvider {
	case "claude":
		return claudeapi.DefaultModel
	case "codex":
		return codexapi.DefaultModel
	default:
		return ""
	}
}

// NewRunbookDrafter builds a RunbookDrafter from cfg. When Provider="llm" and
// a valid LLM provider/transport are configured it returns an LLM drafter;
// otherwise it returns a mock drafter so the handler never receives nil.
func NewRunbookDrafter(cfg Config) ruletypes.RunbookDrafter {
	if cfg.Provider == "llm" {
		provider, err := buildLLMProvider(cfg)
		if err == nil && provider != nil {
			model := cfg.LLMModel
			if model == "" {
				model = defaultModelFor(cfg.LLMProvider)
			}
			return llmrunbookdrafter.New(provider, model)
		}
	}
	return mockrunbookdrafter.New(ruletypes.Runbook{Title: "no-llm-configured"})
}
