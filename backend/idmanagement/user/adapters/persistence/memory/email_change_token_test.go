package memory

import (
	"context"
	"testing"
	"time"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

func TestEmailChangeTokenStore(t *testing.T) {
	ctx := context.Background()
	store := NewEmailChangeTokenStore()

	t.Run("Save and Consume", func(t *testing.T) {
		record := userports.EmailChangeTokenRecord{
			TokenHash: "hash-1",
			Sub:       "user-1",
			NewEmail:  "new@example.com",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		err := store.Save(ctx, record)
		if err != nil {
			t.Fatal(err)
		}

		// 正常ケース
		consumed, err := store.Consume(ctx, "hash-1", time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if consumed == nil {
			t.Fatal("expected to consume token")
		}
		if consumed.NewEmail != "new@example.com" {
			t.Errorf("expected email to be new@example.com, got %q", consumed.NewEmail)
		}

		// 一度 consume すると削除されるため二度目は nil
		again, err := store.Consume(ctx, "hash-1", time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if again != nil {
			t.Error("expected token to be consumed already")
		}
	})

	t.Run("Overwrite on Save and Expired Token", func(t *testing.T) {
		record1 := userports.EmailChangeTokenRecord{
			TokenHash: "hash-old",
			Sub:       "user-2",
			NewEmail:  "old@example.com",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		record2 := userports.EmailChangeTokenRecord{
			TokenHash: "hash-new",
			Sub:       "user-2", // 同じ sub
			NewEmail:  "new@example.com",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		_ = store.Save(ctx, record1)
		_ = store.Save(ctx, record2) // 上書き（古いものは消えるはず）

		// 古いトークンは消えているはず
		oldToken, _ := store.Consume(ctx, "hash-old", time.Now())
		if oldToken != nil {
			t.Error("expected old token to be deleted during overwrite")
		}

		// 期限切れトークンを検証
		expiredRecord := userports.EmailChangeTokenRecord{
			TokenHash: "hash-expired",
			Sub:       "user-3",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		_ = store.Save(ctx, expiredRecord)

		// 現在時刻で consume しようとすると、期限切れなので nil を返し、かつ削除される
		expired, err := store.Consume(ctx, "hash-expired", time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if expired != nil {
			t.Error("expected expired token to return nil")
		}

		// 実際に削除されているか確認
		store.mu.Lock()
		_, ok := store.records["hash-expired"]
		store.mu.Unlock()
		if ok {
			t.Error("expected expired token to be deleted from store")
		}
	})
}
