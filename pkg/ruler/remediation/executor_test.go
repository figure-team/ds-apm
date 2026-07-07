package remediation

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

func TestExecutor_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash executor is linux/unix")
	}
	r := NewExecutor(5*time.Second).Run(context.Background(), "#!/bin/bash\necho hello-world\n", nil, RunMeta{})
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
	r := NewExecutor(5*time.Second).Run(context.Background(), "exit 7", nil, RunMeta{})
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
	r := NewExecutor(5*time.Second).Run(context.Background(), "echo nope", nil, RunMeta{})
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
	r := NewExecutor(200*time.Millisecond).Run(context.Background(), "sleep 10", nil, RunMeta{})
	if !r.TimedOut {
		t.Fatalf("want timeout, got %+v", r)
	}
}

func TestRun_TagsViaFromMeta(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash executor is linux/unix")
	}
	// cliaudit는 env가 비면 no-op이므로, 여기서는 Run이 meta.Via를 받아
	// 정상 동작(빈 Via 기본값 포함)하는지 + 결과가 변하지 않는지 확인한다.
	e := NewExecutor(2 * time.Second)
	res := e.Run(context.Background(), "exit 0", nil, RunMeta{Via: "remediation-exec-llm", Source: "llm-generated", Fingerprint: "fp-1"})
	if res.ExitCode != 0 {
		t.Fatalf("expected exit 0, got %d", res.ExitCode)
	}
	res2 := e.Run(context.Background(), "exit 3", nil, RunMeta{}) // 빈 meta → 기본 Via
	if res2.ExitCode != 3 {
		t.Fatalf("expected exit 3, got %d", res2.ExitCode)
	}
}

func TestExecutorRun_LocalWhenTargetNil(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash executor is linux/unix")
	}
	e := NewExecutor(5 * time.Second)
	res := e.Run(context.Background(), "echo hi", nil, RunMeta{})
	if res.ExitCode != 0 || !contains(res.Output, "hi") {
		t.Fatalf("local run failed: %+v", res)
	}
}

func TestLocalTransport_AppliesArgvPrefix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("local transport runs bash")
	}
	// 프리픽스가 실제 argv 앞에 붙는지 env 래퍼로 관찰한다.
	tr := newLocalTransport(5*time.Second, []string{"env", "DSAPM_SBX=1"})
	out, code, timedOut, err := tr.Exec(context.Background(), `echo "sbx=$DSAPM_SBX"`)
	if err != nil || timedOut || code != 0 {
		t.Fatalf("exec: code=%d timedOut=%v err=%v out=%q", code, timedOut, err, out)
	}
	if !contains(out, "sbx=1") {
		t.Fatalf("prefix not applied: %q", out)
	}
}

func TestExecutorRun_LLMSandboxUnavailable_FailClosed(t *testing.T) {
	e := &Executor{timeout: time.Second, probe: fakeProbe(nil, nil, nil)}
	res := e.Run(context.Background(), "echo hi", nil, RunMeta{Source: ruletypes.RemediationSourceLLMGenerated})
	if res.ExitCode != -1 {
		t.Fatalf("want -1, got %d (out=%q)", res.ExitCode, res.Output)
	}
	if !strings.Contains(res.Output, "샌드박스") {
		t.Fatalf("output must explain sandbox failure: %q", res.Output)
	}
}
