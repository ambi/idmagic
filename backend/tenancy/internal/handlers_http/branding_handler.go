package handlers_http

import (
	"net/http"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/internal/usecases"

	"github.com/labstack/echo/v5"
)

// BrandingResponse は GetTenantBranding / UpdateTenantBranding / asset 操作が返す公開
// 安全な projection (SCL TenantBrandingResponse の双子定義)。未設定フィールドは省略し、
// クライアント側でシステム既定 (IdMagic) に解決する。
type BrandingResponse struct {
	ProductName  string                          `json:"product_name,omitempty"`
	LogoURL      string                          `json:"logo_url,omitempty"`
	FaviconURL   string                          `json:"favicon_url,omitempty"`
	PrimaryColor string                          `json:"primary_color,omitempty"`
	AccentColor  string                          `json:"accent_color,omitempty"`
	FooterLink1  *tenancydomain.TenantFooterLink `json:"footer_link_1,omitempty"`
	FooterLink2  *tenancydomain.TenantFooterLink `json:"footer_link_2,omitempty"`
	FooterText   string                          `json:"footer_text,omitempty"`
	UpdatedAt    *time.Time                      `json:"updated_at,omitempty"`
}

func toBrandingResponse(b *tenancydomain.TenantBranding) BrandingResponse {
	resp := BrandingResponse{
		ProductName: b.ProductName, LogoURL: b.LogoURL, FaviconURL: b.FaviconURL,
		PrimaryColor: b.PrimaryColor, AccentColor: b.AccentColor,
		FooterText: b.FooterText,
	}
	if b.FooterLink1.IsSet() {
		link := b.FooterLink1
		resp.FooterLink1 = &link
	}
	if b.FooterLink2.IsSet() {
		link := b.FooterLink2
		resp.FooterLink2 = &link
	}
	if b.IsConfigured() && !b.UpdatedAt.IsZero() {
		updatedAt := b.UpdatedAt
		resp.UpdatedAt = &updatedAt
	}
	return resp
}

// brandingETag は返送する JSON のハッシュを ETag にする。未設定テナントは全テナント共通の
// 固定値を返し、cross-tenant のキャッシュ混同は URL (tenant 解決済み path) 側で防ぐ。
// 担保するルールは docs/design/application/api-guidelines.md の「キャッシュ検証」。
func brandingETag(b *tenancydomain.TenantBranding) string {
	if b == nil || !b.IsConfigured() {
		return `"branding-default"`
	}
	tag, _ := support.EntityTag(toBrandingResponse(b))
	return tag
}

// handleGetBranding は解決済みテナントの hosted UI branding を返す公開 endpoint。
// 認証を要求しない (login / consent / device 画面が未認証のうちに読む)。branding 未設定
// でも例外を投げず空の projection を返し、hosted login エンドポイントを止めない。
func (d Deps) handleGetBranding(c *echo.Context) error {
	branding, err := tenancyusecases.GetBranding(c.Request().Context(), d.BrandingRepo, support.RequestTenantID(c))
	if err != nil {
		return err
	}
	etag := brandingETag(branding)
	c.Response().Header().Set("ETag", etag)
	c.Response().Header().Set("Cache-Control", "public, max-age=60")
	if match := c.Request().Header.Get("If-None-Match"); match != "" && match == etag {
		return c.NoContent(http.StatusNotModified)
	}
	return support.ETagJSON(c, http.StatusOK, toBrandingResponse(branding), etag)
}

// handleGetBrandingAsset は保存済み branding アセット (ロゴ / favicon) を配信する公開
// endpoint。別テナントまたは削除済み object は未存在として扱う (と同型)。
func (d Deps) handleGetBrandingAsset(c *echo.Context) error {
	if d.BrandingAssetStore == nil {
		return support.WriteProblem(c, http.StatusNotFound, "not_found", "The image does not exist.")
	}
	kind := tenancydomain.TenantBrandingAssetKind(c.Param("kind"))
	if !kind.Valid() {
		return support.WriteProblem(c, http.StatusNotFound, "not_found", "The image does not exist.")
	}
	asset, err := d.BrandingAssetStore.Find(c.Request().Context(), support.RequestTenantID(c), kind, c.Param("id"))
	if err != nil {
		return err
	}
	if asset == nil {
		return support.WriteProblem(c, http.StatusNotFound, "not_found", "The image does not exist.")
	}
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	c.Response().Header().Set("Cache-Control", "private, max-age=3600")
	return c.Blob(http.StatusOK, asset.ContentType, asset.Data)
}
