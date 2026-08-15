package db_postgres

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/securitynotification/domain"
	"github.com/ambi/idmagic/backend/authentication/securitynotification/ports"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

// REQ-AUTHENTICATION-034: 保存した設定が読み戻せ、行が無い場合は「すべて有効」になる。
func TestPreferenceRepositoryRoundTripsAndTreatsAbsenceAsAllEnabled(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	repo := &PreferenceRepository{Pool: db}
	ctx := context.Background()

	found, err := repo.Find(ctx, user.ID)
	if err != nil || found != nil {
		t.Fatalf("Find before any change = %#v, err %v; want a nil row and no error", found, err)
	}

	now := pgfixtures.TestClock()
	prefs, err := domain.NewPreferences(user.ID, []domain.Category{domain.CategoryNewDeviceSignIn}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, prefs); err != nil {
		t.Fatal(err)
	}
	found, err = repo.Find(ctx, user.ID)
	if err != nil || found == nil {
		t.Fatalf("Find = %#v, err %v", found, err)
	}
	if !slices.Equal(found.Disabled, []domain.Category{domain.CategoryNewDeviceSignIn}) {
		t.Fatalf("Disabled = %v, want the saved category", found.Disabled)
	}
	if found.Allows(domain.CategoryNewDeviceSignIn) || !found.Allows(domain.CategoryCredentialChange) {
		t.Fatal("only the saved category may be suppressed")
	}

	// 再保存は行を置き換える (追記ではない)。
	cleared, err := domain.NewPreferences(user.ID, nil, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, cleared); err != nil {
		t.Fatal(err)
	}
	found, err = repo.Find(ctx, user.ID)
	if err != nil || found == nil {
		t.Fatalf("Find after clearing = %#v, err %v", found, err)
	}
	if len(found.Disabled) != 0 {
		t.Fatalf("Disabled after clearing = %v, want empty", found.Disabled)
	}
}

// REQ-AUTHENTICATION-030: 端末は最初の観測だけが「新しい端末」で、以後は既知になる。
func TestKnownDeviceRepositoryReportsOnlyTheFirstObservation(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	alice := pgfixtures.SeedUser(t, db, tenant.ID)
	bob := pgfixtures.SeedUser(t, db, tenant.ID)
	repo := &KnownDeviceRepository{Pool: db}
	ctx := context.Background()
	now := pgfixtures.TestClock()

	device := ports.KnownDevice{
		UserID: alice.ID, DeviceHash: "hash-a", Label: "Chrome / macOS", SeenAt: now,
	}
	first, err := repo.Observe(ctx, device)
	if err != nil || !first {
		t.Fatalf("the first observation = %v, err %v; want true", first, err)
	}

	device.SeenAt = now.Add(time.Hour)
	again, err := repo.Observe(ctx, device)
	if err != nil || again {
		t.Fatalf("the second observation of the same device = %v, err %v; want false", again, err)
	}

	// 同一時刻の再観測でも「新しい端末」にはならない。
	device.SeenAt = now
	same, err := repo.Observe(ctx, device)
	if err != nil || same {
		t.Fatalf("re-observing at the same instant = %v, err %v; want false", same, err)
	}

	// 別のユーザーの同じ端末は、そのユーザーにとっては新しい。
	other, err := repo.Observe(ctx, ports.KnownDevice{
		UserID: bob.ID, DeviceHash: "hash-a", SeenAt: now,
	})
	if err != nil || !other {
		t.Fatalf("another user's first observation = %v, err %v; want true", other, err)
	}
}

// 掃除は最終利用が cutoff より前の行だけを消し、新しい行を残す。
func TestKnownDeviceRepositoryDeletesOnlyIdleRows(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	repo := &KnownDeviceRepository{Pool: db}
	ctx := context.Background()
	now := pgfixtures.TestClock()

	if _, err := repo.Observe(ctx, ports.KnownDevice{
		UserID: user.ID, DeviceHash: "idle", SeenAt: now.Add(-400 * 24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Observe(ctx, ports.KnownDevice{
		UserID: user.ID, DeviceHash: "recent", SeenAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	deleted, err := repo.DeleteIdleBefore(ctx, now.Add(-365*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("DeleteIdleBefore deleted %d row(s), want 1", deleted)
	}
	stillKnown, err := repo.Observe(ctx, ports.KnownDevice{
		UserID: user.ID, DeviceHash: "recent", SeenAt: now,
	})
	if err != nil || stillKnown {
		t.Fatalf("the recent device = %v, err %v; want it to stay known", stillKnown, err)
	}
	returned, err := repo.Observe(ctx, ports.KnownDevice{
		UserID: user.ID, DeviceHash: "idle", SeenAt: now,
	})
	if err != nil || !returned {
		t.Fatalf("the swept device = %v, err %v; want it to count as new again", returned, err)
	}
}
