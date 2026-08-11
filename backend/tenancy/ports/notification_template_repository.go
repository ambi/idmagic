package ports

import (
	"context"

	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
)

// NotificationTemplateRepository は通知テンプレートのテナント上書きを保持する
// (wi-288)。組込み既定カタログとレンダラは shared/notification が所有し、
// テナント単位の設定という関心だけを Tenancy が持つ。
//
// 行が存在しない (tenant_id, template_key, locale) は「組込み既定を使う」を意味し、
// 行の削除がそのまま「既定へのリセット」になる。版管理は持たない。
type NotificationTemplateRepository interface {
	FindByKey(ctx context.Context, tenantID string, key notificationports.TemplateKey, locale string) (*notificationports.TemplateOverride, error)
	// ListAll は tenant の全上書きを返す。カタログの全 key × locale は呼び出し側が
	// 知っているため、ここでは存在する行だけを返す。
	ListAll(ctx context.Context, tenantID string) ([]*notificationports.TemplateOverride, error)
	Save(ctx context.Context, override *notificationports.TemplateOverride) error
	// Delete は上書きを削除し、削除した行があったかを返す。上書きが無い場合も成功する
	// (冪等)。
	Delete(ctx context.Context, tenantID string, key notificationports.TemplateKey, locale string) (bool, error)
}
