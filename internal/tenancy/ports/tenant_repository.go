package ports

import (
	"context"

	"github.com/ambi/idmagic/internal/shared/spec"
)

type TenantRepository interface {
	FindByID(ctx context.Context, id string) (*spec.Tenant, error)
	FindByRealm(ctx context.Context, realm string) (*spec.Tenant, error)
	FindAll(ctx context.Context) ([]*spec.Tenant, error)
	Save(ctx context.Context, tenant *spec.Tenant) error
}
