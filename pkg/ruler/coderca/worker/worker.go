// Package worker runs the coderca engine as a SigNoz factory.Service: a
// polling loop that drains queued runs and periodically reaps expired leases
// (design §5.1 worker pool / §6.3 reaper).
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
)

const (
	DefaultPollEvery   = 5 * time.Second
	DefaultReapEvery   = time.Minute
	DefaultMaxAttempts = 2 // design §6.3
)

// Engine processes one queued run (satisfied by *engine.Engine).
type Engine interface {
	ProcessNext(ctx context.Context) (bool, error)
}

// Reaper sweeps expired leases (satisfied by *runstore.Store).
type Reaper interface {
	Reap(ctx context.Context, p runstore.ReapParams) (int, error)
}

// Worker is the coderca background service.
type Worker struct {
	engine      Engine
	reaper      Reaper
	scope       string
	maxAttempts int
	pollEvery   time.Duration
	reapEvery   time.Duration
	now         func() time.Time
	logger      *slog.Logger
	stop        chan struct{}
	done        chan struct{}
}

// compile-time assertion: Worker satisfies factory.Service.
var _ factory.Service = (*Worker)(nil)

// New builds a Worker. Zero durations fall back to defaults; logger/now may be nil.
func New(engine Engine, reaper Reaper, scope string, pollEvery, reapEvery time.Duration, logger *slog.Logger, now func() time.Time) *Worker {
	if pollEvery <= 0 {
		pollEvery = DefaultPollEvery
	}
	if reapEvery <= 0 {
		reapEvery = DefaultReapEvery
	}
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	if scope == "" {
		scope = "global"
	}
	return &Worker{
		engine: engine, reaper: reaper, scope: scope,
		maxAttempts: DefaultMaxAttempts,
		pollEvery:   pollEvery, reapEvery: reapEvery,
		now:    now,
		logger: logger.With(slog.String("component", "ds-apm-coderca-worker")),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

// Start blocks (factory.Service contract) until Stop or ctx cancel.
func (w *Worker) Start(ctx context.Context) error {
	defer close(w.done)
	poll := time.NewTicker(w.pollEvery)
	defer poll.Stop()
	reap := time.NewTicker(w.reapEvery)
	defer reap.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-w.stop:
			return nil
		case <-poll.C:
			w.drain(ctx)
		case <-reap.C:
			if _, err := w.reaper.Reap(ctx, runstore.ReapParams{Scope: w.scope, Now: w.now(), MaxAttempts: w.maxAttempts}); err != nil {
				w.logger.WarnContext(ctx, "coderca worker: reap failed", slog.Any("err", err))
			}
		}
	}
}

// drain processes queued runs until the queue is empty or capacity is hit.
func (w *Worker) drain(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		default:
		}
		processed, err := w.engine.ProcessNext(ctx)
		if err != nil {
			w.logger.WarnContext(ctx, "coderca worker: process failed", slog.Any("err", err))
			return // retry on next tick
		}
		if !processed {
			return
		}
	}
}

// Stop signals Start to return and waits for it.
func (w *Worker) Stop(ctx context.Context) error {
	close(w.stop)
	select {
	case <-w.done:
	case <-ctx.Done():
	}
	return nil
}
