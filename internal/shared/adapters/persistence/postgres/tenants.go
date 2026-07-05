package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/internal/shared/spec"
)

// TenantRepository (Tenancy)
type TenantRepository struct{ Pool DB }

func (r *TenantRepository) FindByID(ctx context.Context, id string) (*spec.Tenant, error) {
	return scanTenant(r.Pool.QueryRow(ctx, tenantSelect+" WHERE id=$1", id))
}

func (r *TenantRepository) FindByRealm(ctx context.Context, realm string) (*spec.Tenant, error) {
	return scanTenant(r.Pool.QueryRow(ctx, tenantSelect+" WHERE realm=$1", realm))
}

func (r *TenantRepository) FindAll(ctx context.Context) ([]*spec.Tenant, error) {
	rows, err := r.Pool.Query(ctx, tenantSelect+" ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*spec.Tenant{}
	for rows.Next() {
		tenant, err := scanTenant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, tenant)
	}
	return out, rows.Err()
}

func (r *TenantRepository) Save(ctx context.Context, tenant *spec.Tenant) error {
	_, err := r.Pool.Exec(ctx, `INSERT INTO tenants
(id,realm,display_name,status,created_at,updated_at,disabled_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (id) DO UPDATE SET realm=EXCLUDED.realm,display_name=EXCLUDED.display_name,
status=EXCLUDED.status,updated_at=EXCLUDED.updated_at,disabled_at=EXCLUDED.disabled_at`,
		tenant.ID, tenant.Realm, tenant.DisplayName, tenant.Status, tenant.CreatedAt, tenant.UpdatedAt, tenant.DisabledAt)
	return err
}

const tenantSelect = `SELECT id,realm,display_name,status,created_at,updated_at,disabled_at FROM tenants`

func scanTenant(row rowScanner) (*spec.Tenant, error) {
	var tenant spec.Tenant
	err := row.Scan(&tenant.ID, &tenant.Realm, &tenant.DisplayName, &tenant.Status, &tenant.CreatedAt,
		&tenant.UpdatedAt, &tenant.DisabledAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &tenant, tenant.Validate()
}
