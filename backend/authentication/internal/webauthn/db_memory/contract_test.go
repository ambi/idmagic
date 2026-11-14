package db_memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/internal/webauthn/testing_contract"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func TestWebAuthnSessionStoreContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		tenant := &tenantdomain.Tenant{ID: "tenant-a"}
		other := &tenantdomain.Tenant{ID: "tenant-b"}
		return testing_contract.Fixture{
			Store: NewWebAuthnSessionStore(), Context: func(ctx context.Context) context.Context { return tenantports.WithTenant(ctx, tenant, "", "") }, Other: func(ctx context.Context) context.Context { return tenantports.WithTenant(ctx, other, "", "") },
			Now: time.Now().UTC(),
		}
	})
}

func TestWebAuthnCredentialRepositoryContract(t *testing.T) {
	testing_contract.RunCredentialRepository(t, func(*testing.T) testing_contract.CredentialFixture {
		tenant := &tenantdomain.Tenant{ID: "tenant-a"}
		return testing_contract.CredentialFixture{
			Repo:    NewWebAuthnCredentialRepository(),
			Context: func(ctx context.Context) context.Context { return tenantports.WithTenant(ctx, tenant, "", "") },
			Owner:   "user-owner",
			Other:   "user-other",
			Now:     time.Now().UTC().Truncate(time.Microsecond),
		}
	})
}
