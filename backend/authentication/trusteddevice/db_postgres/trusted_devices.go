package db_postgres

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/authentication/trusteddevice/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// TrustedDeviceRepository は信頼済みデバイスの PostgreSQL 実装 (wi-91)。
type TrustedDeviceRepository struct{ Pool sharedpg.DB }

func (r *TrustedDeviceRepository) queries() *Queries { return New(r.Pool) }

func (r *TrustedDeviceRepository) FindBySelector(
	ctx context.Context,
	tenantID, selector string,
) (*domain.TrustedDevice, error) {
	row, err := r.queries().FindTrustedDeviceBySelector(ctx, FindTrustedDeviceBySelectorParams{
		TenantID: tenantID, Selector: selector,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // An unknown selector is an absent device, not an error.
	}
	if err != nil {
		return nil, err
	}
	return toDomain(row), nil
}

func (r *TrustedDeviceRepository) FindByID(
	ctx context.Context,
	tenantID, userID, deviceID string,
) (*domain.TrustedDevice, error) {
	row, err := r.queries().FindTrustedDeviceByID(ctx, FindTrustedDeviceByIDParams{
		TenantID: tenantID, UserID: userID, ID: deviceID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // Another user's device is intentionally treated as absent.
	}
	if err != nil {
		return nil, err
	}
	return toDomain(row), nil
}

func (r *TrustedDeviceRepository) ListActiveByUser(
	ctx context.Context,
	tenantID, userID string,
) ([]*domain.TrustedDevice, error) {
	rows, err := r.queries().ListActiveTrustedDevicesByUser(ctx, ListActiveTrustedDevicesByUserParams{
		TenantID: tenantID, UserID: userID, ExpiresAt: time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.TrustedDevice, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomain(row))
	}
	return out, nil
}

func (r *TrustedDeviceRepository) Save(ctx context.Context, device *domain.TrustedDevice) error {
	return r.queries().UpsertTrustedDevice(ctx, UpsertTrustedDeviceParams{
		ID: device.ID, TenantID: device.TenantID, UserID: device.UserID,
		Selector: device.Selector, VerifierHash: device.VerifierHash,
		Label:     textOrNil(device.Label),
		CreatedAt: device.CreatedAt, LastUsedAt: device.LastUsedAt, ExpiresAt: device.ExpiresAt,
		RevokedAt: timestamptzOrNil(device.RevokedAt), RevokeReason: revokeReasonOrNil(device.RevokeReason),
	})
}

// RevokeAllForUser は未失効の行を 1 度の UPDATE でまとめて失効させる。RETURNING で
// 実際に失効した行だけが返るので、既に失効済みの行に対する再送は自然に no-op になる。
func (r *TrustedDeviceRepository) RevokeAllForUser(
	ctx context.Context,
	tenantID, userID string,
	reason spec.TrustedDeviceRevokeReason,
	now time.Time,
) ([]*domain.TrustedDevice, error) {
	rows, err := r.queries().RevokeTrustedDevicesForUser(ctx, RevokeTrustedDevicesForUserParams{
		TenantID: tenantID, UserID: userID,
		RevokedAt:    pgtype.Timestamptz{Time: now, Valid: true},
		RevokeReason: pgtype.Text{String: string(reason), Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.TrustedDevice, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomain(row))
	}
	return out, nil
}

func (r *TrustedDeviceRepository) DeleteAllForSub(ctx context.Context, userID string) error {
	return r.queries().DeleteTrustedDevicesForSub(ctx, userID)
}

func toDomain(row *TrustedDevice) *domain.TrustedDevice {
	if row == nil {
		return nil
	}
	device := &domain.TrustedDevice{
		ID: row.ID, TenantID: row.TenantID, UserID: row.UserID,
		Selector: row.Selector, VerifierHash: row.VerifierHash, Label: row.Label.String,
		CreatedAt: row.CreatedAt, LastUsedAt: row.LastUsedAt, ExpiresAt: row.ExpiresAt,
		RevokedAt: timestamptzPtr(row.RevokedAt),
	}
	if row.RevokeReason.Valid {
		reason := spec.TrustedDeviceRevokeReason(row.RevokeReason.String)
		device.RevokeReason = &reason
	}
	return device
}

func textOrNil(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func revokeReasonOrNil(reason *spec.TrustedDeviceRevokeReason) pgtype.Text {
	if reason == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: string(*reason), Valid: true}
}

func timestamptzOrNil(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func timestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	value := t.Time
	return &value
}
