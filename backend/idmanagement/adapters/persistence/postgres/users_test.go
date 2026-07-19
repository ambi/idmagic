package postgres

import (
	"context"
	"testing"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
)

func TestUserRepositorySaveAndFind(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	repo := &UserRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	user := &userdomain.User{
		ID:                newUUID(t),
		TenantID:          tenant.ID,
		PreferredUsername: "alice",
		PasswordHash:      "hash",
		Email:             new("alice@example.com"),
		EmailVerified:     true,
		Roles:             []string{"admin"},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := repo.Save(ctx, user); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.FindBySub(ctx, user.ID)
	if err != nil {
		t.Fatalf("find by sub: %v", err)
	}
	if got == nil || got.PreferredUsername != "alice" || got.Email == nil || *got.Email != "alice@example.com" {
		t.Fatalf("unexpected user: %+v", got)
	}

	byName, err := repo.FindByUsername(ctx, tenant.ID, "alice")
	if err != nil || byName == nil || byName.ID != user.ID {
		t.Fatalf("find by username: %v, %+v", err, byName)
	}

	byEmail, err := repo.FindByEmail(ctx, tenant.ID, "ALICE@example.com")
	if err != nil || byEmail == nil || byEmail.ID != user.ID {
		t.Fatalf("find by email (case-insensitive): %v, %+v", err, byEmail)
	}

	all, err := repo.FindAll(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("find all: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("find all len=%d, want 1", len(all))
	}
}
