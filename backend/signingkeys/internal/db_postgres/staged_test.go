package db_postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"

	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
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

//spec:covers REQ-SIGNINGKEYS-003: PostgreSQL の提供元のスコープの列挙は共有契約を満たすこと。
func TestKeyScopesContract(t *testing.T) {
	contract.RunKeyScopes(t, func(t *testing.T) contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		ctx := newContractTenant(t, db)
		store, err := NewKeyStore(ctx, db)
		if err != nil {
			t.Fatal(err)
		}
		return contract.Fixture{Store: store, Context: ctx}
	})
}

// newContractTenant は、パッケージ内のテストが共有するデータベースに、ほかのテストの鍵を見ない
// 新しいテナントを保存し、その文脈を返す。
func newContractTenant(t *testing.T, db postgres.DB) context.Context {
	t.Helper()
	now := pgtest.Now()
	id := uuid.NewString()
	tenant := &tenancydomain.Tenant{ID: id, Realm: "contract-" + id, DisplayName: "Contract", Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := tenancy.NewPostgresModule(db).TenantRepo.Save(context.Background(), tenant); err != nil {
		t.Fatal(err)
	}
	return tenantports.WithTenant(context.Background(), tenant, "", "")
}
