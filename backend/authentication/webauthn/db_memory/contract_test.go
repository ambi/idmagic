package db_memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/webauthn/testing_contract"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestWebAuthnSessionStoreContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		tenant := &tenantdomain.Tenant{ID: "tenant-a"}
		other := &tenantdomain.Tenant{ID: "tenant-b"}
		return testing_contract.Fixture{
			Store: NewWebAuthnSessionStore(), Context: func(ctx context.Context) context.Context { return tenancy.WithTenant(ctx, tenant, "", "") }, Other: func(ctx context.Context) context.Context { return tenancy.WithTenant(ctx, other, "", "") },
			Now: time.Now().UTC(),
		}
	})
}
