package db_postgres_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/oauth2/internal/approval/testing_contract"
	oauth2postgres "github.com/ambi/idmagic/backend/oauth2/internal/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

//spec:covers REQ-OAUTH2-093: 同じ操作の保留の再利用、一人 10 件の上限、並行の起票の直列化、承認後の一回の使用
func TestResourceServerStartContract(t *testing.T) {
	db := pgtest.Require(t)
	testing_contract.RunResourceServerStartContract(t, func(t *testing.T) testing_contract.ResourceServerStartFixture {
		t.Helper()
		tenant := seedTenant(t, db)
		return testing_contract.ResourceServerStartFixture{
			Store:   &oauth2postgres.ApprovalRequestStore{Pool: db},
			Context: tenantports.WithTenant(t.Context(), tenant, "", ""),
			UserID:  seedUser(t, db, tenant.ID).ID, ClientID: seedClient(t, db, tenant.ID).ClientID,
			OtherUser: seedUser(t, db, tenant.ID).ID,
			NewID:     func() string { return newUUID(t) },
		}
	})
}
