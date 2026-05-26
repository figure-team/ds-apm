package localaigenerator

import (
	"context"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

func TestLocal_Generate_MissingSOPReturnsSOPMissingStatus(t *testing.T) {
	gen := New()
	got, err := gen.Generate(context.Background(), ruletypes.AIStrategyRequest{
		IncidentID:       "inc-1",
		AlertFingerprint: "fp-1",
		Labels:           map[string]string{"alertname": "Foo", "service.name": "svc", "project_id": "p", "environment": "prod", "owner_team": "t", "severity": "warning"},
		GeneratedAt:      "2026-05-20T09:00:00Z",
	})
	require.NoError(t, err)
	require.Equal(t, ruletypes.AIStrategyStatusSOPMissing, got.Status)
}

func TestLocal_ImplementsGeneratorInterface(t *testing.T) {
	var _ ruletypes.AIStrategyGenerator = New()
}
