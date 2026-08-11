package db_postgres

import (
	"context"
	"testing"
	"time"

	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestNotificationTemplateRepositorySaveFindListDelete(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTestTenant(t, db, "33333333-3333-3333-3333-333333333331")
	repo := &NotificationTemplateRepository{Pool: db}
	ctx := context.Background()

	// 行が無い組み合わせは「組込み既定を使う」なので nil, nil。
	if existing, err := repo.FindByKey(ctx, tenant.ID, notificationports.TemplateKeyPasswordReset, "ja"); err != nil || existing != nil {
		t.Fatalf("expected no override yet: %+v %v", existing, err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	override := &notificationports.TemplateOverride{
		TenantID: tenant.ID, Key: notificationports.TemplateKeyPasswordReset, Locale: "ja",
		Subject: "【Acme】パスワード再設定", BodyText: "{{reset_url}}", BodyHTML: "<p>{{reset_url}}</p>",
		FromDisplayName: "Acme サポート", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Save(ctx, override); err != nil {
		t.Fatalf("save: %v", err)
	}

	stored, err := repo.FindByKey(ctx, tenant.ID, notificationports.TemplateKeyPasswordReset, "ja")
	if err != nil || stored == nil {
		t.Fatalf("find: %+v %v", stored, err)
	}
	if stored.Subject != override.Subject || stored.BodyText != override.BodyText ||
		stored.BodyHTML != override.BodyHTML || stored.FromDisplayName != override.FromDisplayName {
		t.Fatalf("round trip lost content: %+v", stored)
	}

	// 上書きは (key, locale) 単位。別 locale は独立している。
	if other, err := repo.FindByKey(ctx, tenant.ID, notificationports.TemplateKeyPasswordReset, "en"); err != nil || other != nil {
		t.Fatalf("the ja override leaked into en: %+v %v", other, err)
	}

	// 同じ key の再保存は upsert で、created_at は動かさない。
	updated := *override
	updated.Subject = "【Acme】パスワードの再設定 (改訂)"
	updated.UpdatedAt = now.Add(time.Hour)
	if err := repo.Save(ctx, &updated); err != nil {
		t.Fatalf("re-save: %v", err)
	}
	stored, err = repo.FindByKey(ctx, tenant.ID, notificationports.TemplateKeyPasswordReset, "ja")
	if err != nil || stored == nil {
		t.Fatalf("find after re-save: %+v %v", stored, err)
	}
	if stored.Subject != updated.Subject {
		t.Errorf("subject = %q, want the re-saved value", stored.Subject)
	}
	if !stored.CreatedAt.Equal(now) {
		t.Errorf("created_at moved on upsert: %v", stored.CreatedAt)
	}

	list, err := repo.ListAll(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Key != notificationports.TemplateKeyPasswordReset {
		t.Fatalf("unexpected list: %+v", list)
	}

	// 削除は「既定へのリセット」そのもの。2 回目も成功する冪等操作。
	deleted, err := repo.Delete(ctx, tenant.ID, notificationports.TemplateKeyPasswordReset, "ja")
	if err != nil || !deleted {
		t.Fatalf("delete = %v, %v", deleted, err)
	}
	deleted, err = repo.Delete(ctx, tenant.ID, notificationports.TemplateKeyPasswordReset, "ja")
	if err != nil || deleted {
		t.Fatalf("second delete = %v, %v; want false, nil", deleted, err)
	}
	if stored, err := repo.FindByKey(ctx, tenant.ID, notificationports.TemplateKeyPasswordReset, "ja"); err != nil || stored != nil {
		t.Fatalf("override survived delete: %+v %v", stored, err)
	}
}

// 別テナントの上書きは互いに見えない。
func TestNotificationTemplateRepositoryIsolatesTenants(t *testing.T) {
	db := pgtest.Require(t)
	first := seedTestTenant(t, db, "33333333-3333-3333-3333-333333333332")
	second := seedTestTenant(t, db, "33333333-3333-3333-3333-333333333333")
	repo := &NotificationTemplateRepository{Pool: db}
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := repo.Save(ctx, &notificationports.TemplateOverride{
		TenantID: first.ID, Key: notificationports.TemplateKeyAccountSecurityAlert, Locale: "en",
		Subject: "Security notice", BodyText: "text", BodyHTML: "<p>text</p>",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	if stored, err := repo.FindByKey(ctx, second.ID, notificationports.TemplateKeyAccountSecurityAlert, "en"); err != nil || stored != nil {
		t.Fatalf("cross-tenant read returned %+v (%v)", stored, err)
	}
	list, err := repo.ListAll(ctx, second.ID)
	if err != nil || len(list) != 0 {
		t.Fatalf("cross-tenant list returned %+v (%v)", list, err)
	}
}

// tenants.default_locale は通知の locale 解決の第 2 段。永続化されないと解決順序が
// 実質 2 段になるため、往復を固定する。
func TestTenantRepositoryPersistsDefaultLocale(t *testing.T) {
	db := pgtest.Require(t)
	repo := &TenantRepository{Pool: db}
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	locale := "ja"
	tenant := &domain.Tenant{
		ID: "33333333-3333-3333-3333-333333333334", Realm: "locale-tenant",
		DisplayName: "Locale Test Tenant", Status: domain.TenantStatusActive,
		DefaultLocale: &locale, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Save(ctx, tenant); err != nil {
		t.Fatalf("save: %v", err)
	}

	stored, err := repo.FindByID(ctx, tenant.ID)
	if err != nil || stored == nil {
		t.Fatalf("find: %+v %v", stored, err)
	}
	if stored.DefaultLocale == nil || *stored.DefaultLocale != "ja" {
		t.Fatalf("DefaultLocale = %v, want ja", stored.DefaultLocale)
	}

	// 空へ戻すとシステム既定に戻る。
	stored.DefaultLocale = nil
	if err := repo.Save(ctx, stored); err != nil {
		t.Fatalf("save without locale: %v", err)
	}
	stored, err = repo.FindByID(ctx, tenant.ID)
	if err != nil || stored == nil {
		t.Fatalf("find after clear: %+v %v", stored, err)
	}
	if stored.DefaultLocale != nil {
		t.Fatalf("DefaultLocale = %v, want nil", stored.DefaultLocale)
	}
}
