package ports

import (
	"context"

	"idmagic/internal/shared/spec"
)

type TenantRepository interface {
	FindByID(ctx context.Context, id string) (*spec.Tenant, error)
	FindAll(ctx context.Context) ([]*spec.Tenant, error)
	Save(ctx context.Context, tenant *spec.Tenant) error
}
