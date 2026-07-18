package http

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"

	"github.com/labstack/echo/v5"
)

// completeAfterAuthn は認証済みセッションから OAuth フローの続きを決める。同意が必要な
// 場合は同意画面への遷移を、不要なら認可コード発行後のリダイレクトを返す。
func (d Deps) completeAfterAuthn(
	c *echo.Context,
	req *oauthdomain.AuthorizationRequest,
	client *oauthdomain.OAuth2Client,
	authn *authdomain.AuthenticationContext,
) (authorizationNext, error) {
	prompt := oauthdomain.PromptTokens{}
	if req.Prompt != nil {
		prompt, _ = oauthdomain.ParsePromptTokens(*req.Prompt)
	}
	if authn.AuthenticationPending {
		return authorizationNext{Path: d.pendingAuthPath(c, authn)}, nil
	}
	// first-party クライアント (IdP 自身の管理コンソール / アカウントポータル) は
	// resource owner が IdP 利用者自身であるため consent をスキップする (ADR-061)。
	if d.ConsentRepo != nil && !client.FirstParty {
		consent, _ := d.ConsentRepo.Find(
			c.Request().Context(), support.RequestTenantID(c), authn.UserID, client.ClientID,
		)
		covered := consent != nil &&
			consent.State == oauthdomain.ConsentGranted &&
			consent.RevokedAt == nil &&
			time.Now().Before(consent.ExpiresAt)
		if covered {
			for scope := range strings.FieldsSeq(req.Scope) {
				if !containsString(consent.Scopes, scope) {
					covered = false
					break
				}
			}
		}
		if prompt.Consent {
			covered = false
		}
		// RFC 9396 — 構造化された authorization_details は粗い scope 同意では代替できない。
		// 明示同意を要求し、過去 scope 同意での自動スキップを許さない (fail-closed, ADR-050)。
		if len(req.AuthorizationDetails) > 0 {
			covered = false
		}
		if !covered {
			if prompt.None {
				return authorizationNext{RedirectTo: authorizationErrorURL(req, tenancy.Issuer(c.Request().Context(), d.Issuer), "consent_required", "既存同意が必要です")}, nil
			}
			ctx, cancel := d.OperationContext(c.Request().Context())
			defer cancel()
			if err := d.RequestStore.AttachAuthentication(
				ctx, req.ID, authn.UserID, authn.AuthTime, authn.AMR, authn.ACR, authn.SessionID,
			); err != nil {
				return authorizationNext{}, err
			}
			req.UserID, req.AuthTime = &authn.UserID, &authn.AuthTime
			return authorizationNext{Path: support.TenantRoute(c, "/consent")}, nil
		}
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	redirectTo, err := d.issueCodeURL(ctx, c, req, authn, time.Unix(authn.AuthTime, 0))
	return authorizationNext{RedirectTo: redirectTo}, err
}

func (d Deps) canUseTOTP(c *echo.Context, sub string) bool {
	if d.MfaFactorRepo == nil {
		return false
	}
	factor, err := d.MfaFactorRepo.Find(c.Request().Context(), sub, spec.MfaFactorTOTP)
	return err == nil && factor != nil && factor.Secret != nil && *factor.Secret != ""
}

// clientIsFirstParty は client_id が first-party クライアントかを返す。解決不能なら false
// (fail-closed で割当ゲートを適用する)。
func (d Deps) clientIsFirstParty(ctx context.Context, clientID string) bool {
	if d.ClientRepo == nil {
		return false
	}
	client, err := d.ClientRepo.FindByID(ctx, tenancy.TenantID(ctx), clientID)
	return err == nil && client != nil && client.FirstParty
}

func (d Deps) issueCodeURL(
	ctx context.Context,
	c *echo.Context,
	req *oauthdomain.AuthorizationRequest,
	authn *authdomain.AuthenticationContext,
	authTime time.Time,
) (string, error) {
	iss := tenancy.Issuer(ctx, d.Issuer)
	tenantID := tenancy.TenantID(ctx)
	// 割当ゲート (wi-69): client が Application binding に属する場合、未割当 subject には
	// 認可コードを発行せず access_denied で RP へ返す (fail-closed, AssignmentGatesProtocol)。
	// ただし first-party クライアント (IdP 自身の管理コンソール / アカウントポータル) は
	// resource owner が IdP 利用者自身であり、アプリ割当でログインをゲートしない (ADR-061)。
	if !d.clientIsFirstParty(ctx, req.ClientID) {
		decision, err := d.EvaluateApplicationAccess(
			ctx, tenantID, appdomain.ProtocolBindingOIDC, req.ClientID, authn.UserID, authn, d.ClientIP(c.Request()),
		)
		if err != nil {
			return "", err
		}
		if decision.StepUpRequired {
			if d.Emit != nil {
				d.Emit(&appdomain.AppStepUpRequired{
					At: time.Now().UTC(), TenantID: tenantID, ApplicationID: decision.ApplicationID,
					Protocol: string(appdomain.ProtocolBindingOIDC), Subject: authn.UserID,
				})
			}
			if len(d.secondFactorMethods(c, authn.UserID)) > 0 { //nolint:contextcheck // HTTP request context is required for factor lookup.
				pending, err := d.SessionManager.RequireFactor(ctx, authn.SessionID)
				if err != nil {
					return "", err
				}
				if pending == nil {
					return authorizationErrorURL(req, iss, "login_required", "既存セッションが認証要件を満たしません"), nil
				}
				d.setSessionCookie(c, pending.SessionID)    //nolint:contextcheck // Cookie path is derived from the Echo request.
				return support.TenantRoute(c, "/totp"), nil //nolint:contextcheck // Redirect URL is derived from the Echo request.
			}
			return authorizationErrorURL(req, iss, "access_denied", "アプリケーションのサインインポリシーを満たせません"), nil
		}
		if !decision.Allowed {
			reason := decision.Reason
			if reason == "" {
				reason = "subject not assigned to application"
			}
			if d.Emit != nil && decision.ApplicationID != "" {
				d.Emit(&appdomain.AppAccessDeniedByPolicy{
					At: time.Now().UTC(), TenantID: tenantID, ApplicationID: decision.ApplicationID,
					Protocol: string(appdomain.ProtocolBindingOIDC), Subject: authn.UserID, Reason: reason,
				})
			}
			return authorizationErrorURL(req, iss, "access_denied", "この利用者はアプリケーションにアクセスできません"), nil
		}
	}
	out, err := usecases.CompleteLogin(ctx, usecases.CompleteLoginDeps{
		RequestStore: d.RequestStore,
		CodeStore:    d.CodeStore,
	}, usecases.CompleteLoginInput{
		RequestID: req.ID,
		Sub:       authn.UserID,
		AuthTime:  authTime,
		AMR:       authn.AMR,
		ACR:       authn.ACR,
		Sid:       authn.SessionID,
	})
	if err != nil {
		var oauthErr *usecases.OAuthError
		if errors.As(err, &oauthErr) {
			return authorizationErrorURL(req, iss, oauthErr.Code, oauthErr.Description), nil
		}
		return "", err
	}
	if d.Emit != nil {
		d.Emit(&oauthdomain.AuthorizationCodeIssued{
			At: time.Now().UTC(), TenantID: tenantID, ClientID: req.ClientID, UserID: authn.UserID,
			Scopes: out.Code.Scopes, CodeChallengeMethod: req.CodeChallengeMethod,
		})
	}
	u, _ := url.Parse(out.Request.RedirectURI)
	query := u.Query()
	query.Set("code", out.Code.Code)
	if out.Request.StateParam != nil {
		query.Set("state", *out.Request.StateParam)
	}
	// RFC 9207 §2: Authorization Server Issuer Identification (mix-up 攻撃対策)。
	query.Set("iss", iss)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func authorizationErrorURL(req *oauthdomain.AuthorizationRequest, iss, code, description string) string {
	u, _ := url.Parse(req.RedirectURI)
	query := u.Query()
	query.Set("error", code)
	if description != "" {
		query.Set("error_description", description)
	}
	if req.StateParam != nil {
		query.Set("state", *req.StateParam)
	}
	// RFC 9207 §2: error response も含めて iss を必須にする。
	if iss != "" {
		query.Set("iss", iss)
	}
	u.RawQuery = query.Encode()
	return u.String()
}
