package db_postgres

import (
	"context"
	"testing"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
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

func TestUserRepositoryListPage(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	repo := &UserRepository{Pool: db}
	ctx := context.Background()
	now := testClock()

	for _, name := range []string{"charlie", "alice", "bob", "delta", "echo"} {
		u := &userdomain.User{
			ID: newUUID(t), TenantID: tenant.ID, PreferredUsername: name,
			PasswordHash: "hash", CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.Save(ctx, u); err != nil {
			t.Fatalf("save %s: %v", name, err)
		}
	}

	first, err := repo.ListPage(ctx, tenant.ID, "", "", 2)
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if len(first) != 2 || first[0].PreferredUsername != "alice" || first[1].PreferredUsername != "bob" {
		t.Fatalf("unexpected first page: %+v", first)
	}

	last := first[len(first)-1]
	next, err := repo.ListPage(ctx, tenant.ID, last.PreferredUsername, last.ID, 2)
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(next) != 2 || next[0].PreferredUsername != "charlie" || next[1].PreferredUsername != "delta" {
		t.Fatalf("unexpected continuation page: %+v", next)
	}

	all, err := repo.ListPage(ctx, tenant.ID, "", "", 100)
	if err != nil {
		t.Fatalf("list page all: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}
}
