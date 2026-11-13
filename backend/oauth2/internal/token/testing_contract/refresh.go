package testing_contract

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/internal/token/domain"
	"github.com/ambi/idmagic/backend/oauth2/internal/token/ports"
)

type RefreshFixture struct {
	Store    ports.RefreshTokenStore
	TenantID string
	ClientID string
	UserID   string
	Now      time.Time
}

type NewRefreshFixture func(t *testing.T) RefreshFixture

func RunRefresh(t *testing.T, newFixture NewRefreshFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	record := &domain.RefreshTokenRecord{ID: "00000000-0000-4000-8000-000000000011", TenantID: f.TenantID, Hash: "refresh-hash-contract", FamilyID: "00000000-0000-4000-8000-000000000012", ClientID: f.ClientID, UserID: f.UserID, Scopes: []string{"openid"}, IssuedAt: f.Now, ExpiresAt: f.Now.Add(24 * time.Hour), AbsoluteExpiresAt: f.Now.Add(48 * time.Hour)}
	if err := f.Store.Save(ctx, record); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := f.Store.FindByHash(ctx, record.Hash)
	if err != nil || got == nil || got.ID != record.ID {
		t.Fatalf("FindByHash = (%+v, %v)", got, err)
	}
	if got.AuthTime != nil || got.AMR != nil || got.ACR != nil {
		t.Fatalf("認証の値のない記録に値が現れた: %+v", got)
	}

	authTime, acr := f.Now.Add(-time.Hour).Unix(), "urn:idmagic:acr:mfa"
	authenticated := &domain.RefreshTokenRecord{ID: "00000000-0000-4000-8000-000000000013", TenantID: f.TenantID, Hash: "refresh-hash-contract-authenticated", FamilyID: record.FamilyID, ClientID: f.ClientID, UserID: f.UserID, Scopes: []string{"openid"}, IssuedAt: f.Now, ExpiresAt: f.Now.Add(24 * time.Hour), AbsoluteExpiresAt: f.Now.Add(48 * time.Hour), AuthTime: &authTime, AMR: []string{"pwd", "otp"}, ACR: &acr}
	if err := f.Store.Save(ctx, authenticated); err != nil {
		t.Fatalf("Save authenticated: %v", err)
	}
	got, err = f.Store.FindByHash(ctx, authenticated.Hash)
	if err != nil || got == nil {
		t.Fatalf("FindByHash authenticated = (%+v, %v)", got, err)
	}
	if got.AuthTime == nil || *got.AuthTime != authTime || !slices.Equal(got.AMR, []string{"pwd", "otp"}) || got.ACR == nil || *got.ACR != acr {
		t.Fatalf("認証の値が保たれない: auth_time=%v amr=%v acr=%v", got.AuthTime, got.AMR, got.ACR)
	}

	ids, err := f.Store.RevokeFamily(ctx, record.FamilyID)
	if err != nil || len(ids) != 2 || ids[0] != record.ID {
		t.Fatalf("RevokeFamily = (%+v, %v)", ids, err)
	}
	if ids, err := f.Store.RevokeFamily(ctx, record.FamilyID); err != nil || len(ids) != 0 {
		t.Fatalf("RevokeFamily repeat = (%+v, %v)", ids, err)
	}
}
