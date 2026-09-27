package db_postgres

import (
	"testing"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	"github.com/ambi/idmagic/backend/idmanagement/group/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: &GroupRepository{Pool: db}, TenantID: tenant.ID, UserID: user.ID, Group: &groupdomain.Group{ID: newUUID(t), TenantID: tenant.ID, Name: uniqueID("contract-group"), Roles: []string{}, CreatedAt: testClock(), UpdatedAt: testClock()}, Now: testClock()}
	})
}
