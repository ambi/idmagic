package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/token/domain"
	"github.com/ambi/idmagic/backend/oauth2/token/ports"
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
	record := &domain.RefreshTokenRecord{ID: "00000000-0000-0000-0000-000000000011", TenantID: f.TenantID, Hash: "refresh-hash-contract", FamilyID: "00000000-0000-0000-0000-000000000012", ClientID: f.ClientID, UserID: f.UserID, Scopes: []string{"openid"}, IssuedAt: f.Now, ExpiresAt: f.Now.Add(24 * time.Hour), AbsoluteExpiresAt: f.Now.Add(48 * time.Hour)}
	if err := f.Store.Save(ctx, record); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := f.Store.FindByHash(ctx, record.Hash)
	if err != nil || got == nil || got.ID != record.ID {
		t.Fatalf("FindByHash = (%+v, %v)", got, err)
	}
	ids, err := f.Store.RevokeFamily(ctx, record.FamilyID)
	if err != nil || len(ids) != 1 || ids[0] != record.ID {
		t.Fatalf("RevokeFamily = (%+v, %v)", ids, err)
	}
	if ids, err := f.Store.RevokeFamily(ctx, record.FamilyID); err != nil || len(ids) != 0 {
		t.Fatalf("RevokeFamily repeat = (%+v, %v)", ids, err)
	}
}
