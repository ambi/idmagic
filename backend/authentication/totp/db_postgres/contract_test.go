package db_postgres

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/authentication/totp/testing_contract"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenant := pgfixtures.SeedTenant(t, db)
		user := pgfixtures.SeedUser(t, db, tenant.ID)
		return testing_contract.Fixture{
			Repository: &MfaFactorRepository{Pool: db, Cipher: newTestCipher(t, tenant.ID)},
			Context:    func(ctx context.Context) context.Context { return tenancy.WithTenant(ctx, tenant, "", "") },
			UserID:     user.ID, Now: pgtest.Now(),
		}
	})
}
