package keys_memory_test

import (
	"context"
	"testing"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	keys "github.com/ambi/idmagic/backend/signingkeys/internal/keys_memory"
	ports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

//spec:covers REQ-SIGNINGKEYS-014: テナント、登録と用途が異なれば別の鍵を返し、同じ登録では鍵を維持する。
func TestManagedMcpKeysAreIsolatedByTenantAndRegistration(t *testing.T) {
	store, err := keys.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, tenant := range []string{"one", "two"} {
		base := tenantports.WithTenantID(context.Background(), tenant)
		signing, err := store.GetActiveKey(base)
		if err != nil {
			t.Fatal(err)
		}
		seen[signing.Kid] = true
		for _, scope := range []string{"first", "second"} {
			ctx := ports.WithKeyScope(ports.WithKeyUsage(base, signingdomain.KeyUsageMcpGatewayDPoP), scope)
			key, err := store.GetActiveKey(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if seen[key.Kid] || key.Usage != signingdomain.KeyUsageMcpGatewayDPoP || key.ScopeID != scope || key.TenantID != tenant {
				t.Fatalf("key isolation failed: %+v", key.PublicJWK)
			}
			seen[key.Kid] = true
			again, err := store.GetActiveKey(ctx)
			if err != nil || again.Kid != key.Kid {
				t.Fatalf("registered key changed: %v", err)
			}
		}
	}
}
