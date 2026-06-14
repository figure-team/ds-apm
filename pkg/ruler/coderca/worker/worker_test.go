package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
)

// fakeEngine records ProcessNext call count.
// It returns processed=true for the first `trueFor` calls, then false.
// An error is returned on every call when errOn is true.
type fakeEngine struct {
	mu     sync.Mutex
	calls  int
	trueFor int
	errOn  bool
}

func (f *fakeEngine) ProcessNext(_ context.Context) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.errOn {
		return false, errors.New("engine error")
	}
	if f.calls <= f.trueFor {
		return true, nil
	}
	return false, nil
}

func (f *fakeEngine) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// fakeReaper records Reap calls and the last ReapParams.
type fakeReaper struct {
	mu       sync.Mutex
	calls    int
	lastParams runstore.ReapParams
}

func (f *fakeReaper) Reap(_ context.Context, p runstore.ReapParams) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.lastParams = p
	return 0, nil
}

func (f *fakeReaper) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeReaper) LastParams() runstore.ReapParams {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastParams
}

// TestWorkerDrainsQueueOnTick: pollEvery=10ms, fakeEngine returns true for first 3
// calls (simulating 3 queued runs) then false. After ~100ms, ProcessNext should be
// called at least 4 times (3 drain + at least 1 idle poll).
func TestWorkerDrainsQueueOnTick(t *testing.T) {
	const trueFor = 3
	eng := &fakeEngine{trueFor: trueFor}
	reap := &fakeReaper{}

	w := New(eng, reap, "global", 10*time.Millisecond, time.Hour, nil, nil)

	ctx := context.Background()
	go func() { _ = w.Start(ctx) }()

	time.Sleep(100 * time.Millisecond)
	_ = w.Stop(ctx)

	got := eng.Calls()
	// At minimum: 3 drain calls + 1 idle call (queue empty) = 4
	if got < trueFor+1 {
		t.Errorf("ProcessNext called %d times, want >= %d", got, trueFor+1)
	}
}

// TestWorkerReapsPeriodically: reapEvery=10ms; after Stop, fakeReaper.calls >= 1
// and last ReapParams.MaxAttempts == DefaultMaxAttempts (2).
func TestWorkerReapsPeriodically(t *testing.T) {
	eng := &fakeEngine{}
	reap := &fakeReaper{}

	w := New(eng, reap, "test-scope", time.Hour, 10*time.Millisecond, nil, nil)

	ctx := context.Background()
	go func() { _ = w.Start(ctx) }()

	time.Sleep(100 * time.Millisecond)
	_ = w.Stop(ctx)

	if reap.Calls() < 1 {
		t.Errorf("Reap called %d times, want >= 1", reap.Calls())
	}
	if p := reap.LastParams(); p.MaxAttempts != DefaultMaxAttempts {
		t.Errorf("ReapParams.MaxAttempts = %d, want %d", p.MaxAttempts, DefaultMaxAttempts)
	}
}

// TestWorkerStopUnblocksStart: Start is blocking; Stop must cause Start to return
// within 1 second.
func TestWorkerStopUnblocksStart(t *testing.T) {
	eng := &fakeEngine{}
	reap := &fakeReaper{}

	w := New(eng, reap, "global", time.Hour, time.Hour, nil, nil)

	ctx := context.Background()
	done := make(chan struct{})
	go func() {
		_ = w.Start(ctx)
		close(done)
	}()

	// Give Start a moment to enter its select loop.
	time.Sleep(10 * time.Millisecond)
	_ = w.Stop(ctx)

	select {
	case <-done:
		// OK: Start returned
	case <-time.After(1 * time.Second):
		t.Fatal("Start did not return within 1s after Stop")
	}
}

// TestWorkerSurvivesEngineError: even when ProcessNext returns an error, the worker
// must keep running and call ProcessNext again on subsequent ticks.
func TestWorkerSurvivesEngineError(t *testing.T) {
	eng := &fakeEngine{errOn: true}
	reap := &fakeReaper{}

	w := New(eng, reap, "global", 10*time.Millisecond, time.Hour, nil, nil)

	ctx := context.Background()
	go func() { _ = w.Start(ctx) }()

	// Wait long enough to get multiple ticks.
	time.Sleep(80 * time.Millisecond)
	_ = w.Stop(ctx)

	got := eng.Calls()
	// With 10ms poll and 80ms wait, we expect at least 3 calls.
	if got < 3 {
		t.Errorf("ProcessNext called %d times after errors, want >= 3 (loop must continue)", got)
	}
}
