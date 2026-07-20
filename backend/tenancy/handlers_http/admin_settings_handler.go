package handlers_http

import (
	"net/http"
	"slices"
	"time"

	"github.com/ambi/idmagic/backend/tenancy/domain"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"

	"github.com/labstack/echo/v5"
)

// requireTenantAdmin は actor.tenant_id を権限境界として、admin / system_admin の
// いずれかが actor.tenant_id に居る場合にだけ通す。AdminSettings* permissions の
// allow_when と一致する。
func (d Deps) requireTenantAdmin(c *echo.Context) (*userdomain.User, error) {
	actor, err := d.ResolveAdminActor(c)
	if err != nil {
		return nil, err
	}
	if actor.TenantID != support.RequestTenantID(c) {
		return nil, support.ErrAdminAccessDenied
	}
	if !slices.Contains(actor.Roles, "admin") && !slices.Contains(actor.Roles, "system_admin") {
		return nil, support.ErrAdminAccessDenied
	}
	return actor, nil
}

type AdminSettingsResponse struct {
	TenantID               string                         `json:"tenant_id"`
	Realm                  string                         `json:"realm"`
	DisplayName            string                         `json:"display_name"`
	PasswordPolicyOverride *domain.PasswordPolicyOverride `json:"password_policy_override,omitempty"`
	PasswordPolicyDefaults passwordPolicyDefaults         `json:"password_policy_defaults"`
	Quota                  *domain.TenantQuota            `json:"quota,omitempty"`
	Usage                  *domain.TenantUsage            `json:"usage,omitempty"`
}

type passwordPolicyDefaults struct {
	MinLength    int `json:"min_length"`
	MaxLength    int `json:"max_length"`
	HistoryDepth int `json:"history_depth"`
}

type adminSettingsUpdateRequest struct {
	DisplayName            *string                        `json:"display_name,omitempty"`
	PasswordPolicyOverride *domain.PasswordPolicyOverride `json:"password_policy_override,omitempty"`
}

func (d Deps) handleGetAdminSettings(c *echo.Context) error {
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	tenant, err := d.TenantRepo.FindByID(c.Request().Context(), actor.TenantID)
	if err != nil {
		return err
	}
	if tenant == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "tenant_not_found", "テナントが存在しません")
	}
	if d.QuotaRepo != nil {
		if q, err := d.QuotaRepo.GetQuota(c.Request().Context(), tenant.ID); err == nil {
			tenant.Quota = q
		}
		if u, err := d.QuotaRepo.GetUsage(c.Request().Context(), tenant.ID); err == nil {
			tenant.Usage = u
		}
	}
	resp := d.toAdminSettingsResponse(tenant)
	resp.Quota = tenant.Quota
	resp.Usage = tenant.Usage
	return support.NoStoreJSON(c, http.StatusOK, resp)
}

func (d Deps) handleUpdateAdminSettings(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input adminSettingsUpdateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	now := time.Now().UTC()
	tenant, err := tenantusecases.Update(
		c.Request().Context(), d.TenantRepo, actor.TenantID,
		tenantusecases.UpdateInput{
			DisplayName:            input.DisplayName,
			PasswordPolicyOverride: input.PasswordPolicyOverride,
		},
		d.tenantPolicyFloor(),
		now,
	)
	if err != nil {
		return d.writeTenantError(c, err)
	}
	if d.Emit != nil {
		d.Emit(&domain.TenantUpdated{
			At: now, ActorUserID: actor.ID, TenantID: tenant.ID,
			ChangedFields: adminSettingsChangedFields(input),
		})
	}
	return support.NoStoreJSON(c, http.StatusOK, d.toAdminSettingsResponse(tenant))
}

func (d Deps) toAdminSettingsResponse(t *domain.Tenant) AdminSettingsResponse {
	floor := d.tenantPolicyFloor()
	return AdminSettingsResponse{
		TenantID:               t.ID,
		Realm:                  t.Realm,
		DisplayName:            t.DisplayName,
		PasswordPolicyOverride: t.PasswordPolicyOverride,
		PasswordPolicyDefaults: passwordPolicyDefaults{
			MinLength:    floor.MinLength,
			MaxLength:    floor.MaxLength,
			HistoryDepth: floor.HistoryDepth,
		},
		Quota: t.Quota,
		Usage: t.Usage,
	}
}

func adminSettingsChangedFields(input adminSettingsUpdateRequest) []string {
	fields := []string{}
	if input.DisplayName != nil {
		fields = append(fields, "display_name")
	}
	if input.PasswordPolicyOverride != nil {
		fields = append(fields, "password_policy_override")
	}
	return fields
}
