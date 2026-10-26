package db_postgres

import (
	"testing"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	"github.com/ambi/idmagic/backend/idmanagement/internal/group/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

//spec:covers REQ-IDMANAGEMENT-092, REQ-IDMANAGEMENT-093: PostgreSQL 保存で所属を保って未承認ロールを除き、通常保存で承認の取消しを戻させない。
func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: &GroupRepository{Pool: db}, TenantID: tenant.ID, UserID: user.ID, Group: &groupdomain.Group{ID: newUUID(t), TenantID: tenant.ID, Name: uniqueID("contract-group"), Roles: []string{}, CreatedAt: testClock(), UpdatedAt: testClock()}, Now: testClock()}
	})
}
