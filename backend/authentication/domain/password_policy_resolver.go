package domain

// PasswordPolicyResolver は global default に対してテナント固有の上書きを適用する (Phase 4)。
// Tenant.PasswordPolicyOverride の non-nil フィールドのみが global を上書きし、
// 残りは SCL 値をそのまま使う。

import tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

type PasswordPolicySnapshot struct {
	MinLength    int
	MaxLength    int
	HistoryDepth int
}

// ResolvePasswordPolicy は global default + tenant override をマージして返す。
// tenant が nil または override が空なら global そのまま。
func ResolvePasswordPolicy(tenant *tenancydomain.Tenant, defaults PasswordPolicySnapshot) PasswordPolicySnapshot {
	snapshot := defaults
	if tenant == nil || tenant.PasswordPolicyOverride == nil {
		return snapshot
	}
	o := tenant.PasswordPolicyOverride
	if o.MinLength != nil && *o.MinLength > 0 {
		snapshot.MinLength = *o.MinLength
	}
	if o.MaxLength != nil && *o.MaxLength > 0 {
		snapshot.MaxLength = *o.MaxLength
	}
	if o.HistoryDepth != nil && *o.HistoryDepth > 0 {
		snapshot.HistoryDepth = *o.HistoryDepth
	}
	return snapshot
}
