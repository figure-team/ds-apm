# llmaigenerator

Real LLM-backed implementation of `ruletypes.AIStrategyGenerator`. Used as
an opt-in alternative to `localaigenerator` (deterministic fallback) and
`mockaigenerator` (JSON-driven canned responses).

## Quick start

The factory at `pkg/ruler/aigenerator` is the only entry point production
code should use. Set environment variables:

| Env | Default | Notes |
|---|---|---|
| `DS_APM_AI_GENERATOR` | `local` | set to `llm` to opt in |
| `DS_APM_LLM_PROVIDER` | — | `claude` or `codex` |
| `DS_APM_LLM_TRANSPORT` | — | `api` or `cli` |
| `DS_APM_LLM_MODEL` | provider default | e.g. `claude-sonnet-4-6`, `gpt-5` |
| `DS_APM_LLM_TIMEOUT_SECONDS` | `15` | per-call deadline |
| `DS_APM_LLM_BINARY` | `claude` / `codex` | for `cli` transport |
| `ANTHROPIC_API_KEY` | — | required when provider=claude + transport=api |
| `OPENAI_API_KEY` | — | required when provider=codex + transport=api |

Combinations supported: `claude/api`, `claude/cli`, `codex/api`, `codex/cli`.

## Architecture

```
llmaigenerator.Generator     ← satisfies ruletypes.AIStrategyGenerator
  ├─ prompt.Render(req)       ← deterministic system + user message
  ├─ Provider.Complete(...)   ← one of the four sub-packages
  └─ parse.Strategy(raw)      ← extracts JSON, builds AIStrategy, runs ValidateAIStrategy
```

Provider abstraction:

```go
type Provider interface {
    Complete(ctx context.Context, system, user string) (string, error)
}
```

Implementations:

| Package | Transport | Endpoint / Binary |
|---|---|---|
| `claudeapi` | HTTP | `POST https://api.anthropic.com/v1/messages` |
| `claudecli` | exec | `claude -p <user> --append-system-prompt <system> --model ...` |
| `codexapi` | HTTP | `POST https://api.openai.com/v1/chat/completions` |
| `codexcli` | exec | `codex exec --model <model> <combined-prompt>` |

## v0.1 limitations (TODO v0.2)

- No retry on transient failures (single attempt).
- No streaming.
- No PII redaction pass (`AIStrategyAudit.RedactionApplied` is set to `true`
  to satisfy validator, but no actual redactor runs).
- No quota tracking.
- Korean output only (`Language = "ko-KR"` hard-coded).
- The alertmanager dispatcher does not yet auto-call the generator —
  consumers must invoke via `PreviewAIStrategy` endpoint (see
  `docs/demo/2026-05-21-runbook.md`).

## Tests

All tests run offline:
- `httptest` stub servers for `claudeapi` / `codexapi`
- Temp-dir fake shell scripts for `claudecli` / `codexcli`
- No real API/CLI invocation in CI.

Run: `go test ./pkg/ruler/aigenerator/llmaigenerator/... -count=1`
