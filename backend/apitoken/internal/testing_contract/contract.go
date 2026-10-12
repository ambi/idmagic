// Package testing_contract defines the shared API-token repository contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/apitoken/domain"
	"github.com/ambi/idmagic/backend/apitoken/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type Fixture struct {
	Repository ports.Repository
	TenantA    string
	TenantB    string
	UserID     string
	Now        time.Time
}

type NewFixture func(t *testing.T) Fixture

func newID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	expires := f.Now.Add(time.Hour)
	token := &domain.ApiToken{
		ID: newID(t), TenantID: f.TenantA, UserID: f.UserID, JTI: "jti-" + newID(t),
		ClientID: domain.BuiltinClientID, Scopes: domain.Scopes{domain.ScopeScimUsersRead, domain.ScopeScimUsersWrite},
		Audience: "https://api.example", Description: "SCIM", CreatedAt: f.Now, ExpiresAt: &expires,
	}
	if err := f.Repository.Save(ctx, token); err != nil {
		t.Fatalf("Save: %v", err)
	}
	found, err := f.Repository.FindByJTI(ctx, f.TenantA, token.JTI)
	if err != nil || found == nil || found.ID != token.ID || !found.Scopes.Has(domain.ScopeScimUsersWrite) {
		t.Fatalf("FindByJTI = (%+v, %v), want round trip", found, err)
	}
	if leaked, err := f.Repository.FindByJTI(ctx, f.TenantB, token.JTI); err != nil || leaked != nil {
		t.Fatalf("other tenant FindByJTI = (%+v, %v)", leaked, err)
	}
	listed, err := f.Repository.List(ctx, f.TenantA)
	if err != nil || len(listed) != 1 || listed[0].Description != token.Description {
		t.Fatalf("List = (%+v, %v)", listed, err)
	}
	if err := f.Repository.Revoke(ctx, f.TenantB, token.ID, f.Now); err != nil {
		t.Fatalf("cross-tenant Revoke: %v", err)
	}
	if got, _ := f.Repository.FindByJTI(ctx, f.TenantA, token.JTI); got == nil || got.RevokedAt != nil {
		t.Fatal("cross-tenant revoke changed the token")
	}
	if err := f.Repository.RevokeByJTI(ctx, f.TenantA, token.JTI, f.Now); err != nil {
		t.Fatalf("RevokeByJTI: %v", err)
	}
	if err := f.Repository.RevokeByJTI(ctx, f.TenantA, token.JTI, f.Now); err != nil {
		t.Fatalf("idempotent RevokeByJTI: %v", err)
	}
	if got, _ := f.Repository.FindByJTI(ctx, f.TenantA, token.JTI); got == nil || got.RevokedAt == nil {
		t.Fatalf("revocation tombstone missing: %+v", got)
	}
}
