package remediation

import (
	"context"

	"github.com/SigNoz/signoz/pkg/ruler/remediationtargetstore"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// freezeTargetSnapshot resolves the incident's labels to a remediation target
// via store and, on a hit, freezes its connection parameters onto e (design
// §3.2). A nil store or any Resolve failure (not-found/error) leaves e unchanged
// → local execution (empty TargetID). fail-open at propose (design §5).
//
// Shared by Proposer.Propose and Selector.createExecution so the freeze rule has
// a single definition.
func freezeTargetSnapshot(ctx context.Context, store remediationtargetstore.Store, orgID string, labels map[string]string, e *ruletypes.RemediationExecution) {
	if store == nil {
		return
	}
	tgt, err := store.Resolve(ctx, orgID, labels)
	if err != nil {
		return
	}
	e.TargetID = tgt.ID
	e.TargetHost = tgt.Host
	e.TargetPort = tgt.Port
	e.TargetUser = tgt.User
	e.TargetHostKeyFP = tgt.HostKeyFingerprint
	e.TargetName = tgt.Name
}
