package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/backend/idmanagement/adapters/persistence/postgres/sqlcgen"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
)

// TenantUserAttributeSchemaRepository は tenant ごとの custom 属性定義を保持する
// (ADR-040 / wi-19)。定義一覧は attributes JSONB 列に格納する。クエリは sqlc 生成
// (wi-178, ADR-090); Pool は sqlcgen.DBTX を構造的に満たす。
type TenantUserAttributeSchemaRepository struct{ Pool sharedpg.DB }

func (r *TenantUserAttributeSchemaRepository) FindByTenant(
	ctx context.Context, tenantID string,
) (*idmdomain.TenantUserAttributeSchema, error) {
	row, err := sqlcgen.New(r.Pool).FindTenantUserAttributeSchemaByTenant(ctx, tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s := &idmdomain.TenantUserAttributeSchema{
		TenantID:  row.TenantID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if len(row.Attributes) > 0 {
		if err := json.Unmarshal(row.Attributes, &s.Attributes); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (r *TenantUserAttributeSchemaRepository) Save(ctx context.Context, s *idmdomain.TenantUserAttributeSchema) error {
	attributes, err := json.Marshal(s.Attributes)
	if err != nil {
		return err
	}
	return sqlcgen.New(r.Pool).SaveTenantUserAttributeSchema(ctx, sqlcgen.SaveTenantUserAttributeSchemaParams{
		TenantID:   s.TenantID,
		Attributes: attributes,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	})
}

func (r *TenantUserAttributeSchemaRepository) Delete(ctx context.Context, tenantID string) error {
	return sqlcgen.New(r.Pool).DeleteTenantUserAttributeSchema(ctx, tenantID)
}
