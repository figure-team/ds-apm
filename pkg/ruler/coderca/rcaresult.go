package coderca

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
// M2-3 STUB: always unparseable, Raw dropped → success + raw-retention
// assertions fail (RED).
func ParseRCAResult(raw string) (RCAResult, RunStatus) {
	return RCAResult{}, RunStatusUnparseable
}
