package coderca

import "testing"

func TestShouldSampleAudit(t *testing.T) {
	tests := []struct {
		occurrence, everyN int
		want               bool
	}{
		{1, 100, true},   // always audit the first occurrence
		{2, 100, false},  // suppressed
		{99, 100, false}, // suppressed
		{100, 100, true}, // every 100th
		{101, 100, false},
		{200, 100, true},
		{1, 1, true}, // everyN<=1 → audit every occurrence
		{7, 1, true},
		{5, 0, false}, // everyN<=0 (and not first) → only the first is audited
		{0, 100, false},
		{-3, 100, false},
	}
	for _, tc := range tests {
		if got := ShouldSampleAudit(tc.occurrence, tc.everyN); got != tc.want {
			t.Errorf("ShouldSampleAudit(%d, %d) = %v, want %v", tc.occurrence, tc.everyN, got, tc.want)
		}
	}
}
