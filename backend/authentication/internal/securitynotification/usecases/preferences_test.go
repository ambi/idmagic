package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/authentication/internal/securitynotification/db_memory"
	securitynotificationdomain "github.com/ambi/idmagic/backend/authentication/internal/securitynotification/domain"
)

//spec:covers REQ-AUTHENTICATION-033: 全種別が返り、必須の種別は mandatory かつ常に有効である。
func TestGetPreferencesReturnsEveryCategoryWithItsMandatoryFlag(t *testing.T) {
	t.Parallel()
	deps := PreferenceDeps{Repo: db_memory.NewPreferenceRepository()}

	got, err := GetPreferences(context.Background(), deps, testSub)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(securitynotificationdomain.Categories()) {
		t.Fatalf("returned %d categories, want the whole catalog", len(got))
	}
	for i, preference := range got {
		if preference.Category != securitynotificationdomain.Categories()[i] {
			t.Errorf("position %d = %s, want the catalog order", i, preference.Category)
		}
		if !preference.Enabled {
			t.Errorf("%s is disabled before any change was made", preference.Category)
		}
		if preference.Mandatory != preference.Category.Mandatory() {
			t.Errorf("%s: Mandatory = %v", preference.Category, preference.Mandatory)
		}
	}
}

//spec:covers REQ-AUTHENTICATION-033: 必須の種別を含む更新は保存の前に丸ごと拒否する。
func TestUpdatePreferencesRejectsMandatoryCategoriesWithoutPartialApplication(t *testing.T) {
	t.Parallel()
	repo := db_memory.NewPreferenceRepository()
	deps := PreferenceDeps{Repo: repo}
	ctx := context.Background()

	_, err := UpdatePreferences(ctx, deps, testSub,
		[]securitynotificationdomain.Category{securitynotificationdomain.CategoryNewDeviceSignIn, securitynotificationdomain.CategoryCredentialChange}, testNow())
	if !errors.Is(err, securitynotificationdomain.ErrMandatoryCategory) {
		t.Fatalf("err = %v, want ErrMandatoryCategory", err)
	}
	stored, err := repo.Find(ctx, testSub)
	if err != nil {
		t.Fatal(err)
	}
	if stored != nil {
		t.Fatalf("a rejected update stored %#v; the allowed half must not be applied", stored)
	}
}

//spec:covers REQ-AUTHENTICATION-034: 更新は無効化の集合を置き換え、結果をそのまま読み戻せる。
func TestUpdatePreferencesReplacesTheDisabledSet(t *testing.T) {
	t.Parallel()
	deps := PreferenceDeps{Repo: db_memory.NewPreferenceRepository()}
	ctx := context.Background()

	got, err := UpdatePreferences(ctx, deps, testSub,
		[]securitynotificationdomain.Category{securitynotificationdomain.CategoryNewDeviceSignIn, securitynotificationdomain.CategorySessionRevoked}, testNow())
	if err != nil {
		t.Fatal(err)
	}
	enabled := map[securitynotificationdomain.Category]bool{}
	for _, preference := range got {
		enabled[preference.Category] = preference.Enabled
	}
	if enabled[securitynotificationdomain.CategoryNewDeviceSignIn] || enabled[securitynotificationdomain.CategorySessionRevoked] {
		t.Error("the categories just disabled are still reported as enabled")
	}
	if !enabled[securitynotificationdomain.CategoryMfaChange] {
		t.Error("a mandatory category must stay enabled")
	}

	got, err = UpdatePreferences(ctx, deps, testSub, nil, testNow())
	if err != nil {
		t.Fatal(err)
	}
	for _, preference := range got {
		if !preference.Enabled {
			t.Errorf("%s is still disabled after clearing the set", preference.Category)
		}
	}
}

func TestUpdatePreferencesFailsLoudlyWithoutAStore(t *testing.T) {
	t.Parallel()

	_, err := UpdatePreferences(context.Background(), PreferenceDeps{}, testSub, nil, testNow())
	if !errors.Is(err, ErrPreferencesUnavailable) {
		t.Fatalf("err = %v, want ErrPreferencesUnavailable rather than a silent success", err)
	}
}
