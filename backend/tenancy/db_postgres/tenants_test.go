package db_postgres

import (
	"context"
	"testing"
	"time"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestTenantRepositorySaveAndFind(t *testing.T) {
	db := pgtest.Require(t)
	repo := &TenantRepository{Pool: db}
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	tenant := &domain.Tenant{
		ID:          "11111111-1111-1111-1111-111111111111",
		Realm:       "acme",
		DisplayName: "Acme",
		Status:      domain.TenantStatusActive,
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
	if got.DisplayName != "Acme" || got.Status != domain.TenantStatusActive || got.Realm != "acme" {
		t.Fatalf("unexpected tenant: %+v", got)
	}

	// FindByRealm は不変 UUID キーではなく URL slug で解決する。
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

// REQ-TENANCY-019: an override an administrator saves is persisted. Before the
// columns existed, Save dropped the value, so a tenant's policy had no effect at
// all on a PostgreSQL deployment.
func TestTenantRepositoryPersistsPasswordPolicyOverride(t *testing.T) {
	db := pgtest.Require(t)
	repo := &TenantRepository{Pool: db}
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	minLength := 16
	maxAge := 90
	tenant := &domain.Tenant{
		ID: "33333333-3333-3333-3333-333333333341", Realm: "policy-roundtrip",
		DisplayName: "Policy Roundtrip", Status: domain.TenantStatusActive,
		PasswordPolicyOverride: &domain.PasswordPolicyOverride{
			MinLength: &minLength, MaxAgeDays: &maxAge,
		},
		PasswordPolicyUpdatedAt: &now,
		CreatedAt:               now, UpdatedAt: now,
	}
	if err := repo.Save(ctx, tenant); err != nil {
		t.Fatalf("save: %v", err)
	}
	stored, err := repo.FindByID(ctx, tenant.ID)
	if err != nil || stored == nil {
		t.Fatalf("find: %+v %v", stored, err)
	}
	if stored.PasswordPolicyOverride == nil {
		t.Fatal("password policy override was dropped on save")
	}
	if stored.PasswordPolicyOverride.MinLength == nil || *stored.PasswordPolicyOverride.MinLength != minLength {
		t.Fatalf("min_length = %v, want %d", stored.PasswordPolicyOverride.MinLength, minLength)
	}
	if stored.PasswordPolicyOverride.MaxAgeDays == nil || *stored.PasswordPolicyOverride.MaxAgeDays != maxAge {
		t.Fatalf("max_age_days = %v, want %d", stored.PasswordPolicyOverride.MaxAgeDays, maxAge)
	}
	if stored.PasswordPolicyOverride.MaxLength != nil || stored.PasswordPolicyOverride.HistoryDepth != nil {
		t.Fatalf("unset fields must stay unset: %#v", stored.PasswordPolicyOverride)
	}
	if stored.PasswordPolicyUpdatedAt == nil || !stored.PasswordPolicyUpdatedAt.Equal(now) {
		t.Fatalf("password_policy_updated_at = %v, want %v", stored.PasswordPolicyUpdatedAt, now)
	}

	// Clearing the override is persisted as well (back to NULL).
	stored.PasswordPolicyOverride = nil
	if err := repo.Save(ctx, stored); err != nil {
		t.Fatalf("save cleared override: %v", err)
	}
	cleared, err := repo.FindByID(ctx, tenant.ID)
	if err != nil || cleared == nil {
		t.Fatalf("find after clear: %+v %v", cleared, err)
	}
	if cleared.PasswordPolicyOverride != nil {
		t.Fatalf("override should be cleared: %#v", cleared.PasswordPolicyOverride)
	}
}

func TestTenantRepositoryFindByIDMissing(t *testing.T) {
	db := pgtest.Require(t)
	repo := &TenantRepository{Pool: db}

	got, err := repo.FindByID(context.Background(), "00000000-0000-0000-0000-0000000000ff")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for missing tenant, got %+v", got)
	}
}

// wi-285 / endpoint_style がラウンドトリップし、列を知らない呼び出し元が
// 組み立てたゼロ値の Tenant は 'path' として保存される (NOT NULL + CHECK があるため、
// 空文字列を素通しすると保存が落ちる)。
func TestTenantRepositoryPersistsEndpointStyle(t *testing.T) {
	db := pgtest.Require(t)
	repo := &TenantRepository{Pool: db}
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	zeroValue := &domain.Tenant{
		ID: "33333333-3333-3333-3333-333333333340", Realm: "style-default",
		DisplayName: "Style Default", Status: domain.TenantStatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Save(ctx, zeroValue); err != nil {
		t.Fatalf("save zero-value endpoint style: %v", err)
	}
	stored, err := repo.FindByID(ctx, zeroValue.ID)
	if err != nil || stored == nil {
		t.Fatalf("find: %+v %v", stored, err)
	}
	if stored.EndpointStyle != domain.TenantEndpointStylePath {
		t.Fatalf("EndpointStyle = %q, want %q", stored.EndpointStyle, domain.TenantEndpointStylePath)
	}

	stored.EndpointStyle = domain.TenantEndpointStyleSubdomain
	if err := repo.Save(ctx, stored); err != nil {
		t.Fatalf("save subdomain endpoint style: %v", err)
	}
	stored, err = repo.FindByID(ctx, zeroValue.ID)
	if err != nil || stored == nil {
		t.Fatalf("find after switch: %+v %v", stored, err)
	}
	if stored.EndpointStyle != domain.TenantEndpointStyleSubdomain {
		t.Fatalf("EndpointStyle = %q, want %q", stored.EndpointStyle, domain.TenantEndpointStyleSubdomain)
	}
}
