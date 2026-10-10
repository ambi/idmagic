package db_postgres

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/authentication/session/testing_contract"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenant := pgfixtures.SeedTenant(t, db)
		other := pgfixtures.SeedTenant(t, db)
		user := pgfixtures.SeedUser(t, db, tenant.ID)
		return testing_contract.Fixture{
			Store:   &SessionRepository{Pool: db},
			Context: func(ctx context.Context) context.Context { return tenantports.WithTenant(ctx, tenant, "", "") },
			Other:   func(ctx context.Context) context.Context { return tenantports.WithTenant(ctx, other, "", "") },
			TenantA: tenant.ID, UserID: user.ID, Now: pgtest.Now(),
		}
	})
}
