package db_postgres

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/jackc/pgx/v5"
)

// TokenExchangeGrantStore は交換の許可を PostgreSQL に持つ。tenant は ctx から解決する。
type TokenExchangeGrantStore struct {
	Pool sharedpg.DB
}

func (s *TokenExchangeGrantStore) Save(ctx context.Context, grantHash string, grant domain.TokenExchangeGrant) error {
	return New(s.Pool).InsertOauth2TokenExchangeGrant(ctx, InsertOauth2TokenExchangeGrantParams{
		TenantID: tenantports.TenantID(ctx), GrantHash: grantHash, ClientID: grant.ClientID,
		SubjectTokenID: grant.SubjectTokenID, Resource: grant.Resource, ExpiresAt: grant.ExpiresAt,
	})
}

func (s *TokenExchangeGrantStore) Claim(ctx context.Context, grantHash string, grant domain.TokenExchangeGrant, now time.Time) (bool, error) {
	_, err := New(s.Pool).ClaimOauth2TokenExchangeGrant(ctx, ClaimOauth2TokenExchangeGrantParams{
		TenantID: tenantports.TenantID(ctx), GrantHash: grantHash, ClientID: grant.ClientID,
		SubjectTokenID: grant.SubjectTokenID, Resource: grant.Resource, Now: now,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *TokenExchangeGrantStore) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := New(s.Pool).DeleteExpiredOauth2TokenExchangeGrantsBatch(ctx, DeleteExpiredOauth2TokenExchangeGrantsBatchParams{
		Cutoff:     cutoff,
		BatchLimit: int32(limit), //nolint:gosec // G115: small housekeeping batch size
	})
	return int(deleted), err
}
