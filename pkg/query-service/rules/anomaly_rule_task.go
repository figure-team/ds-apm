package rules

import (
	"time"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

// TaskTypeAnomaly identifies a task that evaluates anomaly rules. Anomaly rules
// are evaluated on the same ClickHouse rule-task loop as threshold rules (the
// "anomaly-ness" lives entirely in AnomalyRule.Eval, which scores series against
// a baseline before applying the threshold), so the constructed task is a
// RuleTask whose Type() is TaskTypeCh. The dedicated constant + constructor give
// the manager a clear seam to dispatch RuleTypeAnomaly without leaking that
// detail into shared dispatch code.
const TaskTypeAnomaly TaskType = "anomaly_ruletask"

// NewAnomalyRuleTask builds the evaluation task for anomaly rules. It delegates
// to NewRuleTask because anomaly rules consume the v5 ClickHouse querier exactly
// like threshold rules.
func NewAnomalyRuleTask(name, file string, frequency time.Duration, rules []Rule, opts *ManagerOptions, notify NotifyFunc, maintenanceStore ruletypes.MaintenanceStore, orgID valuer.UUID) *RuleTask {
	return NewRuleTask(name, file, frequency, rules, opts, notify, maintenanceStore, orgID)
}
