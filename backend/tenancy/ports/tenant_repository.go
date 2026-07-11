package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/tenancy/domain"
)

type TenantRepository interface {
	FindByID(ctx context.Context, id string) (*domain.Tenant, error)
	FindByRealm(ctx context.Context, realm string) (*domain.Tenant, error)
	FindAll(ctx context.Context) ([]*domain.Tenant, error)
	Save(ctx context.Context, tenant *domain.Tenant) error
}
