package coderca

import (
	"encoding/json"
	"strings"
)

// RCAResult is the structured outcome of one code-RCA run, parsed from the CLI
// agent's output (design §7). ProposedFix is a SUGGESTION for human review and
// is never applied. Raw retains the full CLI output for audit — including when
// parsing fails, so an unparseable run is still inspectable.
type RCAResult struct {
	BaselineCommit string // commit the agent reports it analyzed (echoed back)
	RootCause      string
	ProposedFix    string // suggestion only; never applied (HITL)
	Confidence     string // high | medium | low
	Limitations    string
	Raw            string // full CLI output, retained for audit
}

// ParseRCAResult parses raw CLI output into an RCAResult. The agent is
// instructed to emit a single fenced ```json block; the parser takes the LAST
// such block (so a leading example/thinking block is ignored), tolerates prose
// wrapping and a bare JSON object, and normalizes confidence to high|medium|low.
//
// Returns RunStatusDone when a result with a non-empty root cause is recovered,
// else RunStatusUnparseable. Raw is always retained on the returned result.
//
func ParseRCAResult(raw string) (RCAResult, RunStatus) {
	blob, ok := extractJSONBlob(raw)
	if !ok {
		return RCAResult{Raw: raw}, RunStatusUnparseable
	}

	var p struct {
		BaselineCommit string `json:"baseline_commit"`
		RootCause      string `json:"root_cause"`
		ProposedFix    string `json:"proposed_fix"`
		Confidence     string `json:"confidence"`
		Limitations    string `json:"limitations"`
	}
	if err := json.Unmarshal([]byte(blob), &p); err != nil {
		return RCAResult{Raw: raw}, RunStatusUnparseable
	}

	res := RCAResult{
		BaselineCommit: strings.TrimSpace(p.BaselineCommit),
		RootCause:      strings.TrimSpace(p.RootCause),
		ProposedFix:    strings.TrimSpace(p.ProposedFix),
		Confidence:     normalizeConfidence(p.Confidence),
		Limitations:    strings.TrimSpace(p.Limitations),
		Raw:            raw,
	}
	// A result with no root cause is not a usable RCA — treat as unparseable
	// (the structured fields stay zero, Raw is retained).
	if res.RootCause == "" {
		return RCAResult{Raw: raw}, RunStatusUnparseable
	}
	return res, RunStatusDone
}

// extractJSONBlob returns the JSON text to parse. It prefers the LAST fenced
// ```json block (so a leading example/thinking block is ignored); failing that,
// it accepts a bare JSON object that the trimmed output starts with.
func extractJSONBlob(raw string) (string, bool) {
	const marker = "```json"
	if start := strings.LastIndex(raw, marker); start >= 0 {
		rest := raw[start+len(marker):]
		nl := strings.IndexByte(rest, '\n')
		if nl < 0 {
			return "", false
		}
		rest = rest[nl+1:]
		end := strings.Index(rest, "```")
		if end < 0 {
			return "", false
		}
		return rest[:end], true
	}
	if trimmed := strings.TrimSpace(raw); strings.HasPrefix(trimmed, "{") {
		return trimmed, true
	}
	return "", false
}

// normalizeConfidence lower-cases and trims the agent's confidence and clamps
// it to the high|medium|low enum, defaulting unknown/missing to low so a run
// never overstates its certainty.
func normalizeConfidence(c string) string {
	switch strings.ToLower(strings.TrimSpace(c)) {
	case "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}
