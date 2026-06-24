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

func TestExecutor_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	r := NewExecutor(200*time.Millisecond).Run(context.Background(), "sleep 10")
	if !r.TimedOut {
		t.Fatalf("want timeout, got %+v", r)
	}
}
