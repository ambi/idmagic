package db_postgres

import (
	"context"
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	contract "github.com/ambi/idmagic/backend/signingkeys/internal/testing_contract"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

//spec:covers REQ-SIGNINGKEYS-006: PostgreSQL の提供元の準備、切り替えと並行要求は共有契約を満たすこと。
func TestPreparedKeyContract(t *testing.T) {
	contract.Run(t, func(t *testing.T) contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		now := pgtest.Now()
		tenant := &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := tenancy.NewPostgresModule(db).TenantRepo.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
		ctx := tenantports.WithTenant(context.Background(), tenant, "", "")
		store, err := NewKeyStore(ctx, db)
		if err != nil {
			t.Fatal(err)
		}
		return contract.Fixture{Store: store, Context: ctx}
	})
}
