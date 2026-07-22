package db_postgres

import (
	"context"
	"errors"

	"github.com/ambi/idmagic/backend/scim/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
)

// ScimRepository は SCIM user-ref/group-ref を PostgreSQL に永続化する。クエリは
// sqlc 生成 (wi-176, ADR-090); Pool は DBTX を構造的に満たす。
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
