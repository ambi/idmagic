package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"

	"github.com/labstack/echo/v5"
)

// BrandingResponse は GetTenantBranding / UpdateTenantBranding / asset 操作が返す公開
// 安全な projection (SCL TenantBrandingResponse の双子定義)。未設定フィールドは省略し、
// クライアント側でシステム既定 (IdMagic) に解決する (ADR-096)。
type BrandingResponse struct {
	ProductName  string     `json:"product_name,omitempty"`
	LogoURL      string     `json:"logo_url,omitempty"`
	FaviconURL   string     `json:"favicon_url,omitempty"`
	PrimaryColor string     `json:"primary_color,omitempty"`
	AccentColor  string     `json:"accent_color,omitempty"`
	SupportURL   string     `json:"support_url,omitempty"`
	LegalURL     string     `json:"legal_url,omitempty"`
	FooterText   string     `json:"footer_text,omitempty"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

func toBrandingResponse(b *domain.TenantBranding) BrandingResponse {
	resp := BrandingResponse{
		ProductName: b.ProductName, LogoURL: b.LogoURL, FaviconURL: b.FaviconURL,
		PrimaryColor: b.PrimaryColor, AccentColor: b.AccentColor,
		SupportURL: b.SupportURL, LegalURL: b.LegalURL, FooterText: b.FooterText,
	}
	if b.IsConfigured() && !b.UpdatedAt.IsZero() {
		updatedAt := b.UpdatedAt
		resp.UpdatedAt = &updatedAt
	}
	return resp
}

// brandingETag は branding の version を ETag にする。未設定テナントは全テナント共通の
// 固定値を返し、cross-tenant のキャッシュ混同は URL (tenant 解決済み path) 側で防ぐ
// (ADR-096 決定 9)。
func brandingETag(b *domain.TenantBranding) string {
	if b == nil || !b.IsConfigured() || b.UpdatedAt.IsZero() {
		return `"branding-default"`
	}
	return `"branding-` + strconv.FormatInt(b.UpdatedAt.UnixNano(), 10) + `"`
}

// handleGetBranding は解決済みテナントの hosted UI branding を返す公開 endpoint。
// 認証を要求しない (login / consent / device 画面が未認証のうちに読む)。branding 未設定
// でも例外を投げず空の projection を返し、hosted login エンドポイントを止めない
// (ADR-096 決定 8)。
func (d Deps) handleGetBranding(c *echo.Context) error {
	branding, err := tenantusecases.GetBranding(c.Request().Context(), d.BrandingRepo, support.RequestTenantID(c))
	if err != nil {
		return err
	}
	etag := brandingETag(branding)
	c.Response().Header().Set("ETag", etag)
	c.Response().Header().Set("Cache-Control", "public, max-age=60")
	if match := c.Request().Header.Get("If-None-Match"); match != "" && match == etag {
		return c.NoContent(http.StatusNotModified)
	}
	return c.JSON(http.StatusOK, toBrandingResponse(branding))
}

// handleGetBrandingAsset は保存済み branding アセット (ロゴ / favicon) を配信する公開
// endpoint。別テナントまたは削除済み object は未存在として扱う (ADR-096、ADR-073 と同型)。
func (d Deps) handleGetBrandingAsset(c *echo.Context) error {
	if d.BrandingAssetStore == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "画像が存在しません")
	}
	kind := domain.TenantBrandingAssetKind(c.Param("kind"))
	if !kind.Valid() {
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "画像が存在しません")
	}
	asset, err := d.BrandingAssetStore.Find(c.Request().Context(), support.RequestTenantID(c), kind, c.Param("object_key"))
	if err != nil {
		return err
	}
	if asset == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "画像が存在しません")
	}
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	c.Response().Header().Set("Cache-Control", "private, max-age=3600")
	return c.Blob(http.StatusOK, asset.ContentType, asset.Data)
}
