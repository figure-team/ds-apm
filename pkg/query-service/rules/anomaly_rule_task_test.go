package rules

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/instrumentation/instrumentationtest"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

// TestNewAnomalyRuleTask locks the wiring contract the manager depends on at
// integration time: an anomaly rule is evaluated on the ClickHouse rule-task
// loop, carrying the rule it was given.
func TestNewAnomalyRuleTask(t *testing.T) {
	rule := newAnomalyTestRule(t, 3, ruletypes.ValueIsAbove)
	opts := &ManagerOptions{Logger: instrumentationtest.New().Logger()}

	task := NewAnomalyRuleTask("group-1", "file-1", time.Minute, []Rule{rule}, opts, nil, nil, valuer.GenerateUUID())

	require.NotNil(t, task)
	assert.Equal(t, "group-1", task.Name())
	assert.Equal(t, "group-1;file-1", task.Key())
	assert.Equal(t, TaskType(TaskTypeCh), task.Type(), "anomaly rules evaluate on the ClickHouse rule-task loop")
	require.Len(t, task.Rules(), 1)
	assert.Equal(t, ruletypes.RuleTypeAnomaly, task.Rules()[0].Type())
}
