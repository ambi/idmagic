package db_postgres

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/apitoken/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenancypg "github.com/ambi/idmagic/backend/tenancy/db_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func newUUID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestRepositoryRoundTripScopesAndTenantDelete(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	tenant := &tenancydomain.Tenant{
		ID: newUUID(t), Realm: "api-token-" + newUUID(t)[:8], DisplayName: "API Token Test",
		Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := (&tenancypg.TenantRepository{Pool: db}).Save(ctx, tenant); err != nil {
		t.Fatal(err)
	}
	expiresAt := now.Add(time.Hour)
	token := &domain.ApiToken{
		ID: newUUID(t), TenantID: tenant.ID, TokenHash: "hash-" + newUUID(t),
		Scopes:      domain.Scopes{domain.ScopeScimUsersRead, domain.ScopeScimUsersWrite},
		Description: "SCIM", CreatedAt: now, ExpiresAt: &expiresAt,
	}
	repository := &Repository{Pool: db}
	if err := repository.Save(ctx, token); err != nil {
		t.Fatal(err)
	}
	found, err := repository.FindByHash(ctx, token.TokenHash)
	if err != nil || found == nil || found.ID != token.ID || !found.Scopes.Has(domain.ScopeScimUsersWrite) {
		t.Fatalf("found = %+v, err = %v", found, err)
	}
	listed, err := repository.List(ctx, tenant.ID)
	if err != nil || len(listed) != 1 || listed[0].Description != "SCIM" {
		t.Fatalf("listed = %+v, err = %v", listed, err)
	}
	if err := repository.Delete(ctx, newUUID(t), token.ID); err != nil {
		t.Fatal(err)
	}
	if stillThere, _ := repository.FindByHash(ctx, token.TokenHash); stillThere == nil {
		t.Fatal("cross-tenant delete removed token")
	}
	if err := repository.Delete(ctx, tenant.ID, token.ID); err != nil {
		t.Fatal(err)
	}
	if deleted, _ := repository.FindByHash(ctx, token.TokenHash); deleted != nil {
		t.Fatalf("deleted token remains: %+v", deleted)
	}
}
