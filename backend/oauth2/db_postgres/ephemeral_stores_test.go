package db_postgres_test

import (
	"context"
	"testing"
	"time"

	oauth2postgres "github.com/ambi/idmagic/backend/oauth2/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
)

// TestReplayStore は DPoP / client-assertion jti リプレイ予約 (ADR-139) を検証する。
// Valkey SETNX + TTL のパリティ: once-only、tenant / kind 名前空間分離、期限切れ後の再予約。
func TestReplayStore(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	other := seedTenant(t, db)
	ctx := tenancy.WithTenant(context.Background(), tenant, "", "")
	otherCtx := tenancy.WithTenant(context.Background(), other, "", "")
	store := &oauth2postgres.ReplayStore{Pool: db, Kind: "dpop"}
	caStore := &oauth2postgres.ReplayStore{Pool: db, Kind: "client_assertion"}
	now := time.Now().UTC()

	if ok, err := store.RecordIfNew(ctx, "jti-1", 60, now); err != nil || !ok {
		t.Fatalf("first ok=%v err=%v (want true)", ok, err)
	}
	if ok, err := store.RecordIfNew(ctx, "jti-1", 60, now); err != nil || ok {
		t.Fatalf("duplicate ok=%v err=%v (want false)", ok, err)
	}
	if ok, err := store.RecordIfNew(otherCtx, "jti-1", 60, now); err != nil || !ok {
		t.Fatalf("tenant-isolated ok=%v err=%v (want true)", ok, err)
	}
	if ok, err := caStore.RecordIfNew(ctx, "jti-1", 60, now); err != nil || !ok {
		t.Fatalf("kind-isolated ok=%v err=%v (want true)", ok, err)
	}
	if ok, err := store.RecordIfNew(ctx, "jti-1", 60, now.Add(2*time.Minute)); err != nil || !ok {
		t.Fatalf("post-expiry ok=%v err=%v (want true)", ok, err)
	}
	if n, err := store.DeleteExpiredBatch(ctx, now.Add(time.Hour), 100); err != nil || n < 1 {
		t.Fatalf("gc n=%d err=%v (want >=1)", n, err)
	}
}

// TestAccessTokenDenylist は access token 失効リスト (ADR-139) を検証する。
// Add → IsRevoked=true、tenant 分離、期限切れエントリは revoked ではない、GC。
func TestAccessTokenDenylist(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	other := seedTenant(t, db)
	ctx := tenancy.WithTenant(context.Background(), tenant, "", "")
	otherCtx := tenancy.WithTenant(context.Background(), other, "", "")
	d := &oauth2postgres.AccessTokenDenylist{Pool: db}
	now := time.Now().UTC()

	if revoked, err := d.IsRevoked(ctx, "jti-a"); err != nil || revoked {
		t.Fatalf("pre-add revoked=%v err=%v (want false)", revoked, err)
	}
	if err := d.Add(ctx, "jti-a", now.Add(time.Hour)); err != nil {
		t.Fatalf("add: %v", err)
	}
	if revoked, err := d.IsRevoked(ctx, "jti-a"); err != nil || !revoked {
		t.Fatalf("post-add revoked=%v err=%v (want true)", revoked, err)
	}
	if revoked, err := d.IsRevoked(otherCtx, "jti-a"); err != nil || revoked {
		t.Fatalf("cross-tenant revoked=%v err=%v (want false)", revoked, err)
	}
	if err := d.Add(ctx, "jti-exp", now.Add(-time.Second)); err != nil {
		t.Fatalf("add expired: %v", err)
	}
	if revoked, err := d.IsRevoked(ctx, "jti-exp"); err != nil || revoked {
		t.Fatalf("expired revoked=%v err=%v (want false)", revoked, err)
	}
	if n, err := d.DeleteExpiredBatch(ctx, now, 100); err != nil || n < 1 {
		t.Fatalf("gc n=%d err=%v (want >=1)", n, err)
	}
}
