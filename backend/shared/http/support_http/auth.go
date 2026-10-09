package support_http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

var (
	ErrAdminAuthenticationRequired = errors.New("admin authentication required")
	ErrAdminAccessDenied           = errors.New("admin access denied")
)

// ErrAdminAccessRefused は WriteAdminAccessError が拒否の応答を書き終えたことを表す。
// ErrResponseWritten を包むので、応答済みかどうかだけを見たい呼び出し元は
// errors.Is(err, ErrResponseWritten) で足りる。
var ErrAdminAccessRefused = fmt.Errorf("%w: admin access refused", ErrResponseWritten)

type InsufficientScopeError struct{ Required string }

func (e *InsufficientScopeError) Error() string { return "insufficient scope: " + e.Required }

type InvalidTokenError struct{}

func (*InvalidTokenError) Error() string { return "invalid access token" }

// Authenticator はリクエストの認証と、管理 API と account API の認可を判定する。
//
// HTTP の資格情報の解釈、スコープ、ロールの判定はここが持つ。セッション、アクセストークン、
// API トークン、利用者の語彙はそれぞれの Context が持つので、インターフェースとして受け取り、
// 組み立て地点が各 Context のアダプターを結ぶ。nil のインターフェースは、その資格情報の
// 配線がないことを表す。
type Authenticator struct {
	Sessions     SessionAuthenticator
	AccessTokens AccessTokenVerifier
	ApiTokens    ApiTokenAuthenticator
	Principals   PrincipalDirectory
}

// Authentication は、リクエストを認証した結果である。具体的な値は認証した経路が決め、
// ここで読むのは主体、認証時刻、ログインセッション、認証の保留だけである。
type Authentication interface {
	Subject() string
	AuthenticatedAt() int64
	Session() string
	Pending() bool
}

// SessionAuthenticator は、ログインセッションの Cookie からリクエストを認証する。
type SessionAuthenticator interface {
	// ResolveSession は、有効なセッションがなければ nil を返す。
	ResolveSession(ctx context.Context, header http.Header) (Authentication, error)
	// EndSession は、主体が無効になったセッションを終わらせる。
	EndSession(ctx context.Context, sessionID string) error
}

// AccessToken は、イントロスペクションで有効と判定し、失効していないアクセストークンである。
type AccessToken struct {
	Subject  string
	ClientID string
	Scope    string
	Audience []string
	IssuedAt int64
	// Managed は、管理コンソールが発行した API アクセストークンであることを表す。
	Managed bool
	// DPoPBound は、トークンが DPoP の鍵に束縛されていることを表す。DPoPJKT はその鍵の拇印。
	DPoPBound bool
	DPoPJKT   string
}

// DPoPProof は、保護されたリソースへのリクエストが提示した DPoP の証明である
// (RFC 9449 §7)。AccessToken は ath と照合する、クライアントが提示したトークンの文字列。
type DPoPProof struct {
	Header      string
	Method      string
	HTU         string
	AccessToken string
}

// AccessTokenVerifier は、Bearer と DPoP のアクセストークンを検証する。
type AccessTokenVerifier interface {
	// IntrospectAccessToken は、トークンが無効または失効していれば nil を返す。
	IntrospectAccessToken(ctx context.Context, token string) (*AccessToken, error)
	// VerifyDPoPProof は、証明を検証して鍵の拇印を返す。証明がなければエラーを返す。
	VerifyDPoPProof(ctx context.Context, proof DPoPProof) (jkt string, err error)
}

// ApiTokenPrincipal は、管理発行の API アクセストークンの有効な記録である。
type ApiTokenPrincipal struct {
	UserID   string
	ClientID string
	Scopes   []string
}

// ApiTokenAuthenticator は、管理発行の API アクセストークンを有効な記録と照合する。
type ApiTokenAuthenticator interface {
	AuthenticateApiToken(ctx context.Context, token string) (ApiTokenPrincipal, error)
}

// Principal は、認可の判定が読む呼び出し主体である。
type Principal struct {
	ID       string
	TenantID string
	Roles    []string
	Active   bool
}

// PrincipalDirectory は、主体を引き、所属から導かれるロールを合成する。
type PrincipalDirectory interface {
	// FindPrincipal は、主体がいなければ nil を返す。Roles は直接のロールである。
	FindPrincipal(ctx context.Context, subject string) (*Principal, error)
	// EffectiveRoles は、直接のロールに Group 由来のロールを合成して返す。
	EffectiveRoles(ctx context.Context, principal Principal) []string
}

// accessTokenAuthentication は、アクセストークンで認証した結果である。セッションを持たない。
type accessTokenAuthentication struct {
	subject  string
	authTime int64
}

func (a accessTokenAuthentication) Subject() string        { return a.subject }
func (a accessTokenAuthentication) AuthenticatedAt() int64 { return a.authTime }
func (accessTokenAuthentication) Session() string          { return "" }
func (accessTokenAuthentication) Pending() bool            { return false }

// SetBearerChallenge writes one RFC 6750 challenge and advertises the RFC 9728
// metadata endpoint derived from the tenant's canonical issuer and TypeSpec contract.
func SetBearerChallenge(c *echo.Context, errorCode, requiredScope string) {
	params := []string{"Bearer error=" + strconv.Quote(errorCode)}
	if requiredScope != "" {
		params = append(params, "scope="+strconv.Quote(requiredScope))
	}
	if metadataURL := protectedResourceMetadataURL(c); metadataURL != "" {
		params = append(params, "resource_metadata="+strconv.Quote(metadataURL))
	}
	c.Response().Header().Set("WWW-Authenticate", strings.Join(params, ", "))
}

func protectedResourceMetadataURL(c *echo.Context) string {
	issuer := strings.TrimRight(RequestIssuer(c, ""), "/")
	operation, ok := spec.CurrentRuntimeContract().Operation("GetProtectedResourceMetadata")
	if issuer == "" || !ok {
		return ""
	}
	return issuer + operation.Path
}

// WriteAccessTokenError maps RFC 6750 bearer-token failures to their resource
// server response. It returns handled=false for non-token errors.
func WriteAccessTokenError(c *echo.Context, err error) (handled bool, result error) {
	if _, ok := errors.AsType[*InvalidTokenError](err); ok {
		SetBearerChallenge(c, "invalid_token", "")
		return true, WriteProblem(c, http.StatusUnauthorized, "invalid_token", "The access token is invalid.")
	}
	if scopeErr, ok := errors.AsType[*InsufficientScopeError](err); ok {
		SetBearerChallenge(c, "insufficient_scope", scopeErr.Required)
		return true, WriteProblem(c, http.StatusForbidden, "insufficient_scope", "The required scope is missing.")
	}
	return false, nil
}

// Authenticate はリクエストを認証し、有効な主体がリクエスト先テナントに属する場合だけ
// 結果を返す。失効/無効/テナント不一致の主体は未認証 (nil) として扱う (defense-in-depth)。
func (a *Authenticator) Authenticate(c *echo.Context) (Authentication, error) {
	authn, err := a.resolveAuthentication(c)
	if err != nil || authn == nil || a.Principals == nil {
		return authn, err
	}
	ctx := c.Request().Context()
	principal, err := a.Principals.FindPrincipal(ctx, authn.Subject())
	if err != nil {
		return nil, err
	}
	if principal == nil || !principal.Active {
		if a.Sessions != nil && authn.Session() != "" {
			_ = a.Sessions.EndSession(ctx, authn.Session())
		}
		return nil, nil
	}
	// cookie path 分離が破られた場合に備え、リクエスト先のテナントと主体の所属テナントが
	// 一致しない認証は未認証扱い (defense-in-depth)。
	if principal.TenantID != RequestTenantID(c) {
		return nil, nil
	}
	return authn, nil
}

// resolveAuthentication は認証の結果を解決する。OIDC RP 化した portal が提示する Bearer
// access token を優先し、無ければ first-party セッション cookie で解決する (dual-mode)。
// Bearer は緊急セッションログイン経路と併存する。
func (a *Authenticator) resolveAuthentication(c *echo.Context) (Authentication, error) {
	if token, scheme := authorizationToken(c); token != "" {
		return a.authenticateAccessToken(c, token, scheme)
	}
	if a.Sessions == nil {
		return nil, nil
	}
	return a.Sessions.ResolveSession(c.Request().Context(), c.Request().Header)
}

func (a *Authenticator) authenticateAccessToken(c *echo.Context, raw, scheme string) (Authentication, error) {
	if a.AccessTokens == nil {
		return nil, nil
	}
	ctx := c.Request().Context()
	// イントロスペクションと失効の判定 (REQ-OAUTH2-047) は OAuth2 が単独で所有し、
	// この経路と /introspect が同じ規則を共有する。
	token, err := a.AccessTokens.IntrospectAccessToken(ctx, raw)
	if err != nil {
		return nil, err
	}
	if token == nil || token.Subject == "" {
		return nil, &InvalidTokenError{}
	}
	// account スコープの資源はレルムの IdMagic API である (RFC9068-DEFAULT-AUDIENCE)。
	// 別の資源へ宛てたトークンを、同じスコープ名を持つだけで受けない。不一致はスコープ
	// 不足ではなくトークン自体の無効なので、RFC 6750 §3.1 の invalid_token で返す。
	if !accountScopedTokenNamesRealmAPI(c, token) {
		return nil, &InvalidTokenError{}
	}
	var apiToken *ApiTokenPrincipal
	if token.Managed {
		if a.ApiTokens == nil {
			return nil, &InvalidTokenError{}
		}
		principal, err := a.ApiTokens.AuthenticateApiToken(ctx, raw)
		if err != nil {
			return nil, &InvalidTokenError{}
		}
		if principal.UserID != token.Subject || principal.ClientID != token.ClientID {
			return nil, &InvalidTokenError{}
		}
		apiToken = &principal
	}
	// DPoP の検証は証明をリプレイ記録へ書き込むので、ここまでの判定で拒否するトークンの
	// 証明は記録しない。
	if token.DPoPBound {
		if scheme != "dpop" {
			return nil, &InvalidTokenError{}
		}
		// ath is checked against the access token string the client presented,
		// not against any post-introspection representation of it.
		jkt, err := a.AccessTokens.VerifyDPoPProof(ctx, DPoPProof{
			Header: c.Request().Header.Get("DPoP"), Method: c.Request().Method,
			HTU: RequestHTU(c, ""), AccessToken: raw,
		})
		if err != nil || jkt != token.DPoPJKT {
			return nil, &InvalidTokenError{}
		}
	}
	// resource server のスコープ境界 : admin / account API は対応する
	// portal scope を要求する。account portal の token で admin API を叩く等の
	// cross-portal 利用を fail-closed で拒否する。緊急セッション経路は scope を
	// 持たないが、その経路はこの分岐を通らない (role 境界で守る)。
	//
	// API アクセストークンは portal scope を持たず、リソースと操作の粒度スコープだけを
	// 持つ。そこで管理 API では、契約が operation ごとに宣言した粒度スコープが portal
	// scope の代わりになる (REQ-APITOKENS-004)。account API が既に同じ形を採っている。
	fields := strings.Fields(token.Scope)
	if want := requiredPortalScope(c.Request().URL.Path); want != "" {
		switch {
		case want == "idmagic.account":
			required, allowed := requiredAccountScope(c.Request().Method, c.Request().URL.Path)
			if !allowed {
				return nil, &InsufficientScopeError{Required: spec.InteractiveSessionScope}
			}
			if !hasRequiredAccountScope(fields, required, c.Request().URL.Path) {
				return nil, &InsufficientScopeError{Required: required}
			}
		case apiToken != nil:
			if err := requireAdminApiTokenScope(c, spec.CurrentRuntimeContract(), apiToken.Scopes); err != nil {
				return nil, err
			}
		case !slices.Contains(fields, want):
			return nil, &InsufficientScopeError{Required: want}
		}
	}
	// access token 経由は session を持たないので Session は空。管理発行 token の
	// sensitive account scope は token possession 自体を継続的な step-up credential と
	// みなす。通常 OAuth token は元の auth_time を維持する。
	authTime := token.IssuedAt
	if token.Managed {
		authTime = time.Now().UTC().Unix()
	}
	return accessTokenAuthentication{subject: token.Subject, authTime: authTime}, nil
}

// accountScopedTokenNamesRealmAPI は、account スコープを持つトークンの audience がリクエスト先
// レルムの IdMagic API (レルムの発行者識別子) を含むかを返す。account スコープを持たない
// トークンには課さない。リクエスト先レルムを解決できなければ、照合先が無いので満たさない。
func accountScopedTokenNamesRealmAPI(c *echo.Context, token *AccessToken) bool {
	if !slices.ContainsFunc(strings.Fields(token.Scope), spec.IsAccountScope) {
		return true
	}
	realmAPI := RequestIssuer(c, "")
	return realmAPI != "" && slices.Contains(token.Audience, realmAPI)
}

// requiredPortalScope は resource path が Bearer に要求する portal scope を返す。
// /api/auth/account は両 portal の共通 bootstrap だが account read 契約として分類し、
// authenticateAccessToken で idmagic.admin も明示的に許可する。
func requiredPortalScope(path string) string {
	switch {
	case strings.Contains(path, "/api/admin/v1/"):
		return "idmagic.admin"
	case strings.Contains(path, "/api/account/v1/") || strings.HasSuffix(path, "/api/auth/account") || strings.HasSuffix(path, "/api/auth/change-password"):
		return "idmagic.account"
	default:
		return ""
	}
}

func requiredAccountScope(method, path string) (string, bool) {
	switch {
	case strings.Contains(path, "/api/account/v1/step-up/"),
		strings.HasSuffix(path, "/api/account/v1/email/verify"),
		strings.HasSuffix(path, "/api/account/v1/email/verify-context"):
		return "", false
	case strings.Contains(path, "/api/account/v1/mfa/"):
		return "account:mfa:write", true
	case strings.Contains(path, "/api/account/v1/sessions/") && method != http.MethodGet:
		return "account:sessions:write", true
	case strings.Contains(path, "/api/account/v1/consents/") && method != http.MethodGet:
		return "account:consents:write", true
	case strings.HasSuffix(path, "/api/auth/change-password"):
		return "account:password:write", true
	case method == http.MethodPatch || method == http.MethodPut || method == http.MethodPost || method == http.MethodDelete:
		return "account:write", true
	default:
		return "account:read", true
	}
}

func hasRequiredAccountScope(granted []string, required, path string) bool {
	accepted := []string{"idmagic.account", required}
	// AccountContext は両 portal が共有する bootstrap endpoint なので、管理 portal の
	// scope もこの route に限って同等の read scope として扱う。
	if strings.HasSuffix(path, "/api/auth/account") {
		accepted = append(accepted, "idmagic.admin")
	}
	return slices.ContainsFunc(accepted, func(scope string) bool {
		return slices.Contains(granted, scope)
	})
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

// RequireAdmin は認証済み + 有効ロールに admin を含む主体を要求する。
// グループ由来ロールを含めた有効ロールで判定し、主体は直接のロールのまま返す。
func (a *Authenticator) RequireAdmin(c *echo.Context) (*Principal, error) {
	authn, err := a.Authenticate(c)
	if err != nil {
		return nil, err
	}
	if authn == nil || authn.Pending() {
		return nil, ErrAdminAuthenticationRequired
	}
	principal, err := a.Principals.FindPrincipal(c.Request().Context(), authn.Subject())
	if err != nil {
		return nil, err
	}
	if principal == nil || principal.TenantID != RequestTenantID(c) || !principal.Active ||
		!slices.Contains(a.EffectiveRoles(c.Request().Context(), *principal), "admin") {
		return nil, ErrAdminAccessDenied
	}
	return principal, nil
}

// IsControlPlaneActor は、解決済みの主体が制御面のテナント横断操作を行えるかを返す。
// 呼び出し元は Group 由来を合成した有効ロールを actor.Roles に入れてから渡す。
func IsControlPlaneActor(actor *Principal, requestTenantID string) bool {
	return actor != nil && actor.Active &&
		actor.TenantID == tenancydomain.DefaultTenantID &&
		requestTenantID == tenancydomain.DefaultTenantID &&
		slices.Contains(actor.Roles, "system_admin")
}

// RequireControlPlaneUser は、認証を完了した制御面主体を有効ロール付きで返す。
// 所属先と要求先の両方を制御面テナントに限定し、別テナントに保存された
// system_admin がロール名だけでテナント境界を越えないようにする。
func (a *Authenticator) RequireControlPlaneUser(c *echo.Context) (*Principal, error) {
	authn, err := a.Authenticate(c)
	if err != nil {
		return nil, err
	}
	if authn == nil || authn.Pending() {
		return nil, ErrAdminAuthenticationRequired
	}
	principal, err := a.Principals.FindPrincipal(c.Request().Context(), authn.Subject())
	if err != nil {
		return nil, err
	}
	if principal == nil {
		return nil, ErrAdminAccessDenied
	}
	actor := a.withEffectiveRoles(c.Request().Context(), principal)
	if !IsControlPlaneActor(actor, RequestTenantID(c)) {
		return nil, ErrAdminAccessDenied
	}
	return actor, nil
}

// WriteAdminAccessError は管理 API の認証・認可の失敗を応答へ写し、応答を書き終えた
// 合図として ErrAdminAccessRefused を返す。応答を書いた結果 (成功時は nil) を返しては
// ならない。この関数を包んで拒否を判定するヘルパーがあり、そこへ nil を返すと
// `if err != nil { return err }` が素通りし、403 を書いた後も操作が実行される。
// 写像できないエラーはそのまま返し、呼び出し元とエラーハンドラーに委ねる。
func (a *Authenticator) WriteAdminAccessError(c *echo.Context, err error) error {
	if handled, result := WriteAccessTokenError(c, err); handled {
		return refused(result)
	}
	if errors.Is(err, ErrAdminAuthenticationRequired) {
		return refused(WriteProblem(c, http.StatusUnauthorized, "authentication_required", "An authenticated session is required."))
	}
	if errors.Is(err, ErrAdminAccessDenied) {
		return refused(WriteProblem(c, http.StatusForbidden, "access_denied", "Administrator privileges are required."))
	}
	return err
}

// refused は応答の書き込み結果を、呼び出し元が止まれる形へ変える。書き込み自体が
// 失敗していればその失敗を優先する。
func refused(writeErr error) error {
	if writeErr != nil {
		return writeErr
	}
	return ErrAdminAccessRefused
}

// ResolveAdminActor は認証済みかつ有効な主体を、グループ由来ロールを合成した
// 形で返す。ロール別の細かな認可判定 (key reader / settings admin など) を呼び出し側に
// 委ねる管理系ハンドラが、actor の解決だけを共有するために使う。
func (a *Authenticator) ResolveAdminActor(c *echo.Context) (*Principal, error) {
	authn, err := a.Authenticate(c)
	if err != nil {
		return nil, err
	}
	if authn == nil || authn.Pending() {
		return nil, ErrAdminAuthenticationRequired
	}
	principal, err := a.Principals.FindPrincipal(c.Request().Context(), authn.Subject())
	if err != nil {
		return nil, err
	}
	if principal == nil || !principal.Active {
		return nil, ErrAdminAccessDenied
	}
	return a.withEffectiveRoles(c.Request().Context(), principal), nil
}

// RequireAdministrator は admin または system_admin ロールを持つ認証済み主体を要求する。
// admin だけを要求する RequireAdmin と違い、テナントの管理者と system_admin の双方に開く
// 操作が使う。ロールはグループ由来を合成した有効ロールで判定し、合成済みの主体を返す。
func (a *Authenticator) RequireAdministrator(c *echo.Context) (*Principal, error) {
	authn, err := a.Authenticate(c)
	if err != nil {
		return nil, err
	}
	if authn == nil || authn.Pending() {
		return nil, ErrAdminAuthenticationRequired
	}
	principal, err := a.Principals.FindPrincipal(c.Request().Context(), authn.Subject())
	if err != nil {
		return nil, err
	}
	if principal == nil || !principal.Active {
		return nil, ErrAdminAccessDenied
	}
	actor := a.withEffectiveRoles(c.Request().Context(), principal)
	if !slices.Contains(actor.Roles, "admin") && !slices.Contains(actor.Roles, "system_admin") {
		return nil, ErrAdminAccessDenied
	}
	return actor, nil
}

// EffectiveRoles は主体の直接ロールにグループ由来ロールを合成して返す。
func (a *Authenticator) EffectiveRoles(ctx context.Context, principal Principal) []string {
	if a.Principals == nil {
		return principal.Roles
	}
	return a.Principals.EffectiveRoles(ctx, principal)
}

// withEffectiveRoles は Roles を有効ロールへ差し替えた主体の複製を返す。
func (a *Authenticator) withEffectiveRoles(ctx context.Context, principal *Principal) *Principal {
	clone := *principal
	clone.Roles = a.EffectiveRoles(ctx, *principal)
	return &clone
}
