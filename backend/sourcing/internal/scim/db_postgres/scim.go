package db_postgres

import (
	"context"
	"errors"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	scimports "github.com/ambi/idmagic/backend/sourcing/scim/ports"
	"github.com/jackc/pgx/v5"
)

// ScimRepository は SCIM user-ref/group-ref を PostgreSQL に永続化する。クエリは
// sqlc 生成 (wi-176); Pool は DBTX を構造的に満たす。
type ScimRepository struct{ Pool sharedpg.DB }

func (r *ScimRepository) SaveUserRef(ctx context.Context, ref *scimports.ScimUserRef) error {
	return New(r.Pool).SaveScimUserRef(ctx, SaveScimUserRefParams{
		TenantID: ref.TenantID, ScimID: ref.ScimID, UserID: ref.UserID,
	})
}

func (r *ScimRepository) FindUserRefByScimID(ctx context.Context, tenantID, scimID string) (*scimports.ScimUserRef, error) {
	row, err := New(r.Pool).FindScimUserRefByScimID(ctx, FindScimUserRefByScimIDParams{TenantID: tenantID, ScimID: scimID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &scimports.ScimUserRef{TenantID: row.TenantID, ScimID: row.ScimID, UserID: row.UserID}, nil
}

func (r *ScimRepository) FindUserRefByUserID(ctx context.Context, tenantID, userID string) (*scimports.ScimUserRef, error) {
	row, err := New(r.Pool).FindScimUserRefByUserID(ctx, FindScimUserRefByUserIDParams{TenantID: tenantID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &scimports.ScimUserRef{TenantID: row.TenantID, ScimID: row.ScimID, UserID: row.UserID}, nil
}

func (r *ScimRepository) FindUserRefsByUserIDs(ctx context.Context, tenantID string, userIDs []string) ([]*scimports.ScimUserRef, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := New(r.Pool).ListScimUserRefsByUserIDs(ctx, ListScimUserRefsByUserIDsParams{TenantID: tenantID, UserIds: userIDs})
	if err != nil {
		return nil, err
	}
	result := make([]*scimports.ScimUserRef, 0, len(rows))
	for _, row := range rows {
		result = append(result, &scimports.ScimUserRef{TenantID: row.TenantID, ScimID: row.ScimID, UserID: row.UserID})
	}
	return result, nil
}

func (r *ScimRepository) DeleteUserRef(ctx context.Context, tenantID, scimID string) error {
	return New(r.Pool).DeleteScimUserRef(ctx, DeleteScimUserRefParams{TenantID: tenantID, ScimID: scimID})
}

func (r *ScimRepository) SaveGroupRef(ctx context.Context, ref *scimports.ScimGroupRef) error {
	return New(r.Pool).SaveScimGroupRef(ctx, SaveScimGroupRefParams{
		TenantID: ref.TenantID, ScimID: ref.ScimID, GroupID: ref.GroupID,
	})
}

func (r *ScimRepository) FindGroupRefByScimID(ctx context.Context, tenantID, scimID string) (*scimports.ScimGroupRef, error) {
	row, err := New(r.Pool).FindScimGroupRefByScimID(ctx, FindScimGroupRefByScimIDParams{TenantID: tenantID, ScimID: scimID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &scimports.ScimGroupRef{TenantID: row.TenantID, ScimID: row.ScimID, GroupID: row.GroupID}, nil
}

func (r *ScimRepository) FindGroupRefByGroupID(ctx context.Context, tenantID, groupID string) (*scimports.ScimGroupRef, error) {
	row, err := New(r.Pool).FindScimGroupRefByGroupID(ctx, FindScimGroupRefByGroupIDParams{TenantID: tenantID, GroupID: groupID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &scimports.ScimGroupRef{TenantID: row.TenantID, ScimID: row.ScimID, GroupID: row.GroupID}, nil
}

func (r *ScimRepository) DeleteGroupRef(ctx context.Context, tenantID, scimID string) error {
	return New(r.Pool).DeleteScimGroupRef(ctx, DeleteScimGroupRefParams{TenantID: tenantID, ScimID: scimID})
}

// FindGroupRefsByGroupIDs は Group の所有権をまとめて解決する。CSV の計画器が
// 行ごとにリポジトリを引かないための一括問い合わせである。
func (r *ScimRepository) FindGroupRefsByGroupIDs(ctx context.Context, tenantID string, groupIDs []string) ([]*scimports.ScimGroupRef, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	rows, err := New(r.Pool).ListScimGroupRefsByGroupIDs(ctx, ListScimGroupRefsByGroupIDsParams{TenantID: tenantID, GroupIds: groupIDs})
	if err != nil {
		return nil, err
	}
	result := make([]*scimports.ScimGroupRef, 0, len(rows))
	for _, row := range rows {
		result = append(result, &scimports.ScimGroupRef{TenantID: row.TenantID, ScimID: row.ScimID, GroupID: row.GroupID})
	}
	return result, nil
}
