package ports

import (
	"context"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type TenantRepository interface {
	FindByID(ctx context.Context, id string) (*tenancydomain.Tenant, error)
	FindByRealm(ctx context.Context, realm string) (*tenancydomain.Tenant, error)
	FindAll(ctx context.Context) ([]*tenancydomain.Tenant, error)
	Save(ctx context.Context, tenant *tenancydomain.Tenant) error
	Update(ctx context.Context, id string, change func(*tenancydomain.Tenant) (*tenancydomain.Tenant, error)) (*tenancydomain.Tenant, error)
}
