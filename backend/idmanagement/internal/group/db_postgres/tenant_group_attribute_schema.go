package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
)

// TenantGroupAttributeSchemaRepository は tenant ごとの Group custom 属性定義を保持する
// (wi-315)。定義一覧は attributes JSONB 列に格納する。クエリは sqlc 生成; Pool は
// DBTX を構造的に満たす。
type TenantGroupAttributeSchemaRepository struct{ Pool sharedpg.DB }

func (r *TenantGroupAttributeSchemaRepository) Save(ctx context.Context, schema *groupdomain.TenantGroupAttributeSchema) error {
	_, err := r.Update(ctx, schema.TenantID, func(_ *groupdomain.TenantGroupAttributeSchema) (*groupdomain.TenantGroupAttributeSchema, error) {
		return schema, nil
	})
	return err
}

func (r *TenantGroupAttributeSchemaRepository) Update(ctx context.Context, tenantID string, change func(*groupdomain.TenantGroupAttributeSchema) (*groupdomain.TenantGroupAttributeSchema, error)) (*groupdomain.TenantGroupAttributeSchema, error) {
	return sharedpg.InTransaction(ctx, r.Pool, func(tx pgx.Tx) (*groupdomain.TenantGroupAttributeSchema, error) {
		if _, err := New(tx).LockTenantForGroupAttributeSchema(ctx, tenantID); err != nil {
			return nil, err
		}
		bound := &TenantGroupAttributeSchemaRepository{Pool: sharedpg.BindTransaction(tx)}
		current, err := bound.FindByTenant(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		updated, err := change(current)
		if err != nil {
			return nil, err
		}
		return updated, bound.save(ctx, updated)
	})
}

func (r *TenantGroupAttributeSchemaRepository) FindByTenant(
	ctx context.Context, tenantID string,
) (*groupdomain.TenantGroupAttributeSchema, error) {
	row, err := New(r.Pool).FindTenantGroupAttributeSchemaByTenant(ctx, tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s := &groupdomain.TenantGroupAttributeSchema{
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

func (r *TenantGroupAttributeSchemaRepository) save(ctx context.Context, s *groupdomain.TenantGroupAttributeSchema) error {
	attributes, err := json.Marshal(s.Attributes)
	if err != nil {
		return err
	}
	return New(r.Pool).SaveTenantGroupAttributeSchema(ctx, SaveTenantGroupAttributeSchemaParams{
		TenantID:   s.TenantID,
		Attributes: attributes,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	})
}

func (r *TenantGroupAttributeSchemaRepository) Delete(ctx context.Context, tenantID string) error {
	return New(r.Pool).DeleteTenantGroupAttributeSchema(ctx, tenantID)
}
