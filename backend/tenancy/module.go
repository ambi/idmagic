// Package tenancy は Tenancy bounded context の DI 組立を所有する (wi-179)。
// Tenant/TenantUserAttributeSchema の永続化 port を Module 1 個に束ね、中央
// server/routes.go と bootstrap の Dependencies が受け渡す。
package tenancy

import (
	"github.com/ambi/idmagic/backend/tenancy/ports"
)

// Module は tenancy context が所有する永続化 port の束。
type Module struct {
	TenantRepo         ports.TenantRepository
	AttrSchemaRepo     ports.TenantUserAttributeSchemaRepository
	BrandingRepo       ports.TenantBrandingRepository
	BrandingAssetStore ports.TenantBrandingAssetStore
	// NotificationTemplates はテナントによる通知テンプレート上書き (wi-288)。
	NotificationTemplates ports.NotificationTemplateRepository
	QuotaRepo             ports.QuotaRepository
}
