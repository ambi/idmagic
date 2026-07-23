package support_http_test

import (
	"context"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestClientDisplayNameResolverFallbackOrder(t *testing.T) {
	ctx := context.Background()
	clients := oauth2memory.NewClientRepository()
	apps := appmemory.NewApplicationRepository()
	now := time.Now().UTC()

	named := "Admin Console"
	clients.Seed(&oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: "client-named",
		ClientName: &named, ClientType: spec.ClientPublic, CreatedAt: now, UpdatedAt: now,
	})
	// client_name が空白のみのクライアントは Application カタログ名へフォールバックする。
	blank := "   "
	clients.Seed(&oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: "client-catalog",
		ClientName: &blank, ClientType: spec.ClientPublic, CreatedAt: now, UpdatedAt: now,
	})
	if err := apps.Save(ctx, &appdomain.Application{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: "app-1", Name: "Catalog App",
		Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
		Protocol:  &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: "client-catalog"},
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
			if got := r.Resolve(ctx, tenancydomain.DefaultTenantID, tc.clientID); got != tc.want {
				t.Fatalf("Resolve(%q) = %q, want %q", tc.clientID, got, tc.want)
			}
		})
	}

	all := r.ResolveAll(ctx, tenancydomain.DefaultTenantID, []string{"client-named", "client-catalog", "client-named"})
	if all["client-named"] != "Admin Console" || all["client-catalog"] != "Catalog App" {
		t.Fatalf("ResolveAll mismatch: %v", all)
	}
	if len(all) != 2 {
		t.Fatalf("ResolveAll should dedupe: got %d entries", len(all))
	}
}

func TestClientDisplayNameResolverNilSafe(t *testing.T) {
	var r *support.ClientDisplayNameResolver
	if got := r.Resolve(context.Background(), tenancydomain.DefaultTenantID, "abc"); got != "abc" {
		t.Fatalf("nil resolver must fall back to client_id, got %q", got)
	}
}
