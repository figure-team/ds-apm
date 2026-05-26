package ruletypes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// staticGenerator is a minimal AIStrategyGenerator implementation used to
// prove the interface compiles and is callable.
type staticGenerator struct {
	out AIStrategy
}

func (s staticGenerator) Generate(_ context.Context, _ AIStrategyRequest) (AIStrategy, error) {
	return s.out, nil
}

func TestAIStrategyGenerator_InterfaceContract(t *testing.T) {
	var gen AIStrategyGenerator = staticGenerator{out: AIStrategy{StrategyID: "s1"}}
	got, err := gen.Generate(context.Background(), AIStrategyRequest{})
	require.NoError(t, err)
	require.Equal(t, "s1", got.StrategyID)
}
