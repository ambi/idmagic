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
	if revoked, err := f.Store.Revoke(ctx, session.ID, spec.SessionEndAdminRevoke, f.Now); err != nil || !revoked {
		t.Fatalf("Revoke = (%v, %v), want (true, nil)", revoked, err)
	}
	if got, err := f.Store.Find(ctx, session.ID); err != nil || got != nil {
		t.Fatalf("Find revoked = (%+v, %v), want nil", got, err)
	}
	if revoked, err := f.Store.Revoke(ctx, session.ID, spec.SessionEndLogout, f.Now.Add(time.Minute)); err != nil || revoked {
		t.Fatalf("idempotent Revoke = (%v, %v), want (false, nil)", revoked, err)
	}
	owned, err := f.Store.FindOwned(ctx, session.ID, f.UserID)
	if err != nil || owned == nil || owned.RevokedAt == nil || !owned.RevokedAt.Equal(f.Now) ||
		owned.RevokeReason == nil || *owned.RevokeReason != spec.SessionEndAdminRevoke {
		t.Fatalf("FindOwned = (%+v, %v), want the first revocation kept", owned, err)
	}
	if revoked, err := f.Store.Revoke(f.Other(context.Background()), session.ID, spec.SessionEndLogout, f.Now); err != nil || revoked {
		t.Fatalf("cross-tenant Revoke = (%v, %v), want (false, nil)", revoked, err)
	}
	runCountActive(t, f)
}

// runCountActive は、有効なセッションの件数が確認待ちを含み、失効と期限切れ (expires_at が
// 数える時刻ちょうどのものを含む) を除くことを確かめる (REQ-TENANCY-051)。数える時刻を一日先に
// 置き、同じデータベースで期限切れの一括削除を確かめるテストの窓へ行を入れない。
func runCountActive(t *testing.T, f Fixture) {
	t.Helper()
	ctx := f.Context(context.Background())
	tenantA := f.TenantA
	at := f.Now.Add(24 * time.Hour)
	base, err := f.Store.CountActive(ctx, tenantA, at)
	if err != nil {
		t.Fatalf("CountActive: %v", err)
	}
	save := func(ctx context.Context, tenantID string, expiresAt time.Time, pending bool) string {
		t.Helper()
		session := &domain.LoginSession{
			ID: newID(t), TenantID: tenantID, UserID: f.UserID, AuthTime: f.Now.Unix(), AMR: []string{"pwd"},
			ACR: "urn:idmagic:acr:pwd", ExpiresAt: expiresAt, AuthenticationPending: pending,
		}
		if pending {
			session.PendingPurpose = domain.LoginPendingChallenge
		}
		if err := f.Store.Save(ctx, session); err != nil {
			t.Fatalf("Save: %v", err)
		}
		return session.ID
	}
	save(ctx, tenantA, at.Add(time.Second), false)
	save(ctx, tenantA, at.Add(time.Hour), true)
	save(ctx, tenantA, at, false)
	revoked := save(ctx, tenantA, at.Add(time.Hour), false)
	if _, err := f.Store.Revoke(ctx, revoked, spec.SessionEndLogout, at); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if got, err := f.Store.CountActive(ctx, tenantA, at); err != nil || got != base+2 {
		t.Fatalf("CountActive = (%d, %v), want %d (unexpired and pending, not expired at now or revoked)", got, err, base+2)
	}
}
