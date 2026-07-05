package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/internal/shared/spec"
)

func TestTenantRepositorySaveAndFind(t *testing.T) {
	db := requireDB(t)
	repo := &TenantRepository{Pool: db}
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	tenant := &spec.Tenant{
		ID:          "11111111-1111-1111-1111-111111111111",
		Realm:       "acme",
		DisplayName: "Acme",
		Status:      spec.TenantStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.Save(ctx, tenant); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.FindByID(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got == nil {
		t.Fatal("tenant not found after save")
	}
	if got.DisplayName != "Acme" || got.Status != spec.TenantStatusActive || got.Realm != "acme" {
		t.Fatalf("unexpected tenant: %+v", got)
	}

	// FindByRealm は不変 UUID キーではなく URL slug で解決する (ADR-085)。
	byRealm, err := repo.FindByRealm(ctx, "acme")
	if err != nil {
		t.Fatalf("find by realm: %v", err)
	}
	if byRealm == nil || byRealm.ID != tenant.ID {
		t.Fatalf("find by realm mismatch: %+v", byRealm)
	}

	// Update via upsert.
	tenant.DisplayName = "Acme Inc"
	tenant.UpdatedAt = now.Add(time.Minute)
	if err := repo.Save(ctx, tenant); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err = repo.FindByID(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("find after update: %v", err)
	}
	if got.DisplayName != "Acme Inc" {
		t.Fatalf("display name not updated: %q", got.DisplayName)
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("find all: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("expected at least one tenant")
	}
}

func TestTenantRepositoryFindByIDMissing(t *testing.T) {
	db := requireDB(t)
	repo := &TenantRepository{Pool: db}

	got, err := repo.FindByID(context.Background(), "00000000-0000-0000-0000-0000000000ff")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for missing tenant, got %+v", got)
	}
}
