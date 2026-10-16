// Package testing_contract defines the shared login-session persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/authentication/session/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type Fixture struct {
	Store   ports.SessionStore
	Context func(context.Context) context.Context
	Other   func(context.Context) context.Context
	TenantA string
	UserID  string
	Now     time.Time
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
	ctx := f.Context(context.Background())
	session := &domain.LoginSession{
		ID: newID(t), TenantID: f.TenantA, UserID: f.UserID, AuthTime: f.Now.Unix(), AMR: []string{"pwd"},
		ACR: "urn:idmagic:acr:pwd", ExpiresAt: f.Now.Add(time.Hour),
	}
	if err := f.Store.Save(ctx, session); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got, err := f.Store.Find(ctx, session.ID); err != nil || got == nil || got.UserID != f.UserID {
		t.Fatalf("Find = (%+v, %v)", got, err)
	}
	if got, err := f.Store.Find(f.Other(context.Background()), session.ID); err != nil || got != nil {
		t.Fatalf("cross-tenant Find = (%+v, %v)", got, err)
	}
	list, err := f.Store.ListBySub(ctx, f.UserID)
	if err != nil || len(list) != 1 || list[0].ID != session.ID {
		t.Fatalf("ListBySub = (%+v, %v)", list, err)
	}
	if err := f.Store.Revoke(ctx, session.ID, spec.SessionEndAdminRevoke, f.Now); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if got, err := f.Store.Find(ctx, session.ID); err != nil || got != nil {
		t.Fatalf("Find revoked = (%+v, %v), want nil", got, err)
	}
	if err := f.Store.Revoke(ctx, session.ID, spec.SessionEndLogout, f.Now.Add(time.Minute)); err != nil {
		t.Fatalf("idempotent Revoke: %v", err)
	}
	owned, err := f.Store.FindOwned(ctx, session.ID, f.UserID)
	if err != nil || owned == nil || owned.RevokedAt == nil {
		t.Fatalf("FindOwned = (%+v, %v), want revoked tombstone", owned, err)
	}
}
