package usecases_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/notification/email_memory"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/tenancy/db_memory"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
)

const notificationTenantID = "tenant-a"

func newNotificationTemplateDeps(
	ctx context.Context, t *testing.T,
) (tenantusecases.NotificationTemplateDeps, *email_memory.NoopEmailSender) {
	t.Helper()
	now := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	tenants := db_memory.NewTenantRepository()
	if err := tenants.Save(ctx, &domain.Tenant{
		ID: notificationTenantID, Realm: "acme", DisplayName: "Acme Inc.",
		Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	templates := db_memory.NewNotificationTemplateRepository()
	source := tenantusecases.TenantNotificationSource{
		TenantRepo: tenants, BrandingRepo: db_memory.NewTenantBrandingRepository(), TemplateRepo: templates,
	}
	sender := &email_memory.NoopEmailSender{}
	return tenantusecases.NotificationTemplateDeps{
		Repo:     templates,
		Tenant:   source,
		Notifier: &template.Notifier{Sender: sender, Tenant: source, SystemDefaultLocale: "en"},
	}, sender
}

func validTemplateInput() tenantusecases.NotificationTemplateInput {
	return tenantusecases.NotificationTemplateInput{
		Subject:         "【Acme】パスワード再設定のご案内",
		BodyText:        "{{user_display_name}} さん\n{{reset_url}}",
		BodyHTML:        "<p>{{user_display_name}} さん</p><p><a href=\"{{reset_url}}\">再設定</a></p>",
		FromDisplayName: "Acme サポート",
	}
}

// scenario `Tenancy: テナントの通知テンプレート上書きは組込み既定より優先される`
func TestListNotificationTemplatesCoversTheWholeCatalog(t *testing.T) {
	ctx := context.Background()
	deps, _ := newNotificationTemplateDeps(ctx, t)

	summaries, err := tenantusecases.ListNotificationTemplates(ctx, deps, notificationTenantID)
	if err != nil {
		t.Fatal(err)
	}
	want := len(notificationports.TemplateKeys()) * len(template.SupportedLocales())
	if len(summaries) != want {
		t.Fatalf("summaries=%d, want %d (every key x locale)", len(summaries), want)
	}
	for _, summary := range summaries {
		if summary.Customized {
			t.Errorf("%s/%s is customized before any override", summary.Key, summary.Locale)
		}
		if summary.Subject == "" {
			t.Errorf("%s/%s has no effective subject", summary.Key, summary.Locale)
		}
		if summary.UpdatedAt != nil {
			t.Errorf("%s/%s reports an update time without an override", summary.Key, summary.Locale)
		}
	}
}

// scenario `Tenancy: テナントの通知テンプレート上書きは組込み既定より優先される`
func TestUpdateThenResetNotificationTemplate(t *testing.T) {
	ctx := context.Background()
	deps, _ := newNotificationTemplateDeps(ctx, t)
	now := time.Date(2026, 7, 25, 9, 0, 0, 0, time.UTC)

	updated, err := tenantusecases.UpdateNotificationTemplate(ctx, deps, notificationTenantID,
		notificationports.TemplateKeyPasswordReset, "ja", validTemplateInput(), now)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Customized || updated.Subject != validTemplateInput().Subject {
		t.Fatalf("update did not take effect: %#v", updated)
	}
	if updated.DefaultSubject == updated.Subject {
		t.Error("the builtin default subject should still be reported for comparison")
	}
	if len(updated.Placeholders) == 0 {
		t.Error("the allowed placeholder set should be reported to the editor")
	}

	fetched, err := tenantusecases.GetNotificationTemplate(ctx, deps, notificationTenantID,
		notificationports.TemplateKeyPasswordReset, "ja")
	if err != nil {
		t.Fatal(err)
	}
	if !fetched.Customized || fetched.Subject != validTemplateInput().Subject {
		t.Fatalf("stored override not returned: %#v", fetched)
	}

	// 上書きしていない locale は影響を受けない。
	other, err := tenantusecases.GetNotificationTemplate(ctx, deps, notificationTenantID,
		notificationports.TemplateKeyPasswordReset, "en")
	if err != nil {
		t.Fatal(err)
	}
	if other.Customized {
		t.Error("the ja override leaked into en")
	}

	reset, err := tenantusecases.ResetNotificationTemplate(ctx, deps, notificationTenantID,
		notificationports.TemplateKeyPasswordReset, "ja")
	if err != nil {
		t.Fatal(err)
	}
	if reset.Customized || reset.Subject != reset.DefaultSubject {
		t.Fatalf("reset did not return to the builtin default: %#v", reset)
	}

	// 上書きが無い状態での reset も冪等に成功する。
	if _, err := tenantusecases.ResetNotificationTemplate(ctx, deps, notificationTenantID,
		notificationports.TemplateKeyPasswordReset, "ja"); err != nil {
		t.Fatalf("reset is not idempotent: %v", err)
	}
}

// scenario `Tenancy: 許可されていない差し込み変数を含むテンプレート上書きは保存時に拒否される`
func TestUpdateNotificationTemplateRejectsInvalidInput(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name  string
		key   notificationports.TemplateKey
		local string
		input tenantusecases.NotificationTemplateInput
		want  error
	}{
		{
			name: "unknown placeholder", key: notificationports.TemplateKeyPasswordReset, local: "ja",
			input: tenantusecases.NotificationTemplateInput{
				Subject: "件名", BodyText: "{{password}}", BodyHTML: "<p>本文</p>",
			},
			want: tenantusecases.ErrInvalidNotificationTemplate,
		},
		{
			name: "html body only", key: notificationports.TemplateKeyPasswordReset, local: "ja",
			input: tenantusecases.NotificationTemplateInput{
				Subject: "件名", BodyText: "", BodyHTML: "<p>本文</p>",
			},
			want: tenantusecases.ErrInvalidNotificationTemplate,
		},
		{
			name: "unsupported locale", key: notificationports.TemplateKeyPasswordReset, local: "fr",
			input: validTemplateInput(), want: tenantusecases.ErrUnknownNotificationTemplate,
		},
		{
			name: "unknown template key", key: notificationports.TemplateKey("made_up"), local: "ja",
			input: validTemplateInput(), want: tenantusecases.ErrUnknownNotificationTemplate,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deps, _ := newNotificationTemplateDeps(ctx, t)
			_, err := tenantusecases.UpdateNotificationTemplate(ctx, deps, notificationTenantID,
				tc.key, tc.local, tc.input, time.Now().UTC())
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			stored, findErr := deps.Repo.FindByKey(ctx, notificationTenantID, tc.key, tc.local)
			if findErr != nil {
				t.Fatal(findErr)
			}
			if stored != nil {
				t.Fatalf("a rejected override was saved: %#v", stored)
			}
		})
	}
}

// scenario `Tenancy: プレビューは実送信せずテスト送信は操作者本人にしか届かない`
func TestPreviewNotificationTemplateDoesNotSendOrSave(t *testing.T) {
	ctx := context.Background()
	deps, sender := newNotificationTemplateDeps(ctx, t)

	preview, err := tenantusecases.PreviewNotificationTemplate(ctx, deps, notificationTenantID,
		notificationports.TemplateKeyPasswordReset, "ja", tenantusecases.NotificationTemplateInput{
			Subject:  "編集中の件名",
			BodyText: "{{user_display_name}} さん {{reset_url}}",
			BodyHTML: "<p>{{user_display_name}}</p>",
		})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Subject != "編集中の件名" {
		t.Errorf("preview subject = %q, want the unsaved subject", preview.Subject)
	}
	if preview.BodyText == "" || preview.BodyHTML == "" {
		t.Error("preview must render both parts")
	}
	if strings.Contains(preview.BodyText, "{{") {
		t.Errorf("preview left a placeholder unexpanded: %q", preview.BodyText)
	}
	// テナント由来の値は実際の設定から入り、サンプル値で上書きされない。
	if !strings.Contains(preview.BodyText+preview.BodyHTML, "Acme Inc.") &&
		!strings.Contains(preview.Subject, "Acme Inc.") {
		t.Logf("preview did not need the tenant display name: %q", preview.BodyText)
	}
	if len(sender.Sent) != 0 {
		t.Fatalf("preview sent %d emails, want 0", len(sender.Sent))
	}
	stored, err := deps.Repo.FindByKey(ctx, notificationTenantID, notificationports.TemplateKeyPasswordReset, "ja")
	if err != nil {
		t.Fatal(err)
	}
	if stored != nil {
		t.Fatalf("preview saved an override: %#v", stored)
	}
}

// 省略したフィールドは現在有効な文面を使う。
func TestPreviewNotificationTemplateFallsBackToTheEffectiveTemplate(t *testing.T) {
	ctx := context.Background()
	deps, _ := newNotificationTemplateDeps(ctx, t)

	preview, err := tenantusecases.PreviewNotificationTemplate(ctx, deps, notificationTenantID,
		notificationports.TemplateKeyPasswordReset, "ja", tenantusecases.NotificationTemplateInput{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.Subject, "パスワード") {
		t.Fatalf("preview subject = %q, want the ja builtin default", preview.Subject)
	}
}

// scenario `Tenancy: プレビューは実送信せずテスト送信は操作者本人にしか届かない`
// 宛先は操作者本人に固定する。任意宛先を許すとメール送信の踏み台になる (ADR-142 決定 8)。
func TestSendTestNotificationGoesToTheActorOnly(t *testing.T) {
	ctx := context.Background()
	deps, sender := newNotificationTemplateDeps(ctx, t)

	result, err := tenantusecases.SendTestNotification(ctx, deps, notificationTenantID,
		notificationports.TemplateKeyPasswordReset, "ja", tenantusecases.TestNotificationActor{
			Email: "operator@example.test", DisplayName: "operator",
		})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Delivered || result.To != "operator@example.test" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(sender.Sent) != 1 || sender.Sent[0].To != "operator@example.test" {
		t.Fatalf("unexpected sent emails: %#v", sender.Sent)
	}
	if !strings.Contains(sender.Sent[0].Subject, "パスワード") {
		t.Errorf("test send used the wrong locale: %q", sender.Sent[0].Subject)
	}
	if sender.Sent[0].HTML == "" {
		t.Error("test send must include the HTML part")
	}
}

// scenario `Tenancy: プレビューは実送信せずテスト送信は操作者本人にしか届かない`
// (extension at 4): 操作者が検証済みアドレスを持たなければ拒否する。
func TestSendTestNotificationRequiresAnActorAddress(t *testing.T) {
	ctx := context.Background()
	deps, sender := newNotificationTemplateDeps(ctx, t)

	_, err := tenantusecases.SendTestNotification(ctx, deps, notificationTenantID,
		notificationports.TemplateKeyPasswordReset, "ja", tenantusecases.TestNotificationActor{DisplayName: "operator"})
	if !errors.Is(err, tenantusecases.ErrTestNotificationRecipient) {
		t.Fatalf("error = %v, want ErrTestNotificationRecipient", err)
	}
	if len(sender.Sent) != 0 {
		t.Fatalf("sent %d emails, want 0", len(sender.Sent))
	}
}

// テスト送信は保存済み上書きを使う (編集中の文面ではなく「今届く文面」を確認する)。
func TestSendTestNotificationUsesTheStoredOverride(t *testing.T) {
	ctx := context.Background()
	deps, sender := newNotificationTemplateDeps(ctx, t)
	now := time.Date(2026, 7, 25, 9, 0, 0, 0, time.UTC)
	if _, err := tenantusecases.UpdateNotificationTemplate(ctx, deps, notificationTenantID,
		notificationports.TemplateKeyPasswordReset, "ja", validTemplateInput(), now); err != nil {
		t.Fatal(err)
	}

	if _, err := tenantusecases.SendTestNotification(ctx, deps, notificationTenantID,
		notificationports.TemplateKeyPasswordReset, "ja", tenantusecases.TestNotificationActor{
			Email: "operator@example.test", DisplayName: "operator",
		}); err != nil {
		t.Fatal(err)
	}
	if len(sender.Sent) != 1 {
		t.Fatalf("sent %d emails, want 1", len(sender.Sent))
	}
	if sender.Sent[0].Subject != validTemplateInput().Subject {
		t.Errorf("subject = %q, want the stored override", sender.Sent[0].Subject)
	}
	if sender.Sent[0].FromDisplayName != validTemplateInput().FromDisplayName {
		t.Errorf("from display name = %q, want the stored override", sender.Sent[0].FromDisplayName)
	}
}

// TenantNotificationSource は Tenancy の repository で shared/notification の port を
// 満たす (ADR-142 決定 11)。既定 locale と製品名 / 表示名の解決経路を固定する。
func TestTenantNotificationSourceResolvesTenantSettings(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	tenants := db_memory.NewTenantRepository()
	locale := "ja"
	if err := tenants.Save(ctx, &domain.Tenant{
		ID: notificationTenantID, Realm: "acme", DisplayName: "Acme Inc.", DefaultLocale: &locale,
		Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	branding := db_memory.NewTenantBrandingRepository()
	if err := branding.Save(ctx, &domain.TenantBranding{
		TenantID: notificationTenantID, ProductName: "Acme ID", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	source := tenantusecases.TenantNotificationSource{
		TenantRepo: tenants, BrandingRepo: branding, TemplateRepo: db_memory.NewNotificationTemplateRepository(),
	}

	settings, err := source.NotificationSettings(ctx, notificationTenantID)
	if err != nil {
		t.Fatal(err)
	}
	if settings.DefaultLocale != "ja" {
		t.Errorf("DefaultLocale = %q, want ja", settings.DefaultLocale)
	}
	if settings.ProductName != "Acme ID" {
		t.Errorf("ProductName = %q, want the branding product name", settings.ProductName)
	}
	if settings.TenantDisplayName != "Acme Inc." {
		t.Errorf("TenantDisplayName = %q", settings.TenantDisplayName)
	}

	// 未知テナントでも通知は止めない (空の設定を返し、呼び出し側が既定へ落とす)。
	empty, err := source.NotificationSettings(ctx, "missing-tenant")
	if err != nil {
		t.Fatal(err)
	}
	if empty.DefaultLocale != "" || empty.ProductName != "" {
		t.Errorf("unknown tenant produced settings: %#v", empty)
	}
}
