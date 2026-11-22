package handlers_http

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/internal/usecases"

	"github.com/labstack/echo/v5"
)

type tenantCreateRequest struct {
	Realm       string `json:"realm"`
	DisplayName string `json:"display_name"`
}

type tenantUpdateRequest struct {
	DisplayName            *string                               `json:"display_name,omitempty"`
	PasswordPolicyOverride *tenancydomain.PasswordPolicyOverride `json:"password_policy_override,omitempty"`
	MaxDelegationDepth     *int                                  `json:"max_delegation_depth,omitempty"`
	// TrustedDeviceMaxAgeSeconds は信頼済みデバイスの有効期間 (秒)。0 は機能無効。
	TrustedDeviceMaxAgeSeconds *int `json:"trusted_device_max_age_seconds,omitempty"`
}

type tenantEndpointStyleRequest struct {
	EndpointStyle tenancydomain.TenantEndpointStyle `json:"endpoint_style"`
}

func (d Deps) handleListTenants(c *echo.Context) error {
	if _, err := d.RequireControlPlaneUser(c); err != nil {
		return d.WriteControlPlaneAccessError(c, err)
	}
	tenants, err := d.TenantRepo.FindAll(c.Request().Context())
	if err != nil {
		return err
	}
	if d.QuotaRepo != nil {
		for i := range tenants {
			if q, err := d.QuotaRepo.GetQuota(c.Request().Context(), tenants[i].ID); err == nil {
				tenants[i].Quota = q
			}
			if u, err := d.QuotaRepo.GetUsage(c.Request().Context(), tenants[i].ID); err == nil {
				tenants[i].Usage = u
			}
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"tenants": tenants})
}

func (d Deps) handleGetTenant(c *echo.Context) error {
	if _, err := d.RequireControlPlaneUser(c); err != nil {
		return d.WriteControlPlaneAccessError(c, err)
	}
	tenant, err := d.TenantRepo.FindByRealm(c.Request().Context(), c.Param("target_tenant_id"))
	if err != nil {
		return err
	}
	if tenant == nil {
		return support.WriteProblem(c, http.StatusNotFound, "tenant_not_found", "The tenant does not exist.")
	}
	if d.QuotaRepo != nil {
		if q, err := d.QuotaRepo.GetQuota(c.Request().Context(), tenant.ID); err == nil {
			tenant.Quota = q
		}
		if u, err := d.QuotaRepo.GetUsage(c.Request().Context(), tenant.ID); err == nil {
			tenant.Usage = u
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, tenant)
}

// resolveTenantByRealm は admin API path の realm slug を UUID キーのテナントへ解決する。
// usecase は UUID キーで扱うため、handler で realm→UUID を写像する。
func (d Deps) resolveTenantByRealm(c *echo.Context, realm string) (*tenancydomain.Tenant, error) {
	tenant, err := d.TenantRepo.FindByRealm(c.Request().Context(), realm)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, tenancyusecases.ErrTenantNotFound
	}
	return tenant, nil
}

func (d Deps) handleCreateTenant(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireControlPlaneChange(c)
	if err != nil {
		return d.WriteControlPlaneAccessError(c, err)
	}
	var input tenantCreateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	now := time.Now().UTC()
	tenant, err := tenancyusecases.Create(
		c.Request().Context(), d.TenantRepo, input.Realm, input.DisplayName, now,
	)
	if err != nil {
		return d.writeTenantError(c, err)
	}
	if d.Emit != nil {
		d.Emit(&tenancydomain.TenantCreated{At: now, ActorUserID: actor.ID, TenantID: tenant.ID})
	}
	return support.NoStoreJSON(c, http.StatusCreated, tenant)
}

func (d Deps) handleUpdateTenant(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireControlPlaneChange(c)
	if err != nil {
		return d.WriteControlPlaneAccessError(c, err)
	}
	var input tenantUpdateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	target, err := d.resolveTenantByRealm(c, c.Param("target_tenant_id"))
	if err != nil {
		return d.writeTenantError(c, err)
	}
	now := time.Now().UTC()
	tenant, err := tenancyusecases.Update(
		c.Request().Context(), d.TenantRepo, target.ID,
		tenancyusecases.UpdateInput{
			DisplayName:                input.DisplayName,
			PasswordPolicyOverride:     input.PasswordPolicyOverride,
			MaxDelegationDepth:         input.MaxDelegationDepth,
			TrustedDeviceMaxAgeSeconds: input.TrustedDeviceMaxAgeSeconds,
		},
		d.tenantPolicyFloor(),
		now,
	)
	if err != nil {
		return d.writeTenantError(c, err)
	}
	if d.Emit != nil {
		d.Emit(&tenancydomain.TenantUpdated{
			At: now, ActorUserID: actor.ID, TenantID: tenant.ID,
			ChangedFields: tenantChangedFields(input),
		})
	}
	return support.NoStoreJSON(c, http.StatusOK, tenant)
}

// handleSetTenantEndpointStyle は issuer / WebAuthn RP ID / cookie scope を変えるため、
// 通常の PATCH から切り離した endpoint_style の明示的な切替操作。
func (d Deps) handleSetTenantEndpointStyle(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireControlPlaneChange(c)
	if err != nil {
		return d.WriteControlPlaneAccessError(c, err)
	}
	var input tenantEndpointStyleRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	target, err := d.resolveTenantByRealm(c, c.Param("target_tenant_id"))
	if err != nil {
		return d.writeTenantError(c, err)
	}
	now := time.Now().UTC()
	tenant, err := tenancyusecases.SetEndpointStyle(
		c.Request().Context(), d.TenantRepo, target.ID, input.EndpointStyle, d.TenantBaseDomain, now,
	)
	if err != nil {
		return d.writeTenantError(c, err)
	}
	if d.Emit != nil {
		d.Emit(&tenancydomain.TenantEndpointStyleChanged{
			At: now, ActorUserID: actor.ID, TenantID: tenant.ID,
			PreviousEndpointStyle: target.EffectiveEndpointStyle(), EndpointStyle: tenant.EffectiveEndpointStyle(),
		})
	}
	return c.NoContent(http.StatusNoContent)
}

func tenantChangedFields(input tenantUpdateRequest) []string {
	fields := []string{}
	if input.DisplayName != nil {
		fields = append(fields, "display_name")
	}
	if input.PasswordPolicyOverride != nil {
		fields = append(fields, "password_policy_override")
	}
	if input.MaxDelegationDepth != nil {
		fields = append(fields, "max_delegation_depth")
	}
	if input.TrustedDeviceMaxAgeSeconds != nil {
		fields = append(fields, "trusted_device_max_age_seconds")
	}
	return fields
}

func (d Deps) tenantPolicyFloor() tenancyusecases.PolicyFloor {
	return tenancyusecases.PolicyFloor{
		MinLength:    tenancydomain.PasswordPolicyMinLength,
		MaxLength:    tenancydomain.PasswordPolicyMaxLength,
		HistoryDepth: tenancydomain.PasswordPolicyHistoryDepth,
	}
}

func (d Deps) handleDisableTenant(c *echo.Context) error {
	return d.handleSetTenantDisabled(c, true)
}

func (d Deps) handleEnableTenant(c *echo.Context) error {
	return d.handleSetTenantDisabled(c, false)
}

func (d Deps) handleSetTenantDisabled(c *echo.Context, disabled bool) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireControlPlaneChange(c)
	if err != nil {
		return d.WriteControlPlaneAccessError(c, err)
	}
	target, err := d.resolveTenantByRealm(c, c.Param("target_tenant_id"))
	if err != nil {
		return d.writeTenantError(c, err)
	}
	now := time.Now().UTC()
	tenant, err := tenancyusecases.SetDisabled(
		c.Request().Context(), d.TenantRepo, target.ID, disabled, now,
	)
	if err != nil {
		return d.writeTenantError(c, err)
	}
	if d.Emit != nil {
		if disabled {
			d.Emit(&tenancydomain.TenantDisabled{At: now, ActorUserID: actor.ID, TenantID: tenant.ID})
		} else {
			d.Emit(&tenancydomain.TenantEnabled{At: now, ActorUserID: actor.ID, TenantID: tenant.ID})
		}
	}
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) writeTenantError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, tenancyusecases.ErrTenantNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "tenant_not_found", "The tenant does not exist.")
	case errors.Is(err, tenancyusecases.ErrTenantConflict):
		return support.WriteProblem(c, http.StatusConflict, "tenant_conflict", "The tenant ID is already in use.")
	case errors.Is(err, tenancyusecases.ErrInvalidTenantID),
		errors.Is(err, tenancyusecases.ErrDisplayNameEmpty),
		errors.Is(err, tenancyusecases.ErrDefaultTenant),
		errors.Is(err, tenancydomain.ErrEndpointStyleUnknown),
		errors.Is(err, tenancydomain.ErrSubdomainStyleNoBase):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, tenancyusecases.ErrUnsupportedDefaultLocale):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request",
			"The default locale has no bundled translation for notification emails.")
	case errors.Is(err, tenancyusecases.ErrPolicyOverrideWeaker):
		floor := d.tenantPolicyFloor()
		message := fmt.Sprintf(
			"A tenant policy override cannot be weaker than the product baseline "+
				"(min_length≥%d / max_length≤%d / history_depth≥%d), "+
				"max_age_days must be between %d and %d when set, "+
				"max_delegation_depth must be between 1 and %d, "+
				"and trusted_device_max_age_seconds must be between 0 and %d",
			floor.MinLength, floor.MaxLength, floor.HistoryDepth,
			tenancyusecases.PasswordMaxAgeDaysFloor, tenancyusecases.PasswordMaxAgeDaysCeiling,
			tenancydomain.DefaultMaxDelegationDepth,
			tenancydomain.TrustedDeviceMaxAgeCeilingSeconds,
		)
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "policy_override_weaker", message)
	default:
		return err
	}
}

type tenantQuotaUpdateRequest struct {
	Users                *int `json:"users,omitempty"`
	Groups               *int `json:"groups,omitempty"`
	Agents               *int `json:"agents,omitempty"`
	Applications         *int `json:"applications,omitempty"`
	OAuth2Clients        *int `json:"oauth2_clients,omitempty"`
	ActiveSessions       *int `json:"active_sessions,omitempty"`
	Consents             *int `json:"consents,omitempty"`
	ActiveJobs           *int `json:"active_jobs,omitempty"`
	SsfStreams           *int `json:"ssf_streams,omitempty"`
	AuditEventsRetained  *int `json:"audit_events_retained,omitempty"`
	ExportArtifactsBytes *int `json:"export_artifacts_bytes,omitempty"`
}

// handleUpdateTenantQuota はクォータ上限を書き換える制御面の状態変更なので、
// 兄弟の経路と同じく Origin と CSRF トークンの検証を先頭に置く。検証を対象の解決、
// 本文のデコード、保存より前に置くのは、周囲資格情報を持たない呼び出し元に
// 対象の存在や本文の妥当性を観測させないためである。
func (d Deps) handleUpdateTenantQuota(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	ctx := c.Request().Context()
	actor, err := d.RequireControlPlaneChange(c)
	if err != nil {
		return d.WriteControlPlaneAccessError(c, err)
	}
	target, err := d.resolveTenantByRealm(c, c.Param("target_tenant_id"))
	if err != nil {
		return d.writeTenantError(c, err)
	}

	var req tenantQuotaUpdateRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, 400, "invalid_request", "invalid request body: "+err.Error())
	}

	quota := &tenancydomain.TenantQuota{
		Users:                req.Users,
		Groups:               req.Groups,
		Agents:               req.Agents,
		Applications:         req.Applications,
		OAuth2Clients:        req.OAuth2Clients,
		ActiveSessions:       req.ActiveSessions,
		Consents:             req.Consents,
		ActiveJobs:           req.ActiveJobs,
		SsfStreams:           req.SsfStreams,
		AuditEventsRetained:  req.AuditEventsRetained,
		ExportArtifactsBytes: req.ExportArtifactsBytes,
	}
	if quota.HasNegativeLimit() {
		return support.WriteProblem(c, 400, "invalid_request", "quota limits must not be negative")
	}

	if d.QuotaRepo != nil {
		if err := d.QuotaRepo.SetQuota(ctx, target.ID, quota); err != nil {
			return support.WriteServerError(c, err)
		}
	}
	if d.Emit != nil {
		d.Emit(&tenancydomain.TenantQuotaUpdated{
			At: time.Now().UTC(), ActorUserID: actor.ID, TenantID: target.ID,
			ChangedFields: quotaChangedFields(req),
		})
	}

	return support.NoStoreJSON(c, http.StatusOK, quota)
}

// quotaChangedFields は要求に含まれたリソースの名前を、要求の項目の宣言順に返す。
func quotaChangedFields(req tenantQuotaUpdateRequest) []string {
	fields := []string{}
	for _, field := range []struct {
		name  string
		limit *int
	}{
		{"users", req.Users},
		{"groups", req.Groups},
		{"agents", req.Agents},
		{"applications", req.Applications},
		{"oauth2_clients", req.OAuth2Clients},
		{"active_sessions", req.ActiveSessions},
		{"consents", req.Consents},
		{"active_jobs", req.ActiveJobs},
		{"ssf_streams", req.SsfStreams},
		{"audit_events_retained", req.AuditEventsRetained},
		{"export_artifacts_bytes", req.ExportArtifactsBytes},
	} {
		if field.limit != nil {
			fields = append(fields, field.name)
		}
	}
	return fields
}
