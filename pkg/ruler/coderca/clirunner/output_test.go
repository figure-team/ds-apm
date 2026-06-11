package clirunner

import "testing"

func TestReconstructCodexTextPassthrough(t *testing.T) {
	// Non-JSONL (claude-style plain text) is returned unchanged.
	plain := "Here is the result\n```json\n{\"root_cause\":\"x\"}\n```\n"
	if got := reconstructCodexText(plain); got != plain {
		t.Errorf("passthrough failed: got %q, want %q", got, plain)
	}
}

func TestReconstructCodexTextJoinsAgentMessages(t *testing.T) {
	jsonl := `{"type":"item.started","item":{"type":"reasoning"}}` + "\n" +
		`{"type":"item.completed","item":{"type":"agent_message","text":"first "}}` + "\n" +
		`{"type":"item.completed","item":{"type":"agent_message","text":"second"}}`
	got := reconstructCodexText(jsonl)
	if got != "first second" {
		t.Errorf("reconstructed = %q, want %q", got, "first second")
	}
}

func TestReconstructCodexTextJSONLWithoutTextKeepsRaw(t *testing.T) {
	// Valid JSONL but no recognizable text field → keep raw so audit/parse can
	// still attempt it.
	jsonl := `{"type":"item.started","item":{"type":"reasoning"}}`
	if got := reconstructCodexText(jsonl); got != jsonl {
		t.Errorf("got %q, want raw %q", got, jsonl)
	}
}
