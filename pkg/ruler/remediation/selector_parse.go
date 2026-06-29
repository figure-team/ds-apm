package remediation

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// ParseSelectionResponse extracts the single JSON object from a (possibly
// fenced) LLM response, unmarshals it into a RunbookSelectionDecision, and
// validates it (demoting a dangling chosenRunbookId to "none"). Any failure
// returns an error and the caller treats the run as non-actionable (fail-open).
func ParseSelectionResponse(raw string, approvedRunbookIDs map[string]struct{}) (ruletypes.RunbookSelectionDecision, error) {
	body := extractJSONObject(raw)
	if body == "" {
		return ruletypes.RunbookSelectionDecision{}, fmt.Errorf("selector: no JSON object in response")
	}
	var d ruletypes.RunbookSelectionDecision
	if err := json.Unmarshal([]byte(body), &d); err != nil {
		return ruletypes.RunbookSelectionDecision{}, fmt.Errorf("selector: unmarshal: %w", err)
	}
	if strings.TrimSpace(d.ContractVersion) == "" {
		d.ContractVersion = ruletypes.RunbookSelectionContractVersion
	}
	return ruletypes.ValidateRunbookSelectionDecision(d, approvedRunbookIDs)
}

// extractJSONObject returns the substring from the first '{' to the last '}'
// inclusive, stripping code fences and surrounding prose. Returns "" if none.
func extractJSONObject(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end < 0 || end < start {
		return ""
	}
	return s[start : end+1]
}
