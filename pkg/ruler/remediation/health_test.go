package remediation

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// fakeLister는 targetLister의 테스트 구현.
type fakeLister struct {
	mu      sync.Mutex
	targets []ruletypes.RemediationTarget
	err     error
}

func (f *fakeLister) ListAll(context.Context) ([]ruletypes.RemediationTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.targets, f.err
}

func (f *fakeLister) set(ts []ruletypes.RemediationTarget, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.targets, f.err = ts, err
}

func healthTarget(id, fp string) ruletypes.RemediationTarget {
	return ruletypes.RemediationTarget{
		ID: id, OrgID: "org-1", Name: id, Host: "10.0.0.1", Port: 22,
		HostKeyFingerprint: fp,
	}
}

func probeReturning(fp string, err error) ProbeFunc {
	return func(context.Context, string, int, time.Duration) (string, string, error) {
		return fp, "ssh-ed25519", err
	}
}

// 판정 4종: healthy / unreachable / mismatch / 저장지문 공백→healthy.
func TestHealthChecker_SweepJudgments(t *testing.T) {
	cases := []struct {
		name     string
		stored   string
		probeFP  string
		probeErr error
		want     TargetHealthStatus
	}{
		{"healthy", "SHA256:abc", "SHA256:abc", nil, TargetHealthHealthy},
		{"unreachable", "SHA256:abc", "", errors.New("dial tcp: timeout"), TargetHealthUnreachable},
		{"mismatch", "SHA256:abc", "SHA256:zzz", nil, TargetHealthMismatch},
		{"empty stored fingerprint reachable", "", "SHA256:any", nil, TargetHealthHealthy},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lister := &fakeLister{targets: []ruletypes.RemediationTarget{healthTarget("t1", tc.stored)}}
			c := NewHealthChecker(lister, probeReturning(tc.probeFP, tc.probeErr), time.Hour)
			c.SweepOnce(context.Background())
			got := c.Snapshot()["t1"]
			if got.Status != tc.want {
				t.Fatalf("status: got %q want %q (err=%q)", got.Status, tc.want, got.Error)
			}
			if got.CheckedAt.IsZero() {
				t.Fatal("CheckedAt must be set after a sweep")
			}
			if tc.want == TargetHealthUnreachable && got.Error == "" {
				t.Fatal("unreachable must preserve the probe error")
			}
			if tc.want == TargetHealthMismatch && got.Error == "" {
				t.Fatal("mismatch must carry a detail message")
			}
		})
	}
}

// 순회에서 사라진 타겟은 맵에서 제거된다.
func TestHealthChecker_SweepPrunesDeleted(t *testing.T) {
	lister := &fakeLister{targets: []ruletypes.RemediationTarget{
		healthTarget("t1", "SHA256:abc"), healthTarget("t2", "SHA256:abc"),
	}}
	c := NewHealthChecker(lister, probeReturning("SHA256:abc", nil), time.Hour)
	c.SweepOnce(context.Background())
	if len(c.Snapshot()) != 2 {
		t.Fatalf("want 2 states, got %d", len(c.Snapshot()))
	}
	lister.set([]ruletypes.RemediationTarget{healthTarget("t1", "SHA256:abc")}, nil)
	c.SweepOnce(context.Background())
	snap := c.Snapshot()
	if _, ok := snap["t2"]; ok {
		t.Fatal("deleted target t2 must be pruned")
	}
	if _, ok := snap["t1"]; !ok {
		t.Fatal("t1 must survive")
	}
}

// ListAll 실패 시 기존 상태 유지 (해당 순회만 skip).
func TestHealthChecker_ListAllErrorKeepsStates(t *testing.T) {
	lister := &fakeLister{targets: []ruletypes.RemediationTarget{healthTarget("t1", "SHA256:abc")}}
	c := NewHealthChecker(lister, probeReturning("SHA256:abc", nil), time.Hour)
	c.SweepOnce(context.Background())
	lister.set(nil, errors.New("db down"))
	c.SweepOnce(context.Background())
	if got := c.Snapshot()["t1"].Status; got != TargetHealthHealthy {
		t.Fatalf("states must survive a failed ListAll, got %q", got)
	}
}

// CheckedAt이 3×interval보다 오래되면 Snapshot이 unknown으로 강등.
func TestHealthChecker_SnapshotDemotesStale(t *testing.T) {
	lister := &fakeLister{targets: []ruletypes.RemediationTarget{healthTarget("t1", "SHA256:abc")}}
	c := NewHealthChecker(lister, probeReturning("SHA256:abc", nil), time.Minute)
	base := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return base }
	c.SweepOnce(context.Background())
	if got := c.Snapshot()["t1"].Status; got != TargetHealthHealthy {
		t.Fatalf("fresh state: got %q", got)
	}
	c.now = func() time.Time { return base.Add(3*time.Minute + time.Second) }
	if got := c.Snapshot()["t1"].Status; got != TargetHealthUnknown {
		t.Fatalf("stale state must demote to unknown, got %q", got)
	}
}

// nil 리시버는 전 메서드 no-op.
func TestHealthChecker_NilReceiverSafe(t *testing.T) {
	var c *HealthChecker
	if c.Snapshot() != nil {
		t.Fatal("nil checker Snapshot must return nil")
	}
	c.Poke(healthTarget("t1", "SHA256:abc")) // panic하지 않으면 통과
	c.SweepOnce(context.Background())
}

// pokeSync는 순회 없이 단건만 갱신한다.
func TestHealthChecker_PokeSyncUpdatesSingleTarget(t *testing.T) {
	lister := &fakeLister{}
	c := NewHealthChecker(lister, probeReturning("SHA256:abc", nil), time.Hour)
	c.pokeSync(context.Background(), healthTarget("t9", "SHA256:abc"))
	if got := c.Snapshot()["t9"].Status; got != TargetHealthHealthy {
		t.Fatalf("pokeSync must store the probe result, got %q", got)
	}
}

// 이전 순회 진행 중이면 SweepOnce는 프로브 없이 즉시 반환.
func TestHealthChecker_SweepSkipsWhenInFlight(t *testing.T) {
	var calls atomic.Int32
	lister := &fakeLister{targets: []ruletypes.RemediationTarget{healthTarget("t1", "SHA256:abc")}}
	c := NewHealthChecker(lister, func(context.Context, string, int, time.Duration) (string, string, error) {
		calls.Add(1)
		return "SHA256:abc", "ssh-ed25519", nil
	}, time.Hour)
	c.sweeping.Store(true)
	c.SweepOnce(context.Background())
	if calls.Load() != 0 {
		t.Fatalf("in-flight sweep must be skipped, probe called %d times", calls.Load())
	}
}

// 프로브 패닉은 프로세스를 죽이지 않고 unreachable로 격리된다.
func TestHealthChecker_ProbePanicIsolated(t *testing.T) {
	lister := &fakeLister{targets: []ruletypes.RemediationTarget{healthTarget("t1", "SHA256:abc")}}
	c := NewHealthChecker(lister, func(context.Context, string, int, time.Duration) (string, string, error) {
		panic("boom")
	}, time.Hour)
	c.SweepOnce(context.Background()) // panic이 전파되면 여기서 테스트가 죽는다
	got := c.Snapshot()["t1"]
	if got.Status != TargetHealthUnreachable {
		t.Fatalf("panicking probe must read as unreachable, got %q", got.Status)
	}
	if got.Error == "" {
		t.Fatal("panic detail must be preserved in Error")
	}
}

// interval < 10초는 무시하고 기본 300초 사용.
func TestHealthChecker_IntervalClamp(t *testing.T) {
	lister := &fakeLister{}
	if got := NewHealthChecker(lister, nil, 5*time.Second).interval; got != 300*time.Second {
		t.Fatalf("sub-10s interval must clamp to 300s, got %v", got)
	}
	if got := NewHealthChecker(lister, nil, 0).interval; got != 300*time.Second {
		t.Fatalf("zero interval must default to 300s, got %v", got)
	}
	if got := NewHealthChecker(lister, nil, time.Minute).interval; got != time.Minute {
		t.Fatalf("valid interval must be kept, got %v", got)
	}
}
