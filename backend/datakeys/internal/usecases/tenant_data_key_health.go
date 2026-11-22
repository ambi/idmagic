package usecases

import (
	"context"
	"errors"

	datakeysdomain "github.com/ambi/idmagic/backend/datakeys/internal/domain"
	datakeysports "github.com/ambi/idmagic/backend/datakeys/internal/ports"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// ListTenantDataKeyHealthDeps are the dependencies for
// ListTenantDataKeyHealth (wi-97 T007).
type ListTenantDataKeyHealthDeps struct {
	TenantRepo tenantports.TenantRepository
	Repository datakeysports.DataKeyRepository
	Crypto     envelope_crypto.EnvelopeCrypto
}

// ListTenantDataKeyHealth aggregates every tenant's DEK health
// (docs/modules/data-keys/ ListTenantDataKeyHealth): active version,
// status, MasterKeyProvider name, and its current reachability. It never
// returns key material. A tenant with no DataEncryptionKey bootstrapped yet
// (FieldCipher.Encrypt has never run for it) is simply omitted rather than
// reported as unhealthy.
func ListTenantDataKeyHealth(ctx context.Context, deps ListTenantDataKeyHealthDeps) ([]datakeysdomain.TenantDataKeyHealth, error) {
	tenants, err := deps.TenantRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	reachable := deps.Crypto.Healthy(ctx)
	provider := deps.Crypto.Provider()

	out := make([]datakeysdomain.TenantDataKeyHealth, 0, len(tenants))
	for _, tenant := range tenants {
		active, err := deps.Repository.FindActive(ctx, tenant.ID)
		if errors.Is(err, datakeysdomain.ErrNoActiveDataKey) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, datakeysdomain.TenantDataKeyHealth{
			TenantID:          tenant.ID,
			ActiveVersion:     active.Version,
			Status:            active.Status,
			Provider:          provider,
			ProviderReachable: reachable,
			RotatedAt:         active.ActivatedAt,
		})
	}
	return out, nil
}
