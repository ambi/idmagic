package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/oauth2/adapters/persistence/postgres/sqlcgen"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
)

// AuthorizationDetailTypeRepository は RFC 9396 authorization_details の type 定義
// (ADR-050) を PostgreSQL に永続化する。schema は JSONB として保持する。すべての
// 参照はテナント境界に閉じる。クエリは sqlc 生成 (wi-173, ADR-090)。
type AuthorizationDetailTypeRepository struct{ Pool sharedpg.DB }

func textOrNil(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func authorizationDetailTypeFromRow(row *sqlcgen.AuthorizationDetailType) (*domain.AuthorizationDetailType, error) {
	t := &domain.AuthorizationDetailType{
		TenantID:        row.TenantID,
		Type:            row.Type,
		DisplayTemplate: row.DisplayTemplate,
		State:           domain.AuthorizationDetailTypeState(row.State),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
	if row.Description.Valid {
		t.Description = row.Description.String
	}
	if len(row.Schema) > 0 {
		if err := json.Unmarshal(row.Schema, &t.Schema); err != nil {
			return nil, err
		}
	}
	return t, t.Validate()
}

func (r *AuthorizationDetailTypeRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.AuthorizationDetailType, error) {
	rows, err := sqlcgen.New(r.Pool).ListAuthorizationDetailTypesByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.AuthorizationDetailType, 0, len(rows))
	for _, row := range rows {
		t, err := authorizationDetailTypeFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *AuthorizationDetailTypeRepository) FindByType(ctx context.Context, tenantID, detailType string) (*domain.AuthorizationDetailType, error) {
	row, err := sqlcgen.New(r.Pool).GetAuthorizationDetailType(ctx, sqlcgen.GetAuthorizationDetailTypeParams{
		TenantID: tenantID, Type: detailType,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return authorizationDetailTypeFromRow(row)
}

func (r *AuthorizationDetailTypeRepository) Save(ctx context.Context, t *domain.AuthorizationDetailType) error {
	schema, err := json.Marshal(t.Schema)
	if err != nil {
		return err
	}
	return sqlcgen.New(r.Pool).UpsertAuthorizationDetailType(ctx, sqlcgen.UpsertAuthorizationDetailTypeParams{
		TenantID:        t.TenantID,
		Type:            t.Type,
		Description:     textOrNil(&t.Description),
		Schema:          schema,
		DisplayTemplate: t.DisplayTemplate,
		State:           string(t.State),
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	})
}

func (r *AuthorizationDetailTypeRepository) Delete(ctx context.Context, tenantID, detailType string) error {
	return sqlcgen.New(r.Pool).DeleteAuthorizationDetailType(ctx, sqlcgen.DeleteAuthorizationDetailTypeParams{
		TenantID: tenantID, Type: detailType,
	})
}
