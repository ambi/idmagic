// Package usecases は WsFederation bounded context のアプリケーション論理 (wi-142)。
//
// passive sign-in / sign-out と WS-Trust トークン発行の判断 (RP 解決・検証・割当ゲート・
// claim 発行) を HTTP 境界から切り離して所有する。SAML assertion / RSTR / passive form の
// 構築・署名・直列化と、SOAP body 読取・replay・throttle・資格情報検証は adapters が担い、
// 本パッケージは adapter を import しない (oauth2 usecases と同じ依存方向)。
package usecases

import (
	"context"
	"time"

	claimusecases "github.com/ambi/idmagic/backend/claimmapping/usecases"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"
)

// ApplicationAccessDecision は割当ゲートの判定結果。
// support.ApplicationAccessDecision と同一形で、adapter が値変換で橋渡しする。
type ApplicationAccessDecision struct {
	Allowed        bool
	StepUpRequired bool
	ApplicationID  string
	Reason         string
}

// ApplicationGate は binding 経由サインインの割当ゲート評価 (adapter が support.ApplicationGate を橋渡し)。
type ApplicationGate interface {
	EvaluateApplicationAccess(
		ctx context.Context,
		tenantID string,
		bindingType appdomain.ProtocolBindingType,
		bindingKey, sub string,
		authn *authdomain.AuthenticationContext,
		clientIP string,
	) (ApplicationAccessDecision, error)
}

// SignInService は WS-Federation passive sign-in の発行判断を所有する。
type SignInService struct {
	RPRepo   wsfedports.WsFedRelyingPartyRepository
	UserRepo idmports.UserRepository
	Gate     ApplicationGate
	Emit     func(spec.DomainEvent)
}

// SignInOutcomeKind は sign-in 判断の分岐種別。
type SignInOutcomeKind int

const (
	// SignInNeedLogin は未認証 / 再認証が必要でログインへ誘導すべきことを表す。
	SignInNeedLogin SignInOutcomeKind = iota
	// SignInRejected は検証エラーで Status/Message を返すべきことを表す。イベントは発行済み。
	SignInRejected
	// SignInForbidden は割当ゲートで拒否され 403 を返すべきことを表す。イベントは発行済み。
	SignInForbidden
	// SignInIssued は assertion 発行に進んでよいことを表す。
	SignInIssued
)

// SignInOutcome は passive sign-in 判断の結果。
type SignInOutcome struct {
	Kind    SignInOutcomeKind
	Message string // Rejected / Forbidden の応答本文。
	Status  int    // Rejected の HTTP status (400 / 500)。

	// SignInIssued のときの発行データ。
	Validated   feddomain.ValidatedSignIn
	ClaimResult claimusecases.ClaimIssuanceResult
	AuthnMethod feddomain.AuthnMethodClass
	TokenType   feddomain.WsFedTokenType
	Authn       *authdomain.AuthenticationContext
	Now         time.Time
}

// SignInInput は adapter が組み立てて渡す passive sign-in 判断入力。
type SignInInput struct {
	TenantID string
	Request  feddomain.WsFedSignInRequest
	Authn    *authdomain.AuthenticationContext // adapter が header から解決済み。
	ClientIP string
}

// Issue は passive form を発行してよいかを判断する。挙動は旧 HTTP ハンドラの
// handleWsFedSignIn と一致する。
func (s SignInService) Issue(ctx context.Context, in SignInInput) (SignInOutcome, error) {
	tenantID := in.TenantID
	req := in.Request

	if req.Wtrealm == "" {
		s.emit(&feddomain.WsFedSignInRejected{At: time.Now().UTC(), TenantID: tenantID, Reason: "wtrealm is required"})
		return SignInOutcome{Kind: SignInRejected, Message: "wtrealm is required", Status: 400}, nil
	}

	rp, err := s.RPRepo.FindByWtrealm(ctx, tenantID, req.Wtrealm)
	if err != nil {
		return SignInOutcome{}, err
	}
	if rp == nil {
		s.emit(&feddomain.WsFedSignInRejected{At: time.Now().UTC(), TenantID: tenantID, Wtrealm: req.Wtrealm, Reason: "unknown relying party"})
		return SignInOutcome{Kind: SignInRejected, Message: "unknown relying party", Status: 400}, nil
	}

	validated, err := feddomain.ValidateSignIn(req, *rp)
	if err != nil {
		s.emit(&feddomain.WsFedSignInRejected{At: time.Now().UTC(), TenantID: tenantID, Wtrealm: req.Wtrealm, Reason: err.Error()})
		//nolint:nilerr // 検証エラーは 400 の reject outcome へ変換し、呼び出し側には error を返さない。
		return SignInOutcome{Kind: SignInRejected, Message: err.Error(), Status: 400}, nil
	}

	// セッション解決。未認証ならログインへ誘導し、認証後に同じ URL へ戻す。
	authn := in.Authn
	if authn == nil || authn.UserID == "" || authn.AuthenticationPending {
		return SignInOutcome{Kind: SignInNeedLogin}, nil
	}
	user, err := s.UserRepo.FindBySub(ctx, authn.UserID)
	if err != nil {
		return SignInOutcome{}, err
	}
	if user == nil || !user.IsActive() {
		return SignInOutcome{Kind: SignInNeedLogin}, nil
	}

	now := time.Now().UTC()

	// 割当ゲート (wi-69): RP が Application binding に属する場合、未割当 subject には
	// assertion を発行しない (fail-closed, AssignmentGatesProtocol)。
	decision, err := s.Gate.EvaluateApplicationAccess(ctx, tenantID, appdomain.ProtocolBindingWsFed, rp.Wtrealm, authn.UserID, authn, in.ClientIP)
	if err != nil {
		return SignInOutcome{}, err
	}
	if !decision.Allowed {
		reason := decision.Reason
		if decision.StepUpRequired {
			reason = "step-up required by application sign-in policy"
			s.emit(&appdomain.AppStepUpRequired{At: now, TenantID: tenantID, ApplicationID: decision.ApplicationID, Protocol: string(appdomain.ProtocolBindingWsFed), Subject: authn.UserID})
		} else if reason == "" {
			reason = "subject not assigned to application"
		}
		if decision.ApplicationID != "" {
			s.emit(&appdomain.AppAccessDeniedByPolicy{At: now, TenantID: tenantID, ApplicationID: decision.ApplicationID, Protocol: string(appdomain.ProtocolBindingWsFed), Subject: authn.UserID, Reason: reason})
		}
		s.emit(&feddomain.WsFedSignInRejected{At: now, TenantID: tenantID, Wtrealm: rp.Wtrealm, Reason: reason})
		return SignInOutcome{Kind: SignInForbidden, Message: "この利用者はアプリケーションのサインインポリシーを満たしていません"}, nil
	}

	// wfresh: 認証が古すぎれば再認証のためログインへ誘導する。
	if feddomain.RequiresFreshAuth(req.Wfresh, time.Unix(authn.AuthTime, 0), now) {
		return SignInOutcome{Kind: SignInNeedLogin}, nil
	}
	// wauth: 要求された認証方式を尊重する。満たせない方式 (統合 Windows 等) は拒否する。
	authnMethod, err := feddomain.ResolveAuthnMethod(req.Wauth, authn.AMR)
	if err != nil {
		s.emit(&feddomain.WsFedSignInRejected{At: now, TenantID: tenantID, Wtrealm: rp.Wtrealm, Reason: err.Error()})
		//nolint:nilerr // wauth 不適合は 400 の reject outcome へ変換し、呼び出し側には error を返さない。
		return SignInOutcome{Kind: SignInRejected, Message: err.Error(), Status: 400}, nil
	}

	attrs, err := feddomain.ApplyEntraProfile(claimusecases.ResolveUserAttributes(*user), rp.EntraProfile)
	if err != nil {
		s.emit(&feddomain.WsFedSignInRejected{At: now, TenantID: tenantID, Wtrealm: rp.Wtrealm, Reason: "entra profile failed"})
		//nolint:nilerr // entra profile 失敗は 500 の reject outcome へ変換し、呼び出し側には error を返さない。
		return SignInOutcome{Kind: SignInRejected, Message: "entra profile failed", Status: 500}, nil
	}
	result, err := claimusecases.IssueClaims(rp.ClaimPolicy, attrs)
	if err != nil {
		s.emit(&feddomain.WsFedSignInRejected{At: now, TenantID: tenantID, Wtrealm: rp.Wtrealm, Reason: "claim issuance failed"})
		//nolint:nilerr // claim 発行失敗は 500 の reject outcome へ変換し、呼び出し側には error を返さない。
		return SignInOutcome{Kind: SignInRejected, Message: "claim issuance failed", Status: 500}, nil
	}

	return SignInOutcome{
		Kind:        SignInIssued,
		Validated:   validated,
		ClaimResult: result,
		AuthnMethod: authnMethod,
		TokenType:   rp.EffectiveTokenType(),
		Authn:       authn,
		Now:         now,
	}, nil
}

func (s SignInService) emit(event spec.DomainEvent) {
	if s.Emit != nil {
		s.Emit(event)
	}
}
