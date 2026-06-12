package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/instrumentation/instrumentationtest"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

// TestPrepareTaskFunc_DispatchesAnomalyRule locks the manager dispatch wiring:
// a RuleTypeAnomaly postable rule must produce a task carrying an AnomalyRule,
// evaluated on the ClickHouse rule-task loop. Before the wiring, the manager
// rejects anomaly rules as "unsupported".
func TestPrepareTaskFunc_DispatchesAnomalyRule(t *testing.T) {
	logger := instrumentationtest.New().Logger()

	task, err := defaultPrepareTaskFunc(PrepareTaskOptions{
		Rule:        anomalyPostableRule(3, ruletypes.ValueIsAbove),
		TaskName:    "anomaly-1-groupname",
		OrgID:       valuer.GenerateUUID(),
		Logger:      logger,
		ManagerOpts: &ManagerOptions{Logger: logger},
	})

	require.NoError(t, err)
	require.NotNil(t, task)
	require.Len(t, task.Rules(), 1)
	assert.Equal(t, ruletypes.RuleTypeAnomaly, task.Rules()[0].Type())
	assert.Equal(t, TaskType(TaskTypeCh), task.Type())
}
