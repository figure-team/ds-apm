package sqlaihistorystore

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

type historyStore struct {
	sqlstore sqlstore.SQLStore
}

// NewAIStrategyHistoryStore returns an AIStrategyHistoryStore backed by
// the given SQLStore. Migration 078 must have run; the table
// ds_ai_strategy_history is read directly via bun ORM.
func NewAIStrategyHistoryStore(store sqlstore.SQLStore) ruletypes.AIStrategyHistoryStore {
	return &historyStore{sqlstore: store}
}

func (s *historyStore) Upsert(ctx context.Context, orgID string, record ruletypes.AIStrategyHistoryRecord) error {
	storable, err := ruletypes.FromDomainAIStrategyHistoryRecord(orgID, record)
	if err != nil {
		return err
	}
	return s.sqlstore.RunInTxCtx(ctx, nil, func(ctx context.Context) error {
		_, err := s.sqlstore.BunDBCtx(ctx).
			NewInsert().
			Model(storable).
			On("CONFLICT (org_id, incident_id) DO UPDATE").
			Set("alert_fingerprint = EXCLUDED.alert_fingerprint").
			Set("contract_version = EXCLUDED.contract_version").
			Set("payload = EXCLUDED.payload").
			Exec(ctx)
		return err
	})
}

func (s *historyStore) GetLatest(ctx context.Context, orgID string, lookup ruletypes.AIStrategyHistoryLookupRequest) (ruletypes.AIStrategyHistoryRecord, bool, error) {
	storable := new(ruletypes.StorableAIStrategyHistory)
	q := s.sqlstore.BunDBCtx(ctx).NewSelect().Model(storable).Where("org_id = ?", orgID)

	incidentID := strings.TrimSpace(lookup.IncidentID)
	fingerprint := strings.TrimSpace(lookup.AlertFingerprint)

	switch {
	case incidentID != "":
		q = q.Where("incident_id = ?", incidentID)
	case fingerprint != "":
		q = q.Where("alert_fingerprint = ?", fingerprint)
	default:
		return ruletypes.AIStrategyHistoryRecord{}, false, errors.New("history lookup: incidentId or alertFingerprint required")
	}

	err := q.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return ruletypes.AIStrategyHistoryRecord{}, false, nil
	}
	if err != nil {
		return ruletypes.AIStrategyHistoryRecord{}, false, err
	}
	record, err := storable.ToDomain()
	if err != nil {
		return ruletypes.AIStrategyHistoryRecord{}, false, err
	}
	return record, true, nil
}
