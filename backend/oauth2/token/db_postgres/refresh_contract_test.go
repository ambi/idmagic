package db_postgres

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/token/testing_contract"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestRefreshPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	client := pgfixtures.SeedClient(t, db, tenant.ID)
	testing_contract.RunRefresh(t, func(*testing.T) testing_contract.RefreshFixture {
		return testing_contract.RefreshFixture{Store: &RefreshTokenStore{Pool: db}, TenantID: tenant.ID, ClientID: client.ClientID, UserID: user.ID, Now: pgfixtures.TestClock().Add(1000 * 24 * time.Hour)}
	})
}
