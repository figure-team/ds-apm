// Package aihistorystoretest provides an in-memory fake of
// ruletypes.AIStrategyHistoryStore for tests that need a working store
// without a real database.
package aihistorystoretest

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// Fake is an in-memory AIStrategyHistoryStore. Safe for concurrent use.
type Fake struct {
	mu            sync.RWMutex
	byIncident    map[string]ruletypes.AIStrategyHistoryRecord // "org\x00incidentID" -> record
	byFingerprint map[string]ruletypes.AIStrategyHistoryRecord // "org\x00fingerprint" -> record
}

func New() *Fake {
	return &Fake{
		byIncident:    map[string]ruletypes.AIStrategyHistoryRecord{},
		byFingerprint: map[string]ruletypes.AIStrategyHistoryRecord{},
	}
}

func (f *Fake) Upsert(_ context.Context, orgID string, record ruletypes.AIStrategyHistoryRecord) error {
	if strings.TrimSpace(orgID) == "" {
		return errors.New("orgID required")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if record.IncidentID != "" {
		f.byIncident[orgID+"\x00"+record.IncidentID] = record
	}
	if record.AlertFingerprint != "" {
		f.byFingerprint[orgID+"\x00"+record.AlertFingerprint] = record
	}
	return nil
}

func (f *Fake) GetLatest(_ context.Context, orgID string, lookup ruletypes.AIStrategyHistoryLookupRequest) (ruletypes.AIStrategyHistoryRecord, bool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if id := strings.TrimSpace(lookup.IncidentID); id != "" {
		if rec, ok := f.byIncident[orgID+"\x00"+id]; ok {
			return rec, true, nil
		}
		return ruletypes.AIStrategyHistoryRecord{}, false, nil
	}
	if fp := strings.TrimSpace(lookup.AlertFingerprint); fp != "" {
		if rec, ok := f.byFingerprint[orgID+"\x00"+fp]; ok {
			return rec, true, nil
		}
		return ruletypes.AIStrategyHistoryRecord{}, false, nil
	}
	return ruletypes.AIStrategyHistoryRecord{}, false, errors.New("history lookup: incidentId or alertFingerprint required")
}

var _ ruletypes.AIStrategyHistoryStore = (*Fake)(nil)
