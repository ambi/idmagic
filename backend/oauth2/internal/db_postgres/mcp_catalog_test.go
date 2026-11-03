package db_postgres_test

import (
	"testing"

	oauthpg "github.com/ambi/idmagic/backend/oauth2/internal/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/internal/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

//spec:covers REQ-OAUTH2-085: PostgreSQLの条件付き保存は古い操作を復活させない。
func TestMcpCatalogCAS(t *testing.T) {
	db := pgtest.Require(t)
	testing_contract.RunMcpCatalogCAS(t, &oauthpg.McpResourceServerRepository{Pool: db}, seedTenant(t, db).ID, newUUID(t), testClock())
}
