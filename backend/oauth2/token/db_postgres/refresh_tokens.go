package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	oauth2pg "github.com/ambi/idmagic/backend/oauth2/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// RefreshTokenStore は sqlc 生成クエリを利用する OAuth2 refresh token repository。
type RefreshTokenStore struct{ Pool sharedpg.DB }

func (s *RefreshTokenStore) queries() *oauth2pg.Queries { return oauth2pg.New(s.Pool) }

func (s *RefreshTokenStore) FindByHash(ctx context.Context, hash string) (*domain.RefreshTokenRecord, error) {
	row, err := s.queries().GetRefreshTokenByHash(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return refreshFromRow(row)
}

func (s *RefreshTokenStore) Save(ctx context.Context, rec *domain.RefreshTokenRecord) error {
	params, err := refreshInsertParams(rec)
	if err != nil {
		return err
	}
	return s.queries().InsertRefreshToken(ctx, params)
}

func (s *RefreshTokenStore) Rotate(ctx context.Context, parentID string, next *domain.RefreshTokenRecord) (*domain.RefreshTokenRecord, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := oauth2pg.New(tx)
	state, err := queries.GetRefreshTokenRotationState(ctx, parentID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (state.Rotated || state.Revoked)) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := queries.MarkRefreshTokenRotated(ctx, parentID); err != nil {
		return nil, err
	}
	params, err := refreshInsertParams(next)
	if err != nil {
		return nil, err
	}
	if err := queries.InsertRefreshToken(ctx, params); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return next, nil
}

func (s *RefreshTokenStore) RevokeFamily(ctx context.Context, familyID string) error {
	return s.queries().RevokeRefreshTokenFamily(ctx, familyID)
}

func (s *RefreshTokenStore) RevokeBySid(ctx context.Context, sid string) error {
	id := pgtype.UUID{}
	if err := id.Scan(sid); err != nil {
		return err
	}
	return s.queries().RevokeRefreshTokensBySid(ctx, id)
}

func (s *RefreshTokenStore) DeleteAllForSub(ctx context.Context, sub string) error {
	return s.queries().DeleteRefreshTokensForSub(ctx, sub)
}

func refreshFromRow(row *oauth2pg.GetRefreshTokenByHashRow) (*domain.RefreshTokenRecord, error) {
	rec := &domain.RefreshTokenRecord{
		ID: row.ID, Hash: row.Hash, FamilyID: row.FamilyID,
		ClientID: row.ClientID, UserID: row.UserID, IssuedAt: row.IssuedAt, ExpiresAt: row.ExpiresAt,
		AbsoluteExpiresAt: row.AbsoluteExpiresAt, Revoked: row.Revoked, Rotated: row.Rotated,
	}
	if parentID, ok := row.ParentID.(string); ok && parentID != "" {
		rec.ParentID = &parentID
	}
	if sid, ok := row.Sid.(string); ok && sid != "" {
		rec.Sid = &sid
	}
	if row.Resource.Valid {
		rec.Resource = &row.Resource.String
	}
	if err := json.Unmarshal(row.Scopes, &rec.Scopes); err != nil {
		return nil, err
	}
	if len(row.SenderConstraint) > 0 {
		if err := json.Unmarshal(row.SenderConstraint, &rec.SenderConstraint); err != nil {
			return nil, err
		}
	}
	return rec, rec.Validate()
}

func refreshInsertParams(rec *domain.RefreshTokenRecord) (oauth2pg.InsertRefreshTokenParams, error) {
	scopes, err := json.Marshal(rec.Scopes)
	if err != nil {
		return oauth2pg.InsertRefreshTokenParams{}, err
	}
	constraint, err := json.Marshal(rec.SenderConstraint)
	if err != nil {
		return oauth2pg.InsertRefreshTokenParams{}, err
	}
	parentID := pgtype.UUID{}
	if rec.ParentID != nil {
		_ = parentID.Scan(*rec.ParentID)
	}
	sid := pgtype.UUID{}
	if rec.Sid != nil {
		_ = sid.Scan(*rec.Sid)
	}
	resource := pgtype.Text{}
	if rec.Resource != nil {
		resource = pgtype.Text{String: *rec.Resource, Valid: true}
	}
	return oauth2pg.InsertRefreshTokenParams{
		ID: rec.ID, Hash: rec.Hash, FamilyID: rec.FamilyID,
		ParentID: parentID, ClientID: rec.ClientID, UserID: rec.UserID, Scopes: scopes, IssuedAt: rec.IssuedAt,
		ExpiresAt: rec.ExpiresAt, AbsoluteExpiresAt: rec.AbsoluteExpiresAt, Revoked: rec.Revoked, Rotated: rec.Rotated,
		Column13: string(constraint), Sid: sid, Resource: resource,
	}, nil
}
