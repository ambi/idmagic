package db_postgres

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/idmanagement"

	"github.com/ambi/idmagic/backend/authentication/internal/session/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenant := pgfixtures.SeedTenant(t, db)
		other := pgfixtures.SeedTenant(t, db)
		user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db, nil).UserRepo, tenant.ID)
		return testing_contract.Fixture{
			Store:   &SessionRepository{Pool: db},
			Context: func(ctx context.Context) context.Context { return tenantports.WithTenant(ctx, tenant, "", "") },
			Other:   func(ctx context.Context) context.Context { return tenantports.WithTenant(ctx, other, "", "") },
			TenantA: tenant.ID, UserID: user.ID, Now: pgtest.Now(),
		}
	})
}
