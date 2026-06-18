package signoz

import "testing"

func TestNoticeMaxOutputTokens(t *testing.T) {
	cases := []struct {
		name       string
		explicit   int
		budgetUSD  float64
		usdPerMTok float64
		want       int
	}{
		{"all unset uses conservative default", 0, 0, 0, DefaultNoticeMaxOutputTokens},
		{"explicit cap honored", 256, 0, 0, 256},
		{"budget derives tighter cap", 0, 0.001, 15.0, 66},   // 0.001/15 * 1e6 = 66.6 -> 66
		{"explicit beats looser budget", 256, 100, 15.0, 256}, // budget-derived huge -> explicit wins as min
		{"budget beats looser explicit", 4096, 0.001, 15.0, 66},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := noticeMaxOutputTokens(c.explicit, c.budgetUSD, c.usdPerMTok)
			if got != c.want {
				t.Fatalf("noticeMaxOutputTokens(%d,%v,%v) = %d, want %d", c.explicit, c.budgetUSD, c.usdPerMTok, got, c.want)
			}
		})
	}
}
