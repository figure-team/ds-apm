package mockaigenerator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SigNoz/signoz/pkg/ruler/aigenerator/localaigenerator"
	"github.com/SigNoz/signoz/pkg/types/ruletypes"
	"github.com/stretchr/testify/require"
)

func writeFixture(t *testing.T, dir, name, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600))
}

func TestMock_MatchByAlertnameAndService(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "a.json", `{
		"match": {"alertname":"PaymentAPI5xxHigh","service_name":"paymentservice"},
		"strategy": {
			"contractVersion":"ds.ai_strategy.v1","strategyId":"strat-pay",
			"incidentId":"INC-1","status":"ready","language":"ko-KR","confidence":"medium",
			"audit":{"promptVersion":"mock-v1","model":"mock","generatedAt":"2026-05-20T09:00:00Z","redactionApplied":true}
		}
	}`)
	gen, err := New(dir, localaigenerator.New())
	require.NoError(t, err)

	got, err := gen.Generate(context.Background(), ruletypes.AIStrategyRequest{
		IncidentID:       "INC-1",
		AlertFingerprint: "fp",
		Labels: map[string]string{
			"alertname":    "PaymentAPI5xxHigh",
			"service.name": "paymentservice",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "strat-pay", got.StrategyID)
	require.Equal(t, ruletypes.AIStrategyStatusReady, got.Status)
}

func TestMock_NoMatchFallsBackToInjected(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "a.json", `{
		"match": {"alertname":"Other"},
		"strategy": {"contractVersion":"ds.ai_strategy.v1","strategyId":"unused","incidentId":"x","status":"ready","language":"ko-KR","confidence":"low","audit":{"promptVersion":"m","model":"m","generatedAt":"t","redactionApplied":false}}
	}`)
	gen, err := New(dir, localaigenerator.New())
	require.NoError(t, err)

	got, err := gen.Generate(context.Background(), ruletypes.AIStrategyRequest{
		IncidentID:       "inc-x",
		AlertFingerprint: "fp-x",
		Labels:           map[string]string{"alertname": "DoesNotMatch"},
	})
	require.NoError(t, err)
	require.NotEqual(t, "unused", got.StrategyID, "fallback must NOT use the rule")
	require.Equal(t, ruletypes.AIStrategyStatusSOPMissing, got.Status)
}

func TestMock_LoadError_RejectsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "broken.json", `{not json`)
	_, err := New(dir, localaigenerator.New())
	require.Error(t, err)
}

func TestMock_ImplementsGeneratorInterface(t *testing.T) {
	gen, err := New("", localaigenerator.New())
	require.NoError(t, err)
	var _ ruletypes.AIStrategyGenerator = gen
}

func TestMock_LexFirstFileWins(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "b.json", `{
		"match": {"alertname":"Same"},
		"strategy": {"contractVersion":"ds.ai_strategy.v1","strategyId":"second","incidentId":"i","status":"ready","language":"ko-KR","confidence":"low","audit":{"promptVersion":"m","model":"m","generatedAt":"t","redactionApplied":false}}
	}`)
	writeFixture(t, dir, "a.json", `{
		"match": {"alertname":"Same"},
		"strategy": {"contractVersion":"ds.ai_strategy.v1","strategyId":"first","incidentId":"i","status":"ready","language":"ko-KR","confidence":"low","audit":{"promptVersion":"m","model":"m","generatedAt":"t","redactionApplied":false}}
	}`)
	gen, err := New(dir, localaigenerator.New())
	require.NoError(t, err)

	got, err := gen.Generate(context.Background(), ruletypes.AIStrategyRequest{
		IncidentID:       "i",
		AlertFingerprint: "fp",
		Labels:           map[string]string{"alertname": "Same"},
	})
	require.NoError(t, err)
	require.Equal(t, "first", got.StrategyID, "a.json should win over b.json")
}

func TestMock_ScenarioPayment(t *testing.T) {
	gen, err := New("testdata", localaigenerator.New())
	require.NoError(t, err)
	got, err := gen.Generate(context.Background(), ruletypes.AIStrategyRequest{
		IncidentID:       "INC-PAY",
		AlertFingerprint: "fp-pay",
		Labels:           map[string]string{"alertname": "PaymentAPI5xxHigh", "service.name": "paymentservice"},
	})
	require.NoError(t, err)
	require.Equal(t, "mock-pay-001", got.StrategyID)
	require.Equal(t, "SOP-PAY-001", got.SOPID)
}

func TestMock_ScenarioCart(t *testing.T) {
	gen, err := New("testdata", localaigenerator.New())
	require.NoError(t, err)
	got, err := gen.Generate(context.Background(), ruletypes.AIStrategyRequest{
		IncidentID:       "INC-CART",
		AlertFingerprint: "fp-cart",
		Labels:           map[string]string{"alertname": "CartServiceLatencyHigh", "service.name": "cartservice"},
	})
	require.NoError(t, err)
	require.Equal(t, "mock-cart-001", got.StrategyID)
	require.Equal(t, "SOP-CART-001", got.SOPID)
}

func TestMock_ScenarioAd(t *testing.T) {
	gen, err := New("testdata", localaigenerator.New())
	require.NoError(t, err)
	got, err := gen.Generate(context.Background(), ruletypes.AIStrategyRequest{
		IncidentID:       "INC-AD",
		AlertFingerprint: "fp-ad",
		Labels:           map[string]string{"alertname": "AdServiceCPUSat", "service.name": "adservice"},
	})
	require.NoError(t, err)
	require.Equal(t, "mock-ad-001", got.StrategyID)
	require.Equal(t, "SOP-AD-001", got.SOPID)
}
