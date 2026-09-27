package db_postgres

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/token/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancypg "github.com/ambi/idmagic/backend/tenancy/db_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestDenylistPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	now := pgtest.Now()
	tenant := &tenancydomain.Tenant{ID: "55555555-5555-5555-5555-555555555551", Realm: "denylist-contract", DisplayName: "Denylist Contract", Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := (&tenancypg.TenantRepository{Pool: db}).Save(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	testing_contract.RunDenylist(t, func(*testing.T) testing_contract.DenylistFixture {
		return testing_contract.DenylistFixture{Denylist: &AccessTokenDenylist{Pool: db}, Context: tenancy.WithTenant(context.Background(), tenant, "", ""), Now: now.Add(1000 * 24 * time.Hour)}
	})
}
