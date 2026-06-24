package remediation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/SigNoz/signoz/pkg/ruler/remediationstore"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// Proposer creates a proposed RemediationExecution when a bound SOP has an
// approved Runbook and the org has execution enabled. It is fail-open: any
// failure returns (nil, false) so the caller (dispatch hook) keeps delivering
// the alert unchanged.
type Proposer struct {
	store   remediationstore.Store
	baseURL string
	now     func() time.Time
}

// NewProposer constructs a Proposer. now may be nil (falls back to time.Now).
// Trailing slashes on baseURL are trimmed.
func NewProposer(store remediationstore.Store, baseURL string, now func() time.Time) *Proposer {
	if now == nil {
		now = time.Now
	}
	return &Proposer{store: store, baseURL: strings.TrimRight(baseURL, "/"), now: now}
}

// Propose selects the first approved Runbook from doc, creates a proposed
// RemediationExecution via the store, and returns annotation key/value pairs
// that point the operator at the web approval card.
//
// Returns (nil, false) — fail-open — when:
//   - p or p.store is nil
//   - cfg.ExecutionEnabled is false
//   - no approved Runbook with a non-empty script exists
//   - store.Create returns an error
func (p *Proposer) Propose(
	ctx context.Context,
	orgID, incidentID, alertFingerprint string,
	doc ruletypes.SOPDocument,
	cfg ruletypes.RemediationConfig,
) (map[string]string, bool) {
	if p == nil || p.store == nil || !cfg.ExecutionEnabled {
		return nil, false
	}
	rb, ok := firstApprovedRunbook(doc)
	if !ok {
		return nil, false
	}

	now := p.now().UTC()
	e := ruletypes.RemediationExecution{
		ID:               uuid.NewString(),
		OrgID:            orgID,
		IncidentID:       incidentID,
		AlertFingerprint: alertFingerprint,
		SOPID:            doc.SOPID,
		SOPVersion:       doc.Version,
		RunbookID:        rb.ID,
		ScriptSnapshot:   rb.ExecutableScript, // copy approved script verbatim — safety invariant
		Status:           ruletypes.RemediationStatusProposed,
		ProposedAt:       now.Format(time.RFC3339),
		ExpiresAt:        now.Add(time.Duration(cfg.ProposalTTLSeconds) * time.Second).Format(time.RFC3339),
	}
	if err := p.store.Create(ctx, e); err != nil {
		return nil, false // fail-open: never block the alert
	}

	ann := map[string]string{
		alertmanagertypes.IncidentAnnotationRemediationID:            e.ID,
		alertmanagertypes.IncidentAnnotationRemediationScriptSummary: scriptSummary(rb),
		alertmanagertypes.IncidentAnnotationRemediationApproveURL:    p.approveURL(incidentID, e.ID),
	}
	return ann, true
}

// firstApprovedRunbook returns the first Runbook in doc with status==approved
// and a non-empty ExecutableScript. Future extension: sort by Confidence desc.
func firstApprovedRunbook(doc ruletypes.SOPDocument) (ruletypes.Runbook, bool) {
	for _, rb := range doc.Runbooks {
		if rb.Status == ruletypes.RunbookStatusApproved && strings.TrimSpace(rb.ExecutableScript) != "" {
			return rb, true
		}
	}
	return ruletypes.Runbook{}, false
}

// scriptSummary builds a short human-readable description for the annotation.
// The full script is intentionally NOT included in notifications — the operator
// reviews it on the approval card.
func scriptSummary(rb ruletypes.Runbook) string {
	title := strings.TrimSpace(rb.Title)
	if title == "" {
		title = "자동 대응 스크립트"
	}
	return fmt.Sprintf("%s (승인 시 웹 UI에서 실행)", title)
}

// approveURL constructs the web URL where an operator can review and approve
// the proposed remediation execution.
func (p *Proposer) approveURL(incidentID, remediationID string) string {
	return fmt.Sprintf("%s/incidents/%s?remediation=%s", p.baseURL, incidentID, remediationID)
}
