package handlers_http

import (
	"context"
	"net/http"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	wsfedusecases "github.com/ambi/idmagic/backend/wsfederation/usecases"
)

// ApplicationGate は、パッシブサインインの開始を Application の割り当てとサインインポリシーで
// 判定する。判定は Application が所有し、組み立て地点が結ぶ。
type ApplicationGate interface {
	EvaluateApplicationAccess(
		ctx context.Context,
		tenantID string,
		bindingType appdomain.ApplicationProtocolType,
		bindingKey, sub string,
		authn *authdomain.AuthenticationContext,
		clientIP string,
	) (appdomain.ApplicationAccessDecision, error)
	ClientIP(r *http.Request) string
}

// signInService は Deps の依存から passive sign-in usecase を組み立てる。
func (d Deps) signInService() wsfedusecases.SignInService {
	return wsfedusecases.SignInService{
		RPRepo:         d.WsFedRPRepo,
		UserRepo:       d.UserRepo,
		Gate:           gateAdapter{d.ApplicationGate},
		Emit:           d.Emit,
		AttrSchemaRepo: d.AttrSchemaRepo,
	}
}

// signOutService は Deps の依存から sign-out usecase を組み立てる。
func (d Deps) signOutService() wsfedusecases.SignOutService {
	return wsfedusecases.SignOutService{RPRepo: d.WsFedRPRepo}
}

// gateAdapter は Application の割り当てのゲートを usecase の ApplicationGate へ橋渡しする。
type gateAdapter struct{ ApplicationGate }

func (g gateAdapter) EvaluateApplicationAccess(
	ctx context.Context,
	tenantID string,
	bindingType appdomain.ApplicationProtocolType,
	bindingKey, sub string,
	authn *authdomain.AuthenticationContext,
	clientIP string,
) (wsfedusecases.ApplicationAccessDecision, error) {
	dec, err := g.ApplicationGate.EvaluateApplicationAccess(ctx, tenantID, bindingType, bindingKey, sub, authn, clientIP)
	// 項目ごとに写す。このモジュールには step-up の遷移先が無く、信頼済みデバイスの判定も
	// 使わないので、共有の判定に項目が増えてもここは follow しない。
	return wsfedusecases.ApplicationAccessDecision{
		Allowed: dec.Allowed, StepUpRequired: dec.StepUpRequired,
		ApplicationID: dec.ApplicationID, Reason: dec.Reason,
	}, err
}
