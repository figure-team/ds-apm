package sqlremediationtargetstore

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/SigNoz/signoz/pkg/ruler/remediationtargetstore"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/uptrace/bun"
)

// targetRow mirrors the ds_remediation_target DDL (migration 089). ServiceSelectors
// is stored as a newline-joined TEXT column (v1 selectors are simple exact strings).
type targetRow struct {
	bun.BaseModel      `bun:"table:ds_remediation_target"`
	ID                 string `bun:"id"`
	OrgID              string `bun:"org_id"`
	Name               string `bun:"name"`
	Host               string `bun:"host"`
	Port               int    `bun:"port"`
	User               string `bun:"ssh_user"`
	SealedCredential   string `bun:"sealed_credential"`
	CredentialKind     string `bun:"credential_kind"`
	HostKeyFingerprint string `bun:"host_key_fingerprint"`
	ServiceSelectors   string `bun:"service_selectors"`
	CreatedAt          string `bun:"created_at"`
	UpdatedAt          string `bun:"updated_at"`
}

func rowFromDomain(t ruletypes.RemediationTarget) targetRow {
	return targetRow{
		ID: t.ID, OrgID: t.OrgID, Name: t.Name, Host: t.Host, Port: t.Port,
		User: t.User, SealedCredential: t.SealedCredential, CredentialKind: t.CredentialKind,
		HostKeyFingerprint: t.HostKeyFingerprint,
		ServiceSelectors:   strings.Join(t.ServiceSelectors, "\n"),
		CreatedAt:          t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func (r targetRow) toDomain() ruletypes.RemediationTarget {
	var sel []string
	if strings.TrimSpace(r.ServiceSelectors) != "" {
		sel = strings.Split(r.ServiceSelectors, "\n")
	}
	return ruletypes.RemediationTarget{
		ID: r.ID, OrgID: r.OrgID, Name: r.Name, Host: r.Host, Port: r.Port,
		User: r.User, SealedCredential: r.SealedCredential, CredentialKind: r.CredentialKind,
		HostKeyFingerprint: r.HostKeyFingerprint, ServiceSelectors: sel,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

// SQLStore is the bun-backed implementation of remediationtargetstore.Store.
type SQLStore struct {
	sqlstore sqlstore.SQLStore
}

var _ remediationtargetstore.Store = (*SQLStore)(nil)

func New(s sqlstore.SQLStore) *SQLStore { return &SQLStore{sqlstore: s} }

func (s *SQLStore) Create(ctx context.Context, orgID string, t ruletypes.RemediationTarget) error {
	t.OrgID = orgID
	if err := ruletypes.ValidateRemediationTarget(t); err != nil {
		return err
	}
	row := rowFromDomain(t)
	_, err := s.sqlstore.BunDB().NewInsert().Model(&row).Exec(ctx)
	return err
}

func (s *SQLStore) Update(ctx context.Context, orgID string, t ruletypes.RemediationTarget) error {
	t.OrgID = orgID
	if err := ruletypes.ValidateRemediationTarget(t); err != nil {
		return err
	}
	row := rowFromDomain(t)
	_, err := s.sqlstore.BunDB().NewUpdate().Model(&row).
		WherePK().Where("org_id = ?", orgID).Exec(ctx)
	return err
}

func (s *SQLStore) Delete(ctx context.Context, orgID, id string) error {
	_, err := s.sqlstore.BunDB().NewDelete().Model((*targetRow)(nil)).
		Where("org_id = ?", orgID).Where("id = ?", id).Exec(ctx)
	return err
}

func (s *SQLStore) Get(ctx context.Context, orgID, id string) (ruletypes.RemediationTarget, error) {
	var row targetRow
	err := s.sqlstore.BunDB().NewSelect().Model(&row).
		Where("org_id = ?", orgID).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return ruletypes.RemediationTarget{}, sql.ErrNoRows
	}
	if err != nil {
		return ruletypes.RemediationTarget{}, err
	}
	return row.toDomain(), nil
}

func (s *SQLStore) List(ctx context.Context, orgID string) ([]ruletypes.RemediationTarget, error) {
	var rows []targetRow
	err := s.sqlstore.BunDB().NewSelect().Model(&rows).
		Where("org_id = ?", orgID).OrderExpr("name ASC").Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ruletypes.RemediationTarget, len(rows))
	for i, r := range rows {
		out[i] = r.toDomain()
	}
	return out, nil
}

// Resolve reads only service.name (design §3.3) and returns the first target
// (name ASC) whose ServiceSelectors contains that value. not-found otherwise.
func (s *SQLStore) Resolve(ctx context.Context, orgID string, labels map[string]string) (ruletypes.RemediationTarget, error) {
	svc := strings.TrimSpace(labels[alertmanagertypes.IncidentLabelServiceName])
	if svc == "" {
		return ruletypes.RemediationTarget{}, sql.ErrNoRows
	}
	all, err := s.List(ctx, orgID)
	if err != nil {
		return ruletypes.RemediationTarget{}, err
	}
	for _, t := range all {
		for _, sel := range t.ServiceSelectors {
			if strings.EqualFold(strings.TrimSpace(sel), svc) {
				return t, nil
			}
		}
	}
	return ruletypes.RemediationTarget{}, sql.ErrNoRows
}
