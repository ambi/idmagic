package db_postgres

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/apitoken/domain"
	userpg "github.com/ambi/idmagic/backend/idmanagement/user/db_postgres"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
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

func TestRepositoryRoundTripScopesAndTenantRevoke(t *testing.T) {
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
	userID := newUUID(t)
	if err := (&userpg.UserRepository{Pool: db}).Save(ctx, &userdomain.User{
		ID: userID, TenantID: tenant.ID,
		PreferredUsername: "api-token-owner", PasswordHash: "test-hash", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	token := &domain.ApiToken{
		ID: newUUID(t), TenantID: tenant.ID, UserID: userID, JTI: "jti-" + newUUID(t),
		ClientID: "idmagic-api-token", Audience: "https://idp.test/realms/" + tenant.Realm,
		Scopes:      domain.Scopes{domain.ScopeScimUsersRead, domain.ScopeScimUsersWrite},
		Description: "SCIM", CreatedAt: now, ExpiresAt: &expiresAt,
	}
	repository := &Repository{Pool: db}
	if err := repository.Save(ctx, token); err != nil {
		t.Fatal(err)
	}
	found, err := repository.FindByJTI(ctx, tenant.ID, token.JTI)
	if err != nil || found == nil || found.ID != token.ID || !found.Scopes.Has(domain.ScopeScimUsersWrite) {
		t.Fatalf("found = %+v, err = %v", found, err)
	}
	listed, err := repository.List(ctx, tenant.ID)
	if err != nil || len(listed) != 1 || listed[0].Description != "SCIM" {
		t.Fatalf("listed = %+v, err = %v", listed, err)
	}
	if err := repository.Revoke(ctx, newUUID(t), token.ID, now); err != nil {
		t.Fatal(err)
	}
	if stillThere, _ := repository.FindByJTI(ctx, tenant.ID, token.JTI); stillThere == nil || stillThere.RevokedAt != nil {
		t.Fatal("cross-tenant delete removed token")
	}
	if err := repository.Revoke(ctx, tenant.ID, token.ID, now); err != nil {
		t.Fatal(err)
	}
	if revoked, _ := repository.FindByJTI(ctx, tenant.ID, token.JTI); revoked == nil || revoked.RevokedAt == nil {
		t.Fatalf("revocation tombstone missing: %+v", revoked)
	}
}
