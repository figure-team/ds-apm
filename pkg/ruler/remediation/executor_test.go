package remediation

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExecutor_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash executor is linux/unix")
	}
	r := NewExecutor(5*time.Second).Run(context.Background(), "#!/bin/bash\necho hello-world\n")
	if r.ExitCode != 0 || r.TimedOut {
		t.Fatalf("want exit 0, got %+v", r)
	}
	if !strings.Contains(r.Output, "hello-world") {
		t.Fatalf("output missing: %q", r.Output)
	}
}

func TestExecutor_NonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	r := NewExecutor(5*time.Second).Run(context.Background(), "exit 7")
	if r.ExitCode != 7 || r.TimedOut {
		t.Fatalf("want exit 7, got %+v", r)
	}
}

// When the process cannot start (e.g. bash not installed, as on a stock Alpine
// image), the result must be exit -1 AND carry an explanatory snippet so the
// operator does not just see a bare -1 with no output.
func TestExecutor_StartFailureSurfacedInOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	t.Setenv("PATH", "") // make `bash` unresolvable so the process fails to start
	r := NewExecutor(5*time.Second).Run(context.Background(), "echo nope")
	if r.ExitCode != -1 || r.TimedOut {
		t.Fatalf("want exit -1 start failure, got %+v", r)
	}
	if !strings.Contains(r.Output, "실행 실패") {
		t.Fatalf("start error not surfaced in output: %q", r.Output)
	}
}

func TestExecutor_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	r := NewExecutor(200*time.Millisecond).Run(context.Background(), "sleep 10")
	if !r.TimedOut {
		t.Fatalf("want timeout, got %+v", r)
	}
}
