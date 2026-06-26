package ruletypes

import "testing"

func TestDefaultRemediationConfig(t *testing.T) {
	c := DefaultRemediationConfig()
	if c.ExecutionEnabled {
		t.Error("execution must default OFF")
	}
	if c.ProposalTTLSeconds != 1800 || c.ExecTimeoutSeconds != 300 ||
		c.VerifyWindowSeconds != 600 || c.MaxConcurrent != 1 {
		t.Fatalf("unexpected defaults: %+v", c)
	}
}

func TestRemediationConfigWithDefaults(t *testing.T) {
	// Zero numeric fields get backfilled; ExecutionEnabled is preserved as-is.
	c := RemediationConfig{ExecutionEnabled: true}.WithDefaults()
	if !c.ExecutionEnabled {
		t.Error("ExecutionEnabled must be preserved")
	}
	if c.ProposalTTLSeconds != 1800 || c.MaxConcurrent != 1 {
		t.Fatalf("zero fields not backfilled: %+v", c)
	}
	// Explicit values are preserved.
	c2 := RemediationConfig{ExecTimeoutSeconds: 120}.WithDefaults()
	if c2.ExecTimeoutSeconds != 120 {
		t.Fatalf("explicit value clobbered: %+v", c2)
	}
}

func TestValidateRemediationConfig(t *testing.T) {
	// Both toggle states with sane knobs are valid.
	if err := ValidateRemediationConfig(RemediationConfig{ExecutionEnabled: true}.WithDefaults()); err != nil {
		t.Fatalf("enabled default config must be valid: %v", err)
	}
	if err := ValidateRemediationConfig(DefaultRemediationConfig()); err != nil {
		t.Fatalf("default config must be valid: %v", err)
	}
	// Negative knobs are rejected.
	if err := ValidateRemediationConfig(RemediationConfig{ProposalTTLSeconds: -1}); err == nil {
		t.Error("negative proposalTtlSeconds must be rejected")
	}
	if err := ValidateRemediationConfig(RemediationConfig{MaxConcurrent: -5}); err == nil {
		t.Error("negative maxConcurrent must be rejected")
	}
}
