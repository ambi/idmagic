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

// TestAuthorizationRequestStore は /authorize 中間状態の tx 直列化された状態遷移 (ADR-139) を
// 検証する。Save/Find、UpdateState の TransitionAuthorizationCodeFlow、AttachAuthentication、
// 不正遷移エラー、tenant 分離、GC を memory adapter とのパリティで確認する。
func TestAuthorizationRequestStore(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	other := seedTenant(t, db)
	ctx := tenancy.WithTenant(context.Background(), tenant, "", "")
	otherCtx := tenancy.WithTenant(context.Background(), other, "", "")
	store := &oauth2postgres.AuthorizationRequestStore{Pool: db}
	now := time.Now().UTC()

	req := &domain.AuthorizationRequest{
		ID: "req-1", TenantID: tenant.ID, State: spec.AuthFlowReceived, ClientID: "client-1",
		RedirectURI: "https://c/cb", ResponseType: spec.ResponseTypeCode, Scope: "openid",
		ExpiresAt: now.Add(time.Minute),
	}
	if err := store.Save(ctx, req); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := store.Find(ctx, "req-1")
	if err != nil || got == nil || got.State != spec.AuthFlowReceived || got.ClientID != "client-1" {
		t.Fatalf("find: %v %+v", err, got)
	}
	if err := store.UpdateState(ctx, "req-1", spec.AuthFlowAuthenticationPending); err != nil {
		t.Fatalf("update state: %v", err)
	}
	if got, err := store.Find(ctx, "req-1"); err != nil || got == nil || got.State != spec.AuthFlowAuthenticationPending {
		t.Fatalf("find after transition=%+v err=%v (want authentication_pending)", got, err)
	}
	if err := store.AttachAuthentication(ctx, "req-1", "user-1", now.Unix(), []string{"pwd"}, "acr-1", "sid-1"); err != nil {
		t.Fatalf("attach: %v", err)
	}
	got, err = store.Find(ctx, "req-1")
	if err != nil || got == nil || got.UserID == nil || *got.UserID != "user-1" ||
		got.ACR == nil || *got.ACR != "acr-1" || got.Sid == nil || *got.Sid != "sid-1" ||
		len(got.AMR) != 1 || got.AMR[0] != "pwd" {
		t.Fatalf("find after attach=%+v err=%v", got, err)
	}
	// invalid transition: authentication_pending → exchanged is not a legal edge.
	if err := store.UpdateState(ctx, "req-1", spec.AuthFlowExchanged); err == nil {
		t.Fatal("invalid transition: want error")
	}
	// missing id
	if err := store.UpdateState(ctx, "missing", spec.AuthFlowAuthenticationPending); err == nil {
		t.Fatal("update missing: want error")
	}
	// tenant isolation
	if got, err := store.Find(otherCtx, "req-1"); err != nil || got != nil {
		t.Fatalf("cross-tenant find=%+v err=%v (want nil)", got, err)
	}
	// GC
	expired := &domain.AuthorizationRequest{ID: "req-exp", TenantID: tenant.ID, State: spec.AuthFlowReceived, ClientID: "c", ExpiresAt: now.Add(-time.Second)}
	if err := store.Save(ctx, expired); err != nil {
		t.Fatalf("save expired: %v", err)
	}
	if n, err := store.DeleteExpiredBatch(ctx, now, 100); err != nil || n < 1 {
		t.Fatalf("gc n=%d err=%v (want >=1)", n, err)
	}
}
