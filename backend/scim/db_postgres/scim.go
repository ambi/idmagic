package db_postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/scim/db_postgres/sqlcgen"
	"github.com/ambi/idmagic/backend/scim/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// ScimRepository は SCIM token/user-ref/group-ref を PostgreSQL に永続化する。クエリは
// sqlc 生成 (wi-176, ADR-090); Pool は sqlcgen.DBTX を構造的に満たす。
type ScimRepository struct{ Pool sharedpg.DB }

func timestamptzOrNil(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func tokenFromRow(id, tenantID, tokenHash string, description pgtype.Text, createdAt time.Time, expiresAt pgtype.Timestamptz) *ports.ScimToken {
	tok := &ports.ScimToken{
		ID:          id,
		TenantID:    tenantID,
		TokenHash:   tokenHash,
		Description: description.String,
		CreatedAt:   createdAt,
	}
	if expiresAt.Valid {
		expires := expiresAt.Time
		tok.ExpiresAt = &expires
	}
	return tok
}

func (r *ScimRepository) SaveToken(ctx context.Context, token *ports.ScimToken) error {
	return sqlcgen.New(r.Pool).SaveScimToken(ctx, sqlcgen.SaveScimTokenParams{
		ID:          token.ID,
		TenantID:    token.TenantID,
		TokenHash:   token.TokenHash,
		Description: pgtype.Text{String: token.Description, Valid: true},
		CreatedAt:   token.CreatedAt,
		ExpiresAt:   timestamptzOrNil(token.ExpiresAt),
	})
}

func (r *ScimRepository) FindToken(ctx context.Context, tokenHash string) (*ports.ScimToken, error) {
	row, err := sqlcgen.New(r.Pool).FindScimTokenByHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tokenFromRow(row.ID, row.TenantID, row.TokenHash, row.Description, row.CreatedAt, row.ExpiresAt), nil
}

func (r *ScimRepository) ListTokens(ctx context.Context, tenantID string) ([]*ports.ScimToken, error) {
	rows, err := sqlcgen.New(r.Pool).ListScimTokensByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*ports.ScimToken, 0, len(rows))
	for _, row := range rows {
		out = append(out, tokenFromRow(row.ID, row.TenantID, row.TokenHash, row.Description, row.CreatedAt, row.ExpiresAt))
	}
	return out, nil
}

func (r *ScimRepository) DeleteToken(ctx context.Context, tenantID, id string) error {
	return sqlcgen.New(r.Pool).DeleteScimToken(ctx, sqlcgen.DeleteScimTokenParams{TenantID: tenantID, ID: id})
}

func (r *ScimRepository) SaveUserRef(ctx context.Context, ref *ports.ScimUserRef) error {
	return sqlcgen.New(r.Pool).SaveScimUserRef(ctx, sqlcgen.SaveScimUserRefParams{
		TenantID: ref.TenantID, ScimID: ref.ScimID, UserID: ref.UserID,
	})
}

func (r *ScimRepository) FindUserRefByScimID(ctx context.Context, tenantID, scimID string) (*ports.ScimUserRef, error) {
	row, err := sqlcgen.New(r.Pool).FindScimUserRefByScimID(ctx, sqlcgen.FindScimUserRefByScimIDParams{TenantID: tenantID, ScimID: scimID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ports.ScimUserRef{TenantID: row.TenantID, ScimID: row.ScimID, UserID: row.UserID}, nil
}

func (r *ScimRepository) FindUserRefByUserID(ctx context.Context, tenantID, userID string) (*ports.ScimUserRef, error) {
	row, err := sqlcgen.New(r.Pool).FindScimUserRefByUserID(ctx, sqlcgen.FindScimUserRefByUserIDParams{TenantID: tenantID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ports.ScimUserRef{TenantID: row.TenantID, ScimID: row.ScimID, UserID: row.UserID}, nil
}

func (r *ScimRepository) DeleteUserRef(ctx context.Context, tenantID, scimID string) error {
	return sqlcgen.New(r.Pool).DeleteScimUserRef(ctx, sqlcgen.DeleteScimUserRefParams{TenantID: tenantID, ScimID: scimID})
}

func (r *ScimRepository) SaveGroupRef(ctx context.Context, ref *ports.ScimGroupRef) error {
	return sqlcgen.New(r.Pool).SaveScimGroupRef(ctx, sqlcgen.SaveScimGroupRefParams{
		TenantID: ref.TenantID, ScimID: ref.ScimID, GroupID: ref.GroupID,
	})
}

func (r *ScimRepository) FindGroupRefByScimID(ctx context.Context, tenantID, scimID string) (*ports.ScimGroupRef, error) {
	row, err := sqlcgen.New(r.Pool).FindScimGroupRefByScimID(ctx, sqlcgen.FindScimGroupRefByScimIDParams{TenantID: tenantID, ScimID: scimID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ports.ScimGroupRef{TenantID: row.TenantID, ScimID: row.ScimID, GroupID: row.GroupID}, nil
}

func (r *ScimRepository) FindGroupRefByGroupID(ctx context.Context, tenantID, groupID string) (*ports.ScimGroupRef, error) {
	row, err := sqlcgen.New(r.Pool).FindScimGroupRefByGroupID(ctx, sqlcgen.FindScimGroupRefByGroupIDParams{TenantID: tenantID, GroupID: groupID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ports.ScimGroupRef{TenantID: row.TenantID, ScimID: row.ScimID, GroupID: row.GroupID}, nil
}

func (r *ScimRepository) DeleteGroupRef(ctx context.Context, tenantID, scimID string) error {
	return sqlcgen.New(r.Pool).DeleteScimGroupRef(ctx, sqlcgen.DeleteScimGroupRefParams{TenantID: tenantID, ScimID: scimID})
}
