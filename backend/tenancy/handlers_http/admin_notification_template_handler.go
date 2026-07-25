package handlers_http

import (
	"errors"
	"net/http"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"

	"github.com/labstack/echo/v5"
)

type notificationTemplateSummaryResponse struct {
	TemplateKey string     `json:"template_key"`
	Locale      string     `json:"locale"`
	Customized  bool       `json:"customized"`
	Subject     string     `json:"subject"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type notificationTemplateListResponse struct {
	Templates        []notificationTemplateSummaryResponse `json:"templates"`
	SupportedLocales []string                              `json:"supported_locales"`
}

type notificationTemplateDetailResponse struct {
	TemplateKey     string     `json:"template_key"`
	Locale          string     `json:"locale"`
	Customized      bool       `json:"customized"`
	Subject         string     `json:"subject"`
	BodyText        string     `json:"body_text"`
	BodyHTML        string     `json:"body_html"`
	FromDisplayName string     `json:"from_display_name,omitempty"`
	DefaultSubject  string     `json:"default_subject"`
	DefaultBodyText string     `json:"default_body_text"`
	DefaultBodyHTML string     `json:"default_body_html"`
	Placeholders    []string   `json:"placeholders"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

type notificationTemplatePreviewResponse struct {
	Subject         string `json:"subject"`
	BodyText        string `json:"body_text"`
	BodyHTML        string `json:"body_html"`
	FromDisplayName string `json:"from_display_name,omitempty"`
}

type notificationTemplateTestSendResponse struct {
	Delivered bool   `json:"delivered"`
	To        string `json:"to"`
}

// notificationTemplateRequest は更新とプレビューで共用する入力。更新では 3 点セットが
// 必須で、プレビューでは省略が「現在有効な文面を使う」を意味する。テスト送信の宛先を
// 指定するフィールドは意図的に存在しない (ADR-142 決定 8)。
type notificationTemplateRequest struct {
	Subject         string `json:"subject"`
	BodyText        string `json:"body_text"`
	BodyHTML        string `json:"body_html"`
	FromDisplayName string `json:"from_display_name"`
}

func (r notificationTemplateRequest) toInput() tenantusecases.NotificationTemplateInput {
	return tenantusecases.NotificationTemplateInput{
		Subject: r.Subject, BodyText: r.BodyText, BodyHTML: r.BodyHTML,
		FromDisplayName: r.FromDisplayName,
	}
}

func (d Deps) notificationTemplateDeps() tenantusecases.NotificationTemplateDeps {
	return tenantusecases.NotificationTemplateDeps{
		Repo: d.NotificationTemplateRepo,
		Tenant: tenantusecases.TenantNotificationSource{
			TenantRepo: d.TenantRepo, BrandingRepo: d.BrandingRepo, TemplateRepo: d.NotificationTemplateRepo,
		},
		Notifier: d.Notifier,
	}
}

func (d Deps) writeNotificationTemplateError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, tenantusecases.ErrUnknownNotificationTemplate):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request",
			"The notification template key or locale does not exist.")
	case errors.Is(err, tenantusecases.ErrInvalidNotificationTemplate):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_notification_template",
			"The template must set a subject, a text body, and an HTML body, and may only use the listed placeholders.")
	case errors.Is(err, tenantusecases.ErrTestNotificationRecipient):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request",
			"A test message is only sent to your own verified email address, and your account has none.")
	default:
		return err
	}
}

func (d Deps) handleListNotificationTemplates(c *echo.Context) error {
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	summaries, err := tenantusecases.ListNotificationTemplates(c.Request().Context(), d.notificationTemplateDeps(), actor.TenantID)
	if err != nil {
		return d.writeNotificationTemplateError(c, err)
	}
	response := notificationTemplateListResponse{
		Templates:        make([]notificationTemplateSummaryResponse, 0, len(summaries)),
		SupportedLocales: template.SupportedLocales(),
	}
	for _, summary := range summaries {
		response.Templates = append(response.Templates, notificationTemplateSummaryResponse{
			TemplateKey: string(summary.Key), Locale: summary.Locale,
			Customized: summary.Customized, Subject: summary.Subject, UpdatedAt: summary.UpdatedAt,
		})
	}
	return support.NoStoreJSON(c, http.StatusOK, response)
}

func (d Deps) handleGetNotificationTemplate(c *echo.Context) error {
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	detail, err := tenantusecases.GetNotificationTemplate(c.Request().Context(), d.notificationTemplateDeps(),
		actor.TenantID, notificationTemplateKeyParam(c), c.Param("locale"))
	if err != nil {
		return d.writeNotificationTemplateError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toNotificationTemplateDetailResponse(detail))
}

func (d Deps) handleUpdateNotificationTemplate(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input notificationTemplateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	key, locale := notificationTemplateKeyParam(c), c.Param("locale")
	now := time.Now().UTC()
	detail, err := tenantusecases.UpdateNotificationTemplate(c.Request().Context(), d.notificationTemplateDeps(),
		actor.TenantID, key, locale, input.toInput(), now)
	if err != nil {
		return d.writeNotificationTemplateError(c, err)
	}
	if d.Emit != nil {
		d.Emit(&domain.NotificationTemplateUpdated{
			At: now, TenantID: actor.TenantID, ActorUserID: actor.ID,
			TemplateKey: string(key), Locale: locale,
		})
	}
	return support.NoStoreJSON(c, http.StatusOK, toNotificationTemplateDetailResponse(detail))
}

func (d Deps) handleResetNotificationTemplate(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	key, locale := notificationTemplateKeyParam(c), c.Param("locale")
	detail, err := tenantusecases.ResetNotificationTemplate(c.Request().Context(), d.notificationTemplateDeps(),
		actor.TenantID, key, locale)
	if err != nil {
		return d.writeNotificationTemplateError(c, err)
	}
	if d.Emit != nil {
		d.Emit(&domain.NotificationTemplateReset{
			At: time.Now().UTC(), TenantID: actor.TenantID, ActorUserID: actor.ID,
			TemplateKey: string(key), Locale: locale,
		})
	}
	return support.NoStoreJSON(c, http.StatusOK, toNotificationTemplateDetailResponse(detail))
}

func (d Deps) handlePreviewNotificationTemplate(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input notificationTemplateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	preview, err := tenantusecases.PreviewNotificationTemplate(c.Request().Context(), d.notificationTemplateDeps(),
		actor.TenantID, notificationTemplateKeyParam(c), c.Param("locale"), input.toInput())
	if err != nil {
		return d.writeNotificationTemplateError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, notificationTemplatePreviewResponse{
		Subject: preview.Subject, BodyText: preview.BodyText, BodyHTML: preview.BodyHTML,
		FromDisplayName: preview.FromDisplayName,
	})
}

// handleSendTestNotification は宛先をリクエストから一切読まない。操作者本人の検証済み
// アドレスに固定することで、管理者権限が任意宛先メール送信の踏み台になる経路を構造的に
// 作らない (ADR-142 決定 8)。
func (d Deps) handleSendTestNotification(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	recipient := tenantusecases.TestNotificationActor{DisplayName: actor.DisplayName()}
	if actor.Email != nil && actor.EmailVerified {
		recipient.Email = *actor.Email
	}
	result, err := tenantusecases.SendTestNotification(c.Request().Context(), d.notificationTemplateDeps(),
		actor.TenantID, notificationTemplateKeyParam(c), c.Param("locale"), recipient)
	if err != nil {
		return d.writeNotificationTemplateError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, notificationTemplateTestSendResponse{
		Delivered: result.Delivered, To: result.To,
	})
}

func notificationTemplateKeyParam(c *echo.Context) notificationports.TemplateKey {
	return notificationports.TemplateKey(c.Param("template_key"))
}

func toNotificationTemplateDetailResponse(detail *tenantusecases.NotificationTemplateDetail) notificationTemplateDetailResponse {
	return notificationTemplateDetailResponse{
		TemplateKey: string(detail.Key), Locale: detail.Locale, Customized: detail.Customized,
		Subject: detail.Subject, BodyText: detail.BodyText, BodyHTML: detail.BodyHTML,
		FromDisplayName: detail.FromDisplayName,
		DefaultSubject:  detail.DefaultSubject, DefaultBodyText: detail.DefaultBodyText,
		DefaultBodyHTML: detail.DefaultBodyHTML,
		Placeholders:    detail.Placeholders, UpdatedAt: detail.UpdatedAt,
	}
}
