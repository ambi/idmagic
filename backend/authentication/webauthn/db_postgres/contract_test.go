package db_postgres_test

import (
	"context"
	"testing"

	webauthnpg "github.com/ambi/idmagic/backend/authentication/webauthn/db_postgres"
	"github.com/ambi/idmagic/backend/authentication/webauthn/testing_contract"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func TestWebAuthnSessionStoreContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenant := pgfixtures.SeedTenant(t, db)
		other := pgfixtures.SeedTenant(t, db)
		return testing_contract.Fixture{
			Store:   &webauthnpg.WebAuthnSessionStore{Pool: db},
			Context: func(ctx context.Context) context.Context { return tenantports.WithTenant(ctx, tenant, "", "") },
			Other:   func(ctx context.Context) context.Context { return tenantports.WithTenant(ctx, other, "", "") },
			Now:     pgtest.Now(),
		}
	})
}
