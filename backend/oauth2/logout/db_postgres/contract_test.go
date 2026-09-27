package db_postgres

import (
	"context"
	"testing"
	"time"

	authsessionpg "github.com/ambi/idmagic/backend/authentication/session/db_postgres"
	authsessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/oauth2/logout/testing_contract"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	client := pgfixtures.SeedClient(t, db, tenant.ID)
	now := pgtest.Now()
	if err := (&authsessionpg.SessionRepository{Pool: db}).Save(context.Background(), &authsessiondomain.LoginSession{ID: "00000000-0000-0000-0000-000000000002", TenantID: tenant.ID, UserID: user.ID, AuthTime: now.Unix(), AMR: []string{}, ACR: "", ExpiresAt: now.Add(time.Hour), LastSeenAt: now}); err != nil {
		t.Fatalf("seed authentication session: %v", err)
	}
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Sessions: &ClientSessionStore{Pool: db}, Notifications: &NotificationStore{Pool: db}, TenantID: tenant.ID, Now: now, ClientID: client.ClientID}
	})
}
