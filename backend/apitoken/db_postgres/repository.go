package db_postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/apitoken/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

type Repository struct{ Pool sharedpg.DB }

func timestamptzOrNil(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func textOrNil(value string) pgtype.Text { return pgtype.Text{String: value, Valid: value != ""} }

func tokenFromRow(id, tenantID, userID, jti, clientID string, scopeValues []string, audience string,
	dpopJKT, description pgtype.Text, createdAt time.Time, expiresAt, revokedAt pgtype.Timestamptz,
) (*domain.ApiToken, error) {
	scopes, err := domain.ParseScopes(scopeValues)
	if err != nil {
		return nil, err
	}
	token := &domain.ApiToken{
		ID: id, TenantID: tenantID, UserID: userID, JTI: jti, ClientID: clientID,
		Scopes: scopes, Audience: audience, DPoPJKT: dpopJKT.String, Description: description.String, CreatedAt: createdAt,
	}
	if expiresAt.Valid {
		value := expiresAt.Time
		token.ExpiresAt = &value
	}
	if revokedAt.Valid {
		value := revokedAt.Time
		token.RevokedAt = &value
	}
	return token, nil
}

func (r *Repository) Save(ctx context.Context, token *domain.ApiToken) error {
	return New(r.Pool).SaveApiToken(ctx, SaveApiTokenParams{
		ID: token.ID, TenantID: token.TenantID,
		UserID: token.UserID, Jti: token.JTI, ClientID: token.ClientID, Scopes: token.Scopes.Strings(),
		Audience: token.Audience, DpopJkt: textOrNil(token.DPoPJKT), Description: textOrNil(token.Description),
		CreatedAt: token.CreatedAt, ExpiresAt: timestamptzOrNil(token.ExpiresAt), RevokedAt: timestamptzOrNil(token.RevokedAt),
	})
}

func (r *Repository) FindByJTI(ctx context.Context, tenantID, jti string) (*domain.ApiToken, error) {
	row, err := New(r.Pool).FindApiTokenByJTI(ctx, FindApiTokenByJTIParams{TenantID: tenantID, Jti: jti})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tokenFromRow(row.ID, row.TenantID, row.UserID, row.Jti, row.ClientID, row.Scopes, row.Audience, row.DpopJkt, row.Description, row.CreatedAt, row.ExpiresAt, row.RevokedAt)
}

func (r *Repository) List(ctx context.Context, tenantID string) ([]*domain.ApiToken, error) {
	rows, err := New(r.Pool).ListApiTokensByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.ApiToken, 0, len(rows))
	for _, row := range rows {
		token, err := tokenFromRow(row.ID, row.TenantID, row.UserID, row.Jti, row.ClientID, row.Scopes, row.Audience, row.DpopJkt, row.Description, row.CreatedAt, row.ExpiresAt, row.RevokedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, token)
	}
	return result, nil
}

func (r *Repository) Revoke(ctx context.Context, tenantID, id string, at time.Time) error {
	return New(r.Pool).RevokeApiToken(ctx, RevokeApiTokenParams{TenantID: tenantID, ID: id, RevokedAt: timestamptzOrNil(&at)})
}

func (r *Repository) RevokeByJTI(ctx context.Context, tenantID, jti string, at time.Time) error {
	return New(r.Pool).RevokeApiTokenByJTI(ctx, RevokeApiTokenByJTIParams{TenantID: tenantID, Jti: jti, RevokedAt: timestamptzOrNil(&at)})
}
