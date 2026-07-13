package remediation

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// TargetHealthStatus is the badge-level health of a remediation target
// (spec §2.2). Exactly four values; the wire layer serializes them verbatim.
type TargetHealthStatus string

const (
	TargetHealthHealthy     TargetHealthStatus = "healthy"
	TargetHealthUnreachable TargetHealthStatus = "unreachable"
	TargetHealthMismatch    TargetHealthStatus = "mismatch"
	TargetHealthUnknown     TargetHealthStatus = "unknown"
)

// TargetHealth is one target's last probe outcome.
type TargetHealth struct {
	Status    TargetHealthStatus
	Error     string    // unreachable/mismatch일 때 상세
	CheckedAt time.Time // zero면 미확인
}

// ProbeFunc fetches a host-key fingerprint without authenticating.
// nil → FetchHostKeyFingerprint. The seam exists for tests.
type ProbeFunc func(ctx context.Context, host string, port int, timeout time.Duration) (string, string, error)

// targetLister is the checker's own narrow view of the target store.
// Deliberately NOT remediationtargetstore.Store — adding a method to that
// interface breaks every fake in other packages (spec §2.3).
type targetLister interface {
	ListAll(ctx context.Context) ([]ruletypes.RemediationTarget, error)
}

const (
	defaultHealthInterval = 300 * time.Second // fail2ban 기본 임계(5회/10분) 아래 (spec §5)
	minHealthInterval     = 10 * time.Second  // 이보다 짧은 설정은 무시 — 타겟 두들김 방지
	healthDialTimeout     = 5 * time.Second
	healthProbeParallel   = 8
	healthStaleFactor     = 3 // CheckedAt이 이 배수×interval보다 오래되면 unknown 강등
)

// HealthChecker periodically probes every remediation target with the
// no-auth host-key probe and keeps the latest result in memory (spec §2).
// All exported methods are nil-receiver safe so the wire layer can hold a
// concrete *HealthChecker and call through without nil branches (spec §2.4).
type HealthChecker struct {
	lister   targetLister
	probe    ProbeFunc
	interval time.Duration
	now      func() time.Time

	sweeping atomic.Bool
	mu       sync.RWMutex
	states   map[string]TargetHealth // key = target ID
}

// NewHealthChecker builds a checker. probe nil → FetchHostKeyFingerprint;
// interval < 10s(미설정 포함) → 300s.
func NewHealthChecker(lister targetLister, probe ProbeFunc, interval time.Duration) *HealthChecker {
	if probe == nil {
		probe = FetchHostKeyFingerprint
	}
	if interval < minHealthInterval {
		interval = defaultHealthInterval
	}
	return &HealthChecker{
		lister:   lister,
		probe:    probe,
		interval: interval,
		now:      time.Now,
		states:   map[string]TargetHealth{},
	}
}

// Run sweeps immediately, then on every interval tick until ctx is done.
func (c *HealthChecker) Run(ctx context.Context) {
	if c == nil {
		return
	}
	c.SweepOnce(ctx)
	t := time.NewTicker(c.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.SweepOnce(ctx)
		}
	}
}

// SweepOnce probes every listed target (bounded concurrency), merges results,
// and prunes states for targets that no longer exist. A sweep already in
// flight makes this call a no-op (tick skip, spec §2.2).
func (c *HealthChecker) SweepOnce(ctx context.Context) {
	if c == nil || !c.sweeping.CompareAndSwap(false, true) {
		return
	}
	defer c.sweeping.Store(false)

	targets, err := c.lister.ListAll(ctx)
	if err != nil {
		return // 기존 상태 유지 — 다음 tick에 재시도 (spec §2.2)
	}

	results := make(map[string]TargetHealth, len(targets))
	var (
		wg  sync.WaitGroup
		sem = make(chan struct{}, healthProbeParallel)
		rmu sync.Mutex
	)
	for _, t := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(t ruletypes.RemediationTarget) {
			defer wg.Done()
			defer func() { <-sem }()
			h := c.probeOne(ctx, t)
			rmu.Lock()
			results[t.ID] = h
			rmu.Unlock()
		}(t)
	}
	wg.Wait()

	listed := make(map[string]struct{}, len(targets))
	for _, t := range targets {
		listed[t.ID] = struct{}{}
	}
	c.mu.Lock()
	for id := range c.states {
		if _, ok := listed[id]; !ok {
			delete(c.states, id) // 삭제된 타겟 prune
		}
	}
	for id, h := range results {
		c.states[id] = h
	}
	c.mu.Unlock()
}

// probeOne judges a single target (spec §2.2 판정표). A panicking probe must
// not take the process down (spec §2.2 에러 격리) — it reads as unreachable.
func (c *HealthChecker) probeOne(ctx context.Context, t ruletypes.RemediationTarget) (h TargetHealth) {
	defer func() {
		if r := recover(); r != nil {
			h = TargetHealth{
				Status:    TargetHealthUnreachable,
				Error:     fmt.Sprintf("probe panic: %v", r),
				CheckedAt: c.now().UTC(),
			}
		}
	}()
	fp, _, err := c.probe(ctx, t.Host, t.Port, healthDialTimeout)
	checked := c.now().UTC()
	if err != nil {
		return TargetHealth{Status: TargetHealthUnreachable, Error: err.Error(), CheckedAt: checked}
	}
	want := strings.TrimSpace(t.HostKeyFingerprint)
	if want != "" && fp != want {
		return TargetHealth{
			Status:    TargetHealthMismatch,
			Error:     fmt.Sprintf("host key fingerprint mismatch: got %s, want %s", fp, want),
			CheckedAt: checked,
		}
	}
	return TargetHealth{Status: TargetHealthHealthy, CheckedAt: checked}
}

// Snapshot returns a copy of the current states. Entries older than
// 3×interval demote to unknown — a stalled checker must not present stale
// '정상' as fresh (spec §2.2). nil receiver → nil map (fail-open, spec §3).
func (c *HealthChecker) Snapshot() map[string]TargetHealth {
	if c == nil {
		return nil
	}
	cutoff := c.now().UTC().Add(-healthStaleFactor * c.interval)
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]TargetHealth, len(c.states))
	for id, h := range c.states {
		if h.CheckedAt.Before(cutoff) {
			h = TargetHealth{Status: TargetHealthUnknown}
		}
		out[id] = h
	}
	return out
}

// Poke asynchronously probes one target right now — the create/update
// handlers call this so a fresh target doesn't wait a full interval for its
// first badge (spec §2.2). Fire-and-forget: 실패해도 핸들러 응답과 무관.
func (c *HealthChecker) Poke(t ruletypes.RemediationTarget) {
	if c == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), healthDialTimeout+time.Second)
		defer cancel()
		c.pokeSync(ctx, t)
	}()
}

func (c *HealthChecker) pokeSync(ctx context.Context, t ruletypes.RemediationTarget) {
	h := c.probeOne(ctx, t)
	c.mu.Lock()
	c.states[t.ID] = h
	c.mu.Unlock()
}
