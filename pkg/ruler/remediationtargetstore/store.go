// Package remediationtargetstore persists SSH remediation targets and resolves
// an incident's labels to a target (design §3.1).
package remediationtargetstore

import (
	"context"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// Store persists remediation targets per org.
type Store interface {
	Create(ctx context.Context, orgID string, t ruletypes.RemediationTarget) error
	Update(ctx context.Context, orgID string, t ruletypes.RemediationTarget) error
	Delete(ctx context.Context, orgID, id string) error
	Get(ctx context.Context, orgID, id string) (ruletypes.RemediationTarget, error)
	List(ctx context.Context, orgID string) ([]ruletypes.RemediationTarget, error)
	// Resolve maps an incident's alert labels to a target. v1 reads only
	// alertmanagertypes.IncidentLabelServiceName and returns the first target
	// whose ServiceSelectors contains that value; not-found when none match
	// (design §3.1, §3.3). The full labels map is taken (not just serviceName)
	// so future instance-specific resolution needs no signature change (New-3).
	Resolve(ctx context.Context, orgID string, labels map[string]string) (ruletypes.RemediationTarget, error)
}
