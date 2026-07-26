package support_http

import (
	"net/http"
	"strings"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/tenancy"

	"github.com/labstack/echo/v5"
)

// ResolveHostTenant は Host が `{realm}.{TenantBaseDomain}` に一致するリクエストを
// subdomain style のテナントへ解決する (ADR-144)。TenantBaseDomain が空、Host が
// 一致しない、realm が不在、あるいは見つかったテナントが subdomain style でない
// 場合はいずれも 404 とし、default テナントへは落とさない。
//
// fail-open にすると任意の Host ヘッダで default テナントへ到達でき、テナント境界の
// 破りになるため、既定は deny とする。
func (d Deps) ResolveHostTenant(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		realm, ok := d.hostRealm(c.Request().Host)
		if !ok {
			return tenantNotFound(c)
		}
		return d.resolveTenant(c, next, realm, tenancydomain.TenantEndpointStyleSubdomain)
	}
}

// ResolvePathTenant は `/realms/{realm}/...` を path style のテナントへ解決する。
func (d Deps) ResolvePathTenant(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		// subdomain style のテナントの origin から path prefix で別テナントへ抜けられると、
		// origin とテナントが食い違う (その origin の cookie 空間に別テナントの
		// セッションが載る)。Host がテナントに解決する場合、path 経路は使わせない。
		if _, ok := d.hostRealm(c.Request().Host); ok {
			return tenantNotFound(c)
		}
		return d.resolveTenant(c, next, c.Param("tenant_id"), tenancydomain.TenantEndpointStylePath)
	}
}

// ResolveDefaultRealmTenant は path から realm を読まず、固定で default realm の
// テナントを resolve する。URL prefix は /realms/default になり、cookie path が
// そのまま control-plane endpoint を覆う (cross-tenant cookie 漏れを避けるための
// ADR-033 §1 の選択)。テナント横断操作である /realms/default/admin/tenants 等で使う。
func (d Deps) ResolveDefaultRealmTenant(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if _, ok := d.hostRealm(c.Request().Host); ok {
			return tenantNotFound(c)
		}
		return d.resolveTenant(c, next, tenancydomain.DefaultRealm, tenancydomain.TenantEndpointStylePath)
	}
}

// hostRealm は Host から subdomain style の realm ラベルを取り出す。
// TenantBaseDomain が未設定なら常に不一致 — host 解決を丸ごと無効化する。
func (d Deps) hostRealm(host string) (string, bool) {
	base := strings.ToLower(strings.TrimSpace(d.TenantBaseDomain))
	if base == "" {
		return "", false
	}
	hostname := normalizeHost(host)
	suffix := "." + strings.TrimSuffix(base, ".")
	if !strings.HasSuffix(hostname, suffix) {
		return "", false
	}
	label := strings.TrimSuffix(hostname, suffix)
	// 多段ラベル (`a.b.{base}`) は realm ではない。realm は単一 DNS ラベルなので、
	// ここで弾かないと `evil.acme.{base}` が acme に解決してしまう。
	if label == "" || strings.Contains(label, ".") {
		return "", false
	}
	return label, true
}

// normalizeHost は Host ヘッダを照合可能な hostname にする: port と trailing dot を
// 落として lowercase にする (SCL Tenancy.interfaces.ResolveTenant)。
func normalizeHost(host string) string {
	hostname := strings.ToLower(strings.TrimSpace(host))
	if idx := strings.LastIndex(hostname, ":"); idx >= 0 && !strings.Contains(hostname[idx:], "]") {
		hostname = hostname[:idx]
	}
	hostname = strings.Trim(hostname, "[]")
	return strings.TrimSuffix(hostname, ".")
}

// resolveTenant は realm からテナントを解決し、到達経路が そのテナントの正規ロケーション
// と一致することを確かめてから、issuer / URL prefix を ctx に載せる (ADR-144)。
// 内部キーは tenant.ID (UUID)。
func (d Deps) resolveTenant(
	c *echo.Context,
	next echo.HandlerFunc,
	realm string,
	via tenancydomain.TenantEndpointStyle,
) error {
	if d.TenantRepo == nil {
		if realm != tenancydomain.DefaultRealm || via != tenancydomain.TenantEndpointStylePath {
			return tenantNotFound(c)
		}
		tenant := &tenancydomain.Tenant{
			ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
			Status: tenancydomain.TenantStatusActive, EndpointStyle: tenancydomain.TenantEndpointStylePath,
		}
		return d.enterTenant(c, next, tenant)
	}
	tenant, err := d.TenantRepo.FindByRealm(c.Request().Context(), realm)
	if err != nil {
		return err
	}
	if tenant == nil {
		return tenantNotFound(c)
	}
	// 正規ロケーション以外からの到達は「不在」として扱う。存在を漏らさないため、
	// 不在と区別できる応答にはしない。
	if tenant.EffectiveEndpointStyle() != via {
		return tenantNotFound(c)
	}
	if tenant.Status != tenancydomain.TenantStatusActive || tenant.DisabledAt != nil {
		return c.JSON(http.StatusBadRequest, OAuthErrorBody("invalid_request", "tenant is unavailable"))
	}
	return d.enterTenant(c, next, tenant)
}

func (d Deps) enterTenant(c *echo.Context, next echo.HandlerFunc, tenant *tenancydomain.Tenant) error {
	issuer, urlPrefix := d.CanonicalLocation(tenant)
	c.SetRequest(c.Request().WithContext(tenancy.WithTenant(c.Request().Context(), tenant, issuer, urlPrefix)))
	// テナントの正規ロケーションは Host に依存する。共有キャッシュが host をキーに
	// せずに discovery / branding を混ぜないよう、全 tenant 応答に明示する。
	c.Response().Header().Add("Vary", "Host")
	return next(c)
}

func tenantNotFound(c *echo.Context) error {
	return c.JSON(http.StatusNotFound, map[string]string{"error": "tenant_not_found"})
}

// CanonicalLocation はテナントの正規ロケーションを (issuer, URL prefix) として返す
// (ADR-144)。1 テナント = 1 正規ロケーション = 1 issuer なので、discovery 文書の
// issuer は必ずその文書の取得元 URL と一致する (OIDC Discovery 1.0 §4.3)。
func (d Deps) CanonicalLocation(tenant *tenancydomain.Tenant) (issuer, urlPrefix string) {
	base := strings.TrimSuffix(d.Issuer, "/")
	if tenant.EffectiveEndpointStyle() == tenancydomain.TenantEndpointStyleSubdomain {
		scheme := "https"
		if rest, ok := strings.CutPrefix(base, "http://"); ok {
			scheme, base = "http", rest
		}
		// port を保つ: ローカル開発では issuer が http://localhost:5173 の形になり、
		// テナント origin も同じ port でなければ到達できない。
		host := strings.TrimPrefix(base, "https://")
		if idx := strings.Index(host, "/"); idx >= 0 {
			host = host[:idx]
		}
		port := ""
		if idx := strings.LastIndex(host, ":"); idx >= 0 {
			port = host[idx:]
		}
		return scheme + "://" + tenant.Realm + "." + strings.TrimSuffix(d.TenantBaseDomain, ".") + port, ""
	}
	return base + "/realms/" + tenant.Realm, "/realms/" + tenant.Realm
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

// TenantURL はテナントの正規ロケーション配下の絶対 URL を組み立てる (ADR-144)。
// issuer 自体が正規ロケーションの root なので、issuer に path を継ぐだけでよい。
//
// `issuer + TenantRoute(c, path)` と書いてはならない: path style では issuer が
// 既に /realms/{realm} を含むため prefix が二重になる。TenantRoute は同一 origin 内の
// 相対リンク (redirect 先、cookie path) 専用。
func TenantURL(c *echo.Context, path, fallback string) string {
	return strings.TrimRight(RequestIssuer(c, fallback), "/") + path
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

// TenantCookieName は endpoint style に合わせて cookie 名を返す。Subdomain は tenant
// 固有 host でだけ使えるため __Host- prefix を使い、Domain 属性を持たず Path=/ とする。
// Path style は既存の名前を保ち、/realms/{realm} の Path で分離する。
func TenantCookieName(c *echo.Context, name string) string {
	if tenant := tenancy.Tenant(c.Request().Context()); tenant != nil &&
		tenant.EffectiveEndpointStyle() == tenancydomain.TenantEndpointStyleSubdomain {
		return "__Host-" + name
	}
	return name
}

// TenantCookieSecure reports whether the cookie belongs to a subdomain tenant.
// __Host- is valid only with Secure, independently of the process-wide local
// development issuer setting.
func TenantCookieSecure(c *echo.Context) bool {
	tenant := tenancy.Tenant(c.Request().Context())
	return tenant != nil && tenant.EffectiveEndpointStyle() == tenancydomain.TenantEndpointStyleSubdomain
}
