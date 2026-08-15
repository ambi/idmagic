package db_postgres

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/trusteddevice/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

const testMaxAge = 30 * 24 * time.Hour

// selector で 1 行を引き、回転した verifier を保存し直せる。
func TestTrustedDeviceRepositoryRoundTripsAndRotates(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	repo := &TrustedDeviceRepository{Pool: db}
	now := pgfixtures.TestClock()
	ctx := context.Background()

	device, cookie, err := domain.NewTrustedDevice(tenant.ID, user.ID, "Chrome / macOS", testMaxAge, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, device); err != nil {
		t.Fatal(err)
	}
	_, verifier, _ := domain.ParseCookie(cookie)

	found, err := repo.FindBySelector(ctx, tenant.ID, device.Selector)
	if err != nil || found == nil {
		t.Fatalf("FindBySelector = %#v, err %v", found, err)
	}
	if !found.VerifierMatches(verifier) {
		t.Fatal("the stored hash must match the issued verifier")
	}
	if found.Label != "Chrome / macOS" {
		t.Fatalf("Label = %q, want the masked device label", found.Label)
	}

	rotated, err := found.Rotate(now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, found); err != nil {
		t.Fatal(err)
	}
	reloaded, err := repo.FindBySelector(ctx, tenant.ID, device.Selector)
	if err != nil || reloaded == nil {
		t.Fatalf("FindBySelector after rotation = %#v, err %v", reloaded, err)
	}
	if reloaded.VerifierMatches(verifier) || !reloaded.VerifierMatches(rotated) {
		t.Fatal("rotation must invalidate the previous verifier and accept the new one")
	}
}

// 別テナントからは同じ selector でも解決できない (realm cookie の取り違え防止)。
func TestTrustedDeviceRepositoryScopesSelectorToTheTenant(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	other := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	repo := &TrustedDeviceRepository{Pool: db}
	now := pgfixtures.TestClock()
	ctx := context.Background()

	device, _, err := domain.NewTrustedDevice(tenant.ID, user.ID, "", testMaxAge, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, device); err != nil {
		t.Fatal(err)
	}

	found, err := repo.FindBySelector(ctx, other.ID, device.Selector)
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Fatal("a selector from another realm must not resolve")
	}
}

// 一括失効は未失効の行だけを対象にし、再送は 0 件を返す idempotent な操作になる。
func TestTrustedDeviceRepositoryRevokesAllOnce(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	repo := &TrustedDeviceRepository{Pool: db}
	now := pgfixtures.TestClock()
	ctx := context.Background()

	for range 2 {
		device, _, err := domain.NewTrustedDevice(tenant.ID, user.ID, "", testMaxAge, now)
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.Save(ctx, device); err != nil {
			t.Fatal(err)
		}
	}

	revoked, err := repo.RevokeAllForUser(ctx, tenant.ID, user.ID, spec.TrustedDevicePasswordChange, now.Add(time.Hour))
	if err != nil || len(revoked) != 2 {
		t.Fatalf("RevokeAllForUser = %d row(s), err %v, want 2", len(revoked), err)
	}
	again, err := repo.RevokeAllForUser(ctx, tenant.ID, user.ID, spec.TrustedDevicePasswordChange, now.Add(2*time.Hour))
	if err != nil || len(again) != 0 {
		t.Fatalf("re-revoking = %d row(s), err %v, want 0", len(again), err)
	}
	active, err := repo.ListActiveByUser(ctx, tenant.ID, user.ID)
	if err != nil || len(active) != 0 {
		t.Fatalf("ListActiveByUser = %d row(s), err %v, want 0", len(active), err)
	}
}

// FindByID は本人の行だけを返し、他人のデバイス ID の存在を試せない。
func TestTrustedDeviceRepositoryFindByIDScopesToTheOwner(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	owner := pgfixtures.SeedUser(t, db, tenant.ID)
	stranger := pgfixtures.SeedUser(t, db, tenant.ID)
	repo := &TrustedDeviceRepository{Pool: db}
	now := pgfixtures.TestClock()
	ctx := context.Background()

	device, _, err := domain.NewTrustedDevice(tenant.ID, owner.ID, "", testMaxAge, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, device); err != nil {
		t.Fatal(err)
	}

	found, err := repo.FindByID(ctx, tenant.ID, owner.ID, device.ID)
	if err != nil || found == nil {
		t.Fatalf("FindByID(owner) = %#v, err %v", found, err)
	}
	other, err := repo.FindByID(ctx, tenant.ID, stranger.ID, device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if other != nil {
		t.Fatal("another user's device must not resolve by id")
	}
}
