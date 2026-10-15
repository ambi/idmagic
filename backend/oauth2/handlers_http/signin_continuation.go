package handlers_http

import (
	"fmt"
	"net/http"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	signinports "github.com/ambi/idmagic/backend/authentication/signin/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// errUnknownClient は、認可要求のクライアントが見つからず 400 を書き終えたことを表す。
var errUnknownClient = fmt.Errorf("%w: the client of the authorization request does not exist", support.ErrResponseWritten)

// SignInContinuation は、Authentication の対話のログインが認証を終えたあとに、Cookie の
// 認可要求を続ける。組み立て地点が Authentication へ渡す。
type SignInContinuation struct{ d Deps }

// NewSignInContinuation は d の保存先と判定で認可要求を続ける SignInContinuation を返す。
func NewSignInContinuation(d Deps) SignInContinuation { return SignInContinuation{d: d} }

var _ signinports.PendingAuthorizations = SignInContinuation{}

// Find はリクエストの Cookie が指す、まだ認証されていない認可要求を返す。
func (s SignInContinuation) Find(c *echo.Context) (signinports.Authorization, error) {
	req, err := s.d.transactionRequest(c)
	if err != nil {
		return nil, err
	}
	return &pendingAuthorization{d: s.d, req: req}, nil
}

// pendingAuthorization は一つの認可要求の続きである。Requirement で解決したクライアントを、
// 同じ要求の ContinueAfterPassword でも使う。
type pendingAuthorization struct {
	d      Deps
	req    *domain.AuthorizationRequest
	client *domain.OAuth2Client
}

func (p *pendingAuthorization) ClientID() string { return p.req.ClientID }

func (p *pendingAuthorization) Requirement(
	c *echo.Context, authn *authdomain.AuthenticationContext,
) (signinports.SignInRequirement, error) {
	client, err := p.d.ClientRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), p.req.ClientID)
	if err != nil || client == nil {
		if writeErr := support.WriteProblem(c, http.StatusBadRequest, "invalid_transaction", "The client does not exist."); writeErr != nil {
			return signinports.SignInRequirement{}, writeErr
		}
		return signinports.SignInRequirement{}, errUnknownClient
	}
	p.client = client
	if client.FirstParty {
		return signinports.SignInRequirement{TenantDefault: true}, nil
	}
	decision, err := p.d.EvaluateApplicationAccess(
		c.Request().Context(), support.RequestTenantID(c), appdomain.ApplicationProtocolOIDC, p.req.ClientID,
		authn.UserID, authn, p.d.ClientIP(c.Request()),
	)
	if err != nil {
		return signinports.SignInRequirement{}, err
	}
	return signinports.SignInRequirement{
		Allowed: decision.Allowed, StepUpRequired: decision.StepUpRequired,
		TrustedDeviceAllowed: decision.TrustedDeviceAllowed, ApplicationID: decision.ApplicationID,
	}, nil
}

func (p *pendingAuthorization) EnrollmentPolicy(c *echo.Context, applicationID string) (*appdomain.MfaEnrollmentPolicy, error) {
	return p.d.applicationMfaEnrollmentPolicy(c, applicationID)
}

func (p *pendingAuthorization) DeniedRedirect(c *echo.Context, code, description string) string {
	return authorizationErrorURL(p.req, support.RequestIssuer(c, p.d.Issuer), code, description)
}

func (p *pendingAuthorization) ContinueAfterPassword(c *echo.Context, authn *authdomain.AuthenticationContext) error {
	p.req.UserID, p.req.AuthTime, p.req.AMR, p.req.ACR = &authn.UserID, &authn.AuthTime, authn.AMR, &authn.ACR
	return p.continueAuthorization(c, p.client, authn)
}

func (p *pendingAuthorization) ContinueAfterSecondFactor(c *echo.Context, completed *authdomain.AuthenticationContext) error {
	if err := p.d.RequestStore.AttachAuthentication(
		c.Request().Context(), p.req.ID, completed.UserID, completed.AuthTime, completed.AMR, completed.ACR, completed.SessionID,
	); err != nil {
		return writeOAuthError(c, err)
	}
	p.req.UserID, p.req.AuthTime, p.req.AMR, p.req.ACR = &completed.UserID, &completed.AuthTime, completed.AMR, &completed.ACR
	p.req.Sid = &completed.SessionID
	client, err := p.d.ClientRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), p.req.ClientID)
	if err != nil || client == nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_transaction", "The client does not exist.")
	}
	return p.continueAuthorization(c, client, completed)
}

func (p *pendingAuthorization) continueAuthorization(c *echo.Context, client *domain.OAuth2Client, authn *authdomain.AuthenticationContext) error {
	next, err := p.d.completeAfterAuthn(c, p.req, client, authn)
	if err != nil {
		return err
	}
	if next.RedirectTo != "" {
		p.d.clearTransactionCookie(c)
	}
	return writeAuthorizationNext(c, next)
}

// applicationMfaEnrollmentPolicy は、アプリケーションの実効のサインインポリシーから、第二要素を
// 持たない利用者に MFA の登録を始めさせるかの方針を返す。
func (d Deps) applicationMfaEnrollmentPolicy(c *echo.Context, applicationID string) (*appdomain.MfaEnrollmentPolicy, error) {
	if d.ApplicationGate == nil {
		return &appdomain.MfaEnrollmentPolicy{}, nil
	}
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	var appPolicy *appdomain.AppSignInPolicy
	var defaultPolicy *appdomain.TenantDefaultSignInPolicy
	var err error
	if d.ApplicationSignInPolicyRepo != nil {
		appPolicy, err = d.ApplicationSignInPolicyRepo.Get(ctx, tenantID, applicationID)
		if err != nil {
			return nil, err
		}
	}
	if d.DefaultSignInPolicyRepo != nil {
		defaultPolicy, err = d.DefaultSignInPolicyRepo.Get(ctx, tenantID)
		if err != nil {
			return nil, err
		}
	}
	return appdomain.MfaEnrollmentPolicyFromRules(appdomain.EffectiveSignInRules(defaultPolicy, appPolicy)), nil
}

// extractClientIP は共有の support.ExtractClientIP へ委ねる。プロキシの信頼の規則を一か所に保つ。
func extractClientIP(request *http.Request, trustedHops int) string {
	return support.ExtractClientIP(request, trustedHops)
}
