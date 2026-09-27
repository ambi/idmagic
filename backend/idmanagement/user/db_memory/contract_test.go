package db_memory

import (
	"context"
	"testing"
	"time"

	auditmemory "github.com/ambi/idmagic/backend/audit/db_memory"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	passwordmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/idmanagement/user/testing_contract"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
)

func testingContractUser(_ *testing.T, tenantID, username string, now time.Time) *userdomain.User {
	email := username + "@example.com"
	return &userdomain.User{
		ID: username, TenantID: tenantID, PreferredUsername: username,
		PasswordHash: "hash", Email: &email, Roles: []string{"reader"},
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	}
}

func newContractFixture(t *testing.T) testing_contract.Fixture {
	t.Helper()
	users := NewUserRepository()
	history := passwordmemory.NewPasswordHistoryRepository()
	quota := tenancymemory.NewQuotaRepository()
	audit := auditmemory.NewAuditEventStore(10)
	now := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	emailUser := testingContractUser(t, "tenant-b", "email-user", now)
	users.Seed(emailUser)
	actor := testingContractUser(t, "tenant-b", "actor", now)
	users.Seed(actor)

	return testing_contract.Fixture{
		Users:       users,
		EmailTokens: NewEmailChangeTokenStore(users),
		Schemas:     NewTenantUserAttributeSchemaRepository(),
		ImportRows: UserImportRowCommitter{
			Users: users, PasswordHistory: history, Quota: quota, Audit: audit,
		},
		TenantA: "tenant-a", TenantB: "tenant-b", EmailUser: emailUser,
		ActorUserID: actor.ID, Now: now,
		AssertImported: func(t *testing.T, mutation userports.UserImportRowMutation) {
			t.Helper()
			ctx := context.Background()
			if got, err := users.FindBySub(ctx, mutation.After.ID); err != nil || got == nil {
				t.Fatalf("imported user = (%+v, %v)", got, err)
			}
			entries, err := history.Recent(ctx, mutation.After.ID, 1)
			if err != nil || len(entries) != 1 || entries[0].Encoded != mutation.PasswordHistoryHash {
				t.Fatalf("password history = (%+v, %v)", entries, err)
			}
			events, err := audit.List(ctx, auditports.AuditEventQuery{TenantID: mutation.After.TenantID})
			if err != nil || len(events) != 1 || events[0].Type != mutation.AuditEventType {
				t.Fatalf("audit events = (%+v, %v)", events, err)
			}
		},
	}
}

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, newContractFixture)
}
