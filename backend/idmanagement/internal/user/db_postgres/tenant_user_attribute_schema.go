package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
)

// TenantUserAttributeSchemaRepository は tenant ごとの custom 属性定義を保持する
// (wi-19)。定義一覧は attributes JSONB 列に格納する。クエリは sqlc 生成
// (wi-178); Pool は DBTX を構造的に満たす。
type TenantUserAttributeSchemaRepository struct{ Pool sharedpg.DB }

func (r *TenantUserAttributeSchemaRepository) FindByTenant(
	ctx context.Context, tenantID string,
) (*userdomain.TenantUserAttributeSchema, error) {
	return findUserAttributeSchema(ctx, New(r.Pool), tenantID)
}

func findUserAttributeSchema(ctx context.Context, q *Queries, tenantID string) (*userdomain.TenantUserAttributeSchema, error) {
	row, err := q.FindTenantUserAttributeSchemaByTenant(ctx, tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s := &userdomain.TenantUserAttributeSchema{
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

// Save は、定義と一意性の表を一つのトランザクションで保存する。unique の属性の値を、削除されて
// いない User から写し直し、値の重なる属性は AttributeValuesNotUniqueError で拒否する。
func (r *TenantUserAttributeSchemaRepository) Save(ctx context.Context, s *userdomain.TenantUserAttributeSchema) error {
	attributes, err := json.Marshal(s.Attributes)
	if err != nil {
		return err
	}
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	if err := q.LockUniqueAttributeValuesExclusive(ctx, s.TenantID); err != nil {
		return err
	}
	if err := q.SaveTenantUserAttributeSchema(ctx, SaveTenantUserAttributeSchemaParams{
		TenantID:   s.TenantID,
		Attributes: attributes,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}); err != nil {
		return err
	}
	if err := q.DeleteTenantUniqueAttributeValues(ctx, s.TenantID); err != nil {
		return err
	}
	for _, key := range s.UniqueKeys() {
		err := q.CopyUniqueAttributeValues(ctx, CopyUniqueAttributeValuesParams{AttributeKey: key, TenantID: s.TenantID})
		if isUniqueAttributeValueViolation(err) {
			return &userports.AttributeValuesNotUniqueError{Key: key}
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// Delete は、定義とテナントの一意性の表を消す。
func (r *TenantUserAttributeSchemaRepository) Delete(ctx context.Context, tenantID string) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	if err := q.LockUniqueAttributeValuesExclusive(ctx, tenantID); err != nil {
		return err
	}
	if err := q.DeleteTenantUniqueAttributeValues(ctx, tenantID); err != nil {
		return err
	}
	if err := q.DeleteTenantUserAttributeSchema(ctx, tenantID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
