// Package usecases は Saml bounded context のアプリケーション論理 (wi-142)。
//
// SSO sign-in と SLO logout のオーケストレーション (SP 解決・署名検証・割当ゲート・
// claim 発行) を HTTP 境界から切り離して所有する。wire 変換・XML 署名・直列化・cookie /
// redirect などの HTTP 境界処理は adapters が担い、本パッケージは adapter を import しない
// (oauth2 usecases と同じ依存方向)。
package usecases

import (
	"context"
	"time"

	appdomain "github.com/ambi/idmagic/internal/application/domain"
	authdomain "github.com/ambi/idmagic/internal/authentication/domain"
	idmports "github.com/ambi/idmagic/internal/identitymanagement/ports"
	samldomain "github.com/ambi/idmagic/internal/saml/domain"
	samlports "github.com/ambi/idmagic/internal/saml/ports"
	"github.com/ambi/idmagic/internal/shared/spec"
	feddomain "github.com/ambi/idmagic/internal/wsfederation/domain"
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

// SignInService は SAML SSO の発行判断を所有する。
type SignInService struct {
	SPRepo   samlports.SamlServiceProviderRepository
	UserRepo idmports.UserRepository
	Gate     ApplicationGate
	Emit     func(spec.DomainEvent)
}

// SignInOutcomeKind は SSO 判断の分岐種別。
type SignInOutcomeKind int

const (
	// SignInNeedLogin は未認証 / 再認証が必要でログインへ誘導すべきことを表す。
	SignInNeedLogin SignInOutcomeKind = iota
	// SignInRejected はプロトコル / 検証エラーで 400 を返すべきことを表す。イベントは発行済み。
	SignInRejected
	// SignInForbidden は割当ゲートで拒否され 403 を返すべきことを表す。イベントは発行済み。
	SignInForbidden
	// SignInIssued は assertion 発行に進んでよいことを表す。
	SignInIssued
)

// SignInOutcome は SSO 判断の結果。Kind により有効フィールドが変わる。
type SignInOutcome struct {
	Kind    SignInOutcomeKind
	Message string // Rejected(400) / Forbidden(403) の応答本文。

	// SignInIssued のときの発行データ。
	SP          spec.SamlServiceProvider
	Validated   samldomain.ValidatedSignIn
	ClaimResult feddomain.ClaimIssuanceResult
	Authn       *authdomain.AuthenticationContext
	Now         time.Time
}

// SignInInput は adapter が wire から組み立てて渡す SSO 判断入力。
type SignInInput struct {
	TenantID            string
	Request             samldomain.AuthnRequest
	Binding             samldomain.Binding // 空なら IdP-initiated (署名検証しない)。
	RawXML              []byte
	RawQuery            string
	ExpectedDestination string
	Authn               *authdomain.AuthenticationContext // adapter が header から解決済み。
	ClientIP            string
}

// Issue は SAMLResponse を発行してよいかを判断する。SP 解決・署名検証・認証ゲート・割当ゲート・
// claim 発行を順に適用し、拒否 / 要ログイン / 発行可の outcome を返す。挙動は旧 HTTP ハンドラの
// issueForRequest と一致する。
func (s SignInService) Issue(ctx context.Context, in SignInInput) (SignInOutcome, error) {
	sp, err := s.SPRepo.FindByEntityID(ctx, in.TenantID, in.Request.Issuer)
	if err != nil {
		return SignInOutcome{}, err
	}
	if sp == nil {
		return s.rejected(in.TenantID, in.Request.Issuer, "unknown service provider", nil), nil
	}
	if in.Binding != "" {
		if err := samldomain.ValidateRequestSignature(in.Binding, in.RawXML, in.RawQuery, *sp); err != nil {
			return s.rejected(in.TenantID, in.Request.Issuer, err.Error(), nil), nil
		}
	}
	validated, err := samldomain.ValidateSignIn(in.Request, *sp, in.ExpectedDestination)
	if err != nil {
		return s.rejected(in.TenantID, in.Request.Issuer, err.Error(), nil), nil
	}

	authn := in.Authn
	if authn == nil || authn.UserID == "" || authn.AuthenticationPending {
		return SignInOutcome{Kind: SignInNeedLogin}, nil
	}
	now := time.Now().UTC()
	if samldomain.RequiresFreshAuth(in.Request.ForceAuthn, time.Unix(authn.AuthTime, 0).UTC(), now) {
		return SignInOutcome{Kind: SignInNeedLogin}, nil
	}
	user, err := s.UserRepo.FindBySub(ctx, authn.UserID)
	if err != nil {
		return SignInOutcome{}, err
	}
	if user == nil || !user.IsActive() {
		return SignInOutcome{Kind: SignInNeedLogin}, nil
	}

	// 割当ゲート: SP が Application binding に属する場合、未割当 subject には発行しない (fail-closed)。
	decision, err := s.Gate.EvaluateApplicationAccess(ctx, in.TenantID, appdomain.ProtocolBindingSAML, sp.EntityID, authn.UserID, authn, in.ClientIP)
	if err != nil {
		return SignInOutcome{}, err
	}
	if !decision.Allowed {
		reason := decision.Reason
		if decision.StepUpRequired {
			reason = "step-up required by application sign-in policy"
			s.emit(&appdomain.AppStepUpRequired{At: now, TenantID: in.TenantID, ApplicationID: decision.ApplicationID, Protocol: string(appdomain.ProtocolBindingSAML), Subject: authn.UserID})
		} else if reason == "" {
			reason = "subject not assigned to application"
		}
		if decision.ApplicationID != "" {
			s.emit(&appdomain.AppAccessDeniedByPolicy{At: now, TenantID: in.TenantID, ApplicationID: decision.ApplicationID, Protocol: string(appdomain.ProtocolBindingSAML), Subject: authn.UserID, Reason: reason})
		}
		s.emit(&spec.SamlSignInRejected{At: now, TenantID: in.TenantID, EntityID: sp.EntityID, Reason: reason})
		return SignInOutcome{Kind: SignInForbidden, Message: "この利用者はアプリケーションのサインインポリシーを満たしていません"}, nil
	}

	result, err := feddomain.IssueClaims(sp.ClaimPolicy, feddomain.ResolveUserAttributes(*user))
	if err != nil {
		return s.rejected(in.TenantID, sp.EntityID, "claim issuance failed", err), nil
	}
	if validated.NameIDFormat != "" {
		result.NameIDFormat = validated.NameIDFormat
	}

	return SignInOutcome{
		Kind:        SignInIssued,
		SP:          *sp,
		Validated:   validated,
		ClaimResult: result,
		Authn:       authn,
		Now:         now,
	}, nil
}

// rejected は SamlSignInRejected を発行し、400 本文に reason を載せる outcome を返す。
func (s SignInService) rejected(tenantID, entityID, reason string, cause error) SignInOutcome {
	msg := reason
	if cause != nil {
		msg = reason + ": " + cause.Error()
	}
	s.emit(&spec.SamlSignInRejected{At: time.Now().UTC(), TenantID: tenantID, EntityID: entityID, Reason: msg})
	return SignInOutcome{Kind: SignInRejected, Message: reason}
}

func (s SignInService) emit(event spec.DomainEvent) {
	if s.Emit != nil {
		s.Emit(event)
	}
}
