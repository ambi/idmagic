// /api/account/v1/trusted_devices — 本人の信頼済みデバイスの一覧と失効 (wi-91)。
// 一覧は認証済みセッションで読めるが、失効は第二要素を条件付きで飛ばす能力を取り消す
// 機微操作なので、直近のステップアップ再認証を要求する。
package handlers_http

import (
	"errors"
	"net/http"
	"time"

	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	"github.com/ambi/idmagic/backend/authentication/trusteddevice/domain"
	trusteddeviceusecases "github.com/ambi/idmagic/backend/authentication/trusteddevice/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

// trustedDeviceCookie は oauth2 の login 経路が発行する cookie 名と同じでなければならない。
// 一覧で「現在の端末」を示すためにここでも読む。
const trustedDeviceCookie = "idmagic_trusted_device"

type accountTrustedDeviceResponse struct {
	ID         string    `json:"id"`
	Current    bool      `json:"current"`
	Label      string    `json:"label,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// currentSelector は提示された cookie の selector を返す。verifier は照合しない:
// ここでの用途は一覧の current 表示だけで、selector は秘密ではないからである。
func currentSelector(c *echo.Context) string {
	value := ""
	if cookie, err := c.Request().Cookie(support.TenantCookieName(c, trustedDeviceCookie)); err == nil {
		value = cookie.Value
	} else if cookie, err := c.Request().Cookie(trustedDeviceCookie); err == nil {
		value = cookie.Value
	}
	if value == "" {
		return ""
	}
	selector, _, ok := domain.ParseCookie(value)
	if !ok {
		return ""
	}
	return selector
}

// trustedDeviceMaxAge はテナントが明示した有効期間を秒で返す。0 は機能無効で、UI は
// 記憶の導線を出さない。
func trustedDeviceMaxAgeSeconds(d httpdeps.Deps, c *echo.Context) int {
	if d.TenantRepo == nil || d.TrustedDeviceRepo == nil {
		return 0
	}
	tenant, err := d.TenantRepo.FindByID(c.Request().Context(), support.RequestTenantID(c))
	if err != nil || tenant == nil {
		return 0
	}
	return int(tenant.EffectiveTrustedDeviceMaxAge().Seconds())
}

func HandleListTrustedDevices(d httpdeps.Deps, c *echo.Context) error {
	sub, err := httpdeps.RequireAuthenticatedSub(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	devices, err := trusteddeviceusecases.ListActive(
		c.Request().Context(), d.TrustedDeviceDeps(), support.RequestTenantID(c), sub, time.Now().UTC(),
	)
	if err != nil {
		return err
	}
	selector := currentSelector(c)
	out := make([]accountTrustedDeviceResponse, len(devices))
	for i, device := range devices {
		out[i] = accountTrustedDeviceResponse{
			ID: device.ID, Current: selector != "" && device.Selector == selector,
			Label: device.Label, CreatedAt: device.CreatedAt,
			LastUsedAt: device.LastUsedAt, ExpiresAt: device.ExpiresAt,
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"devices":         out,
		"max_age_seconds": trustedDeviceMaxAgeSeconds(d, c),
	})
}

func HandleRevokeTrustedDevice(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	// 記憶の取り消しは認証強度に関わる機微操作。step-up 再認証を要求する。
	sub, err := httpdeps.RequireStepUpSub(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	if err := trusteddeviceusecases.RevokeOne(
		c.Request().Context(), d.TrustedDeviceDeps(), support.RequestTenantID(c), sub,
		c.Param("id"), spec.TrustedDeviceSelfRevoke, time.Now().UTC(),
	); err != nil {
		return writeTrustedDeviceError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func HandleRevokeAllTrustedDevices(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := httpdeps.RequireStepUpSub(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	if err := d.RevokeTrustedDevices(
		c.Request().Context(), support.RequestTenantID(c), sub, spec.TrustedDeviceSelfRevoke,
	); err != nil {
		return err
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func writeTrustedDeviceError(c *echo.Context, err error) error {
	if errors.Is(err, trusteddeviceusecases.ErrTrustedDeviceNotFound) {
		return support.WriteProblem(
			c, http.StatusNotFound, "trusted_device_not_found", "The trusted device does not exist.",
		)
	}
	return httpdeps.WriteAccountError(c, err)
}
