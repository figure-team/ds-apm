// pkg/alertmanager/alertmanagerserver/dlqmanager_test.go
package alertmanagerserver

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/alertmanager/alertmanagernotify/dlq"
)

func writeDLQEntry(t *testing.T, path string, e *dlq.Entry) {
	t.Helper()
	sink, err := dlq.NewJSONLDeadLetterSink(path, dlq.DefaultJSONLDeadLetterMaxSizeBytes)
	require.NoError(t, err)
	require.NoError(t, sink.Write(e))
	require.NoError(t, sink.Close())
}

func makeAlertPayload(t *testing.T) []byte {
	t.Helper()
	alerts := []*types.Alert{
		{Alert: model.Alert{Labels: model.LabelSet{"alertname": "test"}}},
	}
	b, err := json.Marshal(alerts)
	require.NoError(t, err)
	return b
}

func TestDLQManagerListEntries_StatusMerge(t *testing.T) {
	dir := t.TempDir()
	dlqPath := filepath.Join(dir, "alert-dlq.jsonl")

	writeDLQEntry(t, dlqPath, &dlq.Entry{
		EventID:  "evt-pending",
		Channel:  "slack",
		Payload:  makeAlertPayload(t),
		FailedAt: time.Now(),
		Reason:   "timeout",
	})
	writeDLQEntry(t, dlqPath, &dlq.Entry{
		EventID:  "evt-replayed",
		Channel:  "webhook",
		Payload:  makeAlertPayload(t),
		FailedAt: time.Now(),
		Reason:   "connection refused",
	})
	writeDLQEntry(t, dlqPath, &dlq.Entry{
		EventID:  "evt-failed",
		Channel:  "slack",
		Payload:  makeAlertPayload(t),
		FailedAt: time.Now(),
		Reason:   "5xx",
	})

	notifyFn := func(_ context.Context, _ string, _ []*types.Alert) error { return nil }
	mgr, err := newDLQManager(dlqPath, notifyFn)
	require.NoError(t, err)
	defer mgr.Close() //nolint:errcheck

	// mark evt-replayed in ledger, evt-failed in sidecar
	require.True(t, mgr.ledger.MarkIfNew(dlq.IdempotencyKey("evt-replayed", "webhook", 0)))
	mgr.sidecar.Record("evt-failed")

	entries, err := mgr.ListEntries("", "")
	require.NoError(t, err)
	require.Len(t, entries, 3)

	byID := make(map[string]string)
	for _, e := range entries {
		byID[e.EventID] = e.Status
	}
	require.Equal(t, "pending", byID["evt-pending"])
	require.Equal(t, "replayed", byID["evt-replayed"])
	require.Equal(t, "replay_failed", byID["evt-failed"])
}

func TestDLQManagerListEntries_ChannelFilter(t *testing.T) {
	dir := t.TempDir()
	dlqPath := filepath.Join(dir, "alert-dlq.jsonl")

	writeDLQEntry(t, dlqPath, &dlq.Entry{EventID: "a", Channel: "slack", Payload: makeAlertPayload(t), FailedAt: time.Now()})
	writeDLQEntry(t, dlqPath, &dlq.Entry{EventID: "b", Channel: "webhook", Payload: makeAlertPayload(t), FailedAt: time.Now()})

	mgr, err := newDLQManager(dlqPath, func(_ context.Context, _ string, _ []*types.Alert) error { return nil })
	require.NoError(t, err)
	defer mgr.Close() //nolint:errcheck

	entries, err := mgr.ListEntries("slack", "")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "a", entries[0].EventID)
}

func TestDLQManagerReplay_SuccessAndIdempotency(t *testing.T) {
	dir := t.TempDir()
	dlqPath := filepath.Join(dir, "alert-dlq.jsonl")

	writeDLQEntry(t, dlqPath, &dlq.Entry{
		EventID:  "evt-1",
		Channel:  "slack",
		Payload:  makeAlertPayload(t),
		FailedAt: time.Now(),
	})

	calls := 0
	notifyFn := func(_ context.Context, _ string, _ []*types.Alert) error {
		calls++
		return nil
	}
	mgr, err := newDLQManager(dlqPath, notifyFn)
	require.NoError(t, err)
	defer mgr.Close() //nolint:errcheck

	// First replay
	result, err := mgr.ReplayEntries(context.Background(), []string{"evt-1"})
	require.NoError(t, err)
	require.Equal(t, 1, result.Replayed)
	require.Equal(t, 0, result.Skipped)
	require.Equal(t, 0, result.Failed)
	require.Equal(t, 1, calls)

	// Idempotent: second replay skips
	result2, err := mgr.ReplayEntries(context.Background(), []string{"evt-1"})
	require.NoError(t, err)
	require.Equal(t, 0, result2.Replayed)
	require.Equal(t, 1, result2.Skipped)
	require.Equal(t, 1, calls, "notifyFn must not be called again")
}

func TestDLQManagerReplay_NotifyFailureRecordedInSidecar(t *testing.T) {
	dir := t.TempDir()
	dlqPath := filepath.Join(dir, "alert-dlq.jsonl")

	writeDLQEntry(t, dlqPath, &dlq.Entry{
		EventID:  "evt-bad",
		Channel:  "slack",
		Payload:  makeAlertPayload(t),
		FailedAt: time.Now(),
	})

	notifyFn := func(_ context.Context, _ string, _ []*types.Alert) error {
		return errors.New("downstream unavailable")
	}
	mgr, err := newDLQManager(dlqPath, notifyFn)
	require.NoError(t, err)
	defer mgr.Close() //nolint:errcheck

	result, err := mgr.ReplayEntries(context.Background(), []string{"evt-bad"})
	require.NoError(t, err)
	require.Equal(t, 1, result.Failed)
	require.True(t, mgr.sidecar.Has("evt-bad"))
}

func TestFailureSidecarPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "failures")

	s1, err := newFailureSidecar(path)
	require.NoError(t, err)
	s1.Record("evt-x")
	require.NoError(t, s1.Close())

	// Re-open: should still have the record
	s2, err := newFailureSidecar(path)
	require.NoError(t, err)
	defer s2.Close() //nolint:errcheck
	require.True(t, s2.Has("evt-x"))
}

func TestDLQManagerReplay_UnknownIDCountedAsSkipped(t *testing.T) {
	dir := t.TempDir()
	dlqPath := filepath.Join(dir, "alert-dlq.jsonl")

	// Write nothing — empty DLQ
	require.NoError(t, os.WriteFile(dlqPath, nil, 0o644))

	mgr, err := newDLQManager(dlqPath, func(_ context.Context, _ string, _ []*types.Alert) error { return nil })
	require.NoError(t, err)
	defer mgr.Close() //nolint:errcheck

	result, err := mgr.ReplayEntries(context.Background(), []string{"does-not-exist"})
	require.NoError(t, err)
	require.Equal(t, 1, result.Skipped)
}

func TestDLQManagerReplay_FailureIsRetryableAndStatusReflected(t *testing.T) {
	dir := t.TempDir()
	dlqPath := filepath.Join(dir, "alert-dlq.jsonl")

	writeDLQEntry(t, dlqPath, &dlq.Entry{
		EventID:  "evt-retry",
		Channel:  "slack",
		Payload:  makeAlertPayload(t),
		FailedAt: time.Now(),
	})

	fail := true
	notifyFn := func(_ context.Context, _ string, _ []*types.Alert) error {
		if fail {
			return errors.New("downstream unavailable")
		}
		return nil
	}
	mgr, err := newDLQManager(dlqPath, notifyFn)
	require.NoError(t, err)
	defer mgr.Close() //nolint:errcheck

	// First attempt fails: counted as failed, NOT marked replayed.
	res1, err := mgr.ReplayEntries(context.Background(), []string{"evt-retry"})
	require.NoError(t, err)
	require.Equal(t, 1, res1.Failed)
	require.Equal(t, 0, res1.Replayed)

	// Status reflects the failure, not "replayed".
	entries, err := mgr.ListEntries("", "")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "replay_failed", entries[0].Status)

	// A failed entry remains retryable: a later successful attempt replays it.
	fail = false
	res2, err := mgr.ReplayEntries(context.Background(), []string{"evt-retry"})
	require.NoError(t, err)
	require.Equal(t, 1, res2.Replayed)
	require.Equal(t, 0, res2.Skipped)

	// Now status is "replayed".
	entries2, err := mgr.ListEntries("", "")
	require.NoError(t, err)
	require.Equal(t, "replayed", entries2[0].Status)

	// Idempotent: a third attempt is skipped.
	res3, err := mgr.ReplayEntries(context.Background(), []string{"evt-retry"})
	require.NoError(t, err)
	require.Equal(t, 1, res3.Skipped)
	require.Equal(t, 0, res3.Replayed)
}
