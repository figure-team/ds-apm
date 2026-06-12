package clirunner

import (
	"encoding/json"
	"strings"
)

// reconstructCodexText turns codex `exec --json` stdout (a stream of JSONL
// events) back into the agent's message text so the shared RCA parser can find
// the fenced ```json block. It collects string values under the agent-message
// text keys, in line order. If stdout is not JSONL (e.g. claude plain text), or
// no text keys are found, it returns raw unchanged so nothing is lost.
//
// The exact codex event schema is reconciled against the real binary at
// integration; this reducer is deliberately tolerant of the surrounding shape.
//
func reconstructCodexText(raw string) string {
	var parts []string
	sawJSON := false
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var v any
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			continue // not a JSON event line
		}
		sawJSON = true
		collectTextValues(v, &parts)
	}
	if !sawJSON || len(parts) == 0 {
		return raw // not JSONL, or no recognizable text → keep raw
	}
	return strings.Join(parts, "")
}

// codexTextKeys are the JSON object keys whose string values carry agent
// message text in codex's event stream.
var codexTextKeys = map[string]struct{}{
	"text":               {},
	"message":            {},
	"last_agent_message": {},
}

// collectTextValues walks a decoded JSON value and appends, in encounter order,
// every string found under a codexTextKeys key.
func collectTextValues(v any, out *[]string) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if s, ok := val.(string); ok {
				if _, want := codexTextKeys[k]; want {
					*out = append(*out, s)
				}
			}
			collectTextValues(val, out)
		}
	case []any:
		for _, e := range t {
			collectTextValues(e, out)
		}
	}
}
