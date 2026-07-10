package support_test

import (
	"context"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	appmemory "github.com/ambi/idmagic/backend/application/adapters/persistence/memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestClientDisplayNameResolverFallbackOrder(t *testing.T) {
	ctx := context.Background()
	clients := memory.NewClientRepository()
	apps := appmemory.NewApplicationRepository()
	now := time.Now().UTC()

	named := "Admin Console"
	clients.Seed(&oauthdomain.OAuth2Client{
		TenantID: spec.DefaultTenantID, ClientID: "client-named",
		ClientName: &named, ClientType: spec.ClientPublic, CreatedAt: now, UpdatedAt: now,
	})
	// client_name が空白のみのクライアントは Application カタログ名へフォールバックする。
	blank := "   "
	clients.Seed(&oauthdomain.OAuth2Client{
		TenantID: spec.DefaultTenantID, ClientID: "client-catalog",
		ClientName: &blank, ClientType: spec.ClientPublic, CreatedAt: now, UpdatedAt: now,
	})
	if err := apps.Save(ctx, &appdomain.Application{
		TenantID: spec.DefaultTenantID, ApplicationID: "app-1", Name: "Catalog App",
		Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
		Bindings:  []appdomain.ProtocolBinding{{Type: appdomain.ProtocolBindingOIDC, ClientID: "client-catalog"}},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	r := &support.ClientDisplayNameResolver{ClientRepo: clients, ApplicationRepo: apps}

	cases := []struct {
		name     string
		clientID string
		want     string
	}{
		{"client_name を優先", "client-named", "Admin Console"},
		{"client_name が無ければカタログ名", "client-catalog", "Catalog App"},
		{"どちらも無ければ client_id", "00000000-0000-4000-8000-000000000099", "00000000-0000-4000-8000-000000000099"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := r.Resolve(ctx, spec.DefaultTenantID, tc.clientID); got != tc.want {
				t.Fatalf("Resolve(%q) = %q, want %q", tc.clientID, got, tc.want)
			}
		})
	}

	all := r.ResolveAll(ctx, spec.DefaultTenantID, []string{"client-named", "client-catalog", "client-named"})
	if all["client-named"] != "Admin Console" || all["client-catalog"] != "Catalog App" {
		t.Fatalf("ResolveAll mismatch: %v", all)
	}
	if len(all) != 2 {
		t.Fatalf("ResolveAll should dedupe: got %d entries", len(all))
	}
}

func TestClientDisplayNameResolverNilSafe(t *testing.T) {
	var r *support.ClientDisplayNameResolver
	if got := r.Resolve(context.Background(), spec.DefaultTenantID, "abc"); got != "abc" {
		t.Fatalf("nil resolver must fall back to client_id, got %q", got)
	}
}
