package db_postgres

import (
	"context"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
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

	previous, err := repo.ListPageBefore(ctx, tenant.ID, next[0].PreferredUsername, next[0].ID, 2)
	if err != nil {
		t.Fatalf("list previous page: %v", err)
	}
	if len(previous) != 2 || previous[0].PreferredUsername != "alice" || previous[1].PreferredUsername != "bob" {
		t.Fatalf("unexpected previous page: %+v", previous)
	}

	all, err := repo.ListPage(ctx, tenant.ID, "", "", 100)
	if err != nil {
		t.Fatalf("list page all: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}
}

func TestUserRepositoryListPageFiltered(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	otherTenant := seedTenant(t, db)
	repo := &UserRepository{Pool: db}
	ctx := context.Background()
	now := testClock()
	active := idmdomain.UserStatusActive
	disabled := idmdomain.UserStatusDisabled
	for _, fixture := range []struct {
		tenantID, username, name, email string
		roles                           []string
		status                          idmdomain.UserStatus
	}{
		{tenant.ID, "alpha", "Alice Example", "alice@example.com", []string{"billing-admin"}, active},
		{tenant.ID, "bravo", "Bob Example", "bob@example.com", []string{"reader"}, disabled},
		{otherTenant.ID, "other", "Alice Other", "other@example.com", []string{"billing-admin"}, active},
	} {
		name, email := fixture.name, fixture.email
		u := &userdomain.User{
			ID: newUUID(t), TenantID: fixture.tenantID, PreferredUsername: fixture.username,
			PasswordHash: "hash", Name: &name, Email: &email, Roles: fixture.roles,
			Lifecycle: userdomain.UserLifecycle{Status: fixture.status}, CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.Save(ctx, u); err != nil {
			t.Fatalf("save %s: %v", fixture.username, err)
		}
	}

	page, err := repo.ListPageFiltered(ctx, tenant.ID, "ALICE EXAMPLE", &active, "", "", 10)
	if err != nil || len(page) != 1 || page[0].PreferredUsername != "alpha" {
		t.Fatalf("filtered page=%+v err=%v", page, err)
	}
	literalWildcard, err := repo.ListPageFiltered(ctx, tenant.ID, "%", nil, "", "", 10)
	if err != nil || len(literalWildcard) != 0 {
		t.Fatalf("wildcard must be treated literally: page=%+v err=%v", literalWildcard, err)
	}
}
