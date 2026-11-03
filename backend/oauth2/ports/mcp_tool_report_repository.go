package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

// McpToolReportRepository は登録ごとの最新のツールの報告を保つ。登録とは別の記録にして、
// 報告が管理者の編集の競合検査と更新時刻に影響しないようにする。
type McpToolReportRepository interface {
	// Find は報告がなければ nil を返す。
	Find(ctx context.Context, tenantID, resourceServerID string) (*domain.McpToolReport, error)
	// Save は前回の報告を置き換える。
	Save(ctx context.Context, report *domain.McpToolReport) error
	Delete(ctx context.Context, tenantID, resourceServerID string) error
}
