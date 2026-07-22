package exportmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
)

func sampleDetail() runstore.RunDetail {
	return runstore.RunDetail{
		RunSummary: runstore.RunSummary{
			RunID:          "run-1",
			OrgID:          "org-1",
			Service:        "payment-api",
			Status:         coderca.RunStatusDone,
			BaselineCommit: "abcdef1234567890",
			CreatedAt:      1784692800,
		},
		RootCause:   "NPE가 발생했습니다.",
		ProposedFix: "널 체크를 추가하세요.",
		Confidence:  "high",
		Limitations: "재현 로그가 제한적입니다.",
	}
}

func TestFilename(t *testing.T) {
	// 1784692800 = 2026-07-22 04:00:00 UTC = 13:00:00 KST.
	got := Filename("payment-api", 1784692800)
	if want := "2026-07-22_130000_rca_payment-api.md"; got != want {
		t.Fatalf("filename: got %q want %q", got, want)
	}
}

func TestFilenameDiffersPerSecond(t *testing.T) {
	if a, b := Filename("svc", 1784692800), Filename("svc", 1784692801); a == b {
		t.Fatalf("same-day different-time filenames must differ: %q", a)
	}
}

func TestFilenameSanitizesService(t *testing.T) {
	got := Filename(`a/b\c:d*e?"f<g>h| i`, 1784692800)
	svc := strings.SplitN(got, "_rca_", 2)[1]
	for _, bad := range []string{"/", `\`, ":", "*", "?", `"`, "<", ">", "|", " "} {
		if strings.Contains(svc, bad) {
			t.Fatalf("unsanitized %q in %q", bad, got)
		}
	}
}

func TestFilenameEmptyServiceFallsBack(t *testing.T) {
	got := Filename("   ", 1784692800)
	if !strings.HasSuffix(got, "_rca_service.md") {
		t.Fatalf("empty service fallback: %q", got)
	}
}

func TestBuildContainsAllFields(t *testing.T) {
	md := Build(sampleDetail())
	for _, want := range []string{
		"runId: run-1",
		"service: payment-api",
		"confidence: high",
		"baselineCommit: abcdef1234567890",
		"createdAt: ",
		"# 코드 RCA 리포트 — payment-api",
		"## 근본 원인",
		"NPE가 발생했습니다.",
		"## 수정 제안",
		"널 체크를 추가하세요.",
		"## 한계",
		"재현 로그가 제한적입니다.",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q in:\n%s", want, md)
		}
	}
	if !strings.HasPrefix(md, "---\n") {
		t.Fatalf("missing frontmatter open:\n%s", md)
	}
}

func TestBuildOmitsEmptyLimitations(t *testing.T) {
	d := sampleDetail()
	d.Limitations = ""
	if strings.Contains(Build(d), "## 한계") {
		t.Fatal("empty limitations must omit section")
	}
}

func TestWriteCreatesDsHubAndOverwrites(t *testing.T) {
	root := t.TempDir()
	d := sampleDetail()

	p1, err := Write(root, d)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(p1) != filepath.Join(root, "ds-hub") {
		t.Fatalf("wrong dir: %q", p1)
	}

	d.RootCause = "수정된 원인"
	p2, err := Write(root, d)
	if err != nil {
		t.Fatal(err)
	}
	if p1 != p2 {
		t.Fatalf("same run (same createdAt) must overwrite: %q vs %q", p1, p2)
	}
	b, err := os.ReadFile(p2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "수정된 원인") {
		t.Fatal("overwrite did not take effect")
	}

	d.CreatedAt++
	p3, err := Write(root, d)
	if err != nil {
		t.Fatal(err)
	}
	if p3 == p1 {
		t.Fatalf("different createdAt must not collide: %q", p3)
	}
	if _, err := os.Stat(p1); err != nil {
		t.Fatalf("earlier export must survive later export: %v", err)
	}
}
