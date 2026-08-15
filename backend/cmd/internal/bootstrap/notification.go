package bootstrap

import (
	"github.com/ambi/idmagic/backend/shared/notification/template"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
)

// AssembleNotification は通知の送信経路を組み立てる: EmailSender adapter (未設定なら
// cfg.EmailSender / SMTP_* から解決) と、その上に載る Notifier。Notifier は Tenancy の
// repository を通してテナント既定 locale・製品名・テンプレート上書きを解決する。
// API プロセスと worker プロセスの両方がこれを呼ぶ。
//
// cfg.DefaultLocale は locale 解決の最終段。UI の
// ConfiguredDefaultLocale (VITE_DEFAULT_LOCALE) と同じ役割で、未設定なら製品既定
// (FallbackLocale) を使う。LoadSharedConfig が同梱翻訳を持たない値を起動時に fail-fast
// させているため (誤設定を「なぜか英語で届く」として運用中に気付く形にしない)、ここでは
// 検証済みの値を使うだけでよい。
func AssembleNotification(deps *Dependencies, cfg SharedConfig) error {
	if deps.Notification.EmailSender == nil {
		deps.Notification.EmailSender = ResolveEmailSender(cfg)
	}

	systemDefaultLocale := cfg.DefaultLocale
	if systemDefaultLocale == "" {
		systemDefaultLocale = template.FallbackLocale
	}

	// セキュリティ通知は同じ Notifier に載る。ここで決めるのは本文の説明を選ぶ locale の
	// 最終段と、送信を認証中のリクエストから切り離す方法だけである (wi-90)。
	deps.SecurityNotifications.SystemDefaultLocale = systemDefaultLocale
	if deps.SecurityNotifications.Run == nil {
		deps.SecurityNotifications.Run = runDetached
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
