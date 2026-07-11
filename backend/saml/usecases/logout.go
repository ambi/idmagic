package usecases

import (
	"context"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	samlports "github.com/ambi/idmagic/backend/saml/ports"
)

// LogoutService は SAML Single Logout の返送先解決と LogoutRequest 検証を所有する。
type LogoutService struct {
	SPRepo samlports.SamlServiceProviderRepository
}

// ResolveRedirect は IdP-initiated ログアウトの返送先を返す。entityID で SP を解決し、
// 登録済み SingleLogoutService URL があればそれを返す。SP / SLO URL が解決できない、または
// 判定不能なら空文字を返す (open redirect 防止)。エラーは「リダイレクトしない」判断へ畳む。
func (s LogoutService) ResolveRedirect(ctx context.Context, tenantID, entityID, relayState string) string {
	if entityID == "" || s.SPRepo == nil {
		return ""
	}
	sp, err := s.SPRepo.FindByEntityID(ctx, tenantID, entityID)
	if err != nil || sp == nil || sp.SLOURL == "" {
		return ""
	}
	target := sp.SLOURL
	if relayState != "" {
		target += "?RelayState=" + relayState
	}
	return target
}

// LogoutRequestInput は adapter が wire から組み立てて渡す LogoutRequest 検証入力。
type LogoutRequestInput struct {
	TenantID            string
	Request             samldomain.LogoutRequest
	Binding             samldomain.Binding
	RawXML              []byte
	RawQuery            string
	ExpectedDestination string
}

// LogoutRequestDecision は LogoutRequest 検証の結果。
type LogoutRequestDecision struct {
	SP         *samldomain.SamlServiceProvider // 非 nil なら logout を honor し LogoutResponse を返す。
	BadRequest string                          // 非空なら 400 を返す本文。
	EmitLogout bool                            // 未知 SP のとき SamlLogout イベントを発行すべき。
}

// ValidateLogoutRequest は LogoutRequest を検証し、返送すべき SP を解決する。
// SP 未登録 / SLO URL 不在、署名不正、Destination 不一致を fail-closed で弾く。
// 挙動は旧 HTTP ハンドラの handleSamlLogoutRequest の検証部と一致する。
func (s LogoutService) ValidateLogoutRequest(ctx context.Context, in LogoutRequestInput) (LogoutRequestDecision, error) {
	sp, err := s.SPRepo.FindByEntityID(ctx, in.TenantID, in.Request.Issuer)
	if err != nil {
		return LogoutRequestDecision{}, err
	}
	if sp == nil || sp.SLOURL == "" {
		return LogoutRequestDecision{BadRequest: "unknown service provider", EmitLogout: true}, nil
	}
	if err := samldomain.ValidateRequestSignature(in.Binding, in.RawXML, in.RawQuery, *sp); err != nil {
		//nolint:nilerr // 署名検証エラーは 400 の reject outcome へ変換し、呼び出し側には error を返さない。
		return LogoutRequestDecision{BadRequest: err.Error()}, nil
	}
	if in.Request.Destination != "" && in.Request.Destination != in.ExpectedDestination {
		return LogoutRequestDecision{BadRequest: "Destination does not match SLO endpoint"}, nil
	}
	return LogoutRequestDecision{SP: sp}, nil
}
