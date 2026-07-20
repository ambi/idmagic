package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/oauth2/db_postgres/sqlcgen"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// ConsentRepository は Consent を PostgreSQL に永続化する。クエリは sqlc 生成
// (wi-173, ADR-090); Pool は sqlcgen.DBTX を構造的に満たす。
type ConsentRepository struct{ Pool sharedpg.DB }

func consentFromRow(row *sqlcgen.Consent) (*domain.Consent, error) {
	c := &domain.Consent{
		UserID:    row.UserID,
		ClientID:  row.ClientID,
		GrantedAt: row.GrantedAt,
		ExpiresAt: row.ExpiresAt,
	}
	if row.RevokedAt.Valid {
		revokedAt := row.RevokedAt.Time
		c.RevokedAt = &revokedAt
	}
	if err := json.Unmarshal(row.Scopes, &c.Scopes); err != nil {
		return nil, err
	}
	switch {
	case c.RevokedAt != nil:
		c.State = domain.ConsentRevoked
	case !time.Now().Before(c.ExpiresAt):
		c.State = domain.ConsentExpired
	default:
		c.State = domain.ConsentGranted
	}
	return c, nil
}

func (r *ConsentRepository) Find(ctx context.Context, tenantID, sub, clientID string) (*domain.Consent, error) {
	row, err := sqlcgen.New(r.Pool).GetConsent(ctx, sqlcgen.GetConsentParams{UserID: sub, ClientID: clientID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return consentFromRow(row)
}

func (r *ConsentRepository) FindAll(ctx context.Context, tenantID string) ([]*domain.Consent, error) {
	rows, err := sqlcgen.New(r.Pool).ListConsentsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Consent, 0, len(rows))
	for _, row := range rows {
		c, err := consentFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func (r *ConsentRepository) Save(ctx context.Context, tenantID string, c *domain.Consent) error {
	scopes, err := json.Marshal(c.Scopes)
	if err != nil {
		return err
	}
	var revokedAt pgtype.Timestamptz
	if c.RevokedAt != nil {
		revokedAt = pgtype.Timestamptz{Time: *c.RevokedAt, Valid: true}
	}
	return sqlcgen.New(r.Pool).UpsertConsent(ctx, sqlcgen.UpsertConsentParams{
		UserID:    c.UserID,
		ClientID:  c.ClientID,
		Scopes:    scopes,
		GrantedAt: c.GrantedAt,
		ExpiresAt: c.ExpiresAt,
		RevokedAt: revokedAt,
	})
}

func (r *ConsentRepository) Revoke(ctx context.Context, tenantID, sub, clientID string) error {
	return sqlcgen.New(r.Pool).RevokeConsent(ctx, sqlcgen.RevokeConsentParams{UserID: sub, ClientID: clientID})
}

func (r *ConsentRepository) DeleteAllForSub(ctx context.Context, sub string) error {
	return sqlcgen.New(r.Pool).DeleteConsentsForSub(ctx, sub)
}
