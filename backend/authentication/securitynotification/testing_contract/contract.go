// Package testing_contract defines shared persistence behavior for security notifications.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/securitynotification/domain"
	"github.com/ambi/idmagic/backend/authentication/securitynotification/ports"
)

type Fixture struct {
	Preferences ports.PreferenceRepository
	Devices     ports.KnownDeviceRepository
	UserID      string
	OtherUserID string
	Now         time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	t.Run("preferences", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		if got, err := f.Preferences.Find(ctx, f.UserID); err != nil || got != nil {
			t.Fatalf("missing Find = (%+v, %v)", got, err)
		}
		prefs, err := domain.NewPreferences(f.UserID, []domain.Category{domain.CategoryNewDeviceSignIn}, f.Now)
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Preferences.Save(ctx, prefs); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := f.Preferences.Find(ctx, f.UserID)
		if err != nil || got == nil || got.Allows(domain.CategoryNewDeviceSignIn) || !got.Allows(domain.CategoryCredentialChange) {
			t.Fatalf("Find = (%+v, %v), want saved disabled category", got, err)
		}
	})

	t.Run("known devices", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		device := ports.KnownDevice{UserID: f.UserID, DeviceHash: "hash-a", Label: "browser", SeenAt: f.Now}
		first, err := f.Devices.Observe(ctx, device)
		if err != nil || !first {
			t.Fatalf("first Observe = (%v, %v), want true", first, err)
		}
		device.SeenAt = f.Now.Add(time.Hour)
		again, err := f.Devices.Observe(ctx, device)
		if err != nil || again {
			t.Fatalf("second Observe = (%v, %v), want false", again, err)
		}
		other, err := f.Devices.Observe(ctx, ports.KnownDevice{UserID: f.OtherUserID, DeviceHash: "hash-a", SeenAt: f.Now})
		if err != nil || !other {
			t.Fatalf("other user's Observe = (%v, %v), want true", other, err)
		}
		deleted, err := f.Devices.DeleteIdleBefore(ctx, f.Now.Add(30*time.Minute))
		if err != nil || deleted != 1 {
			t.Fatalf("DeleteIdleBefore = (%d, %v), want one other-user row", deleted, err)
		}
	})
}
