package usecases

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/mediavalidation"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// MaxTenantBrandingAssetBytes は branding ロゴ / favicon の最大サイズ (ADR-096、ADR-073
// の Application icon と同じ上限)。
const MaxTenantBrandingAssetBytes = 256 * 1024

var (
	ErrInvalidBranding          = errors.New("invalid tenant branding")
	ErrInvalidBrandingAssetKind = errors.New("branding asset kind must be logo or favicon")
	ErrBrandingAssetRequired    = errors.New("branding asset file is required")
	ErrBrandingAssetTooLarge    = errors.New("branding asset exceeds 256KiB")
	ErrBrandingAssetFormat      = errors.New("branding asset must be PNG, JPEG, WebP, or GIF")
)

// BrandingUpdateInput は branding のテキスト / 色 / リンクの部分更新を表す。nil の
// フィールドは現状維持。空文字列を含む非 nil 値はそのフィールドを更新し、空文字列は
// 未設定に戻すことを意味する (TenantBrandingUpdateRequest の SCL 記述通り)。
type BrandingUpdateInput struct {
	ProductName  *string
	PrimaryColor *string
	AccentColor  *string
	SupportURL   *string
	LegalURL     *string
	FooterText   *string
}

// GetBranding は tenant の branding を返す。未設定なら tenant_id だけを持つ空の
// TenantBranding を返し、呼び出し側 (HTTP 層) が常に non-nil を扱えるようにする
// (TenantUserAttributeSchema と同型のパターン)。
func GetBranding(ctx context.Context, repo tenantports.TenantBrandingRepository, tenantID string) (*domain.TenantBranding, error) {
	branding, err := repo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if branding == nil {
		return &domain.TenantBranding{TenantID: tenantID}, nil
	}
	return branding, nil
}

// UpdateBranding は branding のテキスト / 色 / リンクを部分更新する。色は hex 形式と
// コントラスト比、リンクは https scheme のみを検証し、満たさない値は拒否する (ADR-096)。
func UpdateBranding(
	ctx context.Context, repo tenantports.TenantBrandingRepository,
	tenantID string, input BrandingUpdateInput, now time.Time,
) (*domain.TenantBranding, error) {
	existing, err := repo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	branding := &domain.TenantBranding{TenantID: tenantID}
	if existing != nil {
		branding = &domain.TenantBranding{}
		*branding = *existing
	}
	branding.TenantID = tenantID
	if input.ProductName != nil {
		branding.ProductName = strings.TrimSpace(*input.ProductName)
	}
	if input.PrimaryColor != nil {
		branding.PrimaryColor = strings.TrimSpace(*input.PrimaryColor)
	}
	if input.AccentColor != nil {
		branding.AccentColor = strings.TrimSpace(*input.AccentColor)
	}
	if input.SupportURL != nil {
		branding.SupportURL = strings.TrimSpace(*input.SupportURL)
	}
	if input.LegalURL != nil {
		branding.LegalURL = strings.TrimSpace(*input.LegalURL)
	}
	if input.FooterText != nil {
		branding.FooterText = strings.TrimSpace(*input.FooterText)
	}
	t := normalizeNow(now)
	branding.UpdatedAt = t
	if existing == nil || existing.CreatedAt.IsZero() {
		branding.CreatedAt = t
	}
	if err := branding.Validate(); err != nil {
		return nil, errors.Join(ErrInvalidBranding, err)
	}
	if err := repo.Save(ctx, branding); err != nil {
		return nil, err
	}
	return branding, nil
}

// DetectBrandingAssetContentType は backend/shared/mediavalidation の magic byte 判定に
// 委譲し、branding asset 固有のエラー値にマップする (ADR-096: Application icon と検証
// ロジックを共有する)。
func DetectBrandingAssetContentType(data []byte) (string, error) {
	contentType, err := mediavalidation.DetectImageContentType(data, MaxTenantBrandingAssetBytes)
	switch {
	case errors.Is(err, mediavalidation.ErrImageRequired):
		return "", ErrBrandingAssetRequired
	case errors.Is(err, mediavalidation.ErrImageTooLarge):
		return "", ErrBrandingAssetTooLarge
	case errors.Is(err, mediavalidation.ErrImageFormat):
		return "", ErrBrandingAssetFormat
	case err != nil:
		return "", err
	}
	return contentType, nil
}

// UploadBrandingAssetInput は branding ロゴ / favicon のアップロード入力。
type UploadBrandingAssetInput struct {
	Kind      domain.TenantBrandingAssetKind
	ObjectKey string
	Data      []byte
	URL       string
	Now       time.Time
}

// UploadBrandingAsset は branding ロゴ / favicon を検証・保存し、TenantBranding の
// 参照 (object_key / url) を更新する (ADR-096、ADR-073 と同型)。
func UploadBrandingAsset(
	ctx context.Context, brandingRepo tenantports.TenantBrandingRepository, assetStore tenantports.TenantBrandingAssetStore,
	tenantID string, in UploadBrandingAssetInput,
) (*domain.TenantBranding, error) {
	if !in.Kind.Valid() {
		return nil, ErrInvalidBrandingAssetKind
	}
	contentType, err := DetectBrandingAssetContentType(in.Data)
	if err != nil {
		return nil, err
	}
	now := normalizeNow(in.Now)
	objectKey := strings.TrimSpace(in.ObjectKey)
	if objectKey == "" {
		objectKey, err = spec.NewUUIDv4()
		if err != nil {
			return nil, err
		}
	}
	asset := &domain.TenantBrandingAsset{
		TenantID: tenantID, Kind: in.Kind, ObjectKey: objectKey,
		ContentType: contentType, SizeBytes: len(in.Data), Data: slices.Clone(in.Data),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := assetStore.Save(ctx, asset); err != nil {
		return nil, err
	}

	existing, err := brandingRepo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	branding := &domain.TenantBranding{TenantID: tenantID}
	if existing != nil {
		branding = &domain.TenantBranding{}
		*branding = *existing
	}
	branding.TenantID = tenantID
	switch in.Kind {
	case domain.TenantBrandingAssetKindLogo:
		branding.LogoObjectKey = objectKey
		branding.LogoURL = in.URL
	case domain.TenantBrandingAssetKindFavicon:
		branding.FaviconObjectKey = objectKey
		branding.FaviconURL = in.URL
	}
	if existing == nil || existing.CreatedAt.IsZero() {
		branding.CreatedAt = now
	}
	branding.UpdatedAt = now
	if err := branding.Validate(); err != nil {
		return nil, errors.Join(ErrInvalidBranding, err)
	}
	if err := brandingRepo.Save(ctx, branding); err != nil {
		return nil, err
	}
	return branding, nil
}

// DeleteBrandingAsset は branding の保存済みロゴまたは favicon を削除し、TenantBranding
// の参照を空に戻す。branding が未設定のテナントに対しては no-op (ADR-073 の
// DeleteApplicationIcon と同じく冪等)。
func DeleteBrandingAsset(
	ctx context.Context, brandingRepo tenantports.TenantBrandingRepository, assetStore tenantports.TenantBrandingAssetStore,
	tenantID string, kind domain.TenantBrandingAssetKind, now time.Time,
) (*domain.TenantBranding, error) {
	if !kind.Valid() {
		return nil, ErrInvalidBrandingAssetKind
	}
	if err := assetStore.DeleteByTenant(ctx, tenantID, kind); err != nil {
		return nil, err
	}
	existing, err := brandingRepo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return &domain.TenantBranding{TenantID: tenantID}, nil
	}
	updated := *existing
	updated.TenantID = tenantID
	switch kind {
	case domain.TenantBrandingAssetKindLogo:
		updated.LogoObjectKey = ""
		updated.LogoURL = ""
	case domain.TenantBrandingAssetKindFavicon:
		updated.FaviconObjectKey = ""
		updated.FaviconURL = ""
	}
	updated.UpdatedAt = normalizeNow(now)
	if err := brandingRepo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}
