package support_http

import (
	"net/url"
	"strings"

	"github.com/ambi/idmagic/backend/shared/tenantcontext"
	"github.com/labstack/echo/v5"
)

func RequestTenantID(c *echo.Context) string {
	return tenantcontext.TenantID(c.Request().Context())
}

func RequestIssuer(c *echo.Context, fallback string) string {
	return tenantcontext.Issuer(c.Request().Context(), fallback)
}

// RequestHTU は DPoP proof の htu (RFC 9449 §4.2) として照合する、クエリとフラグメントを
// 含まない絶対 URL を返す。文脈にテナントが無いときは fallback を issuer として使う。
//
// issuer 全体ではなく origin にリクエストのパスを継ぐ。path style の issuer は既に
// /realms/{realm} を含み、パスにも同じ prefix があるので、issuer 全体へ継ぐと二重になる。
// Host ヘッダーからは組まない。正規ロケーションは discovery が広告する URL と同じ出所なので、
// プロキシ配下でもクライアントの URL と一致し、ヘッダーの偽装で期待値を動かせない。
//
// origin を決められなければ "" を返す。DPoP の検証は空の htu を一致とみなさないので、
// その場合の証明は拒否される。
func RequestHTU(c *echo.Context, fallback string) string {
	issuer, err := url.Parse(RequestIssuer(c, fallback))
	if err != nil || issuer.Scheme == "" || issuer.Host == "" {
		return ""
	}
	return issuer.Scheme + "://" + issuer.Host + c.Request().URL.Path
}

// TenantURL はテナントの正規ロケーション配下の絶対 URL を組み立てる。
// issuer 自体が正規ロケーションの root なので、issuer に path を継ぐだけでよい。
//
// `issuer + TenantRoute(c, path)` と書いてはならない: path style では issuer が
// 既に /realms/{realm} を含むため prefix が二重になる。TenantRoute は同一 origin 内の
// 相対リンク (redirect 先、cookie path) 専用。
func TenantURL(c *echo.Context, path, fallback string) string {
	return strings.TrimRight(RequestIssuer(c, fallback), "/") + path
}

func TenantRoute(c *echo.Context, path string) string {
	if prefix := tenantcontext.URLPrefix(c.Request().Context()); prefix != "" {
		return prefix + path
	}
	return path
}

func TenantCookiePath(c *echo.Context) string {
	if prefix := tenantcontext.URLPrefix(c.Request().Context()); prefix != "" {
		return prefix
	}
	return "/"
}

// TenantCookieName は endpoint style に合わせて cookie 名を返す。Subdomain は tenant
// 固有 host でだけ使えるため __Host- prefix を使い、Domain 属性を持たず Path=/ とする。
// Path style は既存の名前を保ち、/realms/{realm} の Path で分離する。
func TenantCookieName(c *echo.Context, name string) string {
	if TenantCookieSecure(c) {
		return "__Host-" + name
	}
	return name
}

// TenantCookieSecure は提供側が要求する Cookie の Secure 属性を返す。
func TenantCookieSecure(c *echo.Context) bool {
	_, _, hostOnly := tenantcontext.HTTPLocation(c.Request().Context())
	return hostOnly
}
