package signoz

// DefaultNoticeMaxOutputTokens is the conservative per-call output cap used for
// SOP customer-notice generation when no explicit cap is configured. Notices
// are short (a titled 공지문), so a tight cap bounds spend and latency.
const DefaultNoticeMaxOutputTokens = 1024

// noticeMaxOutputTokens resolves the effective output-token cap for a single
// customer-notice generation. It is a PREVENTIVE guardrail: capping max_tokens
// bounds worst-case cost without post-hoc usage accounting (the api transport's
// Complete does not return usage).
//
//   - explicit (DS_APM_AINOTICE_MAX_OUTPUT_TOKENS): hard cap when > 0.
//   - budgetUSD + usdPerMTok: when both > 0, derive tokens = budgetUSD/usdPerMTok*1e6
//     and take the tighter (min) of explicit and derived.
//
// When nothing is set, DefaultNoticeMaxOutputTokens applies.
func noticeMaxOutputTokens(explicit int, budgetUSD, usdPerMTok float64) int {
	capTokens := explicit
	if capTokens <= 0 {
		capTokens = DefaultNoticeMaxOutputTokens
	}
	if budgetUSD > 0 && usdPerMTok > 0 {
		derived := int(budgetUSD / usdPerMTok * 1_000_000)
		if derived > 0 && derived < capTokens {
			capTokens = derived
		}
	}
	return capTokens
}
