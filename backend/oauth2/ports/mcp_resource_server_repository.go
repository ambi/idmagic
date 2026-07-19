package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

// McpResourceServerRepository はテナント登録済みの MCP resource server (ADR-055) の
// 永続境界。canonical resource URI・許可 scope・運用状態を保持する。
type McpResourceServerRepository interface {
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.McpResourceServer, error)
	FindByID(ctx context.Context, tenantID, resourceServerID string) (*domain.McpResourceServer, error)
	// FindByResource は resource indicator 検証で使う。tenant 内で resource URI により解決する。
	FindByResource(ctx context.Context, tenantID, resource string) (*domain.McpResourceServer, error)
	Save(ctx context.Context, m *domain.McpResourceServer) error
	Delete(ctx context.Context, tenantID, resourceServerID string) error
}
