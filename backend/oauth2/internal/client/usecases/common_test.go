package usecases

import (
	"context"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func tenantContext(id string) context.Context {
	return tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{
		ID: id, DisplayName: id, Status: tenancydomain.TenantStatusActive, CreatedAt: time.Now().UTC(),
	}, "https://idp.example/realms/"+id, "/realms/"+id)
}
