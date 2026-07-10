package memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestMfaFactorRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewMfaFactorRepository()

	t.Run("Save and Find", func(t *testing.T) {
		secret := "JBSWY3DPEHPK3PXP"
		label := "My Phone TOTP"
		now := time.Now()
		factor := &spec.MfaFactor{
			UserID:     "user-1",
			Type:       spec.MfaFactorTOTP,
			Secret:     &secret,
			Label:      &label,
			CreatedAt:  now,
			LastUsedAt: &now,
		}

		err := repo.Save(ctx, factor)
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.Find(ctx, "user-1", spec.MfaFactorTOTP)
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected MFA factor to be found")
		}
		if found.UserID != "user-1" || found.Type != spec.MfaFactorTOTP {
			t.Errorf("unexpected found user/type: %s / %s", found.UserID, found.Type)
		}
		if found.Secret == nil || *found.Secret != "JBSWY3DPEHPK3PXP" {
			t.Errorf("unexpected secret: %v", found.Secret)
		}
		if found.Label == nil || *found.Label != "My Phone TOTP" {
			t.Errorf("unexpected label: %v", found.Label)
		}

		// 存在しない MFA factor
		notfound, err := repo.Find(ctx, "user-1", spec.MfaFactorWebAuthn)
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil for non-existing factor type")
		}
	})

	t.Run("ListBySub", func(t *testing.T) {
		// すでに user-1 / totp がある。さらに WebAuthn も追加する。
		factorWeb := &spec.MfaFactor{
			UserID:    "user-1",
			Type:      spec.MfaFactorWebAuthn,
			CreatedAt: time.Now(),
		}
		factorOther := &spec.MfaFactor{
			UserID:    "user-other",
			Type:      spec.MfaFactorTOTP,
			CreatedAt: time.Now(),
		}

		_ = repo.Save(ctx, factorWeb)
		_ = repo.Save(ctx, factorOther)

		list, err := repo.ListBySub(ctx, "user-1")
		if err != nil {
			t.Fatal(err)
		}
		// user-1 には 2 件あるはず
		if len(list) != 2 {
			t.Fatalf("expected 2 factors, got %d", len(list))
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, "user-1", spec.MfaFactorTOTP)
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.Find(ctx, "user-1", spec.MfaFactorTOTP)
		if err != nil {
			t.Fatal(err)
		}
		if found != nil {
			t.Error("expected TOTP factor to be deleted")
		}
	})

	t.Run("DeleteAllForSub", func(t *testing.T) {
		// すでに user-1 には WebAuthn がある。
		err := repo.DeleteAllForSub(ctx, "user-1")
		if err != nil {
			t.Fatal(err)
		}

		list, err := repo.ListBySub(ctx, "user-1")
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 0 {
			t.Errorf("expected all factors for user-1 to be deleted, got %d", len(list))
		}

		// user-other の factor は残っているはず
		listOther, _ := repo.ListBySub(ctx, "user-other")
		if len(listOther) != 1 {
			t.Error("expected user-other factor to remain")
		}
	})
}
