package ruletypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCodebaseRCAConfig(t *testing.T) {
	cfg := DefaultCodebaseRCAConfig("org-1")
	assert.Equal(t, "org-1", cfg.OrgID)
	assert.False(t, cfg.Enabled) // 기본 OFF, opt-in (설계 §6.1)
	assert.Equal(t, "high", cfg.MinSeverity)
	assert.Equal(t, 21600, cfg.CooldownWindowSecs) // 6h
	assert.Equal(t, 20, cfg.MaxRunsPerDay)
	assert.Equal(t, 50, cfg.MaxQueueDepth)
	assert.Equal(t, 1, cfg.MaxConcurrentRuns)
	assert.False(t, cfg.AllowUnboundWithoutAnomaly)
}

func TestValidateCodebaseRCAConfig(t *testing.T) {
	valid := DefaultCodebaseRCAConfig("org-1")
	require.NoError(t, ValidateCodebaseRCAConfig(valid))

	bad := valid
	bad.MinSeverity = "nonsense"
	require.Error(t, ValidateCodebaseRCAConfig(bad))

	bad2 := valid
	bad2.MaxRunsPerDay = -1
	require.Error(t, ValidateCodebaseRCAConfig(bad2))

	bad3 := valid
	bad3.OrgID = ""
	require.Error(t, ValidateCodebaseRCAConfig(bad3))

	// 비용 제어 임계값은 0을 허용하지 않음(fail-closed)
	bad4 := valid
	bad4.MaxRunsPerDay = 0
	require.Error(t, ValidateCodebaseRCAConfig(bad4))

	bad5 := valid
	bad5.CooldownWindowSecs = 0
	require.Error(t, ValidateCodebaseRCAConfig(bad5))

	bad6 := valid
	bad6.MaxQueueDepth = 0
	require.Error(t, ValidateCodebaseRCAConfig(bad6))

	bad7 := valid
	bad7.MaxConcurrentRuns = 3
	require.Error(t, ValidateCodebaseRCAConfig(bad7))

	bad8 := valid
	bad8.ContractVersion = ""
	require.Error(t, ValidateCodebaseRCAConfig(bad8))
}

func TestSeverityAtLeast(t *testing.T) {
	assert.True(t, SeverityAtLeast("critical", "high"))
	assert.True(t, SeverityAtLeast("HIGH", "high")) // 대소문자 무시
	assert.False(t, SeverityAtLeast("warning", "high"))
	assert.False(t, SeverityAtLeast("", "high"))       // 라벨 부재 → fail-closed
	assert.False(t, SeverityAtLeast("unknown", "high")) // 미지 등급 → fail-closed
}
