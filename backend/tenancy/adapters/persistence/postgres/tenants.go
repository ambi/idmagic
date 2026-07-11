package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/tenancy/adapters/persistence/postgres/sqlcgen"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

// TenantRepository (Tenancy)。クエリは sqlc 生成 (wi-179, ADR-090); Pool は sqlcgen.DBTX を
// 構造的に満たす。
type TenantRepository struct{ Pool sharedpg.DB }

func (r *TenantRepository) FindByID(ctx context.Context, id string) (*domain.Tenant, error) {
	row, err := sqlcgen.New(r.Pool).FindTenantByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tenantFromRow(row)
}

func (r *TenantRepository) FindByRealm(ctx context.Context, realm string) (*domain.Tenant, error) {
	row, err := sqlcgen.New(r.Pool).FindTenantByRealm(ctx, realm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tenantFromRow(row)
}

func (r *TenantRepository) FindAll(ctx context.Context) ([]*domain.Tenant, error) {
	rows, err := sqlcgen.New(r.Pool).FindAllTenants(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Tenant, 0, len(rows))
	for _, row := range rows {
		tenant, err := tenantFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, tenant)
	}
	return out, nil
}

func (r *TenantRepository) Save(ctx context.Context, tenant *domain.Tenant) error {
	return sqlcgen.New(r.Pool).SaveTenant(ctx, sqlcgen.SaveTenantParams{
		ID:          tenant.ID,
		Realm:       tenant.Realm,
		DisplayName: tenant.DisplayName,
		Status:      string(tenant.Status),
		CreatedAt:   tenant.CreatedAt,
		UpdatedAt:   tenant.UpdatedAt,
		DisabledAt:  timestamptzOrNil(tenant.DisabledAt),
	})
}

func tenantFromRow(row *sqlcgen.Tenant) (*domain.Tenant, error) {
	tenant := &domain.Tenant{
		ID:          row.ID,
		Realm:       row.Realm,
		DisplayName: row.DisplayName,
		Status:      domain.TenantStatus(row.Status),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if row.DisabledAt.Valid {
		disabledAt := row.DisabledAt.Time
		tenant.DisabledAt = &disabledAt
	}
	return tenant, tenant.Validate()
}

func timestamptzOrNil(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
