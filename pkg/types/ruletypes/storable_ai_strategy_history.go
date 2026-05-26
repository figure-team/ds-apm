package ruletypes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
)

// StorableAIStrategyHistory is the bun-backed persistence shape for AIStrategyHistoryRecord.
//
// The full domain object is encoded into the payload column as JSON. A small
// set of flat columns (org_id, incident_id, alert_fingerprint, contract_version)
// mirrors the most-queried fields so lookup queries do not have to parse JSON.
// New fields added to AIStrategyHistoryRecord flow through automatically; only
// the indexed columns need explicit handling here.
type StorableAIStrategyHistory struct {
	bun.BaseModel `bun:"table:ds_ai_strategy_history"`

	OrgID            string `bun:"org_id,pk,notnull,type:text"`
	IncidentID       string `bun:"incident_id,pk,notnull,type:text"`
	AlertFingerprint string `bun:"alert_fingerprint,notnull,default:'',type:text"`
	ContractVersion  string `bun:"contract_version,notnull,type:text"`
	Payload          string `bun:"payload,notnull,type:text"`
}

// FromDomainAIStrategyHistoryRecord builds a StorableAIStrategyHistory scoped to orgID.
// The orgID is required (fail-closed); empty orgID returns an error so callers
// cannot accidentally create cross-tenant rows. ContractVersion must also be
// non-empty (upstream NewAIStrategyHistoryRecord should have set it).
func FromDomainAIStrategyHistoryRecord(orgID string, record AIStrategyHistoryRecord) (*StorableAIStrategyHistory, error) {
	if strings.TrimSpace(orgID) == "" {
		return nil, fmt.Errorf("storable ai strategy history: orgID must not be empty")
	}
	contractVersion := strings.TrimSpace(record.ContractVersion)
	if contractVersion == "" {
		return nil, fmt.Errorf("storable ai strategy history: ContractVersion must not be empty (upstream NewAIStrategyHistoryRecord should have caught this)")
	}
	record.ContractVersion = contractVersion // ensure payload sees the trimmed value too

	payload, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("storable ai strategy history: marshal payload: %w", err)
	}
	return &StorableAIStrategyHistory{
		OrgID:            orgID,
		IncidentID:       record.IncidentID,
		AlertFingerprint: record.AlertFingerprint,
		ContractVersion:  contractVersion,
		Payload:          string(payload),
	}, nil
}

// ToDomain decodes the persisted payload back into an AIStrategyHistoryRecord.
func (s *StorableAIStrategyHistory) ToDomain() (AIStrategyHistoryRecord, error) {
	var record AIStrategyHistoryRecord
	if err := json.Unmarshal([]byte(s.Payload), &record); err != nil {
		return AIStrategyHistoryRecord{}, fmt.Errorf("storable ai strategy history: unmarshal payload: %w", err)
	}
	return record, nil
}
