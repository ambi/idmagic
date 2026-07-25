package template_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/shared/notification/email_memory"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/notification/template"
)

type stubTenantSource struct {
	settings  notificationports.TenantNotificationSettings
	overrides map[string]*notificationports.TemplateOverride
}

func (s stubTenantSource) NotificationSettings(context.Context, string) (notificationports.TenantNotificationSettings, error) {
	return s.settings, nil
}

func (s stubTenantSource) FindTemplateOverride(_ context.Context, _ string, key notificationports.TemplateKey, locale string) (*notificationports.TemplateOverride, error) {
	return s.overrides[string(key)+"/"+locale], nil
}

func newNotifier(t *testing.T, source notificationports.TenantNotificationSource) (*template.Notifier, *email_memory.NoopEmailSender) {
	t.Helper()
	sender := &email_memory.NoopEmailSender{}
	return &template.Notifier{Sender: sender, Tenant: source, SystemDefaultLocale: "en"}, sender
}

func passwordResetNotification() notificationports.Notification {
	return notificationports.Notification{
		TenantID:        "t1",
		To:              "hanako@example.test",
		Key:             notificationports.TemplateKeyPasswordReset,
		RecipientLocale: "ja",
		Vars: map[string]string{
			"user_display_name":  "hanako",
			"reset_url":          "https://idp.test/reset_password?token=abc",
			"expires_in_minutes": "30",
		},
	}
}

// scenario `Tenancy: 日本語ロケールのユーザーには日本語のパスワードリセットメールが届く`
func TestNotifyUsesRecipientLocaleAndSendsTextAndHTML(t *testing.T) {
	notifier, sender := newNotifier(t, stubTenantSource{
		settings: notificationports.TenantNotificationSettings{ProductName: "IdMagic", TenantDisplayName: "Acme"},
	})

	if !notifier.Notify(context.Background(), passwordResetNotification()) {
		t.Fatal("Notify returned false")
	}
	if len(sender.Sent) != 1 {
		t.Fatalf("sent %d messages, want 1", len(sender.Sent))
	}
	sent := sender.Sent[0]
	if sent.To != "hanako@example.test" {
		t.Errorf("to = %q", sent.To)
	}
	if !containsJapanese(sent.Subject) {
		t.Errorf("subject = %q, want the ja builtin template", sent.Subject)
	}
	if sent.Text == "" || sent.HTML == "" {
		t.Errorf("both text and html are required, got text=%q html=%q", sent.Text, sent.HTML)
	}
	if !strings.Contains(sent.Text, "https://idp.test/reset_password?token=abc") {
		t.Errorf("text body has no reset link: %q", sent.Text)
	}
	if !strings.Contains(sent.HTML, "https://idp.test/reset_password?token=abc") {
		t.Errorf("html body has no reset link: %q", sent.HTML)
	}
}

// scenario `Tenancy: 日本語ロケールのユーザーには日本語のパスワードリセットメールが届く`
// (extension at 1): 受信者 locale が未設定ならテナント既定が採用される。
func TestNotifyFallsBackToTenantDefaultLocale(t *testing.T) {
	notifier, sender := newNotifier(t, stubTenantSource{
		settings: notificationports.TenantNotificationSettings{DefaultLocale: "ja", ProductName: "IdMagic"},
	})
	notification := passwordResetNotification()
	notification.RecipientLocale = ""

	if !notifier.Notify(context.Background(), notification) {
		t.Fatal("Notify returned false")
	}
	if !containsJapanese(sender.Sent[0].Subject) {
		t.Fatalf("subject = %q, want the ja builtin template", sender.Sent[0].Subject)
	}
}

// scenario `Tenancy: 日本語ロケールのユーザーには日本語のパスワードリセットメールが届く`
// (extension at 1): 受信者もテナントも未設定ならシステム既定が採用される。
func TestNotifyFallsBackToSystemDefaultLocale(t *testing.T) {
	notifier, sender := newNotifier(t, stubTenantSource{
		settings: notificationports.TenantNotificationSettings{ProductName: "IdMagic"},
	})
	notification := passwordResetNotification()
	notification.RecipientLocale = ""

	if !notifier.Notify(context.Background(), notification) {
		t.Fatal("Notify returned false")
	}
	if containsJapanese(sender.Sent[0].Subject) {
		t.Fatalf("subject = %q, want the en builtin template", sender.Sent[0].Subject)
	}
}

// scenario `Tenancy: テナントの通知テンプレート上書きは組込み既定より優先される`
func TestNotifyPrefersTenantOverrideOverBuiltin(t *testing.T) {
	notifier, sender := newNotifier(t, stubTenantSource{
		settings: notificationports.TenantNotificationSettings{ProductName: "IdMagic"},
		overrides: map[string]*notificationports.TemplateOverride{
			"password_reset/ja": {
				TenantID: "t1", Key: notificationports.TemplateKeyPasswordReset, Locale: "ja",
				Subject:         "【Acme】パスワード再設定のご案内",
				BodyText:        "{{user_display_name}} 様\n{{reset_url}}",
				BodyHTML:        "<p>{{user_display_name}} 様</p><p>{{reset_url}}</p>",
				FromDisplayName: "Acme サポート",
			},
		},
	})

	if !notifier.Notify(context.Background(), passwordResetNotification()) {
		t.Fatal("Notify returned false")
	}
	sent := sender.Sent[0]
	if sent.Subject != "【Acme】パスワード再設定のご案内" {
		t.Errorf("subject = %q, want the tenant override", sent.Subject)
	}
	if sent.FromDisplayName != "Acme サポート" {
		t.Errorf("from display name = %q, want the tenant override", sent.FromDisplayName)
	}
}

// scenario `Tenancy: テナントの通知テンプレート上書きは組込み既定より優先される`
// (extension at 5): ja の上書きは en の受信者に影響しない。
func TestNotifyIgnoresOverrideForAnotherLocale(t *testing.T) {
	notifier, sender := newNotifier(t, stubTenantSource{
		settings: notificationports.TenantNotificationSettings{ProductName: "IdMagic"},
		overrides: map[string]*notificationports.TemplateOverride{
			"password_reset/ja": {Subject: "上書き", BodyText: "本文", BodyHTML: "<p>本文</p>"},
		},
	})
	notification := passwordResetNotification()
	notification.RecipientLocale = "en"

	if !notifier.Notify(context.Background(), notification) {
		t.Fatal("Notify returned false")
	}
	if sender.Sent[0].Subject == "上書き" {
		t.Fatal("the ja override leaked into the en message")
	}
}

// テナント由来の値 (製品名・テナント表示名) は呼び出し元が渡さず Notifier が補う。
func TestNotifyInjectsTenantDerivedVariables(t *testing.T) {
	notifier, sender := newNotifier(t, stubTenantSource{
		settings: notificationports.TenantNotificationSettings{ProductName: "Acme ID", TenantDisplayName: "Acme Inc"},
	})
	notification := passwordResetNotification()
	notification.RecipientLocale = "en"

	if !notifier.Notify(context.Background(), notification) {
		t.Fatal("Notify returned false")
	}
	body := sender.Sent[0].Subject + sender.Sent[0].Text + sender.Sent[0].HTML
	if !strings.Contains(body, "Acme ID") {
		t.Fatalf("product name was not injected: %q", body)
	}
}

// Tenancy 由来の設定が取れない構成 (テナント解決前の経路や単体テスト) でも、通知は
// システム既定 locale と既定の製品名で送れなければならない。
func TestNotifyWorksWithoutTenantSource(t *testing.T) {
	notifier, sender := newNotifier(t, nil)
	notification := passwordResetNotification()
	notification.RecipientLocale = "en"

	if !notifier.Notify(context.Background(), notification) {
		t.Fatal("Notify returned false")
	}
	if sender.Sent[0].Text == "" {
		t.Fatal("no text body")
	}
}

// 変数が足りないまま「リンクの無いメール」を送らない。fail-open な port なので
// 送信失敗と同じく false を返し、呼び出し元には伝播しない。
func TestNotifyRefusesToSendWithMissingVariables(t *testing.T) {
	notifier, sender := newNotifier(t, stubTenantSource{})
	notification := passwordResetNotification()
	delete(notification.Vars, "reset_url")

	if notifier.Notify(context.Background(), notification) {
		t.Fatal("Notify returned true for an unrenderable template")
	}
	if len(sender.Sent) != 0 {
		t.Fatalf("sent %d messages, want 0", len(sender.Sent))
	}
}

// 上書きが壊れている (許可外の変数を含む) 場合も、その上書きは無視して組込み既定で
// 送る。保存時に拒否しているので通常は起きないが、DB を直接書かれた場合でも復旧導線を
// 止めない。
func TestNotifyFallsBackToBuiltinWhenOverrideIsInvalid(t *testing.T) {
	notifier, sender := newNotifier(t, stubTenantSource{
		overrides: map[string]*notificationports.TemplateOverride{
			"password_reset/en": {Subject: "OVERRIDDEN {{password}}", BodyText: "{{password}}", BodyHTML: "<p>{{password}}</p>"},
		},
	})
	notification := passwordResetNotification()
	notification.RecipientLocale = "en"

	if !notifier.Notify(context.Background(), notification) {
		t.Fatal("Notify returned false")
	}
	if strings.Contains(sender.Sent[0].Subject, "OVERRIDDEN") {
		t.Fatalf("the invalid override was used: %q", sender.Sent[0].Subject)
	}
}
