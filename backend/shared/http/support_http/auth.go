package support_http

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

var (
	ErrAdminAuthenticationRequired = errors.New("admin authentication required")
	ErrAdminAccessDenied           = errors.New("admin access denied")
)

type InsufficientScopeError struct{ Required string }

func (e *InsufficientScopeError) Error() string { return "insufficient scope: " + e.Required }

type InvalidTokenError struct{}

func (*InvalidTokenError) Error() string { return "invalid access token" }

// WriteAccessTokenError maps RFC 6750 bearer-token failures to their resource
// server response. It returns handled=false for non-token errors.
func WriteAccessTokenError(c *echo.Context, err error) (handled bool, result error) {
	var tokenErr *InvalidTokenError
	if errors.As(err, &tokenErr) {
		c.Response().Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		return true, WriteBrowserError(c, http.StatusUnauthorized, "invalid_token", "The access token is invalid.")
	}
	var scopeErr *InsufficientScopeError
	if errors.As(err, &scopeErr) {
		c.Response().Header().Set("WWW-Authenticate", `Bearer error="insufficient_scope", scope="`+scopeErr.Required+`"`)
		return true, WriteBrowserError(c, http.StatusForbidden, "insufficient_scope", "The required scope is missing.")
	}
	return false, nil
}

// ResolveAuthentication はリクエストの認証セッションを解決し、対応する有効ユーザが
// リクエスト先テナントに属する場合のみ AuthenticationContext を返す。失効/無効/
// テナント不一致のセッションは未認証 (nil) として扱う (defense-in-depth)。
func (a *Authenticator) ResolveAuthentication(c *echo.Context) (*authdomain.AuthenticationContext, error) {
	authn, err := a.resolveAuthnContext(c)
	if err != nil || authn == nil || a.UserRepo == nil {
		return authn, err
	}
	user, err := a.UserRepo.FindBySub(c.Request().Context(), authn.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive() {
		if a.SessionManager != nil && authn.SessionID != "" {
			_ = a.SessionManager.Store.Revoke(c.Request().Context(), authn.SessionID, spec.SessionEndOther, time.Now().UTC())
		}
		return nil, nil
	}
	// cookie path 分離が破られた場合に備え、リクエスト先のテナントと User の所属テナントが
	// 一致しないセッションは未認証扱い (defense-in-depth)。
	if user.TenantID != RequestTenantID(c) {
		return nil, nil
	}
	return authn, nil
}

// resolveAuthnContext は AuthenticationContext を解決する。OIDC RP 化した portal が
// 提示する Bearer access token を優先し ([[ADR-061]])、無ければ first-party セッション
// cookie で解決する (dual-mode)。Bearer は緊急セッションログイン経路と併存する。
func (a *Authenticator) resolveAuthnContext(c *echo.Context) (*authdomain.AuthenticationContext, error) {
	if token, scheme := authorizationToken(c); token != "" {
		if a.TokenIntrospector == nil {
			return nil, nil
		}
		res, err := a.TokenIntrospector.IntrospectAccessToken(c.Request().Context(), token)
		if err != nil {
			return nil, err
		}
		if res == nil || !res.Active || res.Sub == "" {
			return nil, &InvalidTokenError{}
		}
		if res.Managed {
			if a.ApiTokenAuthenticator == nil {
				return nil, &InvalidTokenError{}
			}
			principal, err := a.ApiTokenAuthenticator.Authenticate(c.Request().Context(), token)
			if err != nil {
				return nil, &InvalidTokenError{}
			}
			if principal.UserID != res.Sub || principal.ClientID != res.ClientID {
				return nil, &InvalidTokenError{}
			}
		}
		if res.SenderConstraint != nil && res.SenderConstraint.Type == spec.SenderConstraintDPoP {
			if scheme != "dpop" || a.DpopReplayStore == nil {
				return nil, &InvalidTokenError{}
			}
			proof, err := tokensjose.VerifyDPoP(c.Request().Context(), c.Request().Header.Get("DPoP"), c.Request().Method, RequestHTU(c, ""), a.DpopReplayStore, time.Now().UTC())
			if err != nil {
				return nil, &InvalidTokenError{}
			}
			if proof == nil || proof.JKT != res.SenderConstraint.JKT {
				return nil, &InvalidTokenError{}
			}
		}
		// resource server のスコープ境界 (ADR-061): admin / account API は対応する
		// portal scope を要求する。account portal の token で admin API を叩く等の
		// cross-portal 利用を fail-closed で拒否する。緊急セッション経路は scope を
		// 持たないが、その経路はこの分岐を通らない (role 境界で守る)。
		fields := strings.Fields(res.Scope)
		if want := requiredPortalScope(c.Request().URL.Path); want != "" {
			if want == "idmagic.account" {
				required, allowed := requiredAccountScope(c.Request().Method, c.Request().URL.Path)
				if !allowed {
					return nil, &InsufficientScopeError{Required: "interactive_session"}
				}
				if !slices.Contains(fields, "idmagic.account") && !slices.Contains(fields, required) {
					return nil, &InsufficientScopeError{Required: required}
				}
			} else if !slices.Contains(fields, want) {
				return nil, &InsufficientScopeError{Required: want}
			}
		}
		// access token 経由は session を持たないので SessionID は空。管理発行 token の
		// sensitive account scope は token possession 自体を継続的な step-up credential と
		// みなす。通常 OAuth token は元の auth_time を維持する。
		authTime := res.Iat
		if res.Managed {
			authTime = time.Now().UTC().Unix()
		}
		return &authdomain.AuthenticationContext{UserID: res.Sub, AuthTime: authTime}, nil
	}
	if a.AuthnResolver == nil {
		return nil, nil
	}
	return a.AuthnResolver.Resolve(
		c.Request().Context(),
		authdomain.HTTPHeadersAdapter{H: c.Request().Header},
	)
}

// requiredPortalScope は resource path が Bearer に要求する portal scope を返す。
// admin / account 以外の API (例: /api/auth/account) は scope を要求しない。
func requiredPortalScope(path string) string {
	switch {
	case strings.Contains(path, "/api/admin/"):
		return "idmagic.admin"
	case strings.Contains(path, "/api/account/") || strings.HasSuffix(path, "/api/auth/account") || strings.HasSuffix(path, "/api/auth/change_password"):
		return "idmagic.account"
	default:
		return ""
	}
}

func requiredAccountScope(method, path string) (string, bool) {
	switch {
	case strings.Contains(path, "/api/account/step_up/"),
		strings.HasSuffix(path, "/api/account/email/verify"),
		strings.HasSuffix(path, "/api/account/email/verify_context"):
		return "", false
	case strings.Contains(path, "/api/account/mfa/"):
		return "account:mfa:write", true
	case strings.Contains(path, "/api/account/sessions/") && method != http.MethodGet:
		return "account:sessions:write", true
	case strings.Contains(path, "/api/account/consents/") && method != http.MethodGet:
		return "account:consents:write", true
	case strings.HasSuffix(path, "/api/auth/change_password"):
		return "account:password:write", true
	case method == http.MethodPatch || method == http.MethodPut || method == http.MethodPost || method == http.MethodDelete:
		return "account:write", true
	default:
		return "account:read", true
	}
}

// bearerToken は Authorization: Bearer <token> を抽出する。無ければ空文字を返す。
func authorizationToken(c *echo.Context) (string, string) {
	h := c.Request().Header.Get("Authorization")
	for _, scheme := range []string{"bearer", "dpop"} {
		prefix := scheme + " "
		if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
			return strings.TrimSpace(h[len(prefix):]), scheme
		}
	}
	return "", ""
}

// RequireAdmin は認証済み + 有効ロールに admin を含むユーザを要求する。
// グループ由来ロールを含めた有効ロールで判定する (ADR-038)。
func (a *Authenticator) RequireAdmin(c *echo.Context) (*userdomain.User, error) {
	authn, err := a.ResolveAuthentication(c)
	if err != nil {
		return nil, err
	}
	if authn == nil || authn.AuthenticationPending {
		return nil, ErrAdminAuthenticationRequired
	}
	user, err := a.UserRepo.FindBySub(c.Request().Context(), authn.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != RequestTenantID(c) || !user.IsActive() ||
		!slices.Contains(a.EffectiveRoles(c.Request().Context(), user), "admin") {
		return nil, ErrAdminAccessDenied
	}
	return user, nil
}

func (a *Authenticator) WriteAdminAccessError(c *echo.Context, err error) error {
	if handled, result := WriteAccessTokenError(c, err); handled {
		return result
	}
	if errors.Is(err, ErrAdminAuthenticationRequired) {
		return WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "An authenticated session is required.")
	}
	if errors.Is(err, ErrAdminAccessDenied) {
		return WriteBrowserError(c, http.StatusForbidden, "access_denied", "Administrator privileges are required.")
	}
	return err
}

// ResolveAdminActor は認証済みかつ有効なユーザを、グループ由来ロールを合成した
// 形で返す。ロール別の細かな認可判定 (key reader / settings admin など) を呼び出し側に
// 委ねる管理系ハンドラが、actor の解決だけを共有するために使う。
func (a *Authenticator) ResolveAdminActor(c *echo.Context) (*userdomain.User, error) {
	authn, err := a.ResolveAuthentication(c)
	if err != nil {
		return nil, err
	}
	if authn == nil || authn.AuthenticationPending {
		return nil, ErrAdminAuthenticationRequired
	}
	user, err := a.UserRepo.FindBySub(c.Request().Context(), authn.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive() {
		return nil, ErrAdminAccessDenied
	}
	return a.WithEffectiveRoles(c.Request().Context(), user), nil
}

// RequireAuditReader は admin または system_admin ロールを持つ認証済みユーザを要求する。
// 監査イベントの閲覧と、そこから派生する認証イベントバケット閲覧が共有する。
func (a *Authenticator) RequireAuditReader(c *echo.Context) (*userdomain.User, error) {
	authn, err := a.ResolveAuthentication(c)
	if err != nil {
		return nil, err
	}
	if authn == nil || authn.AuthenticationPending {
		return nil, ErrAdminAuthenticationRequired
	}
	user, err := a.UserRepo.FindBySub(c.Request().Context(), authn.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive() {
		return nil, ErrAdminAccessDenied
	}
	actor := a.WithEffectiveRoles(c.Request().Context(), user)
	if !slices.Contains(actor.Roles, "admin") && !slices.Contains(actor.Roles, "system_admin") {
		return nil, ErrAdminAccessDenied
	}
	return actor, nil
}

// EffectiveRoles は User の直接ロールにグループ由来ロールを合成して返す (ADR-038)。
func (a *Authenticator) EffectiveRoles(ctx context.Context, user *userdomain.User) []string {
	if a.GroupRepo == nil {
		return user.Roles
	}
	groups, err := a.GroupRepo.ListGroupsByUser(ctx, user.TenantID, user.ID)
	if err != nil {
		return user.Roles
	}
	return groupdomain.EffectiveRoles(user.Roles, groups)
}

// WithEffectiveRoles は Roles を有効ロールへ差し替えた User の複製を返す。
func (a *Authenticator) WithEffectiveRoles(ctx context.Context, user *userdomain.User) *userdomain.User {
	clone := *user
	clone.Roles = a.EffectiveRoles(ctx, user)
	return &clone
}
