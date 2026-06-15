package coderca

import (
	"context"
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// stubTooling is a test double for the AgentTooling port so prompt tests stay
// independent of any concrete agent adapter.
type stubTooling struct{ directive string }

func (s stubTooling) ReadOnlyDirective() string { return s.directive }

var testTooling = stubTooling{directive: "read-only inspection directive (test)"}

func sampleContext() RCAContext {
	return RCAContext{
		OrgID:          "org1",
		Service:        "payments",
		Severity:       "critical",
		Environment:    "production",
		Fingerprint:    "fp-abc123",
		ErrorSignature: "alertname=High5xx|service.name=payments|severity=critical",
		BaselineCommit: "a1b2c3d4",
		Labels:         map[string]string{"alertname": "High5xx", "error_class": "PgTimeout"},
		Annotations:    map[string]string{"summary": "payments 5xx spike"},
	}
}

func TestBuildPromptSystemIsReadOnlyHITL(t *testing.T) {
	system, _ := BuildPrompt(sampleContext(), nil, testTooling)
	low := strings.ToLower(system)

	mustContain := []string{
		"root cause",   // (b) hypothesize a root cause
		"suggestion",   // (c) fix is a suggestion
		"confidence",   // (d) confidence
		"limitations",  // (d) limitations
		"baseline",     // (e) echo the analyzed baseline commit
		"read-only",    // explore-only, no writes
		"```json",      // output format the parser expects
	}
	for _, sub := range mustContain {
		if !strings.Contains(low, strings.ToLower(sub)) {
			t.Errorf("system prompt missing %q", sub)
		}
	}
	// HITL: the agent must be told never to modify files / apply the fix.
	if !strings.Contains(low, "do not") && !strings.Contains(low, "never") {
		t.Error("system prompt must forbid applying changes (HITL)")
	}
	// The json keys the parser consumes must be named so output is parseable.
	for _, key := range []string{"baseline_commit", "root_cause", "proposed_fix", "confidence", "limitations"} {
		if !strings.Contains(system, key) {
			t.Errorf("system prompt must name output key %q", key)
		}
	}
	// Must never instruct a permission bypass.
	if strings.Contains(low, "dangerously-skip-permissions") {
		t.Error("system prompt must not suggest skipping permissions")
	}
}

func TestBuildPromptUserCarriesErrorContext(t *testing.T) {
	_, user := BuildPrompt(sampleContext(), nil, testTooling)

	mustContain := []string{
		"payments",   // service
		"critical",   // severity
		"production", // environment
		"a1b2c3d4",   // baseline commit
		"alertname=High5xx|service.name=payments|severity=critical", // signature
		"PgTimeout",          // a label value
		"payments 5xx spike", // an annotation value
	}
	for _, sub := range mustContain {
		if !strings.Contains(user, sub) {
			t.Errorf("user prompt missing %q", sub)
		}
	}
}

func TestBuildPromptInjectsEvidenceWhenPresent(t *testing.T) {
	ev := []ruletypes.AIEvidenceRef{
		{RefID: "ev-1", Type: "log", Observation: "connection pool exhausted at 14:02"},
	}
	_, withEv := BuildPrompt(sampleContext(), ev, testTooling)
	if !strings.Contains(withEv, "connection pool exhausted at 14:02") {
		t.Errorf("user prompt did not inject evidence observation: %q", withEv)
	}

	_, noEv := BuildPrompt(sampleContext(), nil, testTooling)
	if strings.Contains(noEv, "connection pool exhausted at 14:02") {
		t.Error("evidence text leaked into a no-evidence prompt")
	}
}

func TestBuildPromptIsDeterministic(t *testing.T) {
	rc := sampleContext()
	s1, u1 := BuildPrompt(rc, nil, testTooling)
	s2, u2 := BuildPrompt(rc, nil, testTooling)
	if s1 != s2 || u1 != u2 {
		t.Error("BuildPrompt is not deterministic across calls")
	}
}

func TestBuildPromptInjectsToolingDirective(t *testing.T) {
	const marker = "MARKER-read-via-shell-OK"
	system, _ := BuildPrompt(sampleContext(), nil, stubTooling{directive: marker})
	if !strings.Contains(system, marker) {
		t.Errorf("system prompt did not inject the agent tooling directive; got:\n%s", system)
	}
	// The fixed task/output contract must still be present alongside the
	// injected directive.
	for _, sub := range []string{"```json", "baseline_commit", "read-only"} {
		if !strings.Contains(strings.ToLower(system), strings.ToLower(sub)) {
			t.Errorf("system prompt missing fixed section %q", sub)
		}
	}
}

func TestNoopEvidenceCollectorReturnsNothing(t *testing.T) {
	got, err := NoopEvidenceCollector{}.Collect(context.Background(), sampleContext())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("NoopEvidenceCollector returned %d refs, want 0", len(got))
	}
}
