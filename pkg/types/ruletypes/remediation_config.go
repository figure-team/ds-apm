package ruletypes

import (
	"errors"
	"fmt"
)

// RemediationConfig is the per-org master switch + timing knobs for the
// remediation execution feature (design §6). ExecutionEnabled defaults OFF so
// the feature is inert until an org explicitly opts in.
type RemediationConfig struct {
	ExecutionEnabled    bool  `json:"executionEnabled"`
	ProposalTTLSeconds  int64 `json:"proposalTtlSeconds"`
	ExecTimeoutSeconds  int64 `json:"execTimeoutSeconds"`
	VerifyWindowSeconds int64 `json:"verifyWindowSeconds"`
	MaxConcurrent       int64 `json:"maxConcurrent"`
}

func DefaultRemediationConfig() RemediationConfig {
	return RemediationConfig{
		ExecutionEnabled:    false,
		ProposalTTLSeconds:  1800,
		ExecTimeoutSeconds:  300,
		VerifyWindowSeconds: 600,
		MaxConcurrent:       1,
	}
}

// WithDefaults backfills zero-valued numeric knobs with defaults, preserving any
// explicitly-set values and the ExecutionEnabled flag. Used when a stored config
// row has missing/zero columns.
func (c RemediationConfig) WithDefaults() RemediationConfig {
	d := DefaultRemediationConfig()
	if c.ProposalTTLSeconds == 0 {
		c.ProposalTTLSeconds = d.ProposalTTLSeconds
	}
	if c.ExecTimeoutSeconds == 0 {
		c.ExecTimeoutSeconds = d.ExecTimeoutSeconds
	}
	if c.VerifyWindowSeconds == 0 {
		c.VerifyWindowSeconds = d.VerifyWindowSeconds
	}
	if c.MaxConcurrent == 0 {
		c.MaxConcurrent = d.MaxConcurrent
	}
	return c
}

// ValidateRemediationConfig rejects out-of-range knobs before persistence.
// Zero is tolerated for the numeric knobs (WithDefaults backfills them); only
// negative values and a non-positive concurrency cap are errors. ExecutionEnabled
// needs no validation — any bool is valid.
func ValidateRemediationConfig(c RemediationConfig) error {
	var errs []error
	if c.ProposalTTLSeconds < 0 {
		errs = append(errs, fmt.Errorf("proposalTtlSeconds: must be >= 0 (got %d)", c.ProposalTTLSeconds))
	}
	if c.ExecTimeoutSeconds < 0 {
		errs = append(errs, fmt.Errorf("execTimeoutSeconds: must be >= 0 (got %d)", c.ExecTimeoutSeconds))
	}
	if c.VerifyWindowSeconds < 0 {
		errs = append(errs, fmt.Errorf("verifyWindowSeconds: must be >= 0 (got %d)", c.VerifyWindowSeconds))
	}
	if c.MaxConcurrent < 0 {
		errs = append(errs, fmt.Errorf("maxConcurrent: must be >= 0 (got %d)", c.MaxConcurrent))
	}
	return errors.Join(errs...)
}
