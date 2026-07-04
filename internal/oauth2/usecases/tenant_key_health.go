// ListTenantKeyHealth (system_admin — テナント横断の署名鍵ヘルス)
package usecases

import (
	"context"

	"github.com/ambi/idmagic/internal/oauth2/ports"
	"github.com/ambi/idmagic/internal/shared/spec"
	"github.com/ambi/idmagic/internal/tenancy"
	tenantports "github.com/ambi/idmagic/internal/tenancy/ports"
)

type TenantKeyHealthDeps struct {
	TenantRepo tenantports.TenantRepository
	KeyStore   ports.KeyStore
}

// ListTenantKeyHealth は全テナントの署名鍵ヘルスを集約する。秘密鍵は返さない。
// テナントごとに ctx を差し替えて tenant-aware KeyStore に問い合わせる。
func ListTenantKeyHealth(ctx context.Context, deps TenantKeyHealthDeps) ([]ports.TenantKeyHealth, error) {
	tenants, err := deps.TenantRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ports.TenantKeyHealth, 0, len(tenants))
	for _, t := range tenants {
		tctx := tenancy.WithTenant(ctx, t, "", "")
		keys, err := deps.KeyStore.GetAllKeys(tctx)
		if err != nil {
			return nil, err
		}
		activeKid := ""
		for _, k := range keys {
			if k.Active {
				activeKid = k.Kid
				break
			}
		}
		out = append(out, ports.TenantKeyHealth{
			TenantID:     t.ID,
			Provider:     deps.KeyStore.Provider(),
			Usage:        spec.KeyUsageSigning,
			ActiveKid:    activeKid,
			JWKSKeyCount: len(keys),
			Healthy:      deps.KeyStore.Healthy(tctx),
		})
	}
	return out, nil
}
