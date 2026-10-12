package db_memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/apitoken/domain"
)

func TestRepositoryRoundTripTenantIsolationAndIdempotentRevoke(t *testing.T) {
	repository := NewRepository()
	ctx := context.Background()
	now := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	token := &domain.ApiToken{
		ID: "token-1", TenantID: "tenant-1", JTI: "jti-1",
		Scopes: domain.Scopes{domain.ScopeScimUsersRead}, CreatedAt: now,
	}
	if err := repository.Save(ctx, token); err != nil {
		t.Fatal(err)
	}
	token.Scopes[0] = domain.ScopeScimUsersWrite

	found, err := repository.FindByJTI(ctx, "tenant-1", "jti-1")
	if err != nil || found == nil || !found.Scopes.Has(domain.ScopeScimUsersRead) {
		t.Fatalf("found = %+v, err = %v", found, err)
	}
	found.Scopes[0] = domain.ScopeScimGroupsWrite
	foundAgain, _ := repository.FindByJTI(ctx, "tenant-1", "jti-1")
	if !foundAgain.Scopes.Has(domain.ScopeScimUsersRead) {
		t.Fatalf("repository leaked mutable scope slice: %+v", foundAgain.Scopes)
	}

	otherTenant, err := repository.List(ctx, "tenant-2")
	if err != nil || len(otherTenant) != 0 {
		t.Fatalf("other tenant list = %+v, err = %v", otherTenant, err)
	}
	if err := repository.Revoke(ctx, "tenant-2", "token-1", now); err != nil {
		t.Fatal(err)
	}
	if stillThere, _ := repository.FindByJTI(ctx, "tenant-1", "jti-1"); stillThere == nil || stillThere.RevokedAt != nil {
		t.Fatal("cross-tenant delete removed token")
	}
	if err := repository.Revoke(ctx, "tenant-1", "token-1", now); err != nil {
		t.Fatal(err)
	}
	if err := repository.Revoke(ctx, "tenant-1", "token-1", now); err != nil {
		t.Fatalf("idempotent delete failed: %v", err)
	}
	if revoked, _ := repository.FindByJTI(ctx, "tenant-1", "jti-1"); revoked == nil || revoked.RevokedAt == nil {
		t.Fatalf("revocation tombstone missing: %+v", revoked)
	}
}
