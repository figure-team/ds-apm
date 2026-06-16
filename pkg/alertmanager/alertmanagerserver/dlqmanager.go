// pkg/alertmanager/alertmanagerserver/dlqmanager.go
package alertmanagerserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/prometheus/alertmanager/types"

	"github.com/SigNoz/signoz/pkg/alertmanager/alertmanagernotify/dlq"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
)

// FailureSidecar is an append-only set that records event IDs whose replay
// attempt failed. It mirrors the ReplayLedger design: load from disk on start,
// then append-only writes, in-memory set for O(1) lookup.
type FailureSidecar struct {
	mu   sync.Mutex
	seen map[string]struct{}
	f    *os.File
	w    *bufio.Writer
}

// newFailureSidecar opens (creating if necessary) the sidecar at path and
// rebuilds the in-memory set from existing contents.
func newFailureSidecar(path string) (*FailureSidecar, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("failure sidecar: mkdirall: %w", err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failure sidecar: open: %w", err)
	}
	seen := make(map[string]struct{})
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			seen[line] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failure sidecar: scan: %w", err)
	}
	if _, err := f.Seek(0, 2); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failure sidecar: seek end: %w", err)
	}
	return &FailureSidecar{seen: seen, f: f, w: bufio.NewWriter(f)}, nil
}

// Record durably appends eventID to the sidecar. Silently no-ops for
// empty IDs or when the file is closed.
func (s *FailureSidecar) Record(eventID string) {
	if eventID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f == nil {
		return
	}
	if _, ok := s.seen[eventID]; ok {
		return
	}
	if _, err := fmt.Fprintln(s.w, eventID); err != nil {
		return
	}
	_ = s.w.Flush()
	s.seen[eventID] = struct{}{}
}

// Has reports whether eventID has been recorded as a replay failure.
func (s *FailureSidecar) Has(eventID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.seen[eventID]
	return ok
}

// Close flushes and closes the underlying file.
func (s *FailureSidecar) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f == nil {
		return nil
	}
	flushErr := s.w.Flush()
	closeErr := s.f.Close()
	s.f = nil
	if flushErr != nil {
		return flushErr
	}
	return closeErr
}

// DLQManager coordinates reading, status-merging, and idempotent replay of
// dead-letter entries. It is constructed once per Server and shared across
// API calls. Thread-safe.
type DLQManager struct {
	dlqPath  string
	ledger   *dlq.ReplayLedger
	sidecar  *FailureSidecar
	notifyFn func(ctx context.Context, channel string, alerts []*types.Alert) error
}

// newDLQManager opens the replay ledger and failure sidecar derived from
// dlqPath and returns a ready DLQManager.
func newDLQManager(
	dlqPath string,
	notifyFn func(ctx context.Context, channel string, alerts []*types.Alert) error,
) (*DLQManager, error) {
	ledger, err := dlq.NewReplayLedger(dlqPath + ".replay-ledger")
	if err != nil {
		return nil, fmt.Errorf("dlq manager: ledger: %w", err)
	}
	sidecar, err := newFailureSidecar(dlqPath + ".replay-failures")
	if err != nil {
		_ = ledger.Close()
		return nil, fmt.Errorf("dlq manager: sidecar: %w", err)
	}
	return &DLQManager{
		dlqPath:  dlqPath,
		ledger:   ledger,
		sidecar:  sidecar,
		notifyFn: notifyFn,
	}, nil
}

func (m *DLQManager) status(eventID string) string {
	if m.ledger.Has(eventID) {
		return "replayed"
	}
	if m.sidecar.Has(eventID) {
		return "replay_failed"
	}
	return "pending"
}

// ListEntries reads all DLQ entries and returns them filtered by optional
// channel and status. Empty string means "no filter".
func (m *DLQManager) ListEntries(channel, status string) ([]*alertmanagertypes.DLQEntry, error) {
	raw, err := dlq.ReadEntries(m.dlqPath)
	if err != nil {
		return nil, err
	}
	out := make([]*alertmanagertypes.DLQEntry, 0, len(raw))
	for _, e := range raw {
		s := m.status(e.EventID)
		if channel != "" && e.Channel != channel {
			continue
		}
		if status != "" && s != status {
			continue
		}
		out = append(out, &alertmanagertypes.DLQEntry{
			EventID:  e.EventID,
			Channel:  e.Channel,
			Payload:  e.Payload,
			FailedAt: e.FailedAt,
			Reason:   e.Reason,
			Status:   s,
		})
	}
	return out, nil
}

// ReplayEntries re-delivers the entries matching eventIDs. Idempotency is
// guaranteed via the ReplayLedger: duplicate calls are counted as skipped.
func (m *DLQManager) ReplayEntries(ctx context.Context, eventIDs []string) (*alertmanagertypes.ReplayResult, error) {
	raw, err := dlq.ReadEntries(m.dlqPath)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*dlq.Entry, len(raw))
	for _, e := range raw {
		byID[e.EventID] = e
	}

	result := &alertmanagertypes.ReplayResult{}
	for _, id := range eventIDs {
		e, ok := byID[id]
		if !ok {
			result.Skipped++
			continue
		}
		key := dlq.IdempotencyKey(e.EventID, e.Channel, 0)
		if !m.ledger.MarkIfNew(key) {
			result.Skipped++
			continue
		}
		var alerts []*types.Alert
		if err := json.Unmarshal(e.Payload, &alerts); err != nil {
			m.sidecar.Record(e.EventID)
			result.Failed++
			continue
		}
		if err := m.notifyFn(ctx, e.Channel, alerts); err != nil {
			m.sidecar.Record(e.EventID)
			result.Failed++
		} else {
			result.Replayed++
		}
	}
	return result, nil
}

// Close releases the ledger and sidecar file handles.
func (m *DLQManager) Close() error {
	ledgerErr := m.ledger.Close()
	sidecarErr := m.sidecar.Close()
	if ledgerErr != nil {
		return ledgerErr
	}
	return sidecarErr
}
