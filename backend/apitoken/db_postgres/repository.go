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

func tokenFromRow(
	id, tenantID, tokenHash string,
	scopeValues []string,
	description pgtype.Text,
	createdAt time.Time,
	expiresAt pgtype.Timestamptz,
) (*domain.ApiToken, error) {
	scopes, err := domain.ParseScopes(scopeValues)
	if err != nil {
		return nil, err
	}
	token := &domain.ApiToken{
		ID: id, TenantID: tenantID, TokenHash: tokenHash, Scopes: scopes,
		Description: description.String, CreatedAt: createdAt,
	}
	if expiresAt.Valid {
		value := expiresAt.Time
		token.ExpiresAt = &value
	}
	return token, nil
}

func (r *Repository) Save(ctx context.Context, token *domain.ApiToken) error {
	return New(r.Pool).SaveApiToken(ctx, SaveApiTokenParams{
		ID: token.ID, TenantID: token.TenantID, TokenHash: token.TokenHash,
		Scopes: token.Scopes.Strings(), Description: pgtype.Text{String: token.Description, Valid: token.Description != ""},
		CreatedAt: token.CreatedAt, ExpiresAt: timestamptzOrNil(token.ExpiresAt),
	})
}

func (r *Repository) FindByHash(ctx context.Context, tokenHash string) (*domain.ApiToken, error) {
	row, err := New(r.Pool).FindApiTokenByHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tokenFromRow(row.ID, row.TenantID, row.TokenHash, row.Scopes, row.Description, row.CreatedAt, row.ExpiresAt)
}

func (r *Repository) List(ctx context.Context, tenantID string) ([]*domain.ApiToken, error) {
	rows, err := New(r.Pool).ListApiTokensByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.ApiToken, 0, len(rows))
	for _, row := range rows {
		token, err := tokenFromRow(row.ID, row.TenantID, row.TokenHash, row.Scopes, row.Description, row.CreatedAt, row.ExpiresAt)
		if err != nil {
			return nil, err
		}
		result = append(result, token)
	}
	return result, nil
}

func (r *Repository) Delete(ctx context.Context, tenantID, id string) error {
	return New(r.Pool).DeleteApiToken(ctx, DeleteApiTokenParams{TenantID: tenantID, ID: id})
}
