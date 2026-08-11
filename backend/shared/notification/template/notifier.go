package template

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/logging"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
)

// DefaultProductName は branding の product_name が未設定のときに使うシステム既定。
// hosted UI 側の既定表示 (IdMagic) と揃える。
const DefaultProductName = "IdMagic"

// Notifier は locale 解決 → 文面解決 (上書き > 組込み既定) → 描画 → 送信を 1 本にまとめた
// ports.Notifier 実装。呼び出し元の use case は文面を組み立てない。
type Notifier struct {
	Sender notificationports.EmailSender
	// Tenant は Tenancy 由来の既定 locale / 製品名 / 表示名と上書きの取得元。nil なら
	// システム既定 locale と既定の製品名で送る。
	Tenant notificationports.TenantNotificationSource
	// SystemDefaultLocale は NotificationLocaleResolution の最終段。空なら FallbackLocale。
	SystemDefaultLocale string
}

func (n *Notifier) Notify(ctx context.Context, notification notificationports.Notification) bool {
	if n == nil || n.Sender == nil {
		return false
	}
	settings := n.settings(ctx, notification.TenantID)
	locale := ResolveLocale(notification.RecipientLocale, settings.DefaultLocale, n.SystemDefaultLocale)

	def, err := Builtin(notification.Key, locale)
	if err != nil {
		// 宛先アドレスは PII なのでマスクする。
		logging.Error(ctx, "notification template is unavailable",
			"template_key", string(notification.Key), "locale", locale,
			"to", logging.MaskEmail(notification.To), "error", err)
		return false
	}
	if override := n.override(ctx, notification, locale); override != nil {
		def = *override
	}

	rendered, err := Render(def, mergeVars(map[string]string{
		"product_name":        settings.ProductName,
		"tenant_display_name": settings.TenantDisplayName,
	}, notification.Vars))
	if err != nil {
		// 描画できない文面を「変数が抜けたメール」として配らない。
		logging.Error(ctx, "notification template did not render",
			"template_key", string(notification.Key), "locale", locale,
			"to", logging.MaskEmail(notification.To), "error", err)
		return false
	}
	return n.Sender.SendEmail(ctx, notificationports.EmailMessage{
		To: notification.To, Subject: rendered.Subject, Text: rendered.Text,
		HTML: rendered.HTML, FromDisplayName: rendered.FromDisplayName,
	})
}

// settings はテナント由来の値を引く。引けない場合もメールは送る (通知は復旧導線であり、
// 設定の読み出し失敗で止めない)。製品名は既定へ落とす。
func (n *Notifier) settings(ctx context.Context, tenantID string) notificationports.TenantNotificationSettings {
	settings := notificationports.TenantNotificationSettings{}
	if n.Tenant != nil && tenantID != "" {
		loaded, err := n.Tenant.NotificationSettings(ctx, tenantID)
		if err != nil {
			logging.Error(ctx, "tenant notification settings unavailable", "tenant_id", tenantID, "error", err)
		} else {
			settings = loaded
		}
	}
	if settings.ProductName == "" {
		settings.ProductName = DefaultProductName
	}
	if settings.TenantDisplayName == "" {
		settings.TenantDisplayName = settings.ProductName
	}
	return settings
}

// override はテナント上書きを引く。許可外の変数を含む壊れた上書き (DB を直接書かれた等)
// は無視して組込み既定へ落とし、復旧導線を止めない。
func (n *Notifier) override(ctx context.Context, notification notificationports.Notification, locale string) *Definition {
	if n.Tenant == nil || notification.TenantID == "" {
		return nil
	}
	stored, err := n.Tenant.FindTemplateOverride(ctx, notification.TenantID, notification.Key, locale)
	if err != nil {
		logging.Error(ctx, "notification template override unavailable",
			"tenant_id", notification.TenantID, "template_key", string(notification.Key), "locale", locale, "error", err)
		return nil
	}
	if stored == nil {
		return nil
	}
	def := Definition{
		Subject: stored.Subject, BodyText: stored.BodyText,
		BodyHTML: stored.BodyHTML, FromDisplayName: stored.FromDisplayName,
	}
	if err := ValidateDefinition(notification.Key, def); err != nil {
		logging.Error(ctx, "stored notification template override is invalid; falling back to the builtin default",
			"tenant_id", notification.TenantID, "template_key", string(notification.Key), "locale", locale, "error", err)
		return nil
	}
	return &def
}
