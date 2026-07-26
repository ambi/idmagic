package db_postgres

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/saml/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
)

type SamlIdentityProviderProfileRepository struct{ Pool sharedpg.DB }

func samlIDPProfileFromRow(row *SamlIdentityProviderProfile) *domain.SamlIdentityProviderProfile {
	return &domain.SamlIdentityProviderProfile{
		TenantID: row.TenantID, ProfileID: row.ProfileID, Name: row.Name,
		Mode: domain.IDPProfileMode(row.Mode), IsDefault: row.IsDefault,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (r *SamlIdentityProviderProfileRepository) EnsureDefaultIDPProfile(ctx context.Context, tenantID string) (*domain.SamlIdentityProviderProfile, error) {
	row, err := New(r.Pool).EnsureDefaultSamlIDPProfile(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return samlIDPProfileFromRow(row), nil
}

func (r *SamlIdentityProviderProfileRepository) FindIDPProfileByID(ctx context.Context, tenantID, profileID string) (*domain.SamlIdentityProviderProfile, error) {
	row, err := New(r.Pool).GetSamlIDPProfile(ctx, GetSamlIDPProfileParams{TenantID: tenantID, ProfileID: profileID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return samlIDPProfileFromRow(row), nil
}

func (r *SamlIdentityProviderProfileRepository) ListIDPProfilesByTenant(ctx context.Context, tenantID string) ([]*domain.SamlIdentityProviderProfile, error) {
	if _, err := r.EnsureDefaultIDPProfile(ctx, tenantID); err != nil {
		return nil, err
	}
	rows, err := New(r.Pool).ListSamlIDPProfilesByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.SamlIdentityProviderProfile, 0, len(rows))
	for _, row := range rows {
		out = append(out, samlIDPProfileFromRow(row))
	}
	return out, nil
}

func (r *SamlIdentityProviderProfileRepository) SaveIDPProfile(ctx context.Context, profile *domain.SamlIdentityProviderProfile) error {
	if profile == nil {
		return domain.ErrInvalidIDPProfile
	}
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	existing, err := q.GetSamlIDPProfileForUpdate(ctx, GetSamlIDPProfileForUpdateParams{
		TenantID: profile.TenantID, ProfileID: profile.ProfileID,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err == nil && existing.IsDefault {
		return domain.ErrDefaultIDPProfile
	}
	count, err := q.CountSamlServiceProvidersByIDPProfile(ctx, CountSamlServiceProvidersByIDPProfileParams{
		TenantID: profile.TenantID, IdpProfileID: profile.ProfileID,
	})
	if err != nil {
		return err
	}
	if err := profile.Validate(int(count)); err != nil {
		return err
	}
	now := time.Now().UTC()
	createdAt := profile.CreatedAt
	if existing != nil {
		createdAt = existing.CreatedAt
	} else if createdAt.IsZero() {
		createdAt = now
	}
	if err := q.UpsertSamlIDPProfile(ctx, UpsertSamlIDPProfileParams{
		TenantID: profile.TenantID, ProfileID: profile.ProfileID, Name: profile.Name,
		Mode: string(profile.Mode), IsDefault: profile.IsDefault, CreatedAt: createdAt, UpdatedAt: now,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *SamlIdentityProviderProfileRepository) DeleteIDPProfile(ctx context.Context, tenantID, profileID string) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	profile, err := q.GetSamlIDPProfileForUpdate(ctx, GetSamlIDPProfileForUpdateParams{
		TenantID: tenantID, ProfileID: profileID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if profile.IsDefault {
		return domain.ErrDefaultIDPProfile
	}
	count, err := q.CountSamlServiceProvidersByIDPProfile(ctx, CountSamlServiceProvidersByIDPProfileParams{
		TenantID: tenantID, IdpProfileID: profileID,
	})
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrIDPProfileInUse
	}
	if err := q.DeleteSamlIDPProfile(ctx, DeleteSamlIDPProfileParams{TenantID: tenantID, ProfileID: profileID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
