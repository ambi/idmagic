package db_postgres_test

import (
	"testing"

	postgres "github.com/ambi/idmagic/backend/oauth2/internal/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/internal/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

//spec:covers REQ-OAUTH2-097: 影響度の保存、解除、テナント分離の共有契約を PostgreSQL で満たす。
func TestPermissionImpactPersistence(t *testing.T) {
	db := pgtest.Require(t)
	testing_contract.RunPermissionImpact(t, testing_contract.ImpactFixture{Types: &postgres.AuthorizationDetailTypeRepository{Pool: db}, Servers: &postgres.McpResourceServerRepository{Pool: db}, TenantID: seedTenant(t, db).ID, OtherTenantID: seedTenant(t, db).ID, ServerID: newUUID(t), Now: pgtest.Now()})
}
