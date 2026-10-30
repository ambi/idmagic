package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// McpResourceServerRepository は MCP resource server 登録 を PostgreSQL に
// 永続化する。scopes は JSONB として保持する。すべての参照はテナント境界に閉じる。
// クエリは sqlc 生成 (wi-173)。
type McpResourceServerRepository struct{ Pool sharedpg.DB }

func mcpResourceServerFromRow(row *McpResourceServer) (*domain.McpResourceServer, error) {
	m := &domain.McpResourceServer{
		TenantID:  row.TenantID,
		ID:        row.ID,
		Resource:  row.Resource,
		Name:      row.Name,
		ClientID:  row.ClientID.String,
		State:     domain.McpResourceServerState(row.State),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if len(row.Scopes) > 0 {
		if err := json.Unmarshal(row.Scopes, &m.Scopes); err != nil {
			return nil, err
		}
	}
	if len(row.AccessRules) > 0 {
		if err := json.Unmarshal(row.AccessRules, &m.AccessRules); err != nil {
			return nil, err
		}
	}
	return m, m.Validate()
}

func (r *McpResourceServerRepository) ListAll(ctx context.Context, tenantID string) ([]*domain.McpResourceServer, error) {
	rows, err := New(r.Pool).ListMcpResourceServersByTenant(ctx, tenantID)
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
	row, err := New(r.Pool).GetMcpResourceServer(ctx, GetMcpResourceServerParams{
		TenantID: tenantID, ID: resourceServerID,
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
	row, err := New(r.Pool).GetMcpResourceServerByResource(ctx, GetMcpResourceServerByResourceParams{
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

func (r *McpResourceServerRepository) FindByClientID(ctx context.Context, tenantID, clientID string) (*domain.McpResourceServer, error) {
	row, err := New(r.Pool).GetMcpResourceServerByClientID(ctx, GetMcpResourceServerByClientIDParams{
		TenantID: tenantID, ClientID: pgtype.Text{String: clientID, Valid: clientID != ""},
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
	rules := m.AccessRules
	if rules == nil {
		rules = []domain.ResourceAccessRule{}
	}
	accessRules, err := json.Marshal(rules)
	if err != nil {
		return err
	}
	return New(r.Pool).UpsertMcpResourceServer(ctx, UpsertMcpResourceServerParams{
		TenantID:    m.TenantID,
		ID:          m.ID,
		Resource:    m.Resource,
		Name:        m.Name,
		Scopes:      scopes,
		ClientID:    pgtype.Text{String: m.ClientID, Valid: m.ClientID != ""},
		AccessRules: accessRules,
		State:       string(m.State),
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	})
}

func (r *McpResourceServerRepository) Delete(ctx context.Context, tenantID, resourceServerID string) error {
	return New(r.Pool).DeleteMcpResourceServer(ctx, DeleteMcpResourceServerParams{
		TenantID: tenantID, ID: resourceServerID,
	})
}
