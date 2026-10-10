package db_memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/session/testing_contract"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		tenant := &tenantdomain.Tenant{ID: "tenant-a"}
		other := &tenantdomain.Tenant{ID: "tenant-b"}
		now := time.Now().UTC().Truncate(time.Microsecond)
		store := NewSessionStore()
		store.Clock = func() time.Time { return now }
		return testing_contract.Fixture{
			Store: store, Context: func(ctx context.Context) context.Context { return tenantports.WithTenant(ctx, tenant, "", "") },
			Other: func(ctx context.Context) context.Context { return tenantports.WithTenant(ctx, other, "", "") }, TenantA: tenant.ID, UserID: "user-1", Now: now,
		}
	})
}
