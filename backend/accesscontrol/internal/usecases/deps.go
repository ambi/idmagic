package usecases

import (
	"context"

	accessinternalports "github.com/ambi/idmagic/backend/accesscontrol/internal/ports"
	accessports "github.com/ambi/idmagic/backend/accesscontrol/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// Deps は関係の評価が必要とする境界。Authorizer は AuthZEN ポートで、
// ローカル評価器にもリモート PDP にも差し替えられる。
type Deps struct {
	Tuples     accessinternalports.RelationTupleRepository
	Models     accessinternalports.AuthorizationModelRepository
	Principals accessinternalports.PrincipalStatusResolver
	Authorizer accessports.Authorizer
	Emit       func(spec.DomainEvent)
	// MaxDepth は関係グラフをたどる深さの上限。0 なら domain の既定を使う。
	MaxDepth int
	// MaxEnumeratedResources は ListAccessibleResources が走査する候補の上限。
	// 0 なら DefaultMaxEnumeratedResources を使う。
	MaxEnumeratedResources int
}

func (d Deps) emit(event spec.DomainEvent) {
	if d.Emit != nil {
		d.Emit(event)
	}
}

// resolvePrincipalActive はプリンシパルの有効性を解決する。解決手段が無い、
// または解決に失敗した場合は有効とみなさない (fail-closed)。
func (d Deps) resolvePrincipalActive(ctx context.Context, tenantID, principalType, principalID string) bool {
	if d.Principals == nil {
		return false
	}
	active, err := d.Principals.IsPrincipalActive(ctx, tenantID, principalType, principalID)
	if err != nil {
		return false
	}
	return active
}
