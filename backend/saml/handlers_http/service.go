package handlers_http

import (
	"context"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	samlusecases "github.com/ambi/idmagic/backend/saml/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

// signInService は Deps の依存から SSO usecase を組み立てる。
func (d Deps) signInService() samlusecases.SignInService {
	return samlusecases.SignInService{
		SPRepo:         d.SamlSPRepo,
		ReplayStore:    d.ReplayStore,
		UserRepo:       d.UserRepo,
		Gate:           gateAdapter{d.ApplicationGate},
		Emit:           d.Emit,
		AttrSchemaRepo: d.AttrSchemaRepo,
	}
}

// logoutService は Deps の依存から SLO usecase を組み立てる。
func (d Deps) logoutService() samlusecases.LogoutService {
	return samlusecases.LogoutService{SPRepo: d.SamlSPRepo}
}

// gateAdapter は support.ApplicationGate を usecase の ApplicationGate へ橋渡しする。
// 判定結果は項目ごとに写す。
type gateAdapter struct{ *support.ApplicationGate }

func (g gateAdapter) EvaluateApplicationAccess(
	ctx context.Context,
	tenantID string,
	bindingType appdomain.ApplicationProtocolType,
	bindingKey, sub string,
	authn *authdomain.AuthenticationContext,
	clientIP string,
) (samlusecases.ApplicationAccessDecision, error) {
	dec, err := g.ApplicationGate.EvaluateApplicationAccess(ctx, tenantID, bindingType, bindingKey, sub, authn, clientIP)
	// 項目ごとに写す。この Context には step-up の遷移先が無く、信頼済みデバイスの判定も
	// 使わないので、共有の判定に項目が増えてもここは follow しない。
	return samlusecases.ApplicationAccessDecision{
		Allowed: dec.Allowed, StepUpRequired: dec.StepUpRequired,
		ApplicationID: dec.ApplicationID, Reason: dec.Reason,
	}, err
}
