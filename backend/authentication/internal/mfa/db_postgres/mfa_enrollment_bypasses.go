package db_postgres

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/authentication/mfa/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type MfaEnrollmentBypassRepository struct{ Pool sharedpg.DB }

func (r *MfaEnrollmentBypassRepository) queries() *Queries { return New(r.Pool) }

func (r *MfaEnrollmentBypassRepository) Save(ctx context.Context, bypass *domain.MfaEnrollmentBypass) error {
	if err := bypass.Validate(); err != nil {
		return err
	}
	return r.queries().SaveMfaEnrollmentBypass(ctx, SaveMfaEnrollmentBypassParams{
		ID: bypass.ID, TenantID: bypass.TenantID, UserID: bypass.UserID, IssuedBy: bypass.IssuedBy,
		IssuedAt: bypass.IssuedAt, ExpiresAt: bypass.ExpiresAt,
	})
}

func (r *MfaEnrollmentBypassRepository) FindActive(ctx context.Context, tenantID, userID string, now time.Time) (*domain.MfaEnrollmentBypass, error) {
	row, err := r.queries().FindActiveMfaEnrollmentBypass(ctx, FindActiveMfaEnrollmentBypassParams{TenantID: tenantID, UserID: userID, ExpiresAt: now})
	return enrollmentBypassFromRow(row, err)
}

func (r *MfaEnrollmentBypassRepository) ConsumeActive(ctx context.Context, tenantID, userID string, now time.Time) (*domain.MfaEnrollmentBypass, error) {
	row, err := r.queries().ConsumeActiveMfaEnrollmentBypass(ctx, ConsumeActiveMfaEnrollmentBypassParams{TenantID: tenantID, UserID: userID, ConsumedAt: pgtype.Timestamptz{Time: now, Valid: true}})
	return enrollmentBypassFromRow(row, err)
}

func (r *MfaEnrollmentBypassRepository) RevokeActive(ctx context.Context, tenantID, userID string, now time.Time) (*domain.MfaEnrollmentBypass, error) {
	row, err := r.queries().RevokeActiveMfaEnrollmentBypass(ctx, RevokeActiveMfaEnrollmentBypassParams{TenantID: tenantID, UserID: userID, RevokedAt: pgtype.Timestamptz{Time: now, Valid: true}})
	return enrollmentBypassFromRow(row, err)
}

func (r *MfaEnrollmentBypassRepository) ExpireOpen(ctx context.Context, tenantID, userID string, now time.Time) (*domain.MfaEnrollmentBypass, error) {
	row, err := r.queries().ExpireOpenMfaEnrollmentBypass(ctx, ExpireOpenMfaEnrollmentBypassParams{TenantID: tenantID, UserID: userID, ExpiredAt: pgtype.Timestamptz{Time: now, Valid: true}})
	return enrollmentBypassFromRow(row, err)
}

func enrollmentBypassFromRow(row *MfaEnrollmentBypass, err error) (*domain.MfaEnrollmentBypass, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := &domain.MfaEnrollmentBypass{
		ID: row.ID, TenantID: row.TenantID, UserID: row.UserID, IssuedBy: row.IssuedBy,
		IssuedAt: row.IssuedAt, ExpiresAt: row.ExpiresAt,
		ConsumedAt: timestamptzPtr(row.ConsumedAt), RevokedAt: timestamptzPtr(row.RevokedAt),
		ExpiredAt: timestamptzPtr(row.ExpiredAt),
	}
	return out, out.Validate()
}
