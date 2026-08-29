package db_postgres

import (
	"context"
	"errors"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/sourcing/scim/ports"
	"github.com/jackc/pgx/v5"
)

// ScimRepository は SCIM user-ref/group-ref を PostgreSQL に永続化する。クエリは
// sqlc 生成 (wi-176); Pool は DBTX を構造的に満たす。
type ScimRepository struct{ Pool sharedpg.DB }

func (r *ScimRepository) SaveUserRef(ctx context.Context, ref *ports.ScimUserRef) error {
	return New(r.Pool).SaveScimUserRef(ctx, SaveScimUserRefParams{
		TenantID: ref.TenantID, ScimID: ref.ScimID, UserID: ref.UserID,
	})
}

func (r *ScimRepository) FindUserRefByScimID(ctx context.Context, tenantID, scimID string) (*ports.ScimUserRef, error) {
	row, err := New(r.Pool).FindScimUserRefByScimID(ctx, FindScimUserRefByScimIDParams{TenantID: tenantID, ScimID: scimID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ports.ScimUserRef{TenantID: row.TenantID, ScimID: row.ScimID, UserID: row.UserID}, nil
}

func (r *ScimRepository) FindUserRefByUserID(ctx context.Context, tenantID, userID string) (*ports.ScimUserRef, error) {
	row, err := New(r.Pool).FindScimUserRefByUserID(ctx, FindScimUserRefByUserIDParams{TenantID: tenantID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ports.ScimUserRef{TenantID: row.TenantID, ScimID: row.ScimID, UserID: row.UserID}, nil
}

func (r *ScimRepository) FindUserRefsByUserIDs(ctx context.Context, tenantID string, userIDs []string) ([]*ports.ScimUserRef, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := r.Pool.Query(ctx, `SELECT tenant_id, scim_id, user_id FROM scim_user_refs
        WHERE tenant_id = $1 AND user_id = ANY($2::uuid[])`, tenantID, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*ports.ScimUserRef, 0, len(userIDs))
	for rows.Next() {
		ref := &ports.ScimUserRef{}
		if err := rows.Scan(&ref.TenantID, &ref.ScimID, &ref.UserID); err != nil {
			return nil, err
		}
		result = append(result, ref)
	}
	return result, rows.Err()
}

func (r *ScimRepository) DeleteUserRef(ctx context.Context, tenantID, scimID string) error {
	return New(r.Pool).DeleteScimUserRef(ctx, DeleteScimUserRefParams{TenantID: tenantID, ScimID: scimID})
}

func (r *ScimRepository) SaveGroupRef(ctx context.Context, ref *ports.ScimGroupRef) error {
	return New(r.Pool).SaveScimGroupRef(ctx, SaveScimGroupRefParams{
		TenantID: ref.TenantID, ScimID: ref.ScimID, GroupID: ref.GroupID,
	})
}

func (r *ScimRepository) FindGroupRefByScimID(ctx context.Context, tenantID, scimID string) (*ports.ScimGroupRef, error) {
	row, err := New(r.Pool).FindScimGroupRefByScimID(ctx, FindScimGroupRefByScimIDParams{TenantID: tenantID, ScimID: scimID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ports.ScimGroupRef{TenantID: row.TenantID, ScimID: row.ScimID, GroupID: row.GroupID}, nil
}

func (r *ScimRepository) FindGroupRefByGroupID(ctx context.Context, tenantID, groupID string) (*ports.ScimGroupRef, error) {
	row, err := New(r.Pool).FindScimGroupRefByGroupID(ctx, FindScimGroupRefByGroupIDParams{TenantID: tenantID, GroupID: groupID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ports.ScimGroupRef{TenantID: row.TenantID, ScimID: row.ScimID, GroupID: row.GroupID}, nil
}

func (r *ScimRepository) DeleteGroupRef(ctx context.Context, tenantID, scimID string) error {
	return New(r.Pool).DeleteScimGroupRef(ctx, DeleteScimGroupRefParams{TenantID: tenantID, ScimID: scimID})
}

// FindGroupRefsByGroupIDs は Group の所有権をまとめて解決する。CSV の計画器が
// 行ごとにリポジトリを引かないための一括問い合わせである。
func (r *ScimRepository) FindGroupRefsByGroupIDs(ctx context.Context, tenantID string, groupIDs []string) ([]*ports.ScimGroupRef, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	rows, err := r.Pool.Query(ctx, `SELECT tenant_id, scim_id, group_id FROM scim_group_refs
        WHERE tenant_id = $1 AND group_id = ANY($2::uuid[])`, tenantID, groupIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*ports.ScimGroupRef, 0, len(groupIDs))
	for rows.Next() {
		ref := &ports.ScimGroupRef{}
		if err := rows.Scan(&ref.TenantID, &ref.ScimID, &ref.GroupID); err != nil {
			return nil, err
		}
		result = append(result, ref)
	}
	return result, rows.Err()
}
