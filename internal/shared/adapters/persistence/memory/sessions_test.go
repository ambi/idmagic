package memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/internal/shared/spec"
)

func TestSessionStore(t *testing.T) {
	ctx := context.Background()
	store := NewSessionStore()

	t.Run("Save and Find", func(t *testing.T) {
		sess := &spec.LoginSession{
			ID:                    "sess-1",
			TenantID:              "tenant-1",
			UserID:                "user-1",
			AuthTime:              time.Now().Unix(),
			AMR:                   []string{"pwd"},
			ACR:                   "acr-1",
			AuthenticationPending: false,
			ExpiresAt:             time.Now().Add(1 * time.Hour),
		}

		err := store.Save(ctx, sess)
		if err != nil {
			t.Fatal(err)
		}

		found, err := store.Find(ctx, "sess-1")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected session to be found")
		}
		if found.ID != "sess-1" || found.UserID != "user-1" {
			t.Errorf("unexpected found session: %v", found)
		}

		// 存在しないセッション
		notfound, err := store.Find(ctx, "sess-none")
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil for non-existing session")
		}
	})

	t.Run("Expiration with Clock", func(t *testing.T) {
		sessExpired := &spec.LoginSession{
			ID:                    "sess-expired",
			TenantID:              "tenant-1",
			UserID:                "user-1",
			ExpiresAt:             time.Now().Add(10 * time.Minute),
			AuthenticationPending: false,
		}
		_ = store.Save(ctx, sessExpired)

		// 15分後に時計を進める設定にする
		store.Clock = func() time.Time {
			return time.Now().Add(15 * time.Minute)
		}
		defer func() { store.Clock = nil }() // テスト後に戻す

		// Find すると期限切れのため自動削除され nil が返るはず
		found, err := store.Find(ctx, "sess-expired")
		if err != nil {
			t.Fatal(err)
		}
		if found != nil {
			t.Error("expected nil for expired session")
		}

		// 実際に削除されているか確認
		store.Clock = nil
		foundAgain, _ := store.Find(ctx, "sess-expired")
		if foundAgain != nil {
			t.Error("expected session to be deleted from store")
		}
	})

	t.Run("ListBySub", func(t *testing.T) {
		// すでに user-1 / sess-1 がある
		sessPending := &spec.LoginSession{
			ID:                    "sess-pending",
			UserID:                "user-1",
			AuthenticationPending: true, // Pending
			ExpiresAt:             time.Now().Add(1 * time.Hour),
		}
		sessOtherSub := &spec.LoginSession{
			ID:                    "sess-other",
			UserID:                "user-other",
			AuthenticationPending: false,
			ExpiresAt:             time.Now().Add(1 * time.Hour),
		}
		sessWillExpire := &spec.LoginSession{
			ID:                    "sess-will-expire",
			UserID:                "user-1",
			AuthenticationPending: false,
			ExpiresAt:             time.Now().Add(5 * time.Minute),
		}

		_ = store.Save(ctx, sessPending)
		_ = store.Save(ctx, sessOtherSub)
		_ = store.Save(ctx, sessWillExpire)

		// 正常に ListBySub を実行
		list, err := store.ListBySub(ctx, "user-1")
		if err != nil {
			t.Fatal(err)
		}
		// user-1 に対して、sess-1, sess-will-expire の 2 つが返るはず (sess-pending は Pending なので除外される)
		if len(list) != 2 {
			t.Fatalf("expected 2 active sessions, got %d", len(list))
		}

		// 時計を進めて sess-will-expire を失効させる
		store.Clock = func() time.Time {
			return time.Now().Add(10 * time.Minute)
		}
		defer func() { store.Clock = nil }()

		listExpired, err := store.ListBySub(ctx, "user-1")
		if err != nil {
			t.Fatal(err)
		}
		// sess-will-expire は期限切れで除外かつ削除されるため、sess-1 の 1 つだけが返る
		if len(listExpired) != 1 || listExpired[0].ID != "sess-1" {
			t.Errorf("expected only sess-1 in list, got %v", listExpired)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := store.Delete(ctx, "sess-1")
		if err != nil {
			t.Fatal(err)
		}

		found, _ := store.Find(ctx, "sess-1")
		if found != nil {
			t.Error("expected sess-1 to be deleted")
		}
	})

	t.Run("DeleteAllForSub", func(t *testing.T) {
		err := store.DeleteAllForSub(ctx, "user-other")
		if err != nil {
			t.Fatal(err)
		}

		found, _ := store.Find(ctx, "sess-other")
		if found != nil {
			t.Error("expected sess-other to be deleted")
		}
	})
}
