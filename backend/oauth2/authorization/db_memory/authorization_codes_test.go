package db_memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestAuthorizationCodeStore(t *testing.T) {
	ctx := context.Background()
	store := NewAuthorizationCodeStore()

	t.Run("Save and Find", func(t *testing.T) {
		code := &domain.AuthorizationCodeRecord{
			Code:        "auth-code-123",
			TenantID:    "tenant-1",
			State:       spec.AuthCodeRecordIssued,
			Scopes:      []string{"openid", "profile"},
			RedirectURI: "https://example.com/callback",
		}

		err := store.Save(ctx, code)
		if err != nil {
			t.Fatal(err)
		}

		found, err := store.Find(ctx, "auth-code-123")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected code record to be found")
		}
		if found.Code != "auth-code-123" {
			t.Errorf("expected Code to be 'auth-code-123', got %q", found.Code)
		}
		if len(found.Scopes) != 2 || found.Scopes[0] != "openid" || found.Scopes[1] != "profile" {
			t.Errorf("unexpected scopes: %v", found.Scopes)
		}

		// 存在しないコード
		notfound, err := store.Find(ctx, "auth-code-none")
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil for non-existing code")
		}
	})

	t.Run("Redeem", func(t *testing.T) {
		// 正常ケース
		now := time.Now()
		redeemed, err := store.Redeem(ctx, "auth-code-123", now)
		if err != nil {
			t.Fatal(err)
		}
		if redeemed == nil {
			t.Fatal("expected successful redeem")
		}
		if redeemed.State != spec.AuthCodeRecordRedeemed {
			t.Errorf("expected state to be Redeemed, got %v", redeemed.State)
		}
		if redeemed.RedeemedAt == nil || !redeemed.RedeemedAt.Equal(now.UTC()) {
			t.Errorf("expected RedeemedAt to be %v, got %v", now.UTC(), redeemed.RedeemedAt)
		}

		// すでに Redeem されたコードを再 Redeem (失敗するべき)
		again, err := store.Redeem(ctx, "auth-code-123", now)
		if err != nil {
			t.Fatal(err)
		}
		if again != nil {
			t.Error("expected second redeem to return nil")
		}

		// 存在しないコードの Redeem
		noCode, err := store.Redeem(ctx, "auth-code-none", now)
		if err != nil {
			t.Fatal(err)
		}
		if noCode != nil {
			t.Error("expected nil for redeeming non-existing code")
		}
	})

	t.Run("LinkFamily", func(t *testing.T) {
		code := &domain.AuthorizationCodeRecord{
			Code:  "auth-code-link",
			State: spec.AuthCodeRecordIssued,
		}
		_ = store.Save(ctx, code)

		err := store.LinkFamily(ctx, "auth-code-link", "family-999")
		if err != nil {
			t.Fatal(err)
		}

		found, err := store.Find(ctx, "auth-code-link")
		if err != nil {
			t.Fatal(err)
		}
		if found.IssuedFamilyID == nil || *found.IssuedFamilyID != "family-999" {
			t.Errorf("expected IssuedFamilyID to be 'family-999', got %v", found.IssuedFamilyID)
		}

		// 存在しないコード
		err = store.LinkFamily(ctx, "auth-code-none", "family-000")
		if err == nil {
			t.Error("expected error for linking family on non-existing code")
		}
	})
}
