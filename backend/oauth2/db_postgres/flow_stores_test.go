package db_postgres_test

import (
	"context"
	"testing"
	"time"

	oauth2postgres "github.com/ambi/idmagic/backend/oauth2/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
)

// TestPARStore は Pushed Authorization Request の単発消費 (ADR-139) を検証する。
func TestPARStore(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	other := seedTenant(t, db)
	ctx := tenancy.WithTenant(context.Background(), tenant, "", "")
	otherCtx := tenancy.WithTenant(context.Background(), other, "", "")
	store := &oauth2postgres.PARStore{Pool: db}
	now := time.Now().UTC()

	rec := &domain.PARRecord{
		RequestURI: "urn:par:1", TenantID: tenant.ID, ClientID: "client-1",
		Parameters: map[string]string{"scope": "openid"}, IssuedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	if err := store.Save(ctx, rec); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := store.Find(ctx, "urn:par:1")
	if err != nil || got == nil || got.Used || got.ClientID != "client-1" || got.Parameters["scope"] != "openid" {
		t.Fatalf("find: %v %+v", err, got)
	}
	consumed, err := store.Consume(ctx, "urn:par:1")
	if err != nil || consumed == nil || !consumed.Used {
		t.Fatalf("consume: %v %+v (want used=true)", err, consumed)
	}
	if again, err := store.Consume(ctx, "urn:par:1"); err != nil || again != nil {
		t.Fatalf("second consume=%+v err=%v (want nil)", again, err)
	}
	if got, err := store.Find(ctx, "urn:par:1"); err != nil || got == nil || !got.Used {
		t.Fatalf("find after consume=%+v err=%v (want used=true overlay)", got, err)
	}
	expired := &domain.PARRecord{RequestURI: "urn:par:exp", TenantID: tenant.ID, ClientID: "c", IssuedAt: now, ExpiresAt: now.Add(-time.Second)}
	if err := store.Save(ctx, expired); err != nil {
		t.Fatalf("save expired: %v", err)
	}
	if c, err := store.Consume(ctx, "urn:par:exp"); err != nil || c != nil {
		t.Fatalf("expired consume=%+v err=%v (want nil)", c, err)
	}
	if got, err := store.Find(otherCtx, "urn:par:1"); err != nil || got != nil {
		t.Fatalf("cross-tenant find=%+v err=%v (want nil)", got, err)
	}
	if n, err := store.DeleteExpiredBatch(ctx, now, 100); err != nil || n < 1 {
		t.Fatalf("gc n=%d err=%v (want >=1)", n, err)
	}
}

// TestAuthorizationCodeStore は認可コードの単発 redeem (ADR-139) を検証する。
func TestAuthorizationCodeStore(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	ctx := tenancy.WithTenant(context.Background(), tenant, "", "")
	store := &oauth2postgres.AuthorizationCodeStore{Pool: db}
	now := time.Now().UTC()

	rec := &domain.AuthorizationCodeRecord{
		Code: "code-1", TenantID: tenant.ID, ClientID: "client-1", UserID: "user-1",
		Scopes: []string{"openid"}, RedirectURI: "https://c/cb", State: spec.AuthCodeRecordIssued,
		AuthTime: now.Unix(), IssuedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	if err := store.Save(ctx, rec); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got, err := store.Find(ctx, "code-1"); err != nil || got == nil || got.State != spec.AuthCodeRecordIssued {
		t.Fatalf("find: %v %+v", err, got)
	}
	redeemed, err := store.Redeem(ctx, "code-1", now)
	if err != nil || redeemed == nil || redeemed.State != spec.AuthCodeRecordRedeemed || redeemed.RedeemedAt == nil {
		t.Fatalf("redeem: %v %+v (want redeemed + redeemed_at)", err, redeemed)
	}
	if again, err := store.Redeem(ctx, "code-1", now); err != nil || again != nil {
		t.Fatalf("second redeem=%+v err=%v (want nil, replay)", again, err)
	}
	if got, err := store.Find(ctx, "code-1"); err != nil || got == nil || got.State != spec.AuthCodeRecordRedeemed {
		t.Fatalf("find after redeem=%+v err=%v (want redeemed overlay)", got, err)
	}
	if err := store.LinkFamily(ctx, "code-1", "fam-1"); err != nil {
		t.Fatalf("link family: %v", err)
	}
	if got, err := store.Find(ctx, "code-1"); err != nil || got == nil || got.IssuedFamilyID == nil || *got.IssuedFamilyID != "fam-1" {
		t.Fatalf("find after link=%+v err=%v (want fam-1)", got, err)
	}
	if err := store.LinkFamily(ctx, "missing", "fam-x"); err == nil {
		t.Fatal("link family on missing code: want error")
	}
}

// TestDeviceCodeStore は RFC 8628 device flow の状態遷移 (ADR-139) を検証する。
func TestDeviceCodeStore(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	ctx := tenancy.WithTenant(context.Background(), tenant, "", "")
	store := &oauth2postgres.DeviceCodeStore{Pool: db}
	now := time.Now().UTC()

	rec := &domain.DeviceAuthorization{
		DeviceCodeHash: "hash-1", TenantID: tenant.ID, UserCode: "USER-CODE-1", ClientID: "client-1",
		Scopes: []string{"openid"}, State: spec.DeviceFlowIssued, IntervalSeconds: 5,
		IssuedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	if err := store.Save(ctx, rec); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got, err := store.FindByDeviceCodeHash(ctx, "hash-1"); err != nil || got == nil || got.State != spec.DeviceFlowIssued {
		t.Fatalf("find by hash: %v %+v", err, got)
	}
	if got, err := store.FindByUserCode(ctx, "USER-CODE-1"); err != nil || got == nil || got.DeviceCodeHash != "hash-1" {
		t.Fatalf("find by user code: %v %+v", err, got)
	}
	rec.State = spec.DeviceFlowApproved
	rec.UserID = &user.ID
	if err := store.Update(ctx, rec); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got, err := store.FindByDeviceCodeHash(ctx, "hash-1"); err != nil || got == nil || got.State != spec.DeviceFlowApproved || got.UserID == nil || *got.UserID != user.ID {
		t.Fatalf("find after approve=%+v err=%v (want approved + user)", got, err)
	}
	exchanged, err := store.Exchange(ctx, "hash-1")
	if err != nil || exchanged == nil || exchanged.State != spec.DeviceFlowExchanged {
		t.Fatalf("exchange: %v %+v (want exchanged)", err, exchanged)
	}
	if again, err := store.Exchange(ctx, "hash-1"); err != nil || again != nil {
		t.Fatalf("second exchange=%+v err=%v (want nil)", again, err)
	}
	if err := store.DeleteAllForSub(ctx, user.ID); err != nil {
		t.Fatalf("delete all for sub: %v", err)
	}
	if got, err := store.FindByDeviceCodeHash(ctx, "hash-1"); err != nil || got != nil {
		t.Fatalf("find after delete=%+v err=%v (want nil)", got, err)
	}
}
