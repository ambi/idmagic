package http

import (
	"context"

	authdomain "github.com/ambi/idmagic/internal/authentication/domain"
	"github.com/ambi/idmagic/internal/shared/adapters/http/support"
	"github.com/ambi/idmagic/internal/shared/spec"
	wsfedusecases "github.com/ambi/idmagic/internal/wsfederation/usecases"
)

// signInService は Deps の依存から passive sign-in usecase を組み立てる。
func (d Deps) signInService() wsfedusecases.SignInService {
	return wsfedusecases.SignInService{
		RPRepo:   d.WsFedRPRepo,
		UserRepo: d.UserRepo,
		Gate:     gateAdapter{d.ApplicationGate},
		Emit:     d.Emit,
	}
}

// signOutService は Deps の依存から sign-out usecase を組み立てる。
func (d Deps) signOutService() wsfedusecases.SignOutService {
	return wsfedusecases.SignOutService{RPRepo: d.WsFedRPRepo}
}

// gateAdapter は support.ApplicationGate を usecase の ApplicationGate へ橋渡しする。
type gateAdapter struct{ *support.ApplicationGate }

func (g gateAdapter) EvaluateApplicationAccess(
	ctx context.Context,
	tenantID string,
	bindingType spec.ProtocolBindingType,
	bindingKey, sub string,
	authn *authdomain.AuthenticationContext,
	clientIP string,
) (wsfedusecases.ApplicationAccessDecision, error) {
	dec, err := g.ApplicationGate.EvaluateApplicationAccess(ctx, tenantID, bindingType, bindingKey, sub, authn, clientIP)
	return wsfedusecases.ApplicationAccessDecision(dec), err
}
