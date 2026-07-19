package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/backend/oauth2/adapters/persistence/postgres/sqlcgen"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
)

// McpResourceServerRepository は MCP resource server 登録 (ADR-055) を PostgreSQL に
// 永続化する。scopes は JSONB として保持する。すべての参照はテナント境界に閉じる。
// クエリは sqlc 生成 (wi-173, ADR-090)。
type McpResourceServerRepository struct{ Pool sharedpg.DB }

func mcpResourceServerFromRow(row *sqlcgen.McpResourceServer) (*domain.McpResourceServer, error) {
	m := &domain.McpResourceServer{
		TenantID:         row.TenantID,
		ResourceServerID: row.ResourceServerID,
		Resource:         row.Resource,
		Name:             row.Name,
		State:            domain.McpResourceServerState(row.State),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
	if len(row.Scopes) > 0 {
		if err := json.Unmarshal(row.Scopes, &m.Scopes); err != nil {
			return nil, err
		}
	}
	return m, m.Validate()
}

func (r *McpResourceServerRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.McpResourceServer, error) {
	rows, err := sqlcgen.New(r.Pool).ListMcpResourceServersByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.McpResourceServer, 0, len(rows))
	for _, row := range rows {
		m, err := mcpResourceServerFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *McpResourceServerRepository) FindByID(ctx context.Context, tenantID, resourceServerID string) (*domain.McpResourceServer, error) {
	row, err := sqlcgen.New(r.Pool).GetMcpResourceServer(ctx, sqlcgen.GetMcpResourceServerParams{
		TenantID: tenantID, ResourceServerID: resourceServerID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mcpResourceServerFromRow(row)
}

func (r *McpResourceServerRepository) FindByResource(ctx context.Context, tenantID, resource string) (*domain.McpResourceServer, error) {
	row, err := sqlcgen.New(r.Pool).GetMcpResourceServerByResource(ctx, sqlcgen.GetMcpResourceServerByResourceParams{
		TenantID: tenantID, Resource: resource,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mcpResourceServerFromRow(row)
}

func (r *McpResourceServerRepository) Save(ctx context.Context, m *domain.McpResourceServer) error {
	scopes, err := json.Marshal(m.Scopes)
	if err != nil {
		return err
	}
	return sqlcgen.New(r.Pool).UpsertMcpResourceServer(ctx, sqlcgen.UpsertMcpResourceServerParams{
		TenantID:         m.TenantID,
		ResourceServerID: m.ResourceServerID,
		Resource:         m.Resource,
		Name:             m.Name,
		Scopes:           scopes,
		State:            string(m.State),
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	})
}

func (r *McpResourceServerRepository) Delete(ctx context.Context, tenantID, resourceServerID string) error {
	return sqlcgen.New(r.Pool).DeleteMcpResourceServer(ctx, sqlcgen.DeleteMcpResourceServerParams{
		TenantID: tenantID, ResourceServerID: resourceServerID,
	})
}
