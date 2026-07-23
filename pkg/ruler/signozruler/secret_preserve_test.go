package signozruler

import (
	"testing"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/stretchr/testify/require"
)

// TestPreserveSecret covers preserveSecret's four branches: pass-through for
// non-placeholder input (no fetch), placeholder+found returns the existing
// secret, placeholder+not-found returns "", and placeholder+other error
// propagates the raw fetch error for the caller to wrap.
func TestPreserveSecret(t *testing.T) {
	notFound := errors.Newf(errors.TypeNotFound, errors.CodeNotFound, "nf")
	isNF := func(err error) bool { return errors.Is(err, notFound) }

	// non-placeholder passes through, no fetch
	got, err := preserveSecret("real-key", func() (string, error) { t.Fatal("should not fetch"); return "", nil }, isNF)
	require.NoError(t, err)
	require.Equal(t, "real-key", got)

	// placeholder + found → existing
	got, err = preserveSecret(APIKeyPlaceholder, func() (string, error) { return "stored", nil }, isNF)
	require.NoError(t, err)
	require.Equal(t, "stored", got)

	// placeholder + not-found → ""
	got, err = preserveSecret(APIKeyPlaceholder, func() (string, error) { return "", notFound }, isNF)
	require.NoError(t, err)
	require.Equal(t, "", got)

	// placeholder + other error → propagate
	boom := errors.Newf(errors.TypeInternal, errors.CodeInternal, "boom")
	_, err = preserveSecret(APIKeyPlaceholder, func() (string, error) { return "", boom }, isNF)
	require.Error(t, err)
}
