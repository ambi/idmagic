package db_postgres

import (
	"os"
	"testing"

	"github.com/ambi/idmagic/backend/shared/idempotency"
	contract "github.com/ambi/idmagic/backend/shared/idempotency/testing_contract"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

//spec:covers REQ-SYSTEM-036: 実際の PostgreSQL の一意性、期限、主体とテナントの隔離、期限切れ受付への書き込みの拒否、再暗号化の競合を検査する。
func TestPostgresStoreContract(t *testing.T) {
	pool := pgtest.Require(t)
	newTenant := func() string {
		t.Helper()
		id, err := spec.NewUUIDv4()
		if err != nil {
			t.Fatal(err)
		}
		now := pgtest.Now()
		if err := tenancy.NewPostgresModule(pool).TenantRepo.Save(t.Context(), &tenancydomain.Tenant{ID: id, Realm: "idem-" + id[:8], DisplayName: "Idempotency", Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
		return id
	}
	contract.Run(t, func() idempotency.Store { return &Store{Pool: pool} }, newTenant(), newTenant())
}
