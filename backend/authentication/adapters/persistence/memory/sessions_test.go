package memory

import (
	"context"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestSessionStore(t *testing.T) {
	ctx := context.Background()
	store := NewSessionStore()

	t.Run("Save and Find", func(t *testing.T) {
		sess := &authdomain.LoginSession{
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
		sessExpired := &authdomain.LoginSession{
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

		// Find は期限切れセッションを fail-closed に nil として扱う (行は tombstone/housekeeping
		// 対象であり、Find 自体は物理削除しない、wi-253)。
		found, err := store.Find(ctx, "sess-expired")
		if err != nil {
			t.Fatal(err)
		}
		if found != nil {
			t.Error("expected nil for expired session")
		}

		// FindOwned は期限切れでも行を返す (self-service revoke の idempotency 判定に使う)。
		owned, err := store.FindOwned(ctx, "sess-expired", "user-1")
		if err != nil {
			t.Fatal(err)
		}
		if owned == nil {
			t.Error("expected FindOwned to return the expired row (not physically deleted)")
		}
	})

	t.Run("Revoke is idempotent tombstone", func(t *testing.T) {
		sess := &authdomain.LoginSession{
			ID: "sess-revoke", TenantID: "tenant-1", UserID: "user-1",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		_ = store.Save(ctx, sess)

		now := time.Now().UTC()
		if err := store.Revoke(ctx, "sess-revoke", spec.SessionEndSelfRevoke, now); err != nil {
			t.Fatal(err)
		}
		if found, _ := store.Find(ctx, "sess-revoke"); found != nil {
			t.Error("expected revoked session to be excluded from Find")
		}
		owned, err := store.FindOwned(ctx, "sess-revoke", "user-1")
		if err != nil || owned == nil {
			t.Fatalf("expected FindOwned to return revoked row: %v %v", owned, err)
		}
		if owned.RevokedAt == nil || !owned.RevokedAt.Equal(now) {
			t.Errorf("RevokedAt = %v, want %v", owned.RevokedAt, now)
		}

		// 再失効は idempotent (最初の revoked_at/reason を保持する)。
		later := now.Add(time.Minute)
		if err := store.Revoke(ctx, "sess-revoke", spec.SessionEndAdminRevoke, later); err != nil {
			t.Fatal(err)
		}
		owned, _ = store.FindOwned(ctx, "sess-revoke", "user-1")
		if !owned.RevokedAt.Equal(now) || *owned.RevokeReason != spec.SessionEndSelfRevoke {
			t.Errorf("second revoke overwrote tombstone: %+v", owned)
		}
	})

	t.Run("Touch is coarse-grained", func(t *testing.T) {
		sess := &authdomain.LoginSession{
			ID: "sess-touch", TenantID: "tenant-1", UserID: "user-1",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		_ = store.Save(ctx, sess)

		now := time.Now().UTC()
		_ = store.Touch(ctx, "sess-touch", now)
		found, _ := store.Find(ctx, "sess-touch")
		if !found.LastSeenAt.Equal(now) {
			t.Fatalf("LastSeenAt = %v, want %v", found.LastSeenAt, now)
		}

		soon := now.Add(time.Second)
		_ = store.Touch(ctx, "sess-touch", soon)
		found, _ = store.Find(ctx, "sess-touch")
		if !found.LastSeenAt.Equal(now) {
			t.Errorf("touch within interval must not update LastSeenAt: got %v, want %v", found.LastSeenAt, now)
		}
	})

	t.Run("ListBySub", func(t *testing.T) {
		// すでに user-1 / sess-1 がある
		sessPending := &authdomain.LoginSession{
			ID:                    "sess-pending",
			UserID:                "user-1",
			AuthenticationPending: true, // Pending
			ExpiresAt:             time.Now().Add(1 * time.Hour),
		}
		sessOtherSub := &authdomain.LoginSession{
			ID:                    "sess-other",
			UserID:                "user-other",
			AuthenticationPending: false,
			ExpiresAt:             time.Now().Add(1 * time.Hour),
		}
		sessWillExpire := &authdomain.LoginSession{
			ID:                    "sess-will-expire",
			UserID:                "user-1",
			AuthenticationPending: false,
			ExpiresAt:             time.Now().Add(5 * time.Minute),
		}

		_ = store.Save(ctx, sessPending)
		_ = store.Save(ctx, sessOtherSub)
		_ = store.Save(ctx, sessWillExpire)

		// 正常に ListBySub を実行 (sess-1, sess-expired, sess-touch, sess-will-expire が有効。
		// sess-expired は実時計上はまだ期限内、sess-revoke は revoked なので除外される)。
		list, err := store.ListBySub(ctx, "user-1")
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 4 {
			t.Fatalf("expected 4 active sessions, got %d: %v", len(list), sessionIDs(list))
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
		if len(listExpired) != 2 {
			t.Errorf("expected 2 sessions after sess-will-expire expires, got %d: %v", len(listExpired), sessionIDs(listExpired))
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

	t.Run("DeleteExpiredBatch", func(t *testing.T) {
		store := NewSessionStore()
		now := time.Now()
		_ = store.Save(ctx, &authdomain.LoginSession{ID: "old-1", UserID: "u", ExpiresAt: now.Add(-time.Hour)})
		_ = store.Save(ctx, &authdomain.LoginSession{ID: "old-2", UserID: "u", ExpiresAt: now.Add(-time.Minute)})
		_ = store.Save(ctx, &authdomain.LoginSession{ID: "fresh", UserID: "u", ExpiresAt: now.Add(time.Hour)})

		deleted, err := store.DeleteExpiredBatch(ctx, now, 1)
		if err != nil {
			t.Fatal(err)
		}
		if deleted != 1 {
			t.Fatalf("expected batch limit of 1, got %d", deleted)
		}

		deleted, err = store.DeleteExpiredBatch(ctx, now, 10)
		if err != nil {
			t.Fatal(err)
		}
		if deleted != 1 {
			t.Fatalf("expected 1 remaining expired row, got %d", deleted)
		}

		if owned, _ := store.FindOwned(ctx, "fresh", "u"); owned == nil {
			t.Error("expected fresh (non-expired) session to survive cleanup")
		}
	})
}

func sessionIDs(list []*authdomain.LoginSession) []string {
	ids := make([]string, len(list))
	for i, s := range list {
		ids[i] = s.ID
	}
	return ids
}
