package db_postgres

import (
	"context"
	"testing"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/idmanagement/user/testing_contract"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func newContractFixture(t *testing.T) testing_contract.Fixture {
	t.Helper()
	db := pgtest.Require(t)
	tenantA := seedTenant(t, db)
	tenantB := seedTenant(t, db)
	emailUser := seedUser(t, db, tenantB.ID)
	actor := seedUser(t, db, tenantB.ID)
	return testing_contract.Fixture{
		Users:       &UserRepository{Pool: db},
		EmailTokens: &EmailChangeTokenStore{Pool: db},
		Schemas:     &TenantUserAttributeSchemaRepository{Pool: db},
		ImportRows:  UserImportRowCommitter{Pool: db},
		TenantA:     tenantA.ID, TenantB: tenantB.ID, EmailUser: emailUser,
		ActorUserID: actor.ID, Now: pgtest.Now(),
		AssertImported: func(t *testing.T, mutation userports.UserImportRowMutation) {
			t.Helper()
			assertPostgresImport(t, db, mutation)
		},
	}
}

func assertPostgresImport(t *testing.T, db sharedpg.DB, mutation userports.UserImportRowMutation) {
	t.Helper()
	ctx := context.Background()
	if got, err := (&UserRepository{Pool: db}).FindBySub(ctx, mutation.After.ID); err != nil || got == nil {
		t.Fatalf("imported user = (%+v, %v)", got, err)
	}
	var encoded string
	if err := db.QueryRow(ctx, `SELECT encoded FROM password_history WHERE user_id = $1`, mutation.After.ID).Scan(&encoded); err != nil || encoded != mutation.PasswordHistoryHash {
		t.Fatalf("password history = (%q, %v)", encoded, err)
	}
	var eventType string
	if err := db.QueryRow(ctx, `SELECT type FROM audit_events WHERE tenant_id = $1 AND user_id = $2`, mutation.After.TenantID, mutation.After.ID).Scan(&eventType); err != nil || eventType != mutation.AuditEventType {
		t.Fatalf("audit event = (%q, %v)", eventType, err)
	}
}

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, newContractFixture)
}
