package http

import (
	"context"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	samlusecases "github.com/ambi/idmagic/backend/saml/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
)

// signInService は Deps の依存から SSO usecase を組み立てる。
func (d Deps) signInService() samlusecases.SignInService {
	return samlusecases.SignInService{
		SPRepo:   d.SamlSPRepo,
		UserRepo: d.UserRepo,
		Gate:     gateAdapter{d.ApplicationGate},
		Emit:     d.Emit,
	}
}

// logoutService は Deps の依存から SLO usecase を組み立てる。
func (d Deps) logoutService() samlusecases.LogoutService {
	return samlusecases.LogoutService{SPRepo: d.SamlSPRepo}
}

// gateAdapter は support.ApplicationGate を usecase の ApplicationGate へ橋渡しする。
// 判定結果は同一形なので値変換で写す。
type gateAdapter struct{ *support.ApplicationGate }

func (g gateAdapter) EvaluateApplicationAccess(
	ctx context.Context,
	tenantID string,
	bindingType appdomain.ProtocolBindingType,
	bindingKey, sub string,
	authn *authdomain.AuthenticationContext,
	clientIP string,
) (samlusecases.ApplicationAccessDecision, error) {
	dec, err := g.ApplicationGate.EvaluateApplicationAccess(ctx, tenantID, bindingType, bindingKey, sub, authn, clientIP)
	return samlusecases.ApplicationAccessDecision(dec), err
}
