package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

// TenantRepository (Tenancy)。クエリは sqlc 生成 (wi-179); Pool は DBTX を
// 構造的に満たす。
type TenantRepository struct{ Pool sharedpg.DB }

func (r *TenantRepository) FindByID(ctx context.Context, id string) (*domain.Tenant, error) {
	row, err := New(r.Pool).FindTenantByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tenantFromRow(row)
}

func (r *TenantRepository) FindByRealm(ctx context.Context, realm string) (*domain.Tenant, error) {
	row, err := New(r.Pool).FindTenantByRealm(ctx, realm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tenantFromRow(row)
}

func (r *TenantRepository) FindAll(ctx context.Context) ([]*domain.Tenant, error) {
	rows, err := New(r.Pool).FindAllTenants(ctx)
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
	override, err := marshalPolicyOverride(tenant.PasswordPolicyOverride)
	if err != nil {
		return err
	}
	return New(r.Pool).SaveTenant(ctx, SaveTenantParams{
		ID:            tenant.ID,
		Realm:         tenant.Realm,
		DisplayName:   tenant.DisplayName,
		Status:        string(tenant.Status),
		DefaultLocale: textPtrOrNil(tenant.DefaultLocale),
		// ゼロ値の Tenant を 'path' として書く。列は NOT NULL なので、空文字列を
		// そのまま渡すと CHECK 制約で落ちる。
		EndpointStyle:           string(tenant.EffectiveEndpointStyle()),
		PasswordPolicyOverride:  override,
		PasswordPolicyUpdatedAt: timestamptzOrNil(tenant.PasswordPolicyUpdatedAt),
		CreatedAt:               tenant.CreatedAt,
		UpdatedAt:               tenant.UpdatedAt,
		DisabledAt:              timestamptzOrNil(tenant.DisabledAt),
	})
}

func tenantFromRow(row *Tenant) (*domain.Tenant, error) {
	tenant := &domain.Tenant{
		ID:            row.ID,
		Realm:         row.Realm,
		DisplayName:   row.DisplayName,
		Status:        domain.TenantStatus(row.Status),
		EndpointStyle: domain.TenantEndpointStyle(row.EndpointStyle),
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
	if row.DisabledAt.Valid {
		disabledAt := row.DisabledAt.Time
		tenant.DisabledAt = &disabledAt
	}
	if row.DefaultLocale.Valid && row.DefaultLocale.String != "" {
		defaultLocale := row.DefaultLocale.String
		tenant.DefaultLocale = &defaultLocale
	}
	override, err := unmarshalPolicyOverride(row.PasswordPolicyOverride)
	if err != nil {
		return nil, err
	}
	tenant.PasswordPolicyOverride = override
	if row.PasswordPolicyUpdatedAt.Valid {
		policyUpdatedAt := row.PasswordPolicyUpdatedAt.Time
		tenant.PasswordPolicyUpdatedAt = &policyUpdatedAt
	}
	return tenant, tenant.Validate()
}

// marshalPolicyOverride maps "no override" to SQL NULL, so a cleared override and
// one that was never set are the same row state.
func marshalPolicyOverride(override *domain.PasswordPolicyOverride) ([]byte, error) {
	if override == nil {
		return nil, nil
	}
	return json.Marshal(override)
}

func unmarshalPolicyOverride(raw []byte) (*domain.PasswordPolicyOverride, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var override domain.PasswordPolicyOverride
	if err := json.Unmarshal(raw, &override); err != nil {
		return nil, err
	}
	return &override, nil
}

// textPtrOrNil maps an unset or blank optional string to SQL NULL, so "cleared"
// and "never set" are the same row state.
func textPtrOrNil(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return textOrNil(*value)
}

func timestamptzOrNil(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
