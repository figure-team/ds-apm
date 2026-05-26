package ruletypes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
)

// StorableSOPDocument is the bun-backed persistence shape for SOPDocument.
//
// The full domain object is encoded into the payload column as JSON. A small
// set of flat columns (sop_id, version, title, contract_version, updated_at)
// mirrors the most-queried fields so List/lookup queries do not have to parse
// JSON. New fields added to SOPDocument flow through automatically; only the
// indexed columns need explicit handling here.
type StorableSOPDocument struct {
	bun.BaseModel `bun:"table:ds_sop_documents"`

	OrgID           string `bun:"org_id,pk,notnull,type:text"`
	SOPID           string `bun:"sop_id,pk,notnull,type:text"`
	Version         string `bun:"version,pk,notnull,type:text"`
	ContractVersion string `bun:"contract_version,notnull,type:text"`
	Title           string `bun:"title,notnull,type:text"`
	UpdatedAt       string `bun:"updated_at,notnull,type:text"`
	Payload         string `bun:"payload,notnull,type:text"`
}

// FromDomainSOPDocument builds a StorableSOPDocument scoped to orgID. The
// orgID is required (fail-closed); empty orgID returns an error so callers
// cannot accidentally create cross-tenant rows.
func FromDomainSOPDocument(orgID string, doc SOPDocument) (*StorableSOPDocument, error) {
	if strings.TrimSpace(orgID) == "" {
		return nil, fmt.Errorf("storable sop document: orgID must not be empty")
	}
	contractVersion := strings.TrimSpace(doc.ContractVersion)
	if contractVersion == "" {
		return nil, fmt.Errorf("storable sop document: ContractVersion must not be empty (upstream ValidateSOPDocument should have caught this)")
	}
	doc.ContractVersion = contractVersion // ensure payload sees the trimmed value too

	payload, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("storable sop document: marshal payload: %w", err)
	}
	return &StorableSOPDocument{
		OrgID:           orgID,
		SOPID:           doc.SOPID,
		Version:         doc.Version,
		ContractVersion: contractVersion,
		Title:           doc.Title,
		UpdatedAt:       doc.UpdatedAt,
		Payload:         string(payload),
	}, nil
}

// ToDomain decodes the persisted payload back into a SOPDocument.
func (s *StorableSOPDocument) ToDomain() (SOPDocument, error) {
	var doc SOPDocument
	if err := json.Unmarshal([]byte(s.Payload), &doc); err != nil {
		return SOPDocument{}, fmt.Errorf("storable sop document: unmarshal payload: %w", err)
	}
	return doc, nil
}
