package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/internal/scim/ports"
)

type ScimRepository struct{ Pool DB }

func (r *ScimRepository) GetConfig(ctx context.Context, tenantID string) (*ports.ScimConfig, error) {
	var cfg ports.ScimConfig
	err := r.Pool.QueryRow(ctx, "SELECT tenant_id, enabled, last_sync_at, error_count, created_at, updated_at FROM scim_configs WHERE tenant_id=$1", tenantID).
		Scan(&cfg.TenantID, &cfg.Enabled, &cfg.LastSyncAt, &cfg.ErrorCount, &cfg.CreatedAt, &cfg.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return &ports.ScimConfig{TenantID: tenantID, Enabled: false}, nil
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *ScimRepository) SaveConfig(ctx context.Context, config *ports.ScimConfig) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO scim_configs (tenant_id, enabled, last_sync_at, error_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id) DO UPDATE SET
			enabled=EXCLUDED.enabled,
			last_sync_at=EXCLUDED.last_sync_at,
			error_count=EXCLUDED.error_count,
			updated_at=EXCLUDED.updated_at
	`, config.TenantID, config.Enabled, config.LastSyncAt, config.ErrorCount, config.CreatedAt, config.UpdatedAt)
	return err
}

func (r *ScimRepository) SaveToken(ctx context.Context, token *ports.ScimToken) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO scim_tokens (id, tenant_id, token_hash, description, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			token_hash=EXCLUDED.token_hash,
			description=EXCLUDED.description,
			expires_at=EXCLUDED.expires_at,
			updated_at=now()
	`, token.ID, token.TenantID, token.TokenHash, token.Description, token.CreatedAt, token.ExpiresAt)
	return err
}

func (r *ScimRepository) FindToken(ctx context.Context, tokenHash string) (*ports.ScimToken, error) {
	var tok ports.ScimToken
	err := r.Pool.QueryRow(ctx, "SELECT id, tenant_id, token_hash, description, created_at, expires_at FROM scim_tokens WHERE token_hash=$1", tokenHash).
		Scan(&tok.ID, &tok.TenantID, &tok.TokenHash, &tok.Description, &tok.CreatedAt, &tok.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &tok, nil
}

func (r *ScimRepository) ListTokens(ctx context.Context, tenantID string) ([]*ports.ScimToken, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id, tenant_id, token_hash, description, created_at, expires_at FROM scim_tokens WHERE tenant_id=$1", tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tokens []*ports.ScimToken
	for rows.Next() {
		var tok ports.ScimToken
		if err := rows.Scan(&tok.ID, &tok.TenantID, &tok.TokenHash, &tok.Description, &tok.CreatedAt, &tok.ExpiresAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, &tok)
	}
	return tokens, rows.Err()
}

func (r *ScimRepository) DeleteToken(ctx context.Context, tenantID, id string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM scim_tokens WHERE tenant_id=$1 AND id=$2", tenantID, id)
	return err
}

func (r *ScimRepository) SaveUserRef(ctx context.Context, ref *ports.ScimUserRef) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO scim_user_refs (tenant_id, scim_id, user_sub)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, scim_id) DO UPDATE SET
			user_sub=EXCLUDED.user_sub,
			updated_at=now()
	`, ref.TenantID, ref.ScimID, ref.UserID)
	return err
}

func (r *ScimRepository) FindUserRefByScimID(ctx context.Context, tenantID, scimID string) (*ports.ScimUserRef, error) {
	var ref ports.ScimUserRef
	err := r.Pool.QueryRow(ctx, "SELECT tenant_id, scim_id, user_sub FROM scim_user_refs WHERE tenant_id=$1 AND scim_id=$2", tenantID, scimID).
		Scan(&ref.TenantID, &ref.ScimID, &ref.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func (r *ScimRepository) FindUserRefByUserID(ctx context.Context, tenantID, userID string) (*ports.ScimUserRef, error) {
	var ref ports.ScimUserRef
	err := r.Pool.QueryRow(ctx, "SELECT tenant_id, scim_id, user_sub FROM scim_user_refs WHERE tenant_id=$1 AND user_sub=$2", tenantID, userID).
		Scan(&ref.TenantID, &ref.ScimID, &ref.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func (r *ScimRepository) DeleteUserRef(ctx context.Context, tenantID, scimID string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM scim_user_refs WHERE tenant_id=$1 AND scim_id=$2", tenantID, scimID)
	return err
}

func (r *ScimRepository) SaveGroupRef(ctx context.Context, ref *ports.ScimGroupRef) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO scim_group_refs (tenant_id, scim_id, group_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, scim_id) DO UPDATE SET
			group_id=EXCLUDED.group_id,
			updated_at=now()
	`, ref.TenantID, ref.ScimID, ref.GroupID)
	return err
}

func (r *ScimRepository) FindGroupRefByScimID(ctx context.Context, tenantID, scimID string) (*ports.ScimGroupRef, error) {
	var ref ports.ScimGroupRef
	err := r.Pool.QueryRow(ctx, "SELECT tenant_id, scim_id, group_id FROM scim_group_refs WHERE tenant_id=$1 AND scim_id=$2", tenantID, scimID).
		Scan(&ref.TenantID, &ref.ScimID, &ref.GroupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func (r *ScimRepository) FindGroupRefByGroupID(ctx context.Context, tenantID, groupID string) (*ports.ScimGroupRef, error) {
	var ref ports.ScimGroupRef
	err := r.Pool.QueryRow(ctx, "SELECT tenant_id, scim_id, group_id FROM scim_group_refs WHERE tenant_id=$1 AND group_id=$2", tenantID, groupID).
		Scan(&ref.TenantID, &ref.ScimID, &ref.GroupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func (r *ScimRepository) DeleteGroupRef(ctx context.Context, tenantID, scimID string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM scim_group_refs WHERE tenant_id=$1 AND scim_id=$2", tenantID, scimID)
	return err
}
