package usecases

import (
	"context"
	"errors"

	"github.com/ambi/idmagic/backend/datakeys/domain"
	"github.com/ambi/idmagic/backend/datakeys/ports"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	tenancyports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// ListTenantDataKeyHealthDeps are the dependencies for
// ListTenantDataKeyHealth (wi-97 T007).
type ListTenantDataKeyHealthDeps struct {
	TenantRepo tenancyports.TenantRepository
	Repository ports.DataKeyRepository
	Crypto     envelope_crypto.EnvelopeCrypto
}

// ListTenantDataKeyHealth aggregates every tenant's DEK health
// (spec/contexts/data-keys.yaml ListTenantDataKeyHealth): active version,
// status, MasterKeyProvider name, and its current reachability. It never
// returns key material. A tenant with no DataEncryptionKey bootstrapped yet
// (FieldCipher.Encrypt has never run for it) is simply omitted rather than
// reported as unhealthy.
func ListTenantDataKeyHealth(ctx context.Context, deps ListTenantDataKeyHealthDeps) ([]domain.TenantDataKeyHealth, error) {
	tenants, err := deps.TenantRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	reachable := deps.Crypto.Healthy(ctx)
	provider := deps.Crypto.Provider()

	out := make([]domain.TenantDataKeyHealth, 0, len(tenants))
	for _, tenant := range tenants {
		active, err := deps.Repository.FindActive(ctx, tenant.ID)
		if errors.Is(err, domain.ErrNoActiveDataKey) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, domain.TenantDataKeyHealth{
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
