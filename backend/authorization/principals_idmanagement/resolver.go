// Package principals_idmanagement は代行チェーン上のプリンシパルの有効性を
// IdManagement の記録から解決する。Authorization Context は判断の実体を持たず、
// 記録の正であるこちらへ問い合わせるだけである。
package principals_idmanagement

import (
	"context"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

// Resolver は認可モデルの主体型を IdManagement の Repository へ写す。
// 知らない主体型は有効とみなさない (fail-closed)。
type Resolver struct {
	Agents agentports.AgentRepository
	Users  userports.UserRepository
}

func (r Resolver) IsPrincipalActive(ctx context.Context, tenantID, principalType, principalID string) (bool, error) {
	switch principalType {
	case "agent":
		if r.Agents == nil {
			return false, nil
		}
		agent, err := r.Agents.FindByID(ctx, tenantID, principalID)
		if err != nil || agent == nil {
			return false, err
		}
		return agent.IsActive(), nil
	case "user":
		if r.Users == nil {
			return false, nil
		}
		user, err := r.Users.FindBySub(ctx, principalID)
		if err != nil || user == nil {
			return false, err
		}
		// 別テナントの User を代行者として通さない。
		return user.TenantID == tenantID && user.IsActive(), nil
	default:
		return false, nil
	}
}
