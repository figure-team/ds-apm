package clirunner

// reconstructCodexText turns codex `exec --json` stdout (a stream of JSONL
// events) back into the agent's message text so the shared RCA parser can find
// the fenced ```json block. It collects string values under the agent-message
// text keys, in line order. If stdout is not JSONL (e.g. claude plain text), or
// no text keys are found, it returns raw unchanged so nothing is lost.
//
// The exact codex event schema is reconciled against the real binary at
// integration; this reducer is deliberately tolerant of the surrounding shape.
//
// A3 STUB: returns "" → reconstruction + passthrough assertions fail (RED).
func reconstructCodexText(raw string) string {
	return ""
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
