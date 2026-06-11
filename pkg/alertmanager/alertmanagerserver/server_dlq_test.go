package alertmanagerserver

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/SigNoz/signoz/pkg/alertmanager/nfmanager/nfmanagertest"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes/alertmanagertypestest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// TestDLQWiring is the DoD row-3 acceptance test: when SIGNOZ_DLQ_PATH is
// set, booting the alertmanager server must produce a non-nil DLQ sink and
// create the sink file on disk. This closes the server.go :327 TODO where a
// nil sink was passed to the dispatcher.
func TestDLQWiring(t *testing.T) {
	path := filepath.Join(t.TempDir(), "alert-dlq.jsonl")
	t.Setenv("SIGNOZ_DLQ_PATH", path)

	server, err := New(
		context.Background(),
		slog.New(slog.DiscardHandler),
		prometheus.NewRegistry(),
		NewConfig(),
		"1",
		alertmanagertypestest.NewStateStore(),
		nfmanagertest.NewMock(),
		nil,
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = server.Stop(context.Background()) })

	require.NotNil(t, server.dlqSink, "SIGNOZ_DLQ_PATH set → server must boot with a non-nil DLQ sink")

	_, statErr := os.Stat(path)
	require.NoError(t, statErr, "DLQ sink file must be created at boot when SIGNOZ_DLQ_PATH is set")
}

// TestDLQWiringDisabledWhenUnset guards the seam: with SIGNOZ_DLQ_PATH unset
// the sink must be nil so the dispatcher keeps its pre-DLQ behavior and
// hermetic tests never write to the repo tree.
func TestDLQWiringDisabledWhenUnset(t *testing.T) {
	t.Setenv("SIGNOZ_DLQ_PATH", "")

	server, err := New(
		context.Background(),
		slog.New(slog.DiscardHandler),
		prometheus.NewRegistry(),
		NewConfig(),
		"1",
		alertmanagertypestest.NewStateStore(),
		nfmanagertest.NewMock(),
		nil,
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = server.Stop(context.Background()) })

	require.Nil(t, server.dlqSink, "unset SIGNOZ_DLQ_PATH → DLQ disabled (nil sink)")
}
