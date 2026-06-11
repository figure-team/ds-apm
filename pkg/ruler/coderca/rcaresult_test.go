package coderca

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "rca", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

func TestParseRCAResultFixtures(t *testing.T) {
	cases := []struct {
		file       string
		wantStatus RunStatus
		check      func(t *testing.T, r RCAResult)
	}{
		{
			file:       "clean.txt",
			wantStatus: RunStatusDone,
			check: func(t *testing.T, r RCAResult) {
				if r.BaselineCommit != "a1b2c3d4" {
					t.Errorf("BaselineCommit = %q, want a1b2c3d4", r.BaselineCommit)
				}
				if r.Confidence != "medium" {
					t.Errorf("Confidence = %q, want medium", r.Confidence)
				}
				if !strings.Contains(r.RootCause, "pool is exhausted") {
					t.Errorf("RootCause missing expected text: %q", r.RootCause)
				}
				if r.ProposedFix == "" {
					t.Error("ProposedFix is empty")
				}
				if r.Limitations == "" {
					t.Error("Limitations is empty")
				}
			},
		},
		{
			file:       "wrapped.txt",
			wantStatus: RunStatusDone,
			check: func(t *testing.T, r RCAResult) {
				if r.BaselineCommit != "deadbeef99" {
					t.Errorf("BaselineCommit = %q, want deadbeef99", r.BaselineCommit)
				}
				if r.Confidence != "high" {
					t.Errorf("Confidence = %q, want high", r.Confidence)
				}
				if !strings.Contains(r.RootCause, "nil pointer") {
					t.Errorf("RootCause missing expected text: %q", r.RootCause)
				}
			},
		},
		{
			// The parser must take the LAST json block (real finding), not the
			// leading "expected format" example block.
			file:       "two_blocks.txt",
			wantStatus: RunStatusDone,
			check: func(t *testing.T, r RCAResult) {
				if r.BaselineCommit != "99887766" {
					t.Errorf("BaselineCommit = %q, want 99887766 (last block)", r.BaselineCommit)
				}
				if strings.Contains(r.RootCause, "EXAMPLE") {
					t.Errorf("RootCause took the example block, not the real one: %q", r.RootCause)
				}
				if !strings.Contains(r.RootCause, "Retry loop") {
					t.Errorf("RootCause missing expected text: %q", r.RootCause)
				}
			},
		},
		{
			file:       "bare_json.txt",
			wantStatus: RunStatusDone,
			check: func(t *testing.T, r RCAResult) {
				if r.BaselineCommit != "abcabc12" {
					t.Errorf("BaselineCommit = %q, want abcabc12", r.BaselineCommit)
				}
				if !strings.Contains(r.RootCause, "Missing index") {
					t.Errorf("RootCause missing expected text: %q", r.RootCause)
				}
			},
		},
		{file: "malformed.txt", wantStatus: RunStatusUnparseable},
		{file: "prose_only.txt", wantStatus: RunStatusUnparseable},
		{file: "empty_obj.txt", wantStatus: RunStatusUnparseable},
	}

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			raw := readFixture(t, tc.file)
			got, status := ParseRCAResult(raw)

			if status != tc.wantStatus {
				t.Fatalf("status = %q, want %q", status, tc.wantStatus)
			}
			// Raw is always retained, even when parsing fails (audit).
			if got.Raw != raw {
				t.Errorf("Raw not retained: got %d bytes, want %d", len(got.Raw), len(raw))
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

func TestParseRCAResultConfidenceNormalization(t *testing.T) {
	const fence = "```"
	build := func(conf string) string {
		return fence + "json\n" +
			`{"root_cause":"x","confidence":"` + conf + `"}` + "\n" + fence
	}
	tests := []struct {
		in   string
		want string
	}{
		{in: "high", want: "high"},
		{in: "Medium", want: "medium"},
		{in: "LOW", want: "low"},
		{in: " high ", want: "high"},
		{in: "", want: "low"},          // missing → conservative default
		{in: "uncertain", want: "low"}, // unknown → conservative default
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, status := ParseRCAResult(build(tc.in))
			if status != RunStatusDone {
				t.Fatalf("status = %q, want done", status)
			}
			if got.Confidence != tc.want {
				t.Errorf("Confidence = %q, want %q", got.Confidence, tc.want)
			}
		})
	}
}

func TestParseRCAResultEmptyInput(t *testing.T) {
	got, status := ParseRCAResult("")
	if status != RunStatusUnparseable {
		t.Errorf("status = %q, want unparseable", status)
	}
	if got.Raw != "" {
		t.Errorf("Raw = %q, want empty", got.Raw)
	}
}
