package aigenerator

import (
	"context"
	"testing"

	"github.com/SigNoz/signoz/pkg/ruler/remediation"
)

func TestStoreAwareSelectorProvider_ImplementsResolver(t *testing.T) {
	var _ remediation.ProviderResolver = (*StoreAwareSelectorProvider)(nil)
}

func TestStoreAwareSelectorProvider_EmptyOrg_Errors(t *testing.T) {
	r := NewStoreAwareSelectorProvider(nil, nil)
	if _, _, err := r.Resolve(context.Background(), ""); err == nil {
		t.Fatalf("empty org must error (no provider)")
	}
}
