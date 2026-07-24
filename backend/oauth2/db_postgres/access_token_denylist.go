package db_postgres

import (
	"context"
	"time"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
)

// AccessTokenDenylist は失効した access token の jti を PostgreSQL に保持する (ADR-139)。
// failover で失効が消えると防御後退になるため裏付けテーブルは LOGGED。IsRevoked は
// introspection のたびに走り得る唯一のホット読取だが、per-request の共有ストア参照を
// Postgres に置換する範囲に留める (高 RPS 最適化は ADR-139 §6 の別レイヤ)。
// tenant は ctx から解決し、期限切れマーカーは revoked とみなさない。
type AccessTokenDenylist struct{ Pool sharedpg.DB }

func (d *AccessTokenDenylist) Add(ctx context.Context, jti string, expiresAt time.Time) error {
	return New(d.Pool).AddOauth2AccessTokenDenylist(ctx, AddOauth2AccessTokenDenylistParams{
		TenantID:  tenancy.TenantID(ctx),
		Jti:       jti,
		ExpiresAt: expiresAt,
	})
}

func (d *AccessTokenDenylist) IsRevoked(ctx context.Context, jti string) (bool, error) {
	return New(d.Pool).IsOauth2AccessTokenRevoked(ctx, IsOauth2AccessTokenRevokedParams{
		TenantID: tenancy.TenantID(ctx),
		Jti:      jti,
		Now:      time.Now().UTC(),
	})
}

func (d *AccessTokenDenylist) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := New(d.Pool).DeleteExpiredOauth2AccessTokenDenylistBatch(ctx, DeleteExpiredOauth2AccessTokenDenylistBatchParams{
		Cutoff:     cutoff,
		BatchLimit: int32(limit), //nolint:gosec // G115: small housekeeping batch size
	})
	return int(deleted), err
}
