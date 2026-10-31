// Package http: oauth2 コンテキストの HTTP アダプタ。
//
// OAuth 2.0 / OIDC のプロトコルエンドポイント (authorize/token/introspect/revoke/
// userinfo/par/device/discovery/register) と、認可トランザクションの画面 (transaction/
// consent/end_session)、および client/consent/key の管理 API を所有する。
// 管理ロールの表示は Authorization、監査 (audit_event) の管理 API は audit context が
// 所有する (wi-146)。共有基盤 support.Deps を受け取り router から登録される。
package handlers_http

import (
	"context"
	"net/http"
	"time"

	policydomain "github.com/ambi/idmagic/backend/signinpolicy/domain"

	accessports "github.com/ambi/idmagic/backend/accesscontrol/ports"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	consentusecases "github.com/ambi/idmagic/backend/oauth2/internal/consent/usecases"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/internal/token/usecases"
	logoutports "github.com/ambi/idmagic/backend/oauth2/logout/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	policyports "github.com/ambi/idmagic/backend/signinpolicy/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

// Deps は OAuth2 HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator
	ApplicationGate

	AuthzDetailTypeRepo   oauthports.AuthorizationDetailTypeRepository
	McpResourceServerRepo oauthports.McpResourceServerRepository
	// AgentGuardrailPolicies は Agent の発行の上限である (REQ-OAUTH2-073)。
	AgentGuardrailPolicies oauthports.AgentGuardrailPolicyRepository
	// CrossAppAccessPolicies から IDJAGReplayStore までは ID-JAG の発行と償還の依存である。
	CrossAppAccessPolicies     oauthports.CrossAppAccessPolicyRepository
	IDJAGTrust                 oauthports.IDJAGIssuerTrust
	IDTokenAssertionVerifier   oauthports.IDTokenAssertionVerifier
	IDJAGSigner                oauthports.IDJAGSigner
	IDJAGReplayStore           oauthports.ClientAssertionReplayStore
	ClientRepo                 oauthports.OAuth2ClientRepository
	ConsentRepo                oauthports.ConsentRepository
	ClientDisplayNameResolver  ClientDisplayNameResolver
	KeyStore                   signingports.KeyStore
	TenantRepo                 tenantports.TenantRepository
	PARStore                   oauthports.PARStore
	RequestStore               oauthports.AuthorizationRequestStore
	UserRepo                   userports.UserRepository
	CodeStore                  oauthports.AuthorizationCodeStore
	JWKResolver                *tokens_jose.JWKResolver
	ClientAssertionReplayStore oauthports.ClientAssertionReplayStore
	ClientSessionStore         logoutports.ClientSessionStore
	LogoutNotificationStore    logoutports.LogoutNotificationStore
	JobEnqueuer                jobsports.Enqueuer
	DeviceCodeStore            oauthports.DeviceCodeStore
	ApprovalRequestStore       oauthports.ApprovalRequestStore
	Notifier                   notificationports.Notifier
	DpopReplayStore            oauthports.DpopReplayStore
	RefreshStore               oauthports.RefreshTokenStore
	TokenIssuer                oauthports.TokenIssuer
	AgentRepo                  agentports.AgentRepository
	// RevocationEpochRepo backs Introspect's fail-closed Agent revocation
	// check (wi-58). nil skips the check.
	RevocationEpochRepo ssports.AgentRevocationEpochRepository
	// UserRevocationEpochRepo backs the User revocation check (REQ-SHAREDSIGNALS-012).
	UserRevocationEpochRepo ssports.UserRevocationEpochRepository
	// WorkloadVerifier is the composition-root wiring point for workload
	// identity federation ([[wi-54-workload-identity-federation-spiffe]]).
	// nil rejects subject_token_type=JWT URN token-exchange as unsupported.
	WorkloadVerifier    oauthports.WorkloadTokenVerifier
	TokenIntrospector   oauthports.TokenIntrospector
	IDTokenHintVerifier oauthports.IDTokenHintVerifier
	AccessTokenDenylist oauthports.AccessTokenDenylist
	ManagedTokenRevoker tokenusecases.ManagedTokenRevoker
	BuiltinResource     tokenusecases.BuiltinResource
	AttrSchemaRepo      userports.TenantUserAttributeSchemaRepository
	Authorizer          accessports.Authorizer
	// RelationChecker は /resource-access/evaluate の関係の判定。nil なら評価不能を返す。
	RelationChecker accessports.RelationChecker
	// QuotaRepo enforces the tenant's Hard Quota on oauth2_clients (client
	// registration) and consents (wi-160). nil skips enforcement.
	QuotaRepo tenantports.QuotaRepository

	// AuthnResolver は対話のログインが作ったログインセッションを読む。Authentication が所有し、
	// 組み立て地点が結ぶ。
	AuthnResolver authdomain.AuthenticationContextResolver
	// LoginSessions は /end_session がログインセッションを終わらせる先である。
	LoginSessions LoginSessions
	// SignIn は、既存のセッションに追加の認証を求める先である。
	SignIn SignInSteps

	// ApplicationSignInPolicyRepo と DefaultSignInPolicyRepo は、外部のアプリケーションへの
	// 認可で MFA の登録を始めさせるかの方針を読む。ApplicationGate と同じ保存先を結ぶ。
	ApplicationSignInPolicyRepo policyports.SignInPolicyRepository
	DefaultSignInPolicyRepo     policyports.DefaultSignInPolicyRepository
}

// ResolveAuthentication はリクエストの認証を、AMR や保留の目的まで読める認証の文脈として返す。
func (d Deps) ResolveAuthentication(c *echo.Context) (*authdomain.AuthenticationContext, error) {
	authn, err := d.Authenticate(c)
	if err != nil || authn == nil {
		return nil, err
	}
	return authdomain.ContextOf(authn), nil
}

// ApplicationGate は、フェデレーションの開始を Application の割り当てとサインインポリシーで
// 判定する。判定は Application が所有し、組み立て地点が結ぶ。
type ApplicationGate interface {
	EvaluateApplicationAccess(
		ctx context.Context,
		tenantID string,
		bindingType appdomain.ApplicationProtocolType,
		bindingKey, sub string,
		authn policydomain.SignInAuthentication,
		clientIP string,
	) (appdomain.ApplicationAccessDecision, error)
	ClientIP(r *http.Request) string
}

// SignInSteps は、認可エンドポイントが既存のセッションに追加の認証を求める操作である。
// 対話のログインを所有する Authentication が満たし、組み立て地点が結ぶ。
type SignInSteps interface {
	// EnforceTenantDefault は IdMagic 自身の画面への認可にテナントのデフォルトのポリシーを適用する。
	// true は追加の認証の画面へ移すことを表す。拒否の応答を書いたときはそれを表すエラーを返す。
	EnforceTenantDefault(c *echo.Context, authn *authdomain.AuthenticationContext, allowChallenge bool) (bool, error)
	PendingPath(c *echo.Context, authn *authdomain.AuthenticationContext) string
	CanUseTOTP(c *echo.Context, sub string) bool
	SecondFactorMethods(c *echo.Context, sub string) []string
	// RequireSecondFactor はセッションを第二要素の照合の待ちにして Cookie を発行する。
	// セッションが失効していれば nil を返す。
	RequireSecondFactor(ctx context.Context, c *echo.Context, sessionID string) (*authdomain.AuthenticationContext, error)
	PendingStep(c *echo.Context, authn *authdomain.AuthenticationContext) (kind string, methods []string, canRememberDevice bool)
	ValidReturnTo(c *echo.Context, returnTo string) bool
	// CompleteSignIn は、ほかの経路で成立した認証を記録し、認可を続けずに進ませる画面があれば返す。
	CompleteSignIn(c *echo.Context, user *userdomain.User, authn *authdomain.AuthenticationContext, clientID string, at time.Time) (string, error)
}

// ClientDisplayNameResolver は、同意と認可の画面に出す client_id の表示名を解決する。
// 解決の順序は Application が所有し、組み立て地点が結ぶ。
type ClientDisplayNameResolver interface {
	Resolve(ctx context.Context, tenantID, clientID string) string
	ResolveAll(ctx context.Context, tenantID string, clientIDs []string) map[string]string
}

// RegisterRoutes はテナント解決済みグループに oauth2 コンテキストのエンドポイントを
// 登録する。パス・メソッド・middleware は分割前と一致する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/authorize", d.handleAuthorize)
	g.GET("/authorize/resume", d.handleFederatedResume)
	g.GET("/end_session", d.handleEndSession)
	g.POST("/end_session", d.handleEndSession)
	g.GET("/api/auth/transaction", d.handleTransaction)
	g.POST("/api/auth/consent", d.handleConsentAPI)
	g.GET("/api/auth/device", d.handleDeviceContext)
	g.POST("/api/auth/device", d.handleDeviceAPI)
	g.POST("/token", d.handleToken)
	g.POST("/revoke", d.handleRevoke)
	g.POST("/introspect", d.handleIntrospect)
	g.POST("/resource-access/evaluate", d.handleEvaluateResourceAccess)
	g.GET("/userinfo", d.handleUserInfo)
	g.POST("/userinfo", d.handleUserInfo)
	g.POST("/register", d.handleRegisterClient)
	g.POST("/par", d.handlePAR)
	g.POST("/device_authorization", d.handleDeviceAuthorization)
	g.POST("/bc-authorize", d.handleBackchannelAuthenticate)
	g.GET("/api/account/v1/consents", d.handleListMyConsents)
	g.POST("/api/account/v1/consents/:client_id/revoke", d.handleRevokeMyConsent)
	g.GET("/api/account/v1/delegable-agents", d.handleListDelegableAgents)
	g.GET("/api/account/v1/agent-delegations", d.handleListMyAgentDelegations)
	g.POST("/api/account/v1/agent-delegations", d.handleGrantMyAgentDelegation)
	g.POST("/api/account/v1/agent-delegations/:agent_id/revoke", d.handleRevokeMyAgentDelegation)
	g.GET("/api/account/v1/approval-requests", d.handleListMyApprovalRequests)
	g.POST("/api/account/v1/approval-requests/:id/decision", d.handleDecideMyApprovalRequest)
	g.GET("/.well-known/openid-configuration", d.handleDiscovery)
	g.GET("/.well-known/oauth-authorization-server", d.handleDiscovery)
	g.GET("/session/check", d.handleCheckSessionIframe)
	g.GET("/api/admin/v1/clients", d.handleListAdminOAuth2Clients)
	g.GET("/api/admin/v1/clients/:client_id", d.handleGetAdminOAuth2Client)
	g.POST("/api/admin/v1/clients", d.handleCreateAdminOAuth2Client)
	g.PATCH("/api/admin/v1/clients/:client_id", d.handleUpdateAdminOAuth2Client)
	g.DELETE("/api/admin/v1/clients/:client_id", d.handleDeleteAdminOAuth2Client)
	g.GET("/api/admin/v1/authorization-detail-types", d.handleListAuthorizationDetailTypes)
	g.GET("/api/admin/v1/authorization-detail-types/:type", d.handleGetAuthorizationDetailType)
	g.POST("/api/admin/v1/authorization-detail-types", d.handleCreateAuthorizationDetailType)
	g.PATCH("/api/admin/v1/authorization-detail-types/:type", d.handleUpdateAuthorizationDetailType)
	g.DELETE("/api/admin/v1/authorization-detail-types/:type", d.handleDeleteAuthorizationDetailType)
	g.GET("/api/admin/v1/agent-delegations", d.handleListAdminAgentDelegations)
	g.GET("/api/admin/v1/consents", d.handleListAdminConsents)
	g.GET("/api/admin/v1/consents/:sub/:client_id", d.handleGetAdminConsent)
	g.DELETE("/api/admin/v1/consents/:sub/:client_id", d.handleRevokeAdminConsent)
	g.GET("/api/admin/v1/mcp-resource-servers", d.handleListAdminMcpResourceServers)
	g.GET("/api/admin/v1/mcp-resource-servers/:id", d.handleGetAdminMcpResourceServer)
	g.POST("/api/admin/v1/mcp-resource-servers", d.handleCreateAdminMcpResourceServer)
	g.PATCH("/api/admin/v1/mcp-resource-servers/:id", d.handleUpdateAdminMcpResourceServer)
	g.DELETE("/api/admin/v1/mcp-resource-servers/:id", d.handleDeleteAdminMcpResourceServer)
	g.GET("/api/admin/v1/agents/:agent_id/guardrail-policy", d.handleGetAdminAgentGuardrailPolicy)
	g.PUT("/api/admin/v1/agents/:agent_id/guardrail-policy", d.handlePutAdminAgentGuardrailPolicy)
	g.DELETE("/api/admin/v1/agents/:agent_id/guardrail-policy", d.handleDeleteAdminAgentGuardrailPolicy)
	g.GET("/api/admin/v1/agent-guardrail-default", d.handleGetAdminAgentGuardrailDefault)
	g.PUT("/api/admin/v1/agent-guardrail-default", d.handlePutAdminAgentGuardrailDefault)
	g.DELETE("/api/admin/v1/agent-guardrail-default", d.handleDeleteAdminAgentGuardrailDefault)
	g.GET("/api/admin/v1/cross-app-access/issue-policies", d.handleListAdminCrossAppAccessIssuePolicies)
	g.POST("/api/admin/v1/cross-app-access/issue-policies", d.handleCreateAdminCrossAppAccessIssuePolicy)
	g.DELETE("/api/admin/v1/cross-app-access/issue-policies/:id", d.handleDeleteAdminCrossAppAccessIssuePolicy)
	g.GET("/api/admin/v1/cross-app-access/redeem-policies", d.handleListAdminCrossAppAccessRedeemPolicies)
	g.POST("/api/admin/v1/cross-app-access/redeem-policies", d.handleCreateAdminCrossAppAccessRedeemPolicy)
	g.DELETE("/api/admin/v1/cross-app-access/redeem-policies/:id", d.handleDeleteAdminCrossAppAccessRedeemPolicy)
	g.GET("/.well-known/oauth-protected-resource", d.handleProtectedResourceMetadata)
}

func (d Deps) ConsentDeps() consentusecases.ConsentDeps {
	return consentusecases.ConsentDeps{ConsentRepo: d.ConsentRepo, Emit: d.Emit, QuotaRepo: d.QuotaRepo}
}
