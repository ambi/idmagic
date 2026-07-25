package bootstrap

import (
	"context"
	"testing"

	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

// Notifier は組込み既定カタログだけでなくテナント上書きも見なければならない。ここが
// 繋がっていないと、管理画面で保存した文面が実送信に反映されない (wi-288, ADR-142)。
func TestAssembleNotificationAppliesTenantOverrides(t *testing.T) {
	ctx := context.Background()
	templates := tenancymemory.NewNotificationTemplateRepository()
	tenants := tenancymemory.NewTenantRepository()
	deps := &Dependencies{Tenancy: tenancy.Module{
		TenantRepo:            tenants,
		BrandingRepo:          tenancymemory.NewTenantBrandingRepository(),
		NotificationTemplates: templates,
	}}
	if err := tenants.Save(ctx, &domain.Tenant{
		ID: "tenant-a", Realm: "acme", DisplayName: "Acme Inc.", Status: domain.TenantStatusActive,
	}); err != nil {
		t.Fatal(err)
	}
	if err := templates.Save(ctx, &notificationports.TemplateOverride{
		TenantID: "tenant-a", Key: notificationports.TemplateKeyPasswordReset, Locale: "en",
		Subject: "OVERRIDDEN", BodyText: "{{reset_url}}", BodyHTML: "<p>{{reset_url}}</p>",
	}); err != nil {
		t.Fatal(err)
	}

	if err := AssembleNotification(deps, stubEnv(map[string]string{})); err != nil {
		t.Fatalf("AssembleNotification: %v", err)
	}
	if deps.Notification.EmailSender == nil || deps.Notification.Notifier == nil {
		t.Fatal("both the sender and the notifier must be wired")
	}

	sender := &captureEmailSender{}
	deps.Notification.EmailSender = sender
	if err := AssembleNotification(deps, stubEnv(map[string]string{})); err != nil {
		t.Fatal(err)
	}
	// 明示的に差した sender は上書きしない (テスト・組み込み用途の差し替えを壊さない)。
	if deps.Notification.EmailSender != notificationports.EmailSender(sender) {
		t.Fatal("an explicitly provided sender must be kept")
	}

	if !deps.Notification.Notifier.Notify(ctx, notificationports.Notification{
		TenantID: "tenant-a", To: "alice@example.test",
		Key: notificationports.TemplateKeyPasswordReset, RecipientLocale: "en",
		Vars: map[string]string{
			"user_display_name": "alice", "reset_url": "https://idp.test/reset", "expires_in_minutes": "30",
		},
	}) {
		t.Fatal("Notify returned false")
	}
	if len(sender.sent) != 1 || sender.sent[0].Subject != "OVERRIDDEN" {
		t.Fatalf("the tenant override did not reach the sender: %+v", sender.sent)
	}
}

// テナント既定 locale も Notifier から見えていなければ、解決順序の第 2 段が死ぬ。
func TestAssembleNotificationUsesTenantDefaultLocale(t *testing.T) {
	ctx := context.Background()
	tenants := tenancymemory.NewTenantRepository()
	locale := "ja"
	if err := tenants.Save(ctx, &domain.Tenant{
		ID: "tenant-a", Realm: "acme", DisplayName: "Acme Inc.",
		Status: domain.TenantStatusActive, DefaultLocale: &locale,
	}); err != nil {
		t.Fatal(err)
	}
	sender := &captureEmailSender{}
	deps := &Dependencies{Tenancy: tenancy.Module{
		TenantRepo:            tenants,
		BrandingRepo:          tenancymemory.NewTenantBrandingRepository(),
		NotificationTemplates: tenancymemory.NewNotificationTemplateRepository(),
	}}
	deps.Notification.EmailSender = sender
	if err := AssembleNotification(deps, stubEnv(map[string]string{})); err != nil {
		t.Fatal(err)
	}

	if !deps.Notification.Notifier.Notify(ctx, notificationports.Notification{
		TenantID: "tenant-a", To: "alice@example.test",
		Key: notificationports.TemplateKeyPasswordReset,
		Vars: map[string]string{
			"user_display_name": "alice", "reset_url": "https://idp.test/reset", "expires_in_minutes": "30",
		},
	}) {
		t.Fatal("Notify returned false")
	}
	if len(sender.sent) != 1 {
		t.Fatalf("sent=%d, want 1", len(sender.sent))
	}
	if !containsJapanese(sender.sent[0].Subject) {
		t.Fatalf("subject=%q, want the ja template selected by the tenant default", sender.sent[0].Subject)
	}
}

// DEFAULT_LOCALE は解決順序の最終段。未設定なら製品既定 (en) を使う。
func TestAssembleNotificationHonorsSystemDefaultLocale(t *testing.T) {
	ctx := context.Background()
	sender := &captureEmailSender{}
	deps := &Dependencies{}
	deps.Notification.EmailSender = sender
	if err := AssembleNotification(deps, stubEnv(map[string]string{"DEFAULT_LOCALE": "ja"})); err != nil {
		t.Fatal(err)
	}

	if !deps.Notification.Notifier.Notify(ctx, notificationports.Notification{
		To: "alice@example.test", Key: notificationports.TemplateKeyPasswordReset,
		Vars: map[string]string{
			"user_display_name": "alice", "reset_url": "https://idp.test/reset", "expires_in_minutes": "30",
		},
	}) {
		t.Fatal("Notify returned false")
	}
	if !containsJapanese(sender.sent[0].Subject) {
		t.Fatalf("subject=%q, want the ja template selected by DEFAULT_LOCALE", sender.sent[0].Subject)
	}
}

func TestAssembleNotificationRejectsUnsupportedSystemDefaultLocale(t *testing.T) {
	deps := &Dependencies{}
	err := AssembleNotification(deps, stubEnv(map[string]string{"DEFAULT_LOCALE": "fr"}))
	if err == nil {
		t.Fatal("an unsupported DEFAULT_LOCALE must fail startup rather than silently fall back")
	}
}

type captureEmailSender struct {
	sent []notificationports.EmailMessage
}

func (s *captureEmailSender) SendEmail(_ context.Context, message notificationports.EmailMessage) bool {
	s.sent = append(s.sent, message)
	return true
}

func containsJapanese(value string) bool {
	for _, r := range value {
		if (r >= 0x3040 && r <= 0x30ff) || (r >= 0x4e00 && r <= 0x9fff) {
			return true
		}
	}
	return false
}
