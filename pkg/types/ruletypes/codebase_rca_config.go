package ruletypes

import (
	"fmt"
	"strings"
)

// CodebaseRCAConfigContractVersion versions the CF-11 per-org config payload.
const CodebaseRCAConfigContractVersion = "ds.codebase_rca_config.v1"

// severityRank orders alert severities for the min-severity gate. Unknown or
// missing severities rank 0 so the gate fails closed (design §10). The
// recognized set is critical|error|warning|info, matching the alert routing
// severities (ruletypes.*ThresholdName); "high" was retired (it ranked equal
// to "error") so all surfaces share one severity vocabulary.
var severityRank = map[string]int{
	"critical": 4,
	"error":    3,
	"warning":  2,
	"info":     1,
}

// SeverityAtLeast reports whether severity meets the minimum. Comparison is
// case-insensitive; unknown values never pass (fail-closed).
func SeverityAtLeast(severity, min string) bool {
	s := severityRank[strings.ToLower(strings.TrimSpace(severity))]
	m := severityRank[strings.ToLower(strings.TrimSpace(min))]
	if s == 0 || m == 0 {
		return false
	}
	return s >= m
}

// CodebaseRCAConfig is the per-org CF-11 feature toggle + cost thresholds
// (design §6: "all thresholds live in codebase_config, per-org overridable").
// Agent/model/auth are deployment-level (env), not per-org.
type CodebaseRCAConfig struct {
	ContractVersion string `json:"contractVersion"`
	OrgID           string `json:"orgId"`
	Enabled         bool   `json:"enabled"`
	// MinSeverity gates the trigger predicate (default "error" → error|critical).
	MinSeverity        string `json:"minSeverity"`
	CooldownWindowSecs int    `json:"cooldownWindowSecs"`
	MaxRunsPerDay      int    `json:"maxRunsPerDay"`
	MaxQueueDepth      int    `json:"maxQueueDepth"`
	MaxConcurrentRuns  int    `json:"maxConcurrentRuns"`
	// AllowUnboundWithoutAnomaly revives the legacy unbound+severity trigger
	// without an anomaly signal. Off by default; enabling logs a loud warning
	// (design §10).
	AllowUnboundWithoutAnomaly bool   `json:"allowUnboundWithoutAnomaly"`
	UpdatedAt                  string `json:"updatedAt"` // RFC3339
}

// DefaultCodebaseRCAConfig returns the fail-closed defaults from design §6.
func DefaultCodebaseRCAConfig(orgID string) CodebaseRCAConfig {
	return CodebaseRCAConfig{
		ContractVersion:    CodebaseRCAConfigContractVersion,
		OrgID:              orgID,
		Enabled:            false,
		MinSeverity:        "error",
		CooldownWindowSecs: 21600,
		MaxRunsPerDay:      20,
		MaxQueueDepth:      50,
		MaxConcurrentRuns:  1,
	}
}

// ValidateCodebaseRCAConfig validates a config update.
func ValidateCodebaseRCAConfig(cfg CodebaseRCAConfig) error {
	var errs []string
	if strings.TrimSpace(cfg.ContractVersion) != CodebaseRCAConfigContractVersion {
		errs = append(errs, fmt.Sprintf("contractVersion: must be %q, got %q", CodebaseRCAConfigContractVersion, cfg.ContractVersion))
	}
	if strings.TrimSpace(cfg.OrgID) == "" {
		errs = append(errs, "orgId: must not be empty")
	}
	if _, ok := severityRank[strings.ToLower(strings.TrimSpace(cfg.MinSeverity))]; !ok {
		errs = append(errs, fmt.Sprintf("minSeverity: %q is not one of critical|error|warning|info", cfg.MinSeverity))
	}
	if cfg.CooldownWindowSecs < 1 {
		errs = append(errs, "cooldownWindowSecs: must be >= 1")
	}
	if cfg.MaxRunsPerDay < 1 {
		errs = append(errs, "maxRunsPerDay: must be >= 1")
	}
	if cfg.MaxQueueDepth < 1 {
		errs = append(errs, "maxQueueDepth: must be >= 1")
	}
	if cfg.MaxConcurrentRuns < 0 || cfg.MaxConcurrentRuns > 2 {
		errs = append(errs, "maxConcurrentRuns: must be 0..2 (design §6.3)")
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("codebase RCA config validation: %s", strings.Join(errs, "; "))
}
