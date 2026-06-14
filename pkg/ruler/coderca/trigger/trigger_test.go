package trigger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/ruler/coderca"
	"github.com/SigNoz/signoz/pkg/ruler/coderca/runstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

// fakeCfgStore returns a fixed (cfg, err) pair.
type fakeCfgStore struct {
	cfg     ruletypes.CodebaseRCAConfig
	err     error
	sleepFn func(ctx context.Context) // optional hook to simulate slow Get
}

func (f *fakeCfgStore) Get(ctx context.Context, orgID string) (ruletypes.CodebaseRCAConfig, error) {
	if f.sleepFn != nil {
		f.sleepFn(ctx)
	}
	return f.cfg, f.err
}

func (f *fakeCfgStore) Upsert(ctx context.Context, cfg ruletypes.CodebaseRCAConfig) error {
	return nil
}

// fakeMaps returns nil error when service is in the mapped set, else ErrCodebaseServiceMapNotFound.
type fakeMaps struct {
	mapped map[string]bool // service name → mapped?
}

func (f *fakeMaps) Get(ctx context.Context, orgID, serviceName string) (ruletypes.CodebaseServiceMap, error) {
	if f.mapped[serviceName] {
		return ruletypes.CodebaseServiceMap{}, nil
	}
	return ruletypes.CodebaseServiceMap{}, ruletypes.ErrCodebaseServiceMapNotFound
}

func (f *fakeMaps) Upsert(ctx context.Context, m ruletypes.CodebaseServiceMap) error { return nil }
func (f *fakeMaps) Delete(_ context.Context, _, _ string) error                      { return nil }
func (f *fakeMaps) List(ctx context.Context, orgID string) ([]ruletypes.CodebaseServiceMap, error) {
	return nil, nil
}

// fakeAdmitter records whether Admit/RecordSkip were called and captures AdmitParams.
type fakeAdmitter struct {
	called      bool
	admitParams runstore.AdmitParams
	skipCalled  bool
	skipReason  coderca.SkipReason
	panicOnAdmit bool // if true, Admit panics
}

func (f *fakeAdmitter) Admit(ctx context.Context, p runstore.AdmitParams) (runstore.AdmitResult, error) {
	if f.panicOnAdmit {
		panic("fakeAdmitter: intentional panic")
	}
	f.called = true
	f.admitParams = p
	return runstore.AdmitResult{Admitted: true, RunID: "run-fake-1"}, nil
}

func (f *fakeAdmitter) RecordSkip(ctx context.Context, orgID string, reason coderca.SkipReason, now time.Time) error {
	f.skipCalled = true
	f.skipReason = reason
	return nil
}

// ---------------------------------------------------------------------------
// Label helpers
// ---------------------------------------------------------------------------

func without(m map[string]string, key string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		if k != key {
			out[k] = v
		}
	}
	return out
}

func with(m map[string]string, key, val string) map[string]string {
	out := make(map[string]string, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	out[key] = val
	return out
}

func withFlag(cfg ruletypes.CodebaseRCAConfig) ruletypes.CodebaseRCAConfig {
	cfg.AllowUnboundWithoutAnomaly = true
	return cfg
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestMaybeGateChain(t *testing.T) {
	base := func() ruletypes.CodebaseRCAConfig {
		c := ruletypes.DefaultCodebaseRCAConfig("org-1")
		c.Enabled = true
		return c
	}
	labels := map[string]string{
		"alertname": "PayErr", "service.name": "pay", "severity": "critical", "anomaly": "true",
	}

	cases := []struct {
		name      string
		cfg       ruletypes.CodebaseRCAConfig
		cfgErr    error
		labels    map[string]string
		mapped    bool
		wantAdmit bool
	}{
		{"all gates pass → admit", base(), nil, labels, true, true},
		{"feature off → no admit", ruletypes.DefaultCodebaseRCAConfig("org-1"), nil, labels, true, false},
		{"config store error → fail-closed, no admit", base(), errors.New("db down"), labels, true, false},
		{"no anomaly label → fail-closed", base(), nil, without(labels, "anomaly"), true, false},
		{"anomaly=false → fail-closed", base(), nil, with(labels, "anomaly", "false"), true, false},
		{"below severity → no admit", base(), nil, with(labels, "severity", "warning"), true, false},
		{"severity label absent → fail-closed", base(), nil, without(labels, "severity"), true, false},
		{"no service label → no admit", base(), nil, without(labels, "service.name"), true, false},
		{"unmapped service → no admit (no_repo_mapping)", base(), nil, labels, false, false},
		{"allow_unbound_without_anomaly → anomaly 없이 admit", withFlag(base()), nil, without(labels, "anomaly"), true, true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			admitter := &fakeAdmitter{}
			mapped := map[string]bool{}
			if tc.mapped {
				mapped["pay"] = true
			}
			trig := New(
				&fakeCfgStore{cfg: tc.cfg, err: tc.cfgErr},
				&fakeMaps{mapped: mapped},
				admitter,
				nil,
				func() time.Time { return time.Unix(1_700_000_000, 0) },
			)
			trig.Maybe(context.Background(), "org-1", tc.labels, nil)

			if admitter.called != tc.wantAdmit {
				t.Errorf("Admit called=%v, want %v", admitter.called, tc.wantAdmit)
			}

			if tc.wantAdmit {
				p := admitter.admitParams
				wantDedup := coderca.DedupKey("org-1", "pay", coderca.ErrorSignature(tc.labels))
				if p.DedupKey != wantDedup {
					t.Errorf("DedupKey=%q, want %q", p.DedupKey, wantDedup)
				}
				wantCooldown := time.Duration(tc.cfg.CooldownWindowSecs) * time.Second
				if p.CooldownWindow != wantCooldown {
					t.Errorf("CooldownWindow=%v, want %v", p.CooldownWindow, wantCooldown)
				}
				if p.MaxRunsPerDay != tc.cfg.MaxRunsPerDay {
					t.Errorf("MaxRunsPerDay=%d, want %d", p.MaxRunsPerDay, tc.cfg.MaxRunsPerDay)
				}
			}
		})
	}
}

func TestMaybeNeverPanicsOrBlocks(t *testing.T) {
	t.Run("panic in Admit is recovered", func(t *testing.T) {
		admitter := &fakeAdmitter{panicOnAdmit: true}
		cfg := ruletypes.DefaultCodebaseRCAConfig("org-1")
		cfg.Enabled = true
		trig := New(
			&fakeCfgStore{cfg: cfg},
			&fakeMaps{mapped: map[string]bool{"pay": true}},
			admitter,
			nil,
			nil,
		)
		labels := map[string]string{
			"alertname": "PayErr", "service.name": "pay", "severity": "critical", "anomaly": "true",
		}
		// Must not panic; recover inside Maybe catches it.
		trig.Maybe(context.Background(), "org-1", labels, nil)
	})

	t.Run("slow config store is bounded by maybeTimeout", func(t *testing.T) {
		cfg := ruletypes.DefaultCodebaseRCAConfig("org-1")
		cfg.Enabled = true

		slowStore := &fakeCfgStore{
			cfg: cfg,
			sleepFn: func(ctx context.Context) {
				// Sleep up to 2s; returns early when trigger's internal timeout ctx fires.
				select {
				case <-time.After(2 * time.Second):
				case <-ctx.Done():
				}
			},
		}

		admitter := &fakeAdmitter{}
		trig := New(
			slowStore,
			&fakeMaps{mapped: map[string]bool{"pay": true}},
			admitter,
			nil,
			nil,
		)
		labels := map[string]string{
			"alertname": "PayErr", "service.name": "pay", "severity": "critical", "anomaly": "true",
		}

		start := time.Now()
		trig.Maybe(context.Background(), "org-1", labels, nil)
		elapsed := time.Since(start)

		// maybeTimeout is 1s; allow 500ms overhead for CI.
		if elapsed >= 1500*time.Millisecond {
			t.Errorf("Maybe blocked for %v; expected < 1500ms (maybeTimeout=%v)", elapsed, maybeTimeout)
		}
	})
}

func TestMaybeAnomalyFromAnnotations(t *testing.T) {
	// anomaly only in annotations map → should pass the anomaly gate.
	cfg := ruletypes.DefaultCodebaseRCAConfig("org-1")
	cfg.Enabled = true

	admitter := &fakeAdmitter{}
	trig := New(
		&fakeCfgStore{cfg: cfg},
		&fakeMaps{mapped: map[string]bool{"pay": true}},
		admitter,
		nil,
		nil,
	)

	// labels has NO anomaly key; annotations has anomaly=true.
	labels := map[string]string{
		"alertname": "PayErr", "service.name": "pay", "severity": "critical",
	}
	annotations := map[string]string{
		"anomaly": "true",
	}

	trig.Maybe(context.Background(), "org-1", labels, annotations)

	if !admitter.called {
		t.Error("expected Admit to be called when anomaly is in annotations only, got not called")
	}
}
