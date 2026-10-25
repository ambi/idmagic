package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
)

// ConnectorRepository は Directory Connector を PostgreSQL に保存する。
type ConnectorRepository struct{ Pool sharedpg.DB }

var _ ports.ConnectorRepository = (*ConnectorRepository)(nil)

func (r *ConnectorRepository) Create(ctx context.Context, c domain.Connector) error {
	publicKey, err := encodeKey(c.PublicKey)
	if err != nil {
		return err
	}
	return New(r.Pool).CreateDirectoryConnector(ctx, CreateDirectoryConnectorParams{
		ID: c.ID, TenantID: c.TenantID, DisplayName: c.DisplayName, Status: string(c.Status),
		EnrollmentTokenHash: text(c.EnrollmentTokenHash), EnrollmentExpiresAt: instant(c.EnrollmentExpiresAt),
		PublicKey: publicKey, KeyID: text(c.KeyID), CreatedAt: c.CreatedAt,
		EnrolledAt: optional(c.EnrolledAt), LastSeenAt: optional(c.LastSeenAt), DisabledAt: optional(c.DisabledAt),
		Revision:                         c.Revision,
		AccidentalDeletionCountThreshold: int32(c.Thresholds.Count), AccidentalDeletionPercentThreshold: int32(c.Thresholds.Percent), //nolint:gosec // 検証済みの INTEGER の範囲。
	})
}

func (r *ConnectorRepository) Update(ctx context.Context, c domain.Connector) error {
	publicKey, err := encodeKey(c.PublicKey)
	if err != nil {
		return err
	}
	updated, err := New(r.Pool).UpdateDirectoryConnector(ctx, UpdateDirectoryConnectorParams{
		DisplayName: c.DisplayName, Status: string(c.Status),
		EnrollmentTokenHash: text(c.EnrollmentTokenHash), EnrollmentExpiresAt: instant(c.EnrollmentExpiresAt),
		PublicKey: publicKey, KeyID: text(c.KeyID),
		EnrolledAt: optional(c.EnrolledAt), DisabledAt: optional(c.DisabledAt),
		Revision: c.Revision, TenantID: c.TenantID, ID: c.ID,
		AccidentalDeletionCountThreshold: int32(c.Thresholds.Count), AccidentalDeletionPercentThreshold: int32(c.Thresholds.Percent), //nolint:gosec // 検証済みの INTEGER の範囲。
	})
	if err != nil {
		return err
	}
	if updated == 0 {
		return ports.ErrStaleConnector
	}
	return nil
}

func (r *ConnectorRepository) Find(ctx context.Context, tenantID, id string) (*domain.Connector, error) {
	// 列は uuid なので、形式の外の ID は問い合わせずに不在として扱う。
	if !isUUID(id) {
		return nil, nil
	}
	return found(New(r.Pool).FindDirectoryConnector(ctx, FindDirectoryConnectorParams{TenantID: tenantID, ID: id}))
}

func (r *ConnectorRepository) FindByEnrollmentTokenHash(ctx context.Context, tenantID, tokenHash string) (*domain.Connector, error) {
	return found(New(r.Pool).FindDirectoryConnectorByEnrollmentTokenHash(ctx, FindDirectoryConnectorByEnrollmentTokenHashParams{
		TenantID: tenantID, EnrollmentTokenHash: text(tokenHash),
	}))
}

func (r *ConnectorRepository) List(ctx context.Context, tenantID string) ([]domain.Connector, error) {
	rows, err := New(r.Pool).ListDirectoryConnectors(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Connector, 0, len(rows))
	for _, row := range rows {
		connector, err := toDomain(row)
		if err != nil {
			return nil, err
		}
		result = append(result, connector)
	}
	return result, nil
}

func (r *ConnectorRepository) TouchLastSeen(ctx context.Context, tenantID, id string, at time.Time) error {
	return New(r.Pool).TouchDirectoryConnectorLastSeen(ctx, TouchDirectoryConnectorLastSeenParams{
		TenantID: tenantID, ID: id, LastSeenAt: instant(at),
	})
}

func found(row *DirectoryConnector, err error) (*domain.Connector, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	connector, err := toDomain(row)
	if err != nil {
		return nil, err
	}
	return &connector, nil
}

func toDomain(row *DirectoryConnector) (domain.Connector, error) {
	connector := domain.Connector{
		TenantID: row.TenantID, ID: row.ID, DisplayName: row.DisplayName, Status: domain.ConnectorStatus(row.Status),
		EnrollmentTokenHash: row.EnrollmentTokenHash.String, KeyID: row.KeyID.String,
		CreatedAt: row.CreatedAt.UTC(), Revision: row.Revision,
		Thresholds: domain.DisableThresholds{Count: int(row.AccidentalDeletionCountThreshold), Percent: int(row.AccidentalDeletionPercentThreshold)},
		EnrolledAt: pointer(row.EnrolledAt), LastSeenAt: pointer(row.LastSeenAt), DisabledAt: pointer(row.DisabledAt),
	}
	if row.EnrollmentExpiresAt.Valid {
		connector.EnrollmentExpiresAt = row.EnrollmentExpiresAt.Time.UTC()
	}
	if row.PublicKey != nil {
		if err := json.Unmarshal(row.PublicKey, &connector.PublicKey); err != nil {
			return domain.Connector{}, err
		}
	}
	return connector, nil
}

func encodeKey(key map[string]any) ([]byte, error) {
	if key == nil {
		return nil, nil
	}
	return json.Marshal(key)
}

func text(value string) pgtype.Text { return pgtype.Text{String: value, Valid: value != ""} }

func instant(at time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: at, Valid: !at.IsZero()}
}

func optional(at *time.Time) pgtype.Timestamptz {
	if at == nil {
		return pgtype.Timestamptz{}
	}
	return instant(*at)
}

func pointer(at pgtype.Timestamptz) *time.Time {
	if !at.Valid {
		return nil
	}
	utc := at.Time.UTC()
	return &utc
}

func isUUID(id string) bool { return uuid.Validate(id) == nil }
