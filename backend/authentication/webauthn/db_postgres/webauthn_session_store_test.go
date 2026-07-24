package db_postgres_test

import (
	"context"
	"testing"
	"time"

	webauthnpg "github.com/ambi/idmagic/backend/authentication/webauthn/db_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
)

// TestWebAuthnSessionStore は WebAuthn challenge の短命保持 (ADR-087 / ADR-139) を検証する。
// Valkey GetDel のパリティ: round-trip、一度きり消費、期限切れは nil、tenant 分離、GC。
func TestWebAuthnSessionStore(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	other := pgfixtures.SeedTenant(t, db)
	ctx := tenancy.WithTenant(context.Background(), tenant, "", "")
	otherCtx := tenancy.WithTenant(context.Background(), other, "", "")
	store := &webauthnpg.WebAuthnSessionStore{Pool: db}
	now := time.Now().UTC()
	data := gowebauthn.SessionData{Challenge: "abc", UserID: []byte("user-1")}

	if err := store.Save(ctx, "key-1", data, now.Add(time.Minute)); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := store.Take(ctx, "key-1")
	if err != nil || got == nil {
		t.Fatalf("take: %v %+v (want data)", err, got)
	}
	if got.Challenge != "abc" || string(got.UserID) != "user-1" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got2, err := store.Take(ctx, "key-1"); err != nil || got2 != nil {
		t.Fatalf("second take=%+v err=%v (want nil, once-only)", got2, err)
	}
	if err := store.Save(ctx, "key-exp", data, now.Add(-time.Second)); err != nil {
		t.Fatalf("save expired: %v", err)
	}
	if got, err := store.Take(ctx, "key-exp"); err != nil || got != nil {
		t.Fatalf("expired take=%+v err=%v (want nil)", got, err)
	}
	if err := store.Save(ctx, "key-iso", data, now.Add(time.Minute)); err != nil {
		t.Fatalf("save iso: %v", err)
	}
	if got, err := store.Take(otherCtx, "key-iso"); err != nil || got != nil {
		t.Fatalf("cross-tenant take=%+v err=%v (want nil)", got, err)
	}
	if n, err := store.DeleteExpiredBatch(ctx, now, 100); err != nil || n < 1 {
		t.Fatalf("gc n=%d err=%v (want >=1)", n, err)
	}
}
