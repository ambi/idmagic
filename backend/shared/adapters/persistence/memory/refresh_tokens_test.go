package memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestRefreshTokenStore(t *testing.T) {
	ctx := context.Background()
	store := NewRefreshTokenStore()

	t.Run("Save and FindByHash", func(t *testing.T) {
		rec := &spec.RefreshTokenRecord{
			ID:                "token-1",
			TenantID:          "tenant-1",
			Hash:              "hash-1",
			FamilyID:          "family-1",
			ClientID:          "client-1",
			UserID:            "user-1",
			Scopes:            []string{"openid"},
			IssuedAt:          time.Now(),
			ExpiresAt:         time.Now().Add(24 * time.Hour),
			AbsoluteExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			Revoked:           false,
			Rotated:           false,
		}

		err := store.Save(ctx, rec)
		if err != nil {
			t.Fatal(err)
		}

		found, err := store.FindByHash(ctx, "hash-1")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected refresh token to be found")
		}
		if found.ID != "token-1" {
			t.Errorf("expected ID to be 'token-1', got %q", found.ID)
		}
		if len(found.Scopes) != 1 || found.Scopes[0] != "openid" {
			t.Errorf("unexpected scopes: %v", found.Scopes)
		}

		// 存在しないハッシュ
		notfound, err := store.FindByHash(ctx, "hash-none")
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil for non-existing token hash")
		}

		// SenderConstraint ありのトークン保存・検索
		recWithConstraint := &spec.RefreshTokenRecord{
			ID:   "token-constraint",
			Hash: "hash-constraint",
			SenderConstraint: &spec.SenderConstraint{
				Type: spec.SenderConstraintDPoP,
				JKT:  "jkt-val",
			},
		}
		_ = store.Save(ctx, recWithConstraint)
		foundConstraint, _ := store.FindByHash(ctx, "hash-constraint")
		if foundConstraint.SenderConstraint == nil || foundConstraint.SenderConstraint.JKT != "jkt-val" {
			t.Errorf("expected SenderConstraint.JKT to be 'jkt-val', got %v", foundConstraint.SenderConstraint)
		}
	})

	t.Run("Rotate", func(t *testing.T) {
		newRec := &spec.RefreshTokenRecord{
			ID:       "token-2",
			Hash:     "hash-2",
			FamilyID: "family-1",
			ClientID: "client-1",
			UserID:   "user-1",
		}

		// 正常ケース
		rotated, err := store.Rotate(ctx, "token-1", newRec)
		if err != nil {
			t.Fatal(err)
		}
		if rotated == nil || rotated.ID != "token-2" {
			t.Errorf("expected rotated token token-2, got %v", rotated)
		}

		// 親トークンが Rotated=true になっていることの確認
		parent, _ := store.FindByHash(ctx, "hash-1")
		if !parent.Rotated {
			t.Error("expected parent token to be rotated")
		}

		// すでに Rotated=true のトークンを親にして rotate しようとした場合 (nil, nil を返すはず)
		againRec := &spec.RefreshTokenRecord{ID: "token-3", Hash: "hash-3"}
		again, err := store.Rotate(ctx, "token-1", againRec)
		if err != nil {
			t.Fatal(err)
		}
		if again != nil {
			t.Error("expected nil when rotating already rotated token")
		}

		// 存在しないトークンを親にして rotate (エラーを返すはず)
		_, err = store.Rotate(ctx, "token-none", againRec)
		if err == nil {
			t.Error("expected error when rotating non-existing parent token")
		}
	})

	t.Run("RevokeFamily", func(t *testing.T) {
		err := store.RevokeFamily(ctx, "family-1")
		if err != nil {
			t.Fatal(err)
		}

		t1, _ := store.FindByHash(ctx, "hash-1")
		t2, _ := store.FindByHash(ctx, "hash-2")
		if !t1.Revoked || !t2.Revoked {
			t.Error("expected all tokens in family-1 to be revoked")
		}
	})

	t.Run("DeleteAllForSub", func(t *testing.T) {
		err := store.DeleteAllForSub(ctx, "user-1")
		if err != nil {
			t.Fatal(err)
		}

		t1, _ := store.FindByHash(ctx, "hash-1")
		t2, _ := store.FindByHash(ctx, "hash-2")
		if t1 != nil || t2 != nil {
			t.Error("expected all tokens for user-1 to be deleted")
		}
	})
}
