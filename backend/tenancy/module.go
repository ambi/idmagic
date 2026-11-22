// Package tenancy は Tenancy モジュールの DI 組立を所有する (wi-179)。
// テナントの設定の永続化 port を Module 1 個に束ね、中央
// server/routes.go と bootstrap の Dependencies が受け渡す。
package tenancy

import (
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// Module は tenancy context が所有する永続化 port の束。
type Module struct {
	TenantRepo         tenantports.TenantRepository
	BrandingRepo       tenantports.TenantBrandingRepository
	BrandingAssetStore tenantports.TenantBrandingAssetStore
	// NotificationTemplates はテナントによる通知テンプレート上書き (wi-288)。
	NotificationTemplates tenantports.NotificationTemplateRepository
	QuotaRepo             tenantports.QuotaRepository
}
