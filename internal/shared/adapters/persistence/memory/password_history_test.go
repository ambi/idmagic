package memory

import (
	"context"
	"testing"
	"time"
)

func TestPasswordHistoryRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewPasswordHistoryRepository()

	t.Run("Add and Recent", func(t *testing.T) {
		now := time.Now()
		// 存在しないユーザーの履歴
		recentNone, err := repo.Recent(ctx, "user-none", 5)
		if err != nil {
			t.Fatal(err)
		}
		if len(recentNone) != 0 {
			t.Errorf("expected empty history for user-none, got %d", len(recentNone))
		}

		// 無効な depth (0以下) に対する検証
		invalidRecent, err := repo.Recent(ctx, "user-1", 0)
		if err != nil {
			t.Fatal(err)
		}
		if invalidRecent != nil {
			t.Error("expected nil for non-positive depth")
		}

		// 履歴追加
		err = repo.Add(ctx, "user-1", "hash-1", now)
		if err != nil {
			t.Fatal(err)
		}
		err = repo.Add(ctx, "user-1", "hash-2", now.Add(1*time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		err = repo.Add(ctx, "user-1", "hash-3", now.Add(2*time.Minute))
		if err != nil {
			t.Fatal(err)
		}

		// 直近2件の取得 (新しい順に並んでいるはず: hash-3, hash-2)
		recent, err := repo.Recent(ctx, "user-1", 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(recent) != 2 {
			t.Fatalf("expected 2 history entries, got %d", len(recent))
		}
		if recent[0].Encoded != "hash-3" || recent[1].Encoded != "hash-2" {
			t.Errorf("unexpected recent entries order: %s, %s", recent[0].Encoded, recent[1].Encoded)
		}

		// depth が履歴数より大きい場合の取得
		recentAll, err := repo.Recent(ctx, "user-1", 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(recentAll) != 3 {
			t.Errorf("expected to cap at total history size 3, got %d", len(recentAll))
		}
	})

	t.Run("DeleteAllForSub", func(t *testing.T) {
		// user-1 の履歴を削除
		err := repo.DeleteAllForSub(ctx, "user-1")
		if err != nil {
			t.Fatal(err)
		}

		recent, err := repo.Recent(ctx, "user-1", 5)
		if err != nil {
			t.Fatal(err)
		}
		if len(recent) != 0 {
			t.Errorf("expected empty history after delete, got %d", len(recent))
		}
	})
}
