package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

// McpToolCatalogRepository は登録ごとの最新のツールのカタログを保つ。登録とは別の記録にして、
// カタログが管理者の編集の競合検査と更新時刻に影響しないようにする。
type McpToolCatalogRepository interface {
	// Find はカタログがなければ nil を返す。
	Find(ctx context.Context, tenantID, resourceServerID string) (*domain.McpToolCatalog, error)
	// Save は前回のカタログを置き換える。
	Save(ctx context.Context, catalog *domain.McpToolCatalog) error
	Delete(ctx context.Context, tenantID, resourceServerID string) error
}
