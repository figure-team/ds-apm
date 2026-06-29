package signoz

import (
	"context"
	"log/slog"

	"github.com/prometheus/alertmanager/api/v2/restapi/operations/alert"

	"github.com/SigNoz/signoz/pkg/modules/organization"
	"github.com/SigNoz/signoz/pkg/ruler/remediation"
	"github.com/SigNoz/signoz/pkg/ruler/remediationstore"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

// remediationHookAdapter satisfies dispatchhook.RemediationProposer.
// It loads the per-org RemediationConfig from the store before delegating to
// the Proposer so the hook does not need to carry store state itself.
type remediationHookAdapter struct {
	proposer *remediation.Proposer
	store    remediationstore.Store
}

func (a remediationHookAdapter) MaybePropose(
	ctx context.Context,
	orgID, incidentID, fp string,
	labels map[string]string,
	doc ruletypes.SOPDocument,
) (map[string]string, bool) {
	cfg, err := a.store.GetConfig(ctx, orgID)
	if err != nil {
		// fail-open: don't block alert delivery on a config-load error
		return nil, false
	}
	return a.proposer.Propose(ctx, orgID, incidentID, fp, labels, doc, cfg)
}

// alertStateLookup implements remediation.AlertStateLookup over the alertmanager
// Service.GetAlerts method. It checks whether any active alert with the given
// fingerprint is currently firing.
//
// Known limitation: GetAlerts returns DeprecatedGettableAlerts which embed the
// prometheus/alertmanager model.Alert; State is determined by whether the alert
// appears in the Active set. If the alertmanager server for the org is not yet
// initialised, GetAlerts returns an error and IsFiring returns (false, err) so
// the Verifier treats it as best-effort and skips (leaving rows in succeeded).
type alertStateLookup struct {
	am interface {
		GetAlerts(ctx context.Context, orgID string, params alertmanagertypes.GettableAlertsParams) (alertmanagertypes.DeprecatedGettableAlerts, error)
	}
}

func (l alertStateLookup) IsFiring(ctx context.Context, orgID, fp string) (bool, error) {
	// Use NewGetAlertsParams so Silenced/Inhibited/Active get their defaults
	// (all true) and we filter to active-only by overriding Active.
	p := alert.NewGetAlertsParams()
	active := true
	p.Active = &active
	silenced := false
	p.Silenced = &silenced
	inhibited := false
	p.Inhibited = &inhibited
	params := alertmanagertypes.GettableAlertsParams{GetAlertsParams: p}
	alerts, err := l.am.GetAlerts(ctx, orgID, params)
	if err != nil {
		return false, err
	}
	for _, a := range alerts {
		if a != nil && a.Fingerprint == fp {
			return true, nil
		}
	}
	return false, nil
}

// remediationSelectorAdapter satisfies dispatchhook.RemediationSelector. It runs
// the Selector in a detached goroutine with panic recovery so a slow LLM or a
// bug can never block or crash the dispatch path (fire-and-forget, fail-open).
type remediationSelectorAdapter struct {
	selector *remediation.Selector
	logger   *slog.Logger
}

func (a remediationSelectorAdapter) Maybe(
	ctx context.Context,
	orgID, incidentID, alertFingerprint string,
	labels map[string]string,
	doc ruletypes.SOPDocument,
) {
	if a.selector == nil {
		return
	}
	// Detach: the dispatch context may be cancelled the moment the alert is
	// delivered; the Selector has its own internal timeout.
	go func() {
		defer func() {
			if r := recover(); r != nil && a.logger != nil {
				a.logger.Error("remediation selector panic recovered", "recover", r, "orgId", orgID)
			}
		}()
		_, _ = a.selector.Select(context.WithoutCancel(ctx), orgID, incidentID, alertFingerprint, labels, doc)
	}()
}

// orgLister returns a func(context.Context) []string that lists all org IDs
// owned by this instance. Used by the Verifier background worker.
func orgLister(getter organization.Getter, logger *slog.Logger) func(context.Context) []string {
	return func(ctx context.Context) []string {
		orgs, err := getter.ListByOwnedKeyRange(ctx)
		if err != nil {
			logger.WarnContext(ctx, "remediation verifier: failed to list orgs", "err", err)
			return nil
		}
		ids := make([]string, 0, len(orgs))
		for _, o := range orgs {
			ids = append(ids, o.ID.StringValue())
		}
		return ids
	}
}
