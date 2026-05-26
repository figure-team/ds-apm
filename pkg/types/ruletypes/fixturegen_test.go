package ruletypes

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// TestFixtureGen_Print emits canonical JSON payloads for the seed rows used
// by tests/fixtures/go/*. It runs only when env FIXTURE_GEN=1 is set so it
// stays inert in normal `make test`. To regenerate fixtures:
//
//	FIXTURE_GEN=1 go test ./pkg/types/ruletypes/ -run TestFixtureGen_Print -v
//
// Copy the emitted blocks into tests/fixtures/go/*.yml.
func TestFixtureGen_Print(t *testing.T) {
	if os.Getenv("FIXTURE_GEN") != "1" {
		t.Skip("set FIXTURE_GEN=1 to print canonical fixture payloads")
	}

	// --- SOP documents ----------------------------------------------------
	makeDoc := func(sopID, version, title string) SOPDocument {
		return SOPDocument{
			ContractVersion: SOPDocumentContractVersion,
			SOPID:           sopID,
			Version:         version,
			Title:           title,
			BodyMarkdown:    "## step 1\nbody",
			UpdatedAt:       "2026-05-20T09:00:00Z",
			TenantScope: PilotTenantScope{
				ProjectIDs:   []string{"p"},
				Environments: []string{"prod"},
			},
		}
	}

	sopRows := []struct {
		orgID string
		doc   SOPDocument
	}{
		{"org-1", makeDoc("S1", "v1", "T-S1")},
		{"org-1", makeDoc("S1", "v2", "T-S1")},
		{"org-A", makeDoc("S1", "v1", "org-A doc")},
		{"org-B", makeDoc("S1", "v1", "org-B doc")},
		{"customer-a", paySOP()},
		{"customer-a", cartSOP()},
		{"customer-a", adSOP()},
	}
	fmt.Fprintln(os.Stdout, "# === ds_sop_documents.yml ===")
	for _, r := range sopRows {
		s, err := FromDomainSOPDocument(r.orgID, r.doc)
		if err != nil {
			t.Fatalf("FromDomainSOPDocument: %v", err)
		}
		fmt.Fprintf(os.Stdout, "- org_id: %s\n  sop_id: %s\n  version: %s\n  contract_version: %s\n  title: %s\n  updated_at: %q\n  payload: %s\n",
			s.OrgID, s.SOPID, s.Version, s.ContractVersion, s.Title, s.UpdatedAt, mustQuoteJSON(s.Payload))
	}

	// --- AI strategy history ----------------------------------------------
	makeRec := func(incidentID, fp, headline string) AIStrategyHistoryRecord {
		strategy := AIStrategy{
			ContractVersion:  AIStrategyContractVersion,
			StrategyID:       "strat-" + incidentID,
			IncidentID:       incidentID,
			AlertFingerprint: fp,
			Status:           AIStrategyStatusUnavailable,
			Language:         "ko-KR",
			Confidence:       AIConfidenceLow,
			Headline:         headline,
			Limitations:      []string{"test"},
			Audit: AIStrategyAudit{
				PromptVersion:    "ds-ir-ko-v1",
				Model:            "deterministic-local",
				GeneratedAt:      "2026-05-20T09:00:00Z",
				RedactionApplied: true,
			},
		}
		rec, err := NewAIStrategyHistoryRecord(strategy)
		if err != nil {
			t.Fatalf("NewAIStrategyHistoryRecord(%s): %v", incidentID, err)
		}
		return rec
	}

	histRows := []struct {
		orgID string
		rec   AIStrategyHistoryRecord
	}{
		{"org-1", makeRec("inc-1", "fp-1", "")},
		{"org-A", makeRec("inc-1", "fp-abc", "")},
	}
	fmt.Fprintln(os.Stdout, "# === ds_ai_strategy_history.yml ===")
	for _, r := range histRows {
		s, err := FromDomainAIStrategyHistoryRecord(r.orgID, r.rec)
		if err != nil {
			t.Fatalf("FromDomainAIStrategyHistoryRecord: %v", err)
		}
		fmt.Fprintf(os.Stdout, "- org_id: %s\n  incident_id: %s\n  alert_fingerprint: %s\n  contract_version: %s\n  payload: %s\n",
			s.OrgID, s.IncidentID, s.AlertFingerprint, s.ContractVersion, mustQuoteJSON(s.Payload))
	}
}

func mustQuoteJSON(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func paySOP() SOPDocument {
	return SOPDocument{
		ContractVersion: SOPDocumentContractVersion,
		SOPID:           "SOP-PAY-001",
		Version:         "2026-05-20.1",
		Title:           "Payment 5xx surge",
		BodyMarkdown:    "## step 1\n결제 성공률 dashboard와 PG timeout log 확인",
		UpdatedAt:       "2026-05-20T09:00:00Z",
		TenantScope:     PilotTenantScope{ProjectIDs: []string{"customer-a"}, Environments: []string{"prod"}},
	}
}

func cartSOP() SOPDocument {
	return SOPDocument{
		ContractVersion: SOPDocumentContractVersion,
		SOPID:           "SOP-CART-001",
		Version:         "2026-05-20.1",
		Title:           "Cart latency surge",
		BodyMarkdown:    "## step 1\nRedis cache hit-ratio 확인",
		UpdatedAt:       "2026-05-20T09:00:00Z",
		TenantScope:     PilotTenantScope{ProjectIDs: []string{"customer-a"}, Environments: []string{"prod"}},
	}
}

func adSOP() SOPDocument {
	return SOPDocument{
		ContractVersion: SOPDocumentContractVersion,
		SOPID:           "SOP-AD-001",
		Version:         "2026-05-20.1",
		Title:           "Ad CPU saturation",
		BodyMarkdown:    "## step 1\nJVM GC dashboard 확인",
		UpdatedAt:       "2026-05-20T09:00:00Z",
		TenantScope:     PilotTenantScope{ProjectIDs: []string{"customer-a"}, Environments: []string{"prod"}},
	}
}
