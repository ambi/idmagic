package ports

import (
	"context"
	"slices"
	"time"
)

// TemplateKey は通知の用途を表す固定識別子。SCL
// `Tenancy.NotificationTemplateKey` の双子定義。テナントは key を追加できない。
type TemplateKey string

const (
	TemplateKeyPasswordReset                 TemplateKey = "password_reset"
	TemplateKeyEmailVerification             TemplateKey = "email_verification"
	TemplateKeyEmailChangeConfirmation       TemplateKey = "email_change_confirmation"
	TemplateKeyAccountSecurityAlert          TemplateKey = "account_security_alert"
	TemplateKeyLifecycleWorkflowNotification TemplateKey = "lifecycle_workflow_notification"
)

// TemplateKeys はカタログが持つ全 key を安定した並びで返す。管理 API の一覧と
// カタログ完全性テストが同じ並びを使う。
func TemplateKeys() []TemplateKey {
	return []TemplateKey{
		TemplateKeyPasswordReset,
		TemplateKeyEmailVerification,
		TemplateKeyEmailChangeConfirmation,
		TemplateKeyAccountSecurityAlert,
		TemplateKeyLifecycleWorkflowNotification,
	}
}

func (k TemplateKey) Valid() bool {
	return slices.Contains(TemplateKeys(), k)
}

// TemplateOverride はテナントによる通知テンプレート上書き 1 件。SCL
// `Tenancy.NotificationTemplate` の双子定義。上書きできるのは件名 / テキスト本文 /
// HTML 本文 / 差出人表示名だけで、差出人アドレスはサーバ設定のまま。
type TemplateOverride struct {
	TenantID        string
	Key             TemplateKey
	Locale          string
	Subject         string
	BodyText        string
	BodyHTML        string
	FromDisplayName string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TenantNotificationSettings は通知 1 通の描画に必要なテナント由来の値。
// Tenancy が所有する Tenant / TenantBranding から組み立てる。
type TenantNotificationSettings struct {
	// DefaultLocale は NotificationLocaleResolution の第 2 段。未設定なら空文字列。
	DefaultLocale string
	// ProductName は branding の product_name、未設定ならシステム既定の製品名。
	ProductName string
	// TenantDisplayName は Tenant.display_name。
	TenantDisplayName string
}

// TenantNotificationSource は shared/notification が Tenancy から受け取る境界。
// カタログとレンダラは shared に置き、テナント上書きの永続化は Tenancy が所有する
// ため、shared 側はこの port だけを知る。
type TenantNotificationSource interface {
	NotificationSettings(ctx context.Context, tenantID string) (TenantNotificationSettings, error)
	FindTemplateOverride(ctx context.Context, tenantID string, key TemplateKey, locale string) (*TemplateOverride, error)
}

// Notification は「どの用途のメールを誰に送るか」だけを表す送信要求。文面と locale の
// 解決は Notifier の責務で、呼び出し元は文面を組み立てない。
type Notification struct {
	TenantID string
	To       string
	Key      TemplateKey
	// RecipientLocale は受信者 User の locale 属性 (未設定なら空文字列)。
	// NotificationLocaleResolution の第 1 段。
	RecipientLocale string
	// Vars は template_key の許可集合に含まれる差し込み変数。product_name と
	// tenant_display_name は Notifier が TenantNotificationSource から補うため、
	// 呼び出し元は渡さなくてよい。
	Vars map[string]string
}

// Notifier は EmailSender と同じく fail-open な送信境界。送信できたかどうかだけを返し、
// use case 層に配送エラーを伝播しない (EmailSent イベントで観測する)。
type Notifier interface {
	Notify(ctx context.Context, notification Notification) bool
}
