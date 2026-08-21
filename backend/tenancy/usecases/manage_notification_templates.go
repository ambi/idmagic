package usecases

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// wrapTemplateValidation は長さ違反だけを型付きのまま上へ返す。ErrInvalidNotificationTemplate
// へ包むと「3 点セットと差し込み変数」の汎用メッセージに潰れ、どのフィールドを何文字
// 短くすればよいかが利用者に伝わらない。
func wrapTemplateValidation(err error) error {
	if _, ok := errors.AsType[*spec.LengthError](err); ok {
		return err
	}
	return errors.Join(ErrInvalidNotificationTemplate, err)
}

var (
	// ErrUnknownNotificationTemplate はカタログに無い template_key / locale。
	ErrUnknownNotificationTemplate = errors.New("unknown notification template")
	// ErrInvalidNotificationTemplate は許可外の差し込み変数、または件名 / テキスト /
	// HTML の 3 点セットが揃っていない上書き。
	ErrInvalidNotificationTemplate = errors.New("invalid notification template")
	// ErrTestNotificationRecipient は操作者に検証済みメールアドレスが無い状態での
	// テスト送信。宛先は操作者本人に固定するため、代替の宛先は無い。
	ErrTestNotificationRecipient = errors.New("the acting administrator has no verified email address")
)

// NotificationTemplateDeps は通知テンプレート管理 6 操作の共通依存。
type NotificationTemplateDeps struct {
	Repo   tenantports.NotificationTemplateRepository
	Tenant notificationports.TenantNotificationSource
	// Notifier はテスト送信に使う。プレビューは描画だけなので使わない。
	Notifier notificationports.Notifier
}

// NotificationTemplateInput は上書きの保存とプレビューの入力。プレビューでは空の
// フィールドが「現在有効な文面を使う」を意味し、保存では 3 点セットが必須。
type NotificationTemplateInput struct {
	Subject         string
	BodyText        string
	BodyHTML        string
	FromDisplayName string
}

// NotificationTemplateSummary は一覧の 1 行。SCL `NotificationTemplateSummary` の双子定義。
type NotificationTemplateSummary struct {
	Key        notificationports.TemplateKey
	Locale     string
	Customized bool
	Subject    string
	UpdatedAt  *time.Time
}

// NotificationTemplateDetail は編集用表現。SCL `NotificationTemplateDetail` の双子定義。
// 組込み既定を併せて返すので、UI は差分提示と「既定に戻す」を追加の問い合わせ無しに描ける。
type NotificationTemplateDetail struct {
	Key             notificationports.TemplateKey
	Locale          string
	Customized      bool
	Subject         string
	BodyText        string
	BodyHTML        string
	FromDisplayName string
	DefaultSubject  string
	DefaultBodyText string
	DefaultBodyHTML string
	Placeholders    []string
	UpdatedAt       *time.Time
}

// NotificationTemplatePreview は描画結果。SCL `NotificationTemplatePreviewResponse` の双子定義。
type NotificationTemplatePreview struct {
	Subject         string
	BodyText        string
	BodyHTML        string
	FromDisplayName string
}

// TestNotificationActor は操作した管理者。宛先固定の根拠になるので、呼び出し側は
// 認証済み principal からのみ組み立てる。
type TestNotificationActor struct {
	Email       string
	DisplayName string
}

// TestNotificationResult は SCL `NotificationTemplateTestSendResponse` の双子定義。
type TestNotificationResult struct {
	Delivered bool
	To        string
}

// ListNotificationTemplates はカタログの全 key × 全サポート locale を、上書きの有無つきで
// 返す。行が無い組み合わせも「組込み既定」として列挙する。
func ListNotificationTemplates(
	ctx context.Context, deps NotificationTemplateDeps, tenantID string,
) ([]NotificationTemplateSummary, error) {
	stored, err := deps.Repo.ListAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	overrides := map[string]*notificationports.TemplateOverride{}
	for _, override := range stored {
		overrides[string(override.Key)+"/"+override.Locale] = override
	}

	summaries := make([]NotificationTemplateSummary, 0, len(notificationports.TemplateKeys())*len(template.SupportedLocales()))
	for _, key := range notificationports.TemplateKeys() {
		for _, locale := range template.SupportedLocales() {
			summary := NotificationTemplateSummary{Key: key, Locale: locale}
			if override := overrides[string(key)+"/"+locale]; override != nil {
				updatedAt := override.UpdatedAt
				summary.Customized = true
				summary.Subject = override.Subject
				summary.UpdatedAt = &updatedAt
			} else {
				builtin, err := template.Builtin(key, locale)
				if err != nil {
					return nil, err
				}
				summary.Subject = builtin.Subject
			}
			summaries = append(summaries, summary)
		}
	}
	return summaries, nil
}

// GetNotificationTemplate は 1 件を編集用に返す。
func GetNotificationTemplate(
	ctx context.Context, deps NotificationTemplateDeps, tenantID string,
	key notificationports.TemplateKey, locale string,
) (*NotificationTemplateDetail, error) {
	builtin, override, err := resolveTemplate(ctx, deps, tenantID, key, locale)
	if err != nil {
		return nil, err
	}
	return newTemplateDetail(key, locale, builtin, override), nil
}

// UpdateNotificationTemplate は上書きを全置換で保存する。許可外の差し込み変数と
// 3 点セットの欠けは保存前に拒否する。
func UpdateNotificationTemplate(
	ctx context.Context, deps NotificationTemplateDeps, tenantID string,
	key notificationports.TemplateKey, locale string, input NotificationTemplateInput, now time.Time,
) (*NotificationTemplateDetail, error) {
	builtin, existing, err := resolveTemplate(ctx, deps, tenantID, key, locale)
	if err != nil {
		return nil, err
	}
	def := template.Definition{
		Subject:         strings.TrimSpace(input.Subject),
		BodyText:        input.BodyText,
		BodyHTML:        input.BodyHTML,
		FromDisplayName: strings.TrimSpace(input.FromDisplayName),
	}
	if err := template.ValidateDefinition(key, def); err != nil {
		return nil, wrapTemplateValidation(err)
	}

	override := &notificationports.TemplateOverride{
		TenantID: tenantID, Key: key, Locale: locale,
		Subject: def.Subject, BodyText: def.BodyText, BodyHTML: def.BodyHTML,
		FromDisplayName: def.FromDisplayName,
		CreatedAt:       normalizeNow(now), UpdatedAt: normalizeNow(now),
	}
	if existing != nil && !existing.CreatedAt.IsZero() {
		override.CreatedAt = existing.CreatedAt
	}
	if err := deps.Repo.Save(ctx, override); err != nil {
		return nil, err
	}
	return newTemplateDetail(key, locale, builtin, override), nil
}

// ResetNotificationTemplate は上書きを削除して組込み既定へ戻す。上書きが無い場合も
// 成功する冪等操作。
func ResetNotificationTemplate(
	ctx context.Context, deps NotificationTemplateDeps, tenantID string,
	key notificationports.TemplateKey, locale string,
) (*NotificationTemplateDetail, error) {
	builtin, _, err := resolveTemplate(ctx, deps, tenantID, key, locale)
	if err != nil {
		return nil, err
	}
	if _, err := deps.Repo.Delete(ctx, tenantID, key, locale); err != nil {
		return nil, err
	}
	return newTemplateDetail(key, locale, builtin, nil), nil
}

// PreviewNotificationTemplate は保存前の文面をサンプル値で描画する。送信も保存もしない。
// 省略フィールドは現在有効な文面で埋める。
func PreviewNotificationTemplate(
	ctx context.Context, deps NotificationTemplateDeps, tenantID string,
	key notificationports.TemplateKey, locale string, input NotificationTemplateInput,
) (*NotificationTemplatePreview, error) {
	builtin, override, err := resolveTemplate(ctx, deps, tenantID, key, locale)
	if err != nil {
		return nil, err
	}
	effective := effectiveDefinition(builtin, override)
	def := template.Definition{
		Subject:         firstNonBlank(input.Subject, effective.Subject),
		BodyText:        firstNonBlank(input.BodyText, effective.BodyText),
		BodyHTML:        firstNonBlank(input.BodyHTML, effective.BodyHTML),
		FromDisplayName: firstNonBlank(input.FromDisplayName, effective.FromDisplayName),
	}
	if err := template.ValidateDefinition(key, def); err != nil {
		return nil, wrapTemplateValidation(err)
	}
	rendered, err := template.Render(def, previewVars(ctx, deps, tenantID, key))
	if err != nil {
		return nil, errors.Join(ErrInvalidNotificationTemplate, err)
	}
	return &NotificationTemplatePreview{
		Subject: rendered.Subject, BodyText: rendered.Text, BodyHTML: rendered.HTML,
		FromDisplayName: rendered.FromDisplayName,
	}, nil
}

// SendTestNotification は現在有効な文面を操作者本人へ送る。宛先はリクエストで
// 指定できない。
func SendTestNotification(
	ctx context.Context, deps NotificationTemplateDeps, tenantID string,
	key notificationports.TemplateKey, locale string, actor TestNotificationActor,
) (*TestNotificationResult, error) {
	if _, _, err := resolveTemplate(ctx, deps, tenantID, key, locale); err != nil {
		return nil, err
	}
	to := strings.TrimSpace(actor.Email)
	if to == "" {
		return nil, ErrTestNotificationRecipient
	}
	vars := recipientVars(key)
	vars["user_display_name"] = firstNonBlank(actor.DisplayName, to)
	delivered := deps.Notifier.Notify(ctx, notificationports.Notification{
		TenantID: tenantID, To: to, Key: key,
		// 編集中の locale をそのまま使う。テナント既定やシステム既定に落ちると
		// 「編集した locale とは違う文面」が届き、確認の意味が消える。
		RecipientLocale: locale,
		Vars:            vars,
	})
	return &TestNotificationResult{Delivered: delivered, To: to}, nil
}

// resolveTemplate はカタログの組込み既定と (あれば) テナント上書きを返す。カタログに
// 無い key / locale はここで弾く。
func resolveTemplate(
	ctx context.Context, deps NotificationTemplateDeps, tenantID string,
	key notificationports.TemplateKey, locale string,
) (template.Definition, *notificationports.TemplateOverride, error) {
	builtin, err := template.Builtin(key, locale)
	if err != nil {
		return template.Definition{}, nil, errors.Join(ErrUnknownNotificationTemplate, err)
	}
	override, err := deps.Repo.FindByKey(ctx, tenantID, key, locale)
	if err != nil {
		return template.Definition{}, nil, err
	}
	return builtin, override, nil
}

func effectiveDefinition(builtin template.Definition, override *notificationports.TemplateOverride) template.Definition {
	if override == nil {
		return builtin
	}
	return template.Definition{
		Subject: override.Subject, BodyText: override.BodyText,
		BodyHTML: override.BodyHTML, FromDisplayName: override.FromDisplayName,
	}
}

func newTemplateDetail(
	key notificationports.TemplateKey, locale string,
	builtin template.Definition, override *notificationports.TemplateOverride,
) *NotificationTemplateDetail {
	effective := effectiveDefinition(builtin, override)
	detail := &NotificationTemplateDetail{
		Key: key, Locale: locale, Customized: override != nil,
		Subject: effective.Subject, BodyText: effective.BodyText, BodyHTML: effective.BodyHTML,
		FromDisplayName: effective.FromDisplayName,
		DefaultSubject:  builtin.Subject, DefaultBodyText: builtin.BodyText, DefaultBodyHTML: builtin.BodyHTML,
		Placeholders: template.Placeholders(key),
	}
	if override != nil {
		updatedAt := override.UpdatedAt
		detail.UpdatedAt = &updatedAt
	}
	return detail
}

// previewVars はサンプル値にテナントの実際の製品名 / 表示名を重ねる。実データ (実在の
// 利用者名やトークン) は入れず、ブランディングだけ本物にすることで「実際に届く見た目」に
// 近づける。
func previewVars(ctx context.Context, deps NotificationTemplateDeps, tenantID string, key notificationports.TemplateKey) map[string]string {
	vars := template.SampleVars(key)
	if deps.Tenant == nil {
		return vars
	}
	settings, err := deps.Tenant.NotificationSettings(ctx, tenantID)
	if err != nil {
		return vars
	}
	if settings.ProductName != "" {
		vars["product_name"] = settings.ProductName
	}
	if settings.TenantDisplayName != "" {
		vars["tenant_display_name"] = settings.TenantDisplayName
	}
	return vars
}

// recipientVars はテスト送信で使う差し込み値。product_name / tenant_display_name は
// Notifier がテナントから補うので、サンプル値からは外して実際の値を使わせる。
func recipientVars(key notificationports.TemplateKey) map[string]string {
	vars := map[string]string{}
	maps.Copy(vars, template.SampleVars(key))
	delete(vars, "product_name")
	delete(vars, "tenant_display_name")
	return vars
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// TenantNotificationSource は shared/notification の port を Tenancy の repository で
// 満たす。テナントが存在しない場合も空の設定を返し、通知そのものは
// 止めない (呼び出し側がシステム既定へ落とす)。
type TenantNotificationSource struct {
	TenantRepo   tenantports.TenantRepository
	BrandingRepo tenantports.TenantBrandingRepository
	TemplateRepo tenantports.NotificationTemplateRepository
}

func (s TenantNotificationSource) NotificationSettings(
	ctx context.Context, tenantID string,
) (notificationports.TenantNotificationSettings, error) {
	settings := notificationports.TenantNotificationSettings{}
	if s.TenantRepo != nil {
		tenant, err := s.TenantRepo.FindByID(ctx, tenantID)
		if err != nil {
			return settings, fmt.Errorf("resolve tenant for notification: %w", err)
		}
		if tenant != nil {
			settings.TenantDisplayName = tenant.DisplayName
			if tenant.DefaultLocale != nil {
				settings.DefaultLocale = *tenant.DefaultLocale
			}
		}
	}
	if s.BrandingRepo != nil {
		branding, err := s.BrandingRepo.FindByTenant(ctx, tenantID)
		if err != nil {
			return settings, fmt.Errorf("resolve branding for notification: %w", err)
		}
		if branding != nil {
			settings.ProductName = branding.ProductName
		}
	}
	return settings, nil
}

func (s TenantNotificationSource) FindTemplateOverride(
	ctx context.Context, tenantID string, key notificationports.TemplateKey, locale string,
) (*notificationports.TemplateOverride, error) {
	if s.TemplateRepo == nil {
		return nil, nil
	}
	return s.TemplateRepo.FindByKey(ctx, tenantID, key, locale)
}
