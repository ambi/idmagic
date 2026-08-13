package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestEnsureDefaultAndRejectDefaultDisable(t *testing.T) {
	repo := memory.NewTenantRepository()
	now := time.Now().UTC()
	if err := EnsureDefault(context.Background(), repo, now); err != nil {
		t.Fatal(err)
	}
	tenant, err := repo.FindByID(context.Background(), domain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	if tenant == nil || tenant.Status != domain.TenantStatusActive {
		t.Fatalf("default tenant = %#v", tenant)
	}
	if _, err := SetDisabled(
		context.Background(), repo, domain.DefaultTenantID, true, now,
	); !errors.Is(err, ErrDefaultTenant) {
		t.Fatalf("disable default error = %v", err)
	}
}

func TestTenantLifecycle(t *testing.T) {
	repo := memory.NewTenantRepository()
	tenant, err := Create(context.Background(), repo, "acme", "Acme", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if tenant.Status != domain.TenantStatusActive {
		t.Fatalf("status = %s", tenant.Status)
	}
	tenant, err = SetDisabled(context.Background(), repo, tenant.ID, true, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if tenant.Status != domain.TenantStatusDisabled || tenant.DisabledAt == nil {
		t.Fatalf("disabled tenant = %#v", tenant)
	}
	tenant, err = SetDisabled(context.Background(), repo, tenant.ID, false, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if tenant.Status != domain.TenantStatusActive || tenant.DisabledAt != nil {
		t.Fatalf("enabled tenant = %#v", tenant)
	}
}

// wi-285 / scenario Tenancy.tenant_endpoint_style: endpoint style は破壊的な
// 専用操作でのみ切り替え、subdomain は base domain を持つ配備でしか選べない。
func TestSetEndpointStyle(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewTenantRepository()
	tenant, err := Create(ctx, repo, "acme", "Acme", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SetEndpointStyle(ctx, repo, tenant.ID, domain.TenantEndpointStyleSubdomain, "", time.Now().UTC()); !errors.Is(err, domain.ErrSubdomainStyleNoBase) {
		t.Fatalf("unset base domain error = %v", err)
	}
	updated, err := SetEndpointStyle(ctx, repo, tenant.ID, domain.TenantEndpointStyleSubdomain, "idp.example", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if updated.EndpointStyle != domain.TenantEndpointStyleSubdomain {
		t.Fatalf("endpoint style = %q", updated.EndpointStyle)
	}
}

func TestUpdateAppliesDisplayNameAndPolicyOverride(t *testing.T) {
	repo := memory.NewTenantRepository()
	created, err := Create(context.Background(), repo, "acme", "Acme", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	floor := PolicyFloor{MinLength: 12, MaxLength: 128, HistoryDepth: 5}
	newName := "Acme Inc."
	minLen := 16
	historyDepth := 10
	updated, err := Update(context.Background(), repo, created.ID, UpdateInput{
		DisplayName: &newName,
		PasswordPolicyOverride: &domain.PasswordPolicyOverride{
			MinLength: &minLen, HistoryDepth: &historyDepth,
		},
	}, floor, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if updated.DisplayName != newName {
		t.Fatalf("display_name = %q", updated.DisplayName)
	}
	if updated.PasswordPolicyOverride == nil ||
		updated.PasswordPolicyOverride.MinLength == nil ||
		*updated.PasswordPolicyOverride.MinLength != minLen {
		t.Fatalf("override = %#v", updated.PasswordPolicyOverride)
	}
	if updated.PasswordPolicyOverride.MaxLength != nil {
		t.Fatalf("max_length should remain unset: %#v", updated.PasswordPolicyOverride)
	}
}

func TestUpdateRejectsWeakerPolicyOverride(t *testing.T) {
	floor := PolicyFloor{MinLength: 12, MaxLength: 128, HistoryDepth: 5}
	cases := []struct {
		name     string
		override domain.PasswordPolicyOverride
	}{
		{"shorter min_length", domain.PasswordPolicyOverride{MinLength: new(8)}},
		{"longer max_length", domain.PasswordPolicyOverride{MaxLength: new(256)}},
		{"shorter history_depth", domain.PasswordPolicyOverride{HistoryDepth: new(2)}},
		// REQ-TENANCY-019: an expiry that is too short or too long is rejected by
		// the system bounds.
		{"max_age_days below the system floor", domain.PasswordPolicyOverride{MaxAgeDays: new(PasswordMaxAgeDaysFloor - 1)}},
		{"max_age_days above the system ceiling", domain.PasswordPolicyOverride{MaxAgeDays: new(PasswordMaxAgeDaysCeiling + 1)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := memory.NewTenantRepository()
			created, err := Create(context.Background(), repo, "acme", "Acme", time.Now().UTC())
			if err != nil {
				t.Fatal(err)
			}
			_, err = Update(context.Background(), repo, created.ID, UpdateInput{
				PasswordPolicyOverride: &tc.override,
			}, floor, time.Now().UTC())
			if !errors.Is(err, ErrPolicyOverrideWeaker) {
				t.Fatalf("err = %v, want ErrPolicyOverrideWeaker", err)
			}
		})
	}
}

// REQ-AUTHENTICATION-024: the override change time is recorded because expiry is
// measured from it.
func TestUpdateRecordsPasswordPolicyUpdatedAt(t *testing.T) {
	repo := memory.NewTenantRepository()
	created, err := Create(context.Background(), repo, "acme", "Acme", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if created.PasswordPolicyUpdatedAt != nil {
		t.Fatalf("a fresh tenant has no policy change: %v", created.PasswordPolicyUpdatedAt)
	}
	floor := PolicyFloor{MinLength: 12, MaxLength: 128, HistoryDepth: 5}
	policyAt := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	maxAge := 90
	updated, err := Update(context.Background(), repo, created.ID, UpdateInput{
		PasswordPolicyOverride: &domain.PasswordPolicyOverride{MaxAgeDays: &maxAge},
	}, floor, policyAt)
	if err != nil {
		t.Fatal(err)
	}
	if updated.PasswordPolicyUpdatedAt == nil || !updated.PasswordPolicyUpdatedAt.Equal(policyAt) {
		t.Fatalf("password_policy_updated_at = %v, want %v", updated.PasswordPolicyUpdatedAt, policyAt)
	}
	if updated.PasswordPolicyOverride == nil || updated.PasswordPolicyOverride.MaxAgeDays == nil ||
		*updated.PasswordPolicyOverride.MaxAgeDays != maxAge {
		t.Fatalf("override = %#v", updated.PasswordPolicyOverride)
	}

	// A non-policy update must not move the reference time; moving it would
	// extend the grace window without end.
	newName := "Acme Inc."
	renamed, err := Update(context.Background(), repo, created.ID, UpdateInput{
		DisplayName: &newName,
	}, floor, policyAt.AddDate(0, 0, 30))
	if err != nil {
		t.Fatal(err)
	}
	if renamed.PasswordPolicyUpdatedAt == nil || !renamed.PasswordPolicyUpdatedAt.Equal(policyAt) {
		t.Fatalf("password_policy_updated_at = %v, want it unchanged at %v", renamed.PasswordPolicyUpdatedAt, policyAt)
	}
}

func TestUpdatePreservesUnsetFields(t *testing.T) {
	repo := memory.NewTenantRepository()
	created, err := Create(context.Background(), repo, "acme", "Acme", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	floor := PolicyFloor{MinLength: 12, MaxLength: 128, HistoryDepth: 5}
	minLen := 16
	if _, err := Update(context.Background(), repo, created.ID, UpdateInput{
		PasswordPolicyOverride: &domain.PasswordPolicyOverride{MinLength: &minLen},
	}, floor, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	// 後続の display_name 単独更新で override が消えないこと。
	newName := "Acme Renamed"
	updated, err := Update(context.Background(), repo, created.ID, UpdateInput{
		DisplayName: &newName,
	}, floor, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if updated.DisplayName != newName {
		t.Fatalf("display_name = %q", updated.DisplayName)
	}
	if updated.PasswordPolicyOverride == nil ||
		updated.PasswordPolicyOverride.MinLength == nil ||
		*updated.PasswordPolicyOverride.MinLength != minLen {
		t.Fatalf("override lost: %#v", updated.PasswordPolicyOverride)
	}
}

func TestUpdateClearsOverrideWhenAllFieldsZero(t *testing.T) {
	repo := memory.NewTenantRepository()
	created, err := Create(context.Background(), repo, "acme", "Acme", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	floor := PolicyFloor{MinLength: 12, MaxLength: 128, HistoryDepth: 5}
	// まず override を設定。
	minLen := 20
	if _, err := Update(context.Background(), repo, created.ID, UpdateInput{
		PasswordPolicyOverride: &domain.PasswordPolicyOverride{MinLength: &minLen},
	}, floor, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	// 空 override で送ると global default 継承に戻る。
	updated, err := Update(context.Background(), repo, created.ID, UpdateInput{
		PasswordPolicyOverride: &domain.PasswordPolicyOverride{},
	}, floor, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if updated.PasswordPolicyOverride != nil {
		t.Fatalf("override should be cleared: %#v", updated.PasswordPolicyOverride)
	}
}

// テナント既定 locale は通知の locale 解決の第 2 段。カタログが
// 同梱翻訳を持たない locale は保存時に拒否し、空文字列でシステム既定へ戻す。
func TestUpdateDefaultLocale(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	newRepo := func(t *testing.T) *memory.TenantRepository {
		t.Helper()
		repo := memory.NewTenantRepository()
		if err := repo.Save(ctx, &domain.Tenant{
			ID: "tenant-a", Realm: "acme", DisplayName: "Acme Inc.",
			Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
		return repo
	}
	locale := func(value string) *string { return &value }

	t.Run("sets a supported locale", func(t *testing.T) {
		repo := newRepo(t)
		updated, err := Update(ctx, repo, "tenant-a",
			UpdateInput{DefaultLocale: locale("ja")}, PolicyFloor{}, now)
		if err != nil {
			t.Fatal(err)
		}
		if updated.DefaultLocale == nil || *updated.DefaultLocale != "ja" {
			t.Fatalf("DefaultLocale = %v, want ja", updated.DefaultLocale)
		}
	})

	t.Run("clears back to the system default", func(t *testing.T) {
		repo := newRepo(t)
		if _, err := Update(ctx, repo, "tenant-a",
			UpdateInput{DefaultLocale: locale("ja")}, PolicyFloor{}, now); err != nil {
			t.Fatal(err)
		}
		updated, err := Update(ctx, repo, "tenant-a",
			UpdateInput{DefaultLocale: locale("")}, PolicyFloor{}, now)
		if err != nil {
			t.Fatal(err)
		}
		if updated.DefaultLocale != nil {
			t.Fatalf("DefaultLocale = %v, want nil", updated.DefaultLocale)
		}
	})

	t.Run("rejects a locale with no bundled translation", func(t *testing.T) {
		repo := newRepo(t)
		_, err := Update(ctx, repo, "tenant-a",
			UpdateInput{DefaultLocale: locale("fr")}, PolicyFloor{}, now)
		if !errors.Is(err, ErrUnsupportedDefaultLocale) {
			t.Fatalf("error = %v, want ErrUnsupportedDefaultLocale", err)
		}
		stored, err := repo.FindByID(ctx, "tenant-a")
		if err != nil {
			t.Fatal(err)
		}
		if stored.DefaultLocale != nil {
			t.Fatalf("a rejected locale was saved: %v", stored.DefaultLocale)
		}
	})

	t.Run("omitting the field keeps the current value", func(t *testing.T) {
		repo := newRepo(t)
		if _, err := Update(ctx, repo, "tenant-a",
			UpdateInput{DefaultLocale: locale("ja")}, PolicyFloor{}, now); err != nil {
			t.Fatal(err)
		}
		name := "Acme Corporation"
		updated, err := Update(ctx, repo, "tenant-a",
			UpdateInput{DisplayName: &name}, PolicyFloor{}, now)
		if err != nil {
			t.Fatal(err)
		}
		if updated.DefaultLocale == nil || *updated.DefaultLocale != "ja" {
			t.Fatalf("DefaultLocale = %v, want the untouched ja", updated.DefaultLocale)
		}
	})
}
