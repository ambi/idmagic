// Package testing_contract defines the shared trusted-device repository contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/security/entropy"

	"github.com/ambi/idmagic/backend/authentication/trusteddevice/domain"
	"github.com/ambi/idmagic/backend/authentication/trusteddevice/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type Fixture struct {
	Repository ports.TrustedDeviceRepository
	TenantID   string
	UserID     string
	Now        time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	device, cookie, err := domain.NewTrustedDevice(f.TenantID, f.UserID, "Chrome", 24*time.Hour, f.Now, entropy.Crypto())
	if err != nil {
		t.Fatal(err)
	}
	selector, verifier, ok := domain.ParseCookie(cookie)
	if !ok || selector != device.Selector {
		t.Fatalf("ParseCookie(%q): selector=%q ok=%v", cookie, selector, ok)
	}
	if err := f.Repository.Save(ctx, device); err != nil {
		t.Fatalf("Save: %v", err)
	}
	bySelector, err := f.Repository.FindBySelector(ctx, f.TenantID, device.Selector)
	if err != nil || bySelector == nil || !bySelector.VerifierMatches(verifier) {
		t.Fatalf("FindBySelector = (%+v, %v)", bySelector, err)
	}
	byID, err := f.Repository.FindByID(ctx, f.TenantID, f.UserID, device.ID)
	if err != nil || byID == nil || byID.ID != device.ID {
		t.Fatalf("FindByID = (%+v, %v)", byID, err)
	}
	list, err := f.Repository.ListActiveByUser(ctx, f.TenantID, f.UserID, f.Now)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListActiveByUser = (%+v, %v)", list, err)
	}
	revoked, err := f.Repository.RevokeAllForUser(ctx, f.TenantID, f.UserID, spec.TrustedDeviceAdminRevoke, f.Now.Add(time.Minute))
	if err != nil || len(revoked) != 1 {
		t.Fatalf("RevokeAllForUser: %v", err)
	}
	if list, err := f.Repository.ListActiveByUser(ctx, f.TenantID, f.UserID, f.Now.Add(time.Minute)); err != nil || len(list) != 0 {
		t.Fatalf("active devices after revoke = (%+v, %v)", list, err)
	}
	if err := f.Repository.DeleteAllForSub(ctx, f.UserID); err != nil {
		t.Fatalf("DeleteAllForSub: %v", err)
	}
}
