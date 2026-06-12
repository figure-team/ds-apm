package coderca

import "testing"

func TestErrorSignature(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   string
	}{
		{
			name:   "full set, fixed order",
			labels: map[string]string{"alertname": "High5xx", "service.name": "payments", "severity": "critical", "error_class": "PgTimeout"},
			want:   "alertname=High5xx|service.name=payments|severity=critical|error_class=PgTimeout",
		},
		{
			name: "high-cardinality labels are ignored",
			labels: map[string]string{
				"alertname": "High5xx", "service.name": "payments", "severity": "critical", "error_class": "PgTimeout",
				"pod": "payments-7f9c-abcde", "instance": "10.0.3.41:9090", "replica": "3",
			},
			want: "alertname=High5xx|service.name=payments|severity=critical|error_class=PgTimeout",
		},
		{
			name:   "severity is case-normalized",
			labels: map[string]string{"alertname": "High5xx", "service.name": "payments", "severity": "Critical", "error_class": "PgTimeout"},
			want:   "alertname=High5xx|service.name=payments|severity=critical|error_class=PgTimeout",
		},
		{
			name:   "missing error_class is omitted",
			labels: map[string]string{"alertname": "High5xx", "service.name": "payments", "severity": "warning"},
			want:   "alertname=High5xx|service.name=payments|severity=warning",
		},
		{
			name:   "values are trimmed",
			labels: map[string]string{"alertname": "  High5xx  ", "service.name": "payments", "severity": " CRITICAL "},
			want:   "alertname=High5xx|service.name=payments|severity=critical",
		},
		{
			name:   "empty labels yield empty signature",
			labels: map[string]string{},
			want:   "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ErrorSignature(tc.labels); got != tc.want {
				t.Errorf("ErrorSignature() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestErrorSignatureIgnoresLabelOrder(t *testing.T) {
	// Two pods of the same logical error must produce the same signature so
	// they dedup onto one run (the core volume-control invariant).
	a := ErrorSignature(map[string]string{"alertname": "High5xx", "service.name": "payments", "severity": "critical", "pod": "a"})
	b := ErrorSignature(map[string]string{"alertname": "High5xx", "service.name": "payments", "severity": "critical", "pod": "b"})
	if a != b {
		t.Errorf("signatures differ by high-cardinality label: %q vs %q", a, b)
	}
	if a == "" {
		t.Fatal("signature unexpectedly empty")
	}
}

func TestDedupKey(t *testing.T) {
	sig := "alertname=High5xx|service.name=payments|severity=critical"

	base := DedupKey("org1", "payments", sig)
	if base == "" {
		t.Fatal("DedupKey returned empty")
	}

	// Deterministic.
	if again := DedupKey("org1", "payments", sig); again != base {
		t.Errorf("DedupKey not deterministic: %q vs %q", base, again)
	}

	// Distinct per org / service / signature.
	if DedupKey("org2", "payments", sig) == base {
		t.Error("DedupKey collides across orgs")
	}
	if DedupKey("org1", "orders", sig) == base {
		t.Error("DedupKey collides across services")
	}
	if DedupKey("org1", "payments", sig+"|error_class=Other") == base {
		t.Error("DedupKey collides across signatures")
	}

	// Tenant isolation: no field-boundary confusion (org+service vs orgservice).
	if DedupKey("org1a", "b", sig) == DedupKey("org1", "ab", sig) {
		t.Error("DedupKey conflates org/service boundary")
	}
}
