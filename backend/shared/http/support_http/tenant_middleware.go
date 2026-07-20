package support_http

import (
	"net/http"
	"strings"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/tenancy"

	"github.com/labstack/echo/v5"
)

func (d Deps) ResolveDefaultTenant(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		return d.resolveTenant(c, next, tenancydomain.DefaultRealm, true)
	}
}

func (d Deps) ResolvePathTenant(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		return d.resolveTenant(c, next, c.Param("tenant_id"), false)
	}
}

// ResolveControlPlaneTenant は固定で default realm のテナントを resolve し、URL prefix
// /realms/default を ctx に載せる (cookie path 整合のため)。/realms/default/admin/tenants
// 等の control-plane 経路で使う。
func (d Deps) ResolveControlPlaneTenant(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		return d.resolveTenant(c, next, tenancydomain.DefaultRealm, false)
	}
}

// resolveTenant は URL の realm slug からテナントを解決し、issuer / URL prefix を realm
// 語彙で組み立てて ctx に載せる (ADR-085)。内部キーは tenant.ID (UUID)。
func (d Deps) resolveTenant(c *echo.Context, next echo.HandlerFunc, realm string, bare bool) error {
	urlPrefix := ""
	if !bare {
		urlPrefix = "/realms/" + realm
	}
	if d.TenantRepo == nil {
		if realm != tenancydomain.DefaultRealm {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "tenant_not_found"})
		}
		tenant := &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, Status: tenancydomain.TenantStatusActive}
		issuer := tenantIssuer(d.Issuer, tenant.Realm)
		if bare && d.LegacyBareIssuer {
			issuer = strings.TrimSuffix(d.Issuer, "/")
		}
		c.SetRequest(c.Request().WithContext(tenancy.WithTenant(c.Request().Context(), tenant, issuer, urlPrefix)))
		return next(c)
	}
	tenant, err := d.TenantRepo.FindByRealm(c.Request().Context(), realm)
	if err != nil {
		return err
	}
	if tenant == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "tenant_not_found"})
	}
	if tenant.Status != tenancydomain.TenantStatusActive || tenant.DisabledAt != nil {
		return c.JSON(http.StatusBadRequest, OAuthErrorBody("invalid_request", "tenant is unavailable"))
	}
	issuer := tenantIssuer(d.Issuer, tenant.Realm)
	if bare && d.LegacyBareIssuer {
		issuer = strings.TrimSuffix(d.Issuer, "/")
	}
	c.SetRequest(c.Request().WithContext(tenancy.WithTenant(c.Request().Context(), tenant, issuer, urlPrefix)))
	return next(c)
}

func tenantIssuer(base, realm string) string {
	return strings.TrimSuffix(base, "/") + "/realms/" + realm
}

func RequestTenantID(c *echo.Context) string {
	return tenancy.TenantID(c.Request().Context())
}

func RequestIssuer(c *echo.Context, fallback string) string {
	return tenancy.Issuer(c.Request().Context(), fallback)
}

// RequestHTU は DPoP proof の htu (RFC 9449 §4.2) として用いる、
// クエリ・フラグメント無しの絶対 URL を返す。
// テナント prefix `/realms/{id}` を含むパスでもクライアントが送ったままに復元する。
func RequestHTU(c *echo.Context, base string) string {
	return strings.TrimRight(base, "/") + c.Request().URL.Path
}

func TenantRoute(c *echo.Context, path string) string {
	if prefix := tenancy.URLPrefix(c.Request().Context()); prefix != "" {
		return prefix + path
	}
	return path
}

func TenantCookiePath(c *echo.Context) string {
	if prefix := tenancy.URLPrefix(c.Request().Context()); prefix != "" {
		return prefix
	}
	return "/"
}
