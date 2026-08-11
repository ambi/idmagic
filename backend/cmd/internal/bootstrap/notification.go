package bootstrap

import (
	"fmt"
	"strings"

	"github.com/ambi/idmagic/backend/shared/notification/template"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
)

// AssembleNotification は通知の送信経路を組み立てる: EmailSender adapter (未設定なら
// EMAIL_SENDER / SMTP_* から解決) と、その上に載る Notifier。Notifier は Tenancy の
// repository を通してテナント既定 locale・製品名・テンプレート上書きを解決する。
// API プロセスと worker プロセスの両方がこれを呼ぶ。
//
// DEFAULT_LOCALE は locale 解決の最終段。UI の
// ConfiguredDefaultLocale (VITE_DEFAULT_LOCALE) と同じ役割で、未設定なら製品既定
// (FallbackLocale) を使う。同梱翻訳を持たない値は silent fallback にせず起動を失敗させる
// (誤設定を「なぜか英語で届く」として運用中に気付く形にしない)。
func AssembleNotification(deps *Dependencies, getenv func(string) string) error {
	if deps.Notification.EmailSender == nil {
		sender, err := ResolveEmailSender(getenv)
		if err != nil {
			return fmt.Errorf("resolve email sender: %w", err)
		}
		deps.Notification.EmailSender = sender
	}

	systemDefaultLocale := template.FallbackLocale
	if configured := strings.TrimSpace(getenv("DEFAULT_LOCALE")); configured != "" {
		if !template.LocaleSupported(configured) {
			return fmt.Errorf("unsupported DEFAULT_LOCALE=%q (want one of %v)", configured, template.SupportedLocales())
		}
		systemDefaultLocale = configured
	}

	deps.Notification.Notifier = &template.Notifier{
		Sender: deps.Notification.EmailSender,
		Tenant: tenantusecases.TenantNotificationSource{
			TenantRepo:   deps.Tenancy.TenantRepo,
			BrandingRepo: deps.Tenancy.BrandingRepo,
			TemplateRepo: deps.Tenancy.NotificationTemplates,
		},
		SystemDefaultLocale: systemDefaultLocale,
	}
	return nil
}
