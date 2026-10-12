package db_postgres

import (
	"context"
	"errors"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	"github.com/jackc/pgx/v5"
)

// SsfTransmitterConfigRepository は SsfTransmitterConfig を PostgreSQL に永続化する。
type SsfTransmitterConfigRepository struct{ Pool sharedpg.DB }

func transmitterConfigFromRow(row *SsfTransmitterConfig) *ssdomain.SsfTransmitterConfig {
	return &ssdomain.SsfTransmitterConfig{
		StreamID: row.StreamID, DeliveryEndpoint: row.DeliveryEndpoint, Audience: row.Audience,
		DeliveryAuthorization: textPtrOrNil(row.DeliveryAuthorization),
		MaxDeliveryAttempts:   int(row.MaxDeliveryAttempts),
	}
}

func (r *SsfTransmitterConfigRepository) FindByStream(ctx context.Context, tenantID, streamID string) (*ssdomain.SsfTransmitterConfig, error) {
	row, err := New(r.Pool).FindSsfTransmitterConfigByStream(ctx, FindSsfTransmitterConfigByStreamParams{TenantID: tenantID, StreamID: streamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return transmitterConfigFromRow(row), nil
}

func (r *SsfTransmitterConfigRepository) Save(ctx context.Context, tenantID string, c *ssdomain.SsfTransmitterConfig) error {
	return New(r.Pool).SaveSsfTransmitterConfig(ctx, SaveSsfTransmitterConfigParams{
		StreamID: c.StreamID, TenantID: tenantID, DeliveryEndpoint: c.DeliveryEndpoint, Audience: c.Audience,
		DeliveryAuthorization: textOrNilPtr(c.DeliveryAuthorization),
		MaxDeliveryAttempts:   int32(c.MaxDeliveryAttempts), //nolint:gosec // G115: admin-configured small positive count
	})
}

func (r *SsfTransmitterConfigRepository) Delete(ctx context.Context, tenantID, streamID string) error {
	return New(r.Pool).DeleteSsfTransmitterConfig(ctx, DeleteSsfTransmitterConfigParams{TenantID: tenantID, StreamID: streamID})
}
