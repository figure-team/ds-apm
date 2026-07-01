package sqlremediationtargetstore

import (
	"context"
	"testing"

	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstoretest"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// newTestStore spins up a real in-memory SQLite store with the
// ds_remediation_target table created (DDL mirrors migration 089). Mirrors
// sqlremediationstore/store_test.go's setup helper — applies DDL directly
// against sqlitesqlstoretest.New(t), then returns a ready SQLStore.
func newTestStore(t *testing.T) *SQLStore {
	t.Helper()
	s := sqlitesqlstoretest.New(t)
	_, err := s.BunDB().ExecContext(context.Background(), `
CREATE TABLE ds_remediation_target (
  id TEXT NOT NULL,
  org_id TEXT NOT NULL,
  name TEXT NOT NULL,
  host TEXT NOT NULL,
  port INTEGER NOT NULL,
  ssh_user TEXT NOT NULL,
  sealed_credential TEXT NOT NULL,
  credential_kind TEXT NOT NULL,
  host_key_fingerprint TEXT NOT NULL,
  service_selectors TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (org_id, id)
)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(s)
}

func sampleTarget() ruletypes.RemediationTarget {
	return ruletypes.RemediationTarget{
		ID: "3f2504e0-4f89-41d3-9a0c-0305e82c3301", OrgID: "org-1",
		Name: "prod-web-01", Host: "10.0.0.5", Port: 22, User: "deploy",
		SealedCredential: "c2VhbGVk", CredentialKind: ruletypes.RemediationCredentialKindPrivateKey,
		HostKeyFingerprint: "SHA256:abc", ServiceSelectors: []string{"payment", "cart"},
		CreatedAt: "2026-07-01T00:00:00Z", UpdatedAt: "2026-07-01T00:00:00Z",
	}
}

func TestCreateGet_RoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	if err := s.Create(ctx, "org-1", sampleTarget()); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := s.Get(ctx, "org-1", sampleTarget().ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Host != "10.0.0.5" || len(got.ServiceSelectors) != 2 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestResolve_ByServiceNameFirstMatch(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	_ = s.Create(ctx, "org-1", sampleTarget())
	// service.name 키(점)로 조회 — 상수와 동일해야 함.
	got, err := s.Resolve(ctx, "org-1", map[string]string{"service.name": "cart"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.ID != sampleTarget().ID {
		t.Fatalf("expected match, got %+v", got)
	}
}

func TestResolve_NotFoundWhenNoServiceLabel(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	_ = s.Create(ctx, "org-1", sampleTarget())
	if _, err := s.Resolve(ctx, "org-1", map[string]string{"severity": "critical"}); err == nil {
		t.Fatal("expected not-found when service.name label absent")
	}
}
