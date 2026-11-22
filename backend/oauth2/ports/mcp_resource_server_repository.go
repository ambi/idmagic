package ports

import (
	"context"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
)

// McpResourceServerRepository はテナント登録済みの MCP resource server の
// 永続境界。canonical resource URI・許可 scope・運用状態を保持する。
type McpResourceServerRepository interface {
	ListAll(ctx context.Context, tenantID string) ([]*oauthdomain.McpResourceServer, error)
	FindByID(ctx context.Context, tenantID, resourceServerID string) (*oauthdomain.McpResourceServer, error)
	// FindByResource は resource indicator 検証で使う。tenant 内で resource URI により解決する。
	FindByResource(ctx context.Context, tenantID, resource string) (*oauthdomain.McpResourceServer, error)
	// FindByClientID は評価を求めたクライアントに束縛した登録を解決する。
	FindByClientID(ctx context.Context, tenantID, clientID string) (*oauthdomain.McpResourceServer, error)
	Save(ctx context.Context, m *oauthdomain.McpResourceServer) error
	SaveIfUnchanged(ctx context.Context, m *oauthdomain.McpResourceServer, expectedUpdatedAt time.Time) (bool, error)
	Delete(ctx context.Context, tenantID, resourceServerID string) error
}
