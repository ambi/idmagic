// /api/account/v1/notification_preferences — 本人によるセキュリティ通知の受信設定
// (wi-90)。取得は認証済みセッションで読めるが、更新は通知を止める操作であり、乗っ取りの
// 直後に最初に行われる操作でもあるため、直近のステップアップ再認証を要求する。
package handlers_http

import (
	"errors"
	"net/http"
	"time"

	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	"github.com/ambi/idmagic/backend/authentication/securitynotification/domain"
	securitynotificationusecases "github.com/ambi/idmagic/backend/authentication/securitynotification/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type categoryPreferenceResponse struct {
	Category  string `json:"category"`
	Mandatory bool   `json:"mandatory"`
	Enabled   bool   `json:"enabled"`
}

type notificationPreferencesResponse struct {
	Categories []categoryPreferenceResponse `json:"categories"`
}

type updateNotificationPreferencesRequest struct {
	DisabledCategories []string `json:"disabled_categories"`
}

func HandleGetNotificationPreferences(d httpdeps.Deps, c *echo.Context) error {
	sub, err := httpdeps.RequireAuthenticatedSub(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	preferences, err := securitynotificationusecases.GetPreferences(
		c.Request().Context(), d.NotificationPreferenceDeps(), sub,
	)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, toResponse(preferences))
}

func HandleUpdateNotificationPreferences(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	// 通知を止めることは、乗っ取りに気づく手立てを外すことでもある。
	sub, err := httpdeps.RequireStepUpSub(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	var input updateNotificationPreferencesRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	disabled := make([]domain.Category, 0, len(input.DisabledCategories))
	for _, value := range input.DisabledCategories {
		disabled = append(disabled, domain.Category(value))
	}
	preferences, err := securitynotificationusecases.UpdatePreferences(
		c.Request().Context(), d.NotificationPreferenceDeps(), sub, disabled, time.Now().UTC(),
	)
	if err != nil {
		return writeNotificationPreferenceError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toResponse(preferences))
}

func toResponse(preferences []securitynotificationusecases.CategoryPreference) notificationPreferencesResponse {
	categories := make([]categoryPreferenceResponse, 0, len(preferences))
	for _, preference := range preferences {
		categories = append(categories, categoryPreferenceResponse{
			Category:  string(preference.Category),
			Mandatory: preference.Mandatory,
			Enabled:   preference.Enabled,
		})
	}
	return notificationPreferencesResponse{Categories: categories}
}

func writeNotificationPreferenceError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrMandatoryCategory):
		return support.WriteProblem(c, http.StatusBadRequest, "mandatory_notification_category",
			"This security notification cannot be turned off.")
	case errors.Is(err, domain.ErrUnknownCategory):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request",
			"The request names a notification category that does not exist.")
	default:
		return httpdeps.WriteAccountError(c, err)
	}
}
