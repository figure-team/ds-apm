package auditor

import (
	"context"
	"fmt"

	"github.com/SigNoz/signoz/pkg/types/audittypes"
)

// AuditFunc is the narrow slice of pkg/auditor.Auditor the sink needs
// (fire-and-forget, drop-on-full upstream).
type AuditFunc func(ctx context.Context, event audittypes.AuditEvent)

// DSSink bridges coderca audit records to the CF-6 auditor service.
type DSSink struct {
	audit AuditFunc
}

// NewDSSink builds the sink over auditor.Audit.
func NewDSSink(audit AuditFunc) *DSSink {
	return &DSSink{audit: audit}
}

// Record maps the record onto a ds audit event and emits it.
func (s *DSSink) Record(ctx context.Context, rec AuditRecord) {
	if s.audit == nil {
		return
	}
	s.audit(ctx, audittypes.AuditEvent{
		Timestamp: rec.At,
		EventName: audittypes.NewEventName("coderca.run", audittypes.ActionUpdate),
		Body: fmt.Sprintf(
			"coderca run finalized: run=%s org=%s service=%s status=%s outcome=%s detail=%s",
			rec.RunID, rec.OrgID, rec.Service, rec.Status, rec.Outcome, rec.Detail,
		),
	})
}

var _ Sink = (*DSSink)(nil)
