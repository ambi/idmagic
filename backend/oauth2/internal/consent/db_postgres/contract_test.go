package db_postgres

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"

	clientpostgres "github.com/ambi/idmagic/backend/oauth2/internal/client/db_postgres"

	"github.com/ambi/idmagic/backend/oauth2/internal/consent/domain"
	"github.com/ambi/idmagic/backend/oauth2/internal/consent/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db, nil).UserRepo, tenant.ID)
	client := pgfixtures.SeedClient(t, &clientpostgres.OAuth2ClientRepository{Pool: db}, tenant.ID)
	now := pgfixtures.TestClock()
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: &ConsentRepository{Pool: db}, TenantID: tenant.ID, Consent: &domain.Consent{UserID: user.ID, ClientID: client.ClientID, Scopes: []string{"openid"}, State: domain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(1000 * 24 * time.Hour)}}
	})
}
