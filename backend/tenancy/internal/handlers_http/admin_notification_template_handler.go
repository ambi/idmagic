package handlers_http

import (
	"errors"
	"net/http"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/internal/usecases"

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
// 指定するフィールドは意図的に存在しない。
type notificationTemplateRequest struct {
	Subject         string `json:"subject"`
	BodyText        string `json:"body_text"`
	BodyHTML        string `json:"body_html"`
	FromDisplayName string `json:"from_display_name"`
}

func (r notificationTemplateRequest) toInput() tenancyusecases.NotificationTemplateInput {
	return tenancyusecases.NotificationTemplateInput{
		Subject: r.Subject, BodyText: r.BodyText, BodyHTML: r.BodyHTML,
		FromDisplayName: r.FromDisplayName,
	}
}

func (d Deps) notificationTemplateDeps() tenancyusecases.NotificationTemplateDeps {
	return tenancyusecases.NotificationTemplateDeps{
		Repo: d.NotificationTemplateRepo,
		Tenant: tenancyusecases.TenantNotificationSource{
			TenantRepo: d.TenantRepo, BrandingRepo: d.BrandingRepo, TemplateRepo: d.NotificationTemplateRepo,
		},
		Notifier: d.Notifier,
	}
}

func (d Deps) writeNotificationTemplateError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, tenancyusecases.ErrUnknownNotificationTemplate):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request",
			"The notification template key or locale does not exist.")
	case errors.Is(err, tenancyusecases.ErrInvalidNotificationTemplate):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_notification_template",
			"The template must set a subject, a text body, and an HTML body, and may only use the listed placeholders.")
	case errors.Is(err, tenancyusecases.ErrTestNotificationRecipient):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request",
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
	summaries, err := tenancyusecases.ListNotificationTemplates(c.Request().Context(), d.notificationTemplateDeps(), actor.TenantID)
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
	detail, err := tenancyusecases.GetNotificationTemplate(c.Request().Context(), d.notificationTemplateDeps(),
		actor.TenantID, notificationTemplateKeyParam(c), c.Param("locale"))
	if err != nil {
		return d.writeNotificationTemplateError(c, err)
	}
	return support.ETagJSON(c, http.StatusOK, toNotificationTemplateDetailResponse(detail))
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
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	key, locale := notificationTemplateKeyParam(c), c.Param("locale")
	now := time.Now().UTC()
	condition, err := support.ReadIfMatch(c)
	if err != nil {
		return err
	}
	update := input.toInput()
	update.CheckCurrent = func(current *tenancyusecases.NotificationTemplateDetail) error {
		return condition.Check(toNotificationTemplateDetailResponse(current))
	}
	detail, err := tenancyusecases.UpdateNotificationTemplate(c.Request().Context(), d.notificationTemplateDeps(),
		actor.TenantID, key, locale, update, now)
	if err != nil {
		return d.writeNotificationTemplateError(c, err)
	}
	if d.Emit != nil {
		d.Emit(&tenancydomain.NotificationTemplateUpdated{
			At: now, TenantID: actor.TenantID, ActorUserID: actor.ID,
			TemplateKey: string(key), Locale: locale,
		})
	}
	return support.ETagJSON(c, http.StatusOK, toNotificationTemplateDetailResponse(detail))
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
	condition, err := support.ReadIfMatch(c)
	if err != nil {
		return err
	}
	detail, err := tenancyusecases.ResetNotificationTemplate(c.Request().Context(), d.notificationTemplateDeps(),
		actor.TenantID, key, locale, func(current *tenancyusecases.NotificationTemplateDetail) error {
			return condition.Check(toNotificationTemplateDetailResponse(current))
		})
	if err != nil {
		return d.writeNotificationTemplateError(c, err)
	}
	if d.Emit != nil {
		d.Emit(&tenancydomain.NotificationTemplateReset{
			At: time.Now().UTC(), TenantID: actor.TenantID, ActorUserID: actor.ID,
			TemplateKey: string(key), Locale: locale,
		})
	}
	return support.ETagJSON(c, http.StatusOK, toNotificationTemplateDetailResponse(detail))
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
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	preview, err := tenancyusecases.PreviewNotificationTemplate(c.Request().Context(), d.notificationTemplateDeps(),
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
// 作らない。
func (d Deps) handleSendTestNotification(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	// 認可が返す主体は宛先を持たないので、操作者本人の記録を引き直す。認可の後に記録が
	// 消えていれば、主体がいない場合と同じく拒否する。
	operator, err := d.TestNotificationActor(c.Request().Context(), actor.ID)
	if err != nil {
		return err
	}
	if operator == nil {
		return d.WriteAdminAccessError(c, support.ErrAdminAccessDenied)
	}
	recipient := tenancyusecases.TestNotificationActor{DisplayName: operator.DisplayName, Email: operator.Email}
	result, err := tenancyusecases.SendTestNotification(c.Request().Context(), d.notificationTemplateDeps(),
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

func toNotificationTemplateDetailResponse(detail *tenancyusecases.NotificationTemplateDetail) notificationTemplateDetailResponse {
	return notificationTemplateDetailResponse{
		TemplateKey: string(detail.Key), Locale: detail.Locale, Customized: detail.Customized,
		Subject: detail.Subject, BodyText: detail.BodyText, BodyHTML: detail.BodyHTML,
		FromDisplayName: detail.FromDisplayName,
		DefaultSubject:  detail.DefaultSubject, DefaultBodyText: detail.DefaultBodyText,
		DefaultBodyHTML: detail.DefaultBodyHTML,
		Placeholders:    detail.Placeholders, UpdatedAt: detail.UpdatedAt,
	}
}
