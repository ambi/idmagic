package usecases_test

import (
	"context"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"

	authnmemory "github.com/ambi/idmagic/backend/authentication/internal/db_memory"
	securitynotificationmemory "github.com/ambi/idmagic/backend/authentication/internal/securitynotification/db_memory"
	securitynotificationports "github.com/ambi/idmagic/backend/authentication/internal/securitynotification/ports"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/internal/session/db_memory"
	"github.com/ambi/idmagic/backend/authentication/internal/usecases"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
)

func daysAgo(now time.Time, d int) time.Time {
	return now.Add(-time.Duration(d) * 24 * time.Hour)
}

func TestRetentionSweepDeletesOldBuckets(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	store := authnmemory.NewAuthEventBucketStore()
	// 91 日前の窓と直近の窓を作る。
	if _, err := store.Record(ctx, "failed_login", tenancydomain.DefaultTenantID, "old-key", daysAgo(now, 91)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Record(ctx, "failed_login", tenancydomain.DefaultTenantID, "fresh-key", now); err != nil {
		t.Fatal(err)
	}
	res, err := usecases.RunRetentionSweep(ctx, usecases.RetentionStores{Buckets: store}, usecases.DefaultRetentionPolicy(), now)
	if err != nil {
		t.Fatal(err)
	}
	if res.Buckets != 1 {
		t.Fatalf("deleted buckets=%d, want 1", res.Buckets)
	}
	buckets, err := store.List(ctx, tenancydomain.DefaultTenantID, time.Time{}, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) != 1 || buckets[0].KeyHash != "fresh-key" {
		t.Fatalf("remaining buckets=%#v, want only fresh-key", buckets)
	}
}

// TestRetentionSweepDeletesExpiredSessions: wi-253 Plan §7 の housekeeping cleanup を
// retention sweep に統合する。SessionDays (既定 90 日) を LoginSession の
// tombstone/期限切れ行の物理削除 cutoff に転用する。
func TestRetentionSweepDeletesExpiredSessions(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	store := sessionmemory.NewSessionStore()

	mustSave := func(id string, expiresAt time.Time) {
		t.Helper()
		if err := store.Save(ctx, &sessiondomain.LoginSession{
			ID: id, UserID: "user-1", AMR: []string{"pwd"}, ACR: "urn:idmagic:acr:pwd",
			ExpiresAt: expiresAt,
		}); err != nil {
			t.Fatal(err)
		}
	}
	// 90 日 cutoff: 89 日前に失効した行は残り、91 日前に失効した行は消える。
	mustSave("expired-89", daysAgo(now, 89))
	mustSave("expired-91", daysAgo(now, 91))
	mustSave("active", now.Add(time.Hour))

	res, err := usecases.RunRetentionSweep(ctx, usecases.RetentionStores{Sessions: store}, usecases.DefaultRetentionPolicy(), now)
	if err != nil {
		t.Fatal(err)
	}
	if res.Sessions != 1 {
		t.Fatalf("deleted sessions=%d, want 1", res.Sessions)
	}
	if owned, _ := store.FindOwned(ctx, "expired-91", "user-1"); owned != nil {
		t.Error("expired-91 should be purged past the 90-day cutoff")
	}
	if owned, _ := store.FindOwned(ctx, "expired-89", "user-1"); owned == nil {
		t.Error("expired-89 should survive within the 90-day cutoff")
	}
	if owned, _ := store.FindOwned(ctx, "active", "user-1"); owned == nil {
		t.Error("active session should survive")
	}
}

// TestRetentionSweepDeletesIdleKnownSignInDevices: 既知のサインイン端末はサインイン履歴と
// 同じ 365 日で掃除する (wi-90)。履歴から消えた端末を「既知」と呼び続けないためである。
func TestRetentionSweepDeletesIdleKnownSignInDevices(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	store := securitynotificationmemory.NewKnownDeviceRepository()

	observe := func(hash string, seenAt time.Time) {
		t.Helper()
		if _, err := store.Observe(ctx, securitynotificationports.KnownDevice{
			UserID: "user-1", DeviceHash: hash, SeenAt: seenAt,
		}); err != nil {
			t.Fatal(err)
		}
	}
	observe("recent", daysAgo(now, 364))
	observe("idle", daysAgo(now, 366))

	res, err := usecases.RunRetentionSweep(ctx, usecases.RetentionStores{KnownDevices: store}, usecases.DefaultRetentionPolicy(), now)
	if err != nil {
		t.Fatal(err)
	}
	if res.KnownDevices != 1 {
		t.Fatalf("deleted known devices=%d, want 1", res.KnownDevices)
	}
	// 掃除された端末は次のサインインで再び「新しい端末」になり、残った端末はならない。
	swept, err := store.Observe(ctx, securitynotificationports.KnownDevice{UserID: "user-1", DeviceHash: "idle", SeenAt: now})
	if err != nil || !swept {
		t.Errorf("the swept device = %v, err %v; want it to count as new again", swept, err)
	}
	kept, err := store.Observe(ctx, securitynotificationports.KnownDevice{UserID: "user-1", DeviceHash: "recent", SeenAt: now})
	if err != nil || kept {
		t.Errorf("the device within the window = %v, err %v; want it to stay known", kept, err)
	}
}
