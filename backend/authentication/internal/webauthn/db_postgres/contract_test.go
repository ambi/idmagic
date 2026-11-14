package db_postgres_test

import (
	"context"
	"testing"

	webauthnpg "github.com/ambi/idmagic/backend/authentication/internal/webauthn/db_postgres"
	"github.com/ambi/idmagic/backend/authentication/internal/webauthn/testing_contract"
	"github.com/ambi/idmagic/backend/idmanagement"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
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

//spec:covers REQ-AUTHENTICATION-038: 既に保存された credential ID の保存を、本人か他人かを問わず ErrWebAuthnCredentialExists で拒否し、既存の行を変えないこと
func TestWebAuthnCredentialRepositoryContract(t *testing.T) {
	testing_contract.RunCredentialRepository(t, func(t *testing.T) testing_contract.CredentialFixture {
		t.Helper()
		db := pgtest.Require(t)
		tenant := pgfixtures.SeedTenant(t, db)
		users := idmanagement.NewPostgresModule(db).UserRepo
		return testing_contract.CredentialFixture{
			Repo:    &webauthnpg.WebAuthnCredentialRepository{Pool: db},
			Context: func(ctx context.Context) context.Context { return tenantports.WithTenant(ctx, tenant, "", "") },
			Owner:   pgfixtures.SeedUser(t, users, tenant.ID).ID,
			Other:   pgfixtures.SeedUser(t, users, tenant.ID).ID,
			Now:     pgtest.Now(),
		}
	})
}
