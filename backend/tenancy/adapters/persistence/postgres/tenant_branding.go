package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/tenancy/adapters/persistence/postgres/sqlcgen"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

// TenantBrandingRepository は branding config を PostgreSQL に保存する (wi-89,
// ADR-096)。tenant_brandings は個別 nullable 列を持つ専用テーブルで、tenants には
// 列を追加しない。クエリは sqlc 生成。
type TenantBrandingRepository struct{ Pool sharedpg.DB }

func (r *TenantBrandingRepository) FindByTenant(ctx context.Context, tenantID string) (*domain.TenantBranding, error) {
	row, err := sqlcgen.New(r.Pool).FindTenantBrandingByTenant(ctx, tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.TenantBranding{
		TenantID:         row.TenantID,
		ProductName:      row.ProductName.String,
		LogoObjectKey:    row.LogoObjectKey.String,
		LogoURL:          row.LogoUrl.String,
		FaviconObjectKey: row.FaviconObjectKey.String,
		FaviconURL:       row.FaviconUrl.String,
		PrimaryColor:     row.PrimaryColor.String,
		AccentColor:      row.AccentColor.String,
		FooterLink1:      domain.TenantFooterLink{Label: row.FooterLink1Label.String, URL: row.FooterLink1Url.String},
		FooterLink2:      domain.TenantFooterLink{Label: row.FooterLink2Label.String, URL: row.FooterLink2Url.String},
		FooterText:       row.FooterText.String,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

func (r *TenantBrandingRepository) Save(ctx context.Context, branding *domain.TenantBranding) error {
	return sqlcgen.New(r.Pool).SaveTenantBranding(ctx, sqlcgen.SaveTenantBrandingParams{
		TenantID:         branding.TenantID,
		ProductName:      textOrNil(branding.ProductName),
		LogoObjectKey:    textOrNil(branding.LogoObjectKey),
		LogoUrl:          textOrNil(branding.LogoURL),
		FaviconObjectKey: textOrNil(branding.FaviconObjectKey),
		FaviconUrl:       textOrNil(branding.FaviconURL),
		PrimaryColor:     textOrNil(branding.PrimaryColor),
		AccentColor:      textOrNil(branding.AccentColor),
		FooterLink1Label: textOrNil(branding.FooterLink1.Label),
		FooterLink1Url:   textOrNil(branding.FooterLink1.URL),
		FooterLink2Label: textOrNil(branding.FooterLink2.Label),
		FooterLink2Url:   textOrNil(branding.FooterLink2.URL),
		FooterText:       textOrNil(branding.FooterText),
		CreatedAt:        branding.CreatedAt,
		UpdatedAt:        branding.UpdatedAt,
	})
}

func textOrNil(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// TenantBrandingAssetStore は branding ロゴ / favicon blob を PostgreSQL に保存する
// (wi-89, ADR-096)。ADR-073 の application_icons と同型だが専用テーブルに分離する。
type TenantBrandingAssetStore struct{ Pool sharedpg.DB }

func (s *TenantBrandingAssetStore) Save(ctx context.Context, asset *domain.TenantBrandingAsset) error {
	return sqlcgen.New(s.Pool).UpsertTenantBrandingAsset(ctx, sqlcgen.UpsertTenantBrandingAssetParams{
		TenantID: asset.TenantID, Kind: string(asset.Kind), ObjectKey: asset.ObjectKey,
		ContentType: asset.ContentType, SizeBytes: int32(asset.SizeBytes), //nolint:gosec // G115: asset size is bounded by upload limits, well under int32 max
		Data: asset.Data, CreatedAt: asset.CreatedAt, UpdatedAt: asset.UpdatedAt,
	})
}

func (s *TenantBrandingAssetStore) Find(ctx context.Context, tenantID string, kind domain.TenantBrandingAssetKind, objectKey string) (*domain.TenantBrandingAsset, error) {
	row, err := sqlcgen.New(s.Pool).GetTenantBrandingAsset(ctx, sqlcgen.GetTenantBrandingAssetParams{
		TenantID: tenantID, Kind: string(kind), ObjectKey: objectKey,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.TenantBrandingAsset{
		TenantID: row.TenantID, Kind: domain.TenantBrandingAssetKind(row.Kind), ObjectKey: row.ObjectKey,
		ContentType: row.ContentType, SizeBytes: int(row.SizeBytes), Data: row.Data,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (s *TenantBrandingAssetStore) DeleteByTenant(ctx context.Context, tenantID string, kind domain.TenantBrandingAssetKind) error {
	return sqlcgen.New(s.Pool).DeleteTenantBrandingAssetsByKind(ctx, sqlcgen.DeleteTenantBrandingAssetsByKindParams{
		TenantID: tenantID, Kind: string(kind),
	})
}
