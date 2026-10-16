package db_postgres

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"

	"github.com/ambi/idmagic/backend/idmanagement"

	clientpostgres "github.com/ambi/idmagic/backend/oauth2/internal/client/db_postgres"

	authsessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/oauth2/internal/logout/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db, nil).UserRepo, tenant.ID)
	client := pgfixtures.SeedClient(t, &clientpostgres.OAuth2ClientRepository{Pool: db}, tenant.ID)
	now := pgtest.Now()
	if err := authentication.NewPostgresModule(db, nil, nil).SessionStore.Save(context.Background(), &authsessiondomain.LoginSession{ID: "00000000-0000-0000-0000-000000000002", TenantID: tenant.ID, UserID: user.ID, AuthTime: now.Unix(), AMR: []string{}, ACR: "", ExpiresAt: now.Add(time.Hour), LastSeenAt: now}); err != nil {
		t.Fatalf("seed authentication session: %v", err)
	}
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Sessions: &ClientSessionStore{Pool: db}, Notifications: &NotificationStore{Pool: db}, TenantID: tenant.ID, Now: now, ClientID: client.ClientID}
	})
}
