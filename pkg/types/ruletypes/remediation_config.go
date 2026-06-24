package ruletypes

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
