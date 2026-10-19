package usecases

import (
	"context"

	passworddomain "github.com/ambi/idmagic/backend/authentication/password/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// ValidateInitialPassword は管理者が指定した初回のパスワードを検証する。変更時の履歴照合は行わず、
// ロールの検証より先に呼ぶので、ハッシュの生成もここでは行わない。
func ValidateInitialPassword(ctx context.Context, password string) error {
	policy := passworddomain.PolicyForTenant(tenantports.Tenant(ctx))
	result := passworddomain.ValidatePasswordWith(password, policy)
	if !result.OK {
		return &passworddomain.PasswordPolicyError{Violations: result.Violations}
	}
	return nil
}
