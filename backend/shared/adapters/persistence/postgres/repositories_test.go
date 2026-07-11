package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

func TestRefreshTokenStoreSaveFindRotate(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	client := seedClient(t, db, tenant.ID)
	store := &RefreshTokenStore{Pool: db}
	ctx := context.Background()

	now := testClock()
	family := newUUID(t)
	rec := &domain.RefreshTokenRecord{
		ID:                newUUID(t),
		TenantID:          tenant.ID,
		Hash:              uniqueID("hash"),
		FamilyID:          family,
		ClientID:          client.ClientID,
		UserID:            user.ID,
		Scopes:            []string{"openid", "offline_access"},
		IssuedAt:          now,
		ExpiresAt:         now.Add(time.Hour),
		AbsoluteExpiresAt: now.AddDate(0, 0, 30),
	}
	if err := store.Save(ctx, rec); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := store.FindByHash(ctx, rec.Hash)
	if err != nil || got == nil || got.ID != rec.ID {
		t.Fatalf("find by hash: %v %+v", err, got)
	}

	next := &domain.RefreshTokenRecord{
		ID:                newUUID(t),
		TenantID:          tenant.ID,
		Hash:              uniqueID("hash"),
		FamilyID:          family,
		ParentID:          &rec.ID,
		ClientID:          client.ClientID,
		UserID:            user.ID,
		Scopes:            []string{"openid", "offline_access"},
		IssuedAt:          now.Add(time.Minute),
		ExpiresAt:         now.Add(2 * time.Hour),
		AbsoluteExpiresAt: now.AddDate(0, 0, 30),
	}
	rotated, err := store.Rotate(ctx, rec.ID, next)
	if err != nil || rotated == nil {
		t.Fatalf("rotate: %v %+v", err, rotated)
	}

	// 2度目のローテーションは親が rotated 済みのため nil を返す (reuse detection)。
	again, err := store.Rotate(ctx, rec.ID, next)
	if err != nil || again != nil {
		t.Fatalf("second rotate should be nil: %v %+v", err, again)
	}

	if err := store.RevokeFamily(ctx, family); err != nil {
		t.Fatalf("revoke family: %v", err)
	}
	got, err = store.FindByHash(ctx, next.Hash)
	if err != nil || got == nil || !got.Revoked {
		t.Fatalf("expected revoked token: %v %+v", err, got)
	}

	if err := store.DeleteAllForSub(ctx, user.ID); err != nil {
		t.Fatalf("delete all for sub: %v", err)
	}
}
