package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	"github.com/jackc/pgx/v5"
)

// SsfReceiverConfigRepository は SsfReceiverConfig を PostgreSQL に永続化する。
type SsfReceiverConfigRepository struct{ Pool sharedpg.DB }

func receiverConfigFromRow(row *SsfReceiverConfig) (*ssdomain.SsfReceiverConfig, error) {
	c := &ssdomain.SsfReceiverConfig{
		StreamID: row.StreamID, TrustedIssuer: row.TrustedIssuer, JWKSURI: textPtrOrNil(row.JwksUri),
	}
	if len(row.Jwks) > 0 {
		var jwks map[string]any
		if err := json.Unmarshal(row.Jwks, &jwks); err != nil {
			return nil, err
		}
		c.JWKS = jwks
	}
	if err := json.Unmarshal(row.AcceptedAudiences, &c.AcceptedAudiences); err != nil {
		return nil, err
	}
	return c, c.Validate()
}

func (r *SsfReceiverConfigRepository) FindByStream(ctx context.Context, tenantID, streamID string) (*ssdomain.SsfReceiverConfig, error) {
	row, err := New(r.Pool).FindSsfReceiverConfigByStream(ctx, FindSsfReceiverConfigByStreamParams{TenantID: tenantID, StreamID: streamID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return receiverConfigFromRow(row)
}

func (r *SsfReceiverConfigRepository) Save(ctx context.Context, tenantID string, c *ssdomain.SsfReceiverConfig) error {
	audiences := c.AcceptedAudiences
	if audiences == nil {
		audiences = []string{}
	}
	audiencesJSON, err := json.Marshal(audiences)
	if err != nil {
		return err
	}
	var jwksJSON []byte
	if c.JWKS != nil {
		jwksJSON, err = json.Marshal(c.JWKS)
		if err != nil {
			return err
		}
	}
	return New(r.Pool).SaveSsfReceiverConfig(ctx, SaveSsfReceiverConfigParams{
		StreamID: c.StreamID, TenantID: tenantID, TrustedIssuer: c.TrustedIssuer,
		JwksUri: textOrNilPtr(c.JWKSURI), Jwks: jwksJSON, AcceptedAudiences: audiencesJSON,
	})
}

func (r *SsfReceiverConfigRepository) Delete(ctx context.Context, tenantID, streamID string) error {
	return New(r.Pool).DeleteSsfReceiverConfig(ctx, DeleteSsfReceiverConfigParams{TenantID: tenantID, StreamID: streamID})
}
