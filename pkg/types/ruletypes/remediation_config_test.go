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
