// Package ports は Saml モジュールの永続境界 (port) を定義する (wi-29)。
package ports

import (
	"context"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
)

// SamlServiceProviderRepository は SAML 2.0 service provider 登録の永続境界 (wi-29)。
// entityID で識別し、許可 ACS・audience・NameID format・claim policy を保持する。
type SamlServiceProviderRepository interface {
	// FindByEntityID は entityID に一致する SP を返す。存在しなければ (nil, nil)。
	FindByEntityID(ctx context.Context, tenantID, entityID string) (*samldomain.SamlServiceProvider, error)
	// ListAll はテナント内の SP を entityID 昇順で返す。
	ListAll(ctx context.Context, tenantID string) ([]*samldomain.SamlServiceProvider, error)
	// Save は SP を upsert する。
	Save(ctx context.Context, sp *samldomain.SamlServiceProvider) error
	// Delete は entityID に一致する SP を削除する (冪等)。
	Delete(ctx context.Context, tenantID, entityID string) error
}
