package db_memory

import (
	"context"
	"testing"
	"time"

	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// storeTenant は、この store が PostgreSQL 実装と同じく tenant_id で絞ることを
// テスト側でも明示する。db_postgres の同名テストと同じく、lookup も更新も
// リクエストのテナント文脈の下で行う。
const storeTenant = "tenant-1"

func tenantContext(tenantID string) context.Context {
	return tenantports.WithTenant(
		context.Background(), &tenancydomain.Tenant{ID: tenantID, Realm: tenantID}, "", "",
	)
}

func TestSessionStore(t *testing.T) {
	ctx := tenantContext(storeTenant)
	store := NewSessionStore()

	t.Run("Save and Find", func(t *testing.T) {
		sess := &sessiondomain.LoginSession{
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

		found, err := store.Find(ctx, "sess-1", time.Now().UTC())
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
		notfound, err := store.Find(ctx, "sess-none", time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil for non-existing session")
		}
	})

	t.Run("Expiration with Clock", func(t *testing.T) {
		sessExpired := &sessiondomain.LoginSession{
			ID:                    "sess-expired",
			TenantID:              "tenant-1",
			UserID:                "user-1",
			ExpiresAt:             time.Now().Add(10 * time.Minute),
			AuthenticationPending: false,
		}
		_ = store.Save(ctx, sessExpired)

		// 15 分後の時刻で判定する
		later := time.Now().Add(15 * time.Minute)

		// Find は期限切れセッションを fail-closed に nil として扱う (行は tombstone/housekeeping
		// 対象であり、Find 自体は物理削除しない、wi-253)。
		found, err := store.Find(ctx, "sess-expired", later)
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
		sess := &sessiondomain.LoginSession{
			ID: "sess-revoke", TenantID: "tenant-1", UserID: "user-1",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		_ = store.Save(ctx, sess)

		now := time.Now().UTC()
		if _, err := store.Revoke(ctx, "sess-revoke", spec.SessionEndSelfRevoke, now); err != nil {
			t.Fatal(err)
		}
		if found, _ := store.Find(ctx, "sess-revoke", time.Now().UTC()); found != nil {
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
		if _, err := store.Revoke(ctx, "sess-revoke", spec.SessionEndAdminRevoke, later); err != nil {
			t.Fatal(err)
		}
		owned, _ = store.FindOwned(ctx, "sess-revoke", "user-1")
		if !owned.RevokedAt.Equal(now) || *owned.RevokeReason != spec.SessionEndSelfRevoke {
			t.Errorf("second revoke overwrote tombstone: %+v", owned)
		}
	})

	t.Run("Touch is coarse-grained", func(t *testing.T) {
		sess := &sessiondomain.LoginSession{
			ID: "sess-touch", TenantID: "tenant-1", UserID: "user-1",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		_ = store.Save(ctx, sess)

		now := time.Now().UTC()
		_ = store.Touch(ctx, "sess-touch", now)
		found, _ := store.Find(ctx, "sess-touch", time.Now().UTC())
		if !found.LastSeenAt.Equal(now) {
			t.Fatalf("LastSeenAt = %v, want %v", found.LastSeenAt, now)
		}

		soon := now.Add(time.Second)
		_ = store.Touch(ctx, "sess-touch", soon)
		found, _ = store.Find(ctx, "sess-touch", time.Now().UTC())
		if !found.LastSeenAt.Equal(now) {
			t.Errorf("touch within interval must not update LastSeenAt: got %v, want %v", found.LastSeenAt, now)
		}
	})

	t.Run("ListBySub", func(t *testing.T) {
		// すでに user-1 / sess-1 がある
		sessPending := &sessiondomain.LoginSession{
			ID:                    "sess-pending",
			TenantID:              storeTenant,
			UserID:                "user-1",
			AuthenticationPending: true, // Pending
			ExpiresAt:             time.Now().Add(1 * time.Hour),
		}
		sessOtherSub := &sessiondomain.LoginSession{
			ID:                    "sess-other",
			TenantID:              storeTenant,
			UserID:                "user-other",
			AuthenticationPending: false,
			ExpiresAt:             time.Now().Add(1 * time.Hour),
		}
		sessWillExpire := &sessiondomain.LoginSession{
			ID:                    "sess-will-expire",
			TenantID:              storeTenant,
			UserID:                "user-1",
			AuthenticationPending: false,
			ExpiresAt:             time.Now().Add(5 * time.Minute),
		}

		_ = store.Save(ctx, sessPending)
		_ = store.Save(ctx, sessOtherSub)
		_ = store.Save(ctx, sessWillExpire)

		// 正常に ListBySub を実行 (sess-1, sess-expired, sess-touch, sess-will-expire が有効。
		// sess-expired は実時計上はまだ期限内、sess-revoke は revoked なので除外される)。
		list, err := store.ListBySub(ctx, "user-1", time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 4 {
			t.Fatalf("expected 4 active sessions, got %d: %v", len(list), sessionIDs(list))
		}

		// 10 分後の時刻で判定し、sess-will-expire を失効させる
		listExpired, err := store.ListBySub(ctx, "user-1", time.Now().Add(10*time.Minute))
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

		found, _ := store.Find(ctx, "sess-other", time.Now().UTC())
		if found != nil {
			t.Error("expected sess-other to be deleted")
		}
	})

	t.Run("DeleteExpiredBatch", func(t *testing.T) {
		store := NewSessionStore()
		now := time.Now()
		_ = store.Save(ctx, &sessiondomain.LoginSession{ID: "old-1", TenantID: storeTenant, UserID: "u", ExpiresAt: now.Add(-time.Hour)})
		_ = store.Save(ctx, &sessiondomain.LoginSession{ID: "old-2", TenantID: storeTenant, UserID: "u", ExpiresAt: now.Add(-time.Minute)})
		_ = store.Save(ctx, &sessiondomain.LoginSession{ID: "fresh", TenantID: storeTenant, UserID: "u", ExpiresAt: now.Add(time.Hour)})

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

// 別テナントの行は、id と所有者が分かっていても参照も更新もできない。
// PostgreSQL 実装は同じ分離を tenant_id 述語で持つ。ここに同じ検査が無いと、
// メモリ構成でだけテナント境界が消えていることを誰も検出できない。
func TestSessionStoreIsolatesTenants(t *testing.T) {
	store := NewSessionStore()
	owner := tenantContext("tenant-owner")
	intruder := tenantContext("tenant-intruder")
	if err := store.Save(owner, &sessiondomain.LoginSession{
		ID: "sess-owned", TenantID: "tenant-owner", UserID: "user-1",
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	if found, _ := store.Find(intruder, "sess-owned", time.Now().UTC()); found != nil {
		t.Error("別テナントの Find が行を返した")
	}
	if owned, _ := store.FindOwned(intruder, "sess-owned", "user-1"); owned != nil {
		t.Error("別テナントの FindOwned が行を返した")
	}
	if list, _ := store.ListBySub(intruder, "user-1", time.Now().UTC()); len(list) != 0 {
		t.Errorf("別テナントの ListBySub が %d 件返した", len(list))
	}

	if _, err := store.Revoke(intruder, "sess-owned", spec.SessionEndAdminRevoke, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteAllForSub(intruder, "user-1"); err != nil {
		t.Fatal(err)
	}
	owned, err := store.FindOwned(owner, "sess-owned", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if owned == nil {
		t.Fatal("別テナントの DeleteAllForSub が行を消した")
	}
	if owned.RevokedAt != nil {
		t.Error("別テナントの Revoke が tombstone を書いた")
	}
}

func sessionIDs(list []*sessiondomain.LoginSession) []string {
	ids := make([]string, len(list))
	for i, s := range list {
		ids[i] = s.ID
	}
	return ids
}
