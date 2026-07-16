package http

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"

	"github.com/labstack/echo/v5"
)

type brandingUpdateRequest struct {
	ProductName  *string                  `json:"product_name,omitempty"`
	PrimaryColor *string                  `json:"primary_color,omitempty"`
	AccentColor  *string                  `json:"accent_color,omitempty"`
	FooterLink1  *domain.TenantFooterLink `json:"footer_link_1,omitempty"`
	FooterLink2  *domain.TenantFooterLink `json:"footer_link_2,omitempty"`
	FooterText   *string                  `json:"footer_text,omitempty"`
}

func brandingChangedFields(input brandingUpdateRequest) []string {
	fields := []string{}
	if input.ProductName != nil {
		fields = append(fields, "product_name")
	}
	if input.PrimaryColor != nil {
		fields = append(fields, "primary_color")
	}
	if input.AccentColor != nil {
		fields = append(fields, "accent_color")
	}
	if input.FooterLink1 != nil {
		fields = append(fields, "footer_link_1")
	}
	if input.FooterLink2 != nil {
		fields = append(fields, "footer_link_2")
	}
	if input.FooterText != nil {
		fields = append(fields, "footer_text")
	}
	return fields
}

func (d Deps) writeBrandingError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, tenantusecases.ErrInvalidBranding):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_branding",
			"ブランディング設定が不正です (色は#rrggbb形式、リンクはhttpsのみ有効です)")
	case errors.Is(err, tenantusecases.ErrInvalidBrandingAssetKind):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "kind は logo または favicon を指定してください")
	case errors.Is(err, tenantusecases.ErrBrandingAssetRequired):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "画像ファイルを指定してください")
	case errors.Is(err, tenantusecases.ErrBrandingAssetTooLarge):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "画像は256KiB以下にしてください")
	case errors.Is(err, tenantusecases.ErrBrandingAssetFormat):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "画像はPNG/JPEG/WebP/GIFのいずれかにしてください")
	default:
		return err
	}
}

func (d Deps) handleUpdateBranding(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input brandingUpdateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	branding, err := tenantusecases.UpdateBranding(c.Request().Context(), d.BrandingRepo, actor.TenantID, tenantusecases.BrandingUpdateInput{
		ProductName:  input.ProductName,
		PrimaryColor: input.PrimaryColor,
		AccentColor:  input.AccentColor,
		FooterLink1:  input.FooterLink1,
		FooterLink2:  input.FooterLink2,
		FooterText:   input.FooterText,
	}, time.Now().UTC())
	if err != nil {
		return d.writeBrandingError(c, err)
	}
	if d.Emit != nil {
		d.Emit(&domain.TenantBrandingUpdated{
			At: time.Now().UTC(), ActorUserID: actor.ID, TenantID: actor.TenantID,
			ChangedFields: brandingChangedFields(input),
		})
	}
	return support.NoStoreJSON(c, http.StatusOK, toBrandingResponse(branding))
}

func (d Deps) handleUploadBrandingAsset(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	kind := domain.TenantBrandingAssetKind(c.Param("kind"))
	if !kind.Valid() {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "kind は logo または favicon を指定してください")
	}
	file, err := c.FormFile("file")
	if err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "画像ファイルを指定してください")
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(src, tenantusecases.MaxTenantBrandingAssetBytes+1))
	if closeErr := src.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	objectKey, err := spec.NewUUIDv4()
	if err != nil {
		return err
	}
	assetURL := support.TenantRoute(c, "/tenant-branding-assets/"+string(kind)+"/"+objectKey)
	branding, err := tenantusecases.UploadBrandingAsset(c.Request().Context(), d.BrandingRepo, d.BrandingAssetStore, actor.TenantID, tenantusecases.UploadBrandingAssetInput{
		Kind: kind, ObjectKey: objectKey, Data: data, URL: assetURL, Now: time.Now().UTC(),
	})
	if err != nil {
		return d.writeBrandingError(c, err)
	}
	if d.Emit != nil {
		d.Emit(&domain.TenantBrandingUpdated{
			At: time.Now().UTC(), ActorUserID: actor.ID, TenantID: actor.TenantID, ChangedFields: []string{string(kind)},
		})
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"branding": toBrandingResponse(branding)})
}

func (d Deps) handleDeleteBrandingAsset(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	kind := domain.TenantBrandingAssetKind(c.Param("kind"))
	if !kind.Valid() {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "kind は logo または favicon を指定してください")
	}
	branding, err := tenantusecases.DeleteBrandingAsset(
		c.Request().Context(), d.BrandingRepo, d.BrandingAssetStore, actor.TenantID, kind, time.Now().UTC(),
	)
	if err != nil {
		return d.writeBrandingError(c, err)
	}
	if d.Emit != nil {
		d.Emit(&domain.TenantBrandingUpdated{
			At: time.Now().UTC(), ActorUserID: actor.ID, TenantID: actor.TenantID, ChangedFields: []string{string(kind)},
		})
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"branding": toBrandingResponse(branding)})
}
