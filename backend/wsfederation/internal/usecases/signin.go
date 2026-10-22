// Package usecases は WsFederation モジュールのアプリケーション論理 (wi-142)。
//
// passive sign-in / sign-out と WS-Trust トークン発行の判断 (RP 解決・検証・割当ゲート・
// claim 発行) を HTTP 境界から切り離して所有する。SAML assertion / RSTR / passive form の
// 構築・署名・直列化と、SOAP body 読取・replay・throttle・資格情報検証は adapters が担い、
// 本パッケージは adapter を import しない (oauth2 usecases と同じ依存方向)。
package usecases

import (
	"context"
	"time"

	claimports "github.com/ambi/idmagic/backend/claimmapping/ports"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"
)

// ApplicationAccessDecision は割当ゲートの判定結果。
// Application の appdomain.ApplicationAccessDecision と同一形で、adapter が値変換で橋渡しする。
type ApplicationAccessDecision struct {
	Allowed        bool
	StepUpRequired bool
	ApplicationID  string
	Reason         string
}

// ApplicationGate は binding 経由サインインの割当ゲート評価 (adapter が Application の割り当てのゲートを橋渡し)。
type ApplicationGate interface {
	EvaluateApplicationAccess(
		ctx context.Context,
		tenantID string,
		bindingType appdomain.ApplicationProtocolType,
		bindingKey, sub string,
		authn *authdomain.AuthenticationContext,
		clientIP string,
	) (ApplicationAccessDecision, error)
}

// SignInService は WS-Federation passive sign-in の発行判断を所有する。
type SignInService struct {
	RPRepo         wsfedports.WsFedRelyingPartyRepository
	UserRepo       userports.UserRepository
	Gate           ApplicationGate
	Emit           func(spec.DomainEvent)
	AttrSchemaRepo claimports.TenantAttributeSchemaRepo
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
	Message string // Rejected / Forbidden のレスポンスボディ。
	Status  int    // Rejected の HTTP status (400 / 500)。

	// SignInIssued のときの発行データ。
	Validated feddomain.ValidatedSignIn
	// Issuer は Assertion の発行者。空ならテナントの発行者を使う。
	Issuer      string
	ClaimResult claimdomain.ClaimIssuanceResult
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
	reason, err := enforceApplicationAccess(ctx, s.Gate, s.Emit, tenantID, rp.Wtrealm, authn.UserID, authn, in.ClientIP, now)
	if err != nil {
		return SignInOutcome{}, err
	}
	if reason != "" {
		s.emit(&feddomain.WsFedSignInRejected{At: now, TenantID: tenantID, Wtrealm: rp.Wtrealm, Reason: reason})
		return SignInOutcome{Kind: SignInForbidden, Message: "The user does not meet the application's sign-in policy requirements."}, nil
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

	issued, reject, err := issueForRelyingParty(ctx, tenantID, s.AttrSchemaRepo, *rp, *user)
	if err != nil {
		return SignInOutcome{}, err
	}
	if reject != "" {
		s.emit(&feddomain.WsFedSignInRejected{At: now, TenantID: tenantID, Wtrealm: rp.Wtrealm, Reason: reject})
		return SignInOutcome{Kind: SignInRejected, Message: reject, Status: 500}, nil
	}

	return SignInOutcome{
		Kind:        SignInIssued,
		Validated:   validated,
		Issuer:      issued.issuer,
		ClaimResult: issued.claims,
		AuthnMethod: authnMethod,
		TokenType:   rp.EffectiveTokenType(),
		Authn:       authn,
		Now:         now,
	}, nil
}

// rpIssuance は、RP へ発行するクレームと発行者である。発行者が空ならテナントの発行者を使う。
type rpIssuance struct {
	claims claimdomain.ClaimIssuanceResult
	issuer string
}

// issueForRelyingParty は、パッシブサインインと WS-Trust に共通する、RP のクレームと発行者の決定である。
// Entra の定型設定を持つ RP では、ImmutableID を導き、利用者の UPN のドメインの IssuerUri を発行者にする。
// 発行できないときは拒否の理由を返す。
func issueForRelyingParty(
	ctx context.Context, tenantID string, attrSchemaRepo claimports.TenantAttributeSchemaRepo,
	rp feddomain.WsFedRelyingParty, user userdomain.User,
) (rpIssuance, string, error) {
	defs, err := claimports.ResolveTenantAttributeDefs(ctx, tenantID, attrSchemaRepo)
	if err != nil {
		return rpIssuance{}, "", err
	}
	// 構成の後に属性スキーマが変わっても、重なりうるか利用者が書き換えられる値では発行しない。
	if rp.EntraProfile != nil {
		if err := feddomain.CheckEntraSourceAnchorAttribute(defs, rp.EntraProfile); err != nil {
			//nolint:nilerr // 守られていない sourceAnchor は拒否の理由として返す。
			return rpIssuance{}, "entra source anchor not protected", nil
		}
	}
	attrs, err := feddomain.ApplyEntraProfile(claimdomain.ResolveUserAttributes(user), rp.EntraProfile)
	if err != nil {
		//nolint:nilerr // ImmutableID を導けないことは拒否の理由として返し、呼び出し側には error を返さない。
		return rpIssuance{}, "entra profile failed", nil
	}
	var issuer string
	if rp.EntraProfile != nil {
		if issuer, err = rp.EntraProfile.IssuerFor(user.PreferredUsername); err != nil {
			//nolint:nilerr // フェデレーションしていないドメインの利用者は拒否の理由として返す。
			return rpIssuance{}, "entra domain not federated", nil
		}
	}
	result, err := claimdomain.IssueClaimsWithFloor(rp.ClaimPolicy, attrs, feddomain.EntraAttributeDefs(defs, rp.EntraProfile))
	if err != nil {
		//nolint:nilerr // claim 発行失敗 (fail-closed floor 違反を含む) は拒否の理由として返す。
		return rpIssuance{}, "claim issuance failed", nil
	}
	return rpIssuance{claims: result, issuer: issuer}, "", nil
}

func (s SignInService) emit(event spec.DomainEvent) {
	if s.Emit != nil {
		s.Emit(event)
	}
}

// enforceApplicationAccess は wtrealm の RP が属するアプリケーションの割り当てとサインインポリシーで
// 発行を判定する。パッシブサインインと WS-Trust が同じ判定とイベントを使う。拒否したときは
// AppStepUpRequired と AppAccessDeniedByPolicy を発行して理由を返し、許可したときは空の理由を返す。
func enforceApplicationAccess(
	ctx context.Context, gate ApplicationGate, emit func(spec.DomainEvent),
	tenantID, wtrealm, sub string, authn *authdomain.AuthenticationContext, clientIP string, now time.Time,
) (string, error) {
	decision, err := gate.EvaluateApplicationAccess(ctx, tenantID, appdomain.ApplicationProtocolWsFed, wtrealm, sub, authn, clientIP)
	if err != nil {
		return "", err
	}
	if decision.Allowed {
		return "", nil
	}
	send := func(event spec.DomainEvent) {
		if emit != nil {
			emit(event)
		}
	}
	reason := decision.Reason
	if decision.StepUpRequired {
		reason = "step-up required by application sign-in policy"
		send(&appdomain.AppStepUpRequired{At: now, TenantID: tenantID, ApplicationID: decision.ApplicationID, Protocol: string(appdomain.ApplicationProtocolWsFed), Subject: sub})
	} else if reason == "" {
		reason = "subject not assigned to application"
	}
	if decision.ApplicationID != "" {
		send(&appdomain.AppAccessDeniedByPolicy{At: now, TenantID: tenantID, ApplicationID: decision.ApplicationID, Protocol: string(appdomain.ApplicationProtocolWsFed), Subject: sub, Reason: reason})
	}
	return reason, nil
}
