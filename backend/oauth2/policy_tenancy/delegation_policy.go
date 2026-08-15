// Package policy_tenancy は OAuth2 の委譲ポリシー解決を Tenancy の記録へ接続する。
// 上限の値と既定の継承は Tenant 集約が所有し、OAuth2 は解決結果だけを受け取る。
package policy_tenancy

import (
	"context"
	"errors"

	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// ErrTenantNotResolved はテナントを解決できなかったことを表す。呼び出し側は
// システム既定へ退避せず交換を拒否する。
var ErrTenantNotResolved = errors.New("tenant could not be resolved for delegation policy")

// DelegationPolicyResolver は ports.DelegationPolicyResolver を Tenant Repository で満たす。
type DelegationPolicyResolver struct {
	Tenants tenantports.TenantRepository
}

func (r DelegationPolicyResolver) MaxDelegationDepth(ctx context.Context, tenantID string) (int, error) {
	if r.Tenants == nil {
		return 0, ErrTenantNotResolved
	}
	tenant, err := r.Tenants.FindByID(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	if tenant == nil {
		return 0, ErrTenantNotResolved
	}
	return tenant.EffectiveMaxDelegationDepth(), nil
}
