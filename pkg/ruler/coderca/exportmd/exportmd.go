// Package exportmd renders a finished Code-RCA run as a markdown artifact and
// writes it under <artifactRoot>/ds-hub/issues/ for hand-off to external
// consumers (ds-navi).
package exportmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
)

// subDir is the ds-hub subfolder RCA artifacts land in; it groups exported
// runs as issues alongside other ds-hub content.
const (
	dirName = "ds-hub"
	subDir  = "issues"
)

// kst pins filename dates and frontmatter timestamps to Asia/Seoul regardless
// of the server/container timezone (a UTC container would shift the date for
// runs created between 00:00 and 09:00 KST).
var kst = time.FixedZone("KST", 9*60*60)

// unsafeChars collapses path separators, Windows-forbidden characters,
// whitespace and control characters into a single dash.
var unsafeChars = regexp.MustCompile(`[\\/:*?"<>|[:cntrl:]\s]+`)

func sanitizeService(s string) string {
	s = unsafeChars.ReplaceAllString(strings.TrimSpace(s), "-")
	s = strings.Trim(s, "-.")
	if s == "" {
		return "service"
	}
	return s
}

// Filename returns "<YYYY-MM-DD>_<HHmmss>_rca_<service>.md"; the timestamp is
// the run's creation time in KST. Including 시분초 keeps same-day re-exports of
// the same service from overwriting each other.
func Filename(service string, createdAtUnix int64) string {
	stamp := time.Unix(createdAtUnix, 0).In(kst).Format("2006-01-02_150405")
	return fmt.Sprintf("%s_rca_%s.md", stamp, sanitizeService(service))
}

// Build renders YAML frontmatter (runId/service/createdAt/confidence/
// baselineCommit) followed by 근본 원인·수정 제안·한계 sections. 한계 is
// omitted when empty.
func Build(d runstore.RunDetail) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "runId: %s\n", d.RunID)
	fmt.Fprintf(&b, "service: %s\n", d.Service)
	fmt.Fprintf(&b, "createdAt: %s\n", time.Unix(d.CreatedAt, 0).In(kst).Format(time.RFC3339))
	fmt.Fprintf(&b, "confidence: %s\n", d.Confidence)
	fmt.Fprintf(&b, "baselineCommit: %s\n", d.BaselineCommit)
	b.WriteString("---\n\n")
	fmt.Fprintf(&b, "# 코드 RCA 리포트 — %s\n\n", d.Service)
	b.WriteString("## 근본 원인\n\n")
	b.WriteString(strings.TrimSpace(d.RootCause))
	b.WriteString("\n\n## 수정 제안\n\n")
	b.WriteString(strings.TrimSpace(d.ProposedFix))
	b.WriteString("\n")
	if strings.TrimSpace(d.Limitations) != "" {
		b.WriteString("\n## 한계\n\n")
		b.WriteString(strings.TrimSpace(d.Limitations))
		b.WriteString("\n")
	}
	return b.String()
}

// Write renders the artifact into <artifactRoot>/ds-hub/issues/, creating the
// folders when missing. Re-exporting the same run overwrites its file (identical
// createdAt → identical name); runs created at different times never collide.
// Returns the written path.
func Write(artifactRoot string, d runstore.RunDetail) (string, error) {
	dir := filepath.Join(artifactRoot, dirName, subDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("exportmd: create %s: %w", dir, err)
	}
	path := filepath.Join(dir, Filename(d.Service, d.CreatedAt))
	if err := os.WriteFile(path, []byte(Build(d)), 0o644); err != nil {
		return "", fmt.Errorf("exportmd: write %s: %w", path, err)
	}
	return path, nil
}
