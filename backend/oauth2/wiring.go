package oauth2

import (
	"net/http"

	"github.com/labstack/echo/v5"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	signinports "github.com/ambi/idmagic/backend/authentication/signin/ports"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	cimdhttp "github.com/ambi/idmagic/backend/oauth2/internal/client/cimd_http"
	clientpostgres "github.com/ambi/idmagic/backend/oauth2/internal/client/db_postgres"
	consentpostgres "github.com/ambi/idmagic/backend/oauth2/internal/consent/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/internal/db_postgres"
	oauth2http "github.com/ambi/idmagic/backend/oauth2/internal/handlers_http"
	logoutmemory "github.com/ambi/idmagic/backend/oauth2/internal/logout/db_memory"
	logoutpostgres "github.com/ambi/idmagic/backend/oauth2/internal/logout/db_postgres"
	logoutpush "github.com/ambi/idmagic/backend/oauth2/internal/logout/push_http"
	tokenpostgres "github.com/ambi/idmagic/backend/oauth2/internal/token/db_postgres"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/internal/token/usecases"
	tokensjose "github.com/ambi/idmagic/backend/oauth2/internal/tokens_jose"
	logoutports "github.com/ambi/idmagic/backend/oauth2/logout/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	policyports "github.com/ambi/idmagic/backend/signinpolicy/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// JWTSigner は起動時に組み立てる OAuth2 の署名器である。
type JWTSigner = tokensjose.JWTSigner

// NewJWTSigner は OAuth2 のトークンの組み立てと署名を行うアダプターを返す。
func NewJWTSigner(issuer string, ks signingports.KeyStore) *JWTSigner {
	return tokensjose.NewJWTSigner(issuer, ks)
}

// NewMemoryModule は OAuth2 が所有する保存先をメモリで作る。トークンの発行器、検証器、
// イベントの出力先は組み立て地点が設定する。
func NewMemoryModule() Module {
	return Module{
		ClientRepo:                 db_memory.NewClientRepository(),
		ConsentRepo:                db_memory.NewConsentRepository(),
		AuthzDetailTypeRepo:        db_memory.NewAuthorizationDetailTypeRepository(),
		McpResourceServerRepo:      db_memory.NewMcpResourceServerRepository(),
		RequestStore:               db_memory.NewAuthorizationRequestStore(),
		CodeStore:                  db_memory.NewAuthorizationCodeStore(),
		PARStore:                   db_memory.NewPARStore(),
		RefreshStore:               db_memory.NewRefreshTokenStore(),
		DeviceCodeStore:            db_memory.NewDeviceCodeStore(),
		ApprovalRequestStore:       db_memory.NewApprovalRequestStore(),
		DpopReplayStore:            db_memory.NewDpopReplayStore(),
		ClientAssertionReplayStore: db_memory.NewClientAssertionReplayStore(),
		ClientSessionStore:         logoutmemory.NewClientSessionStore(),
		LogoutNotificationStore:    logoutmemory.NewLogoutNotificationStore(),
		AccessTokenDenylist:        db_memory.NewAccessTokenDenylist(),
	}
}

// NewPostgresModule は OAuth2 が所有する保存先を PostgreSQL で作る。トークンの発行器、検証器、
// イベントの出力先は組み立て地点が設定する。
func NewPostgresModule(pool postgres.DB) Module {
	return Module{
		ClientRepo:                 &clientpostgres.OAuth2ClientRepository{Pool: pool},
		ConsentRepo:                &consentpostgres.ConsentRepository{Pool: pool},
		AuthzDetailTypeRepo:        &db_postgres.AuthorizationDetailTypeRepository{Pool: pool},
		McpResourceServerRepo:      &db_postgres.McpResourceServerRepository{Pool: pool},
		RequestStore:               &db_postgres.AuthorizationRequestStore{Pool: pool},
		CodeStore:                  &db_postgres.AuthorizationCodeStore{Pool: pool},
		PARStore:                   &db_postgres.PARStore{Pool: pool},
		RefreshStore:               &tokenpostgres.RefreshTokenStore{Pool: pool},
		DeviceCodeStore:            &db_postgres.DeviceCodeStore{Pool: pool},
		ApprovalRequestStore:       &db_postgres.ApprovalRequestStore{Pool: pool},
		DpopReplayStore:            &db_postgres.ReplayStore{Pool: pool, Kind: "dpop"},
		ClientAssertionReplayStore: &db_postgres.ReplayStore{Pool: pool, Kind: "client_assertion"},
		ClientSessionStore:         &logoutpostgres.ClientSessionStore{Pool: pool},
		LogoutNotificationStore:    &logoutpostgres.NotificationStore{Pool: pool},
		AccessTokenDenylist:        &db_postgres.AccessTokenDenylist{Pool: pool},
	}
}

// WithClientMetadataDocuments は、登録されていない client_id が Client ID Metadata Document の
// URL なら文書を取得して解決するよう、クライアントの保存先を包んだ Module を返す。
// 登録済みのクライアントの読み書きは包む前の保存先へそのまま委ねる。
func (m Module) WithClientMetadataDocuments() Module {
	m.ClientRepo = &cimdhttp.ClientRepositoryWithCIMD{OAuth2ClientRepository: m.ClientRepo, Fetcher: cimdhttp.NewFetcher()}
	return m
}

// BindClientMetadataEmit は、Client ID Metadata Document の解決と拒否のイベントの発行先を設定する。
// 発行先は保存先より後に組み立てられるので、組み立て地点が後から結ぶ。
// WithClientMetadataDocuments で包んでいなければ何もしない。
func (m Module) BindClientMetadataEmit(emit func(spec.DomainEvent)) {
	if repo, ok := m.ClientRepo.(*cimdhttp.ClientRepositoryWithCIMD); ok {
		repo.Emit = emit
	}
}

// NewBackChannelLogoutClient は、RP の back-channel logout の宛先へログアウトトークンを送る
// クライアントを返す。httpClient が nil なら既定の HTTP クライアントを使う。
func NewBackChannelLogoutClient(httpClient *http.Client) logoutports.BackChannelLogoutClient {
	return logoutpush.NewBackChannelLogoutClient(httpClient)
}

// BuiltinResource は、組み立て地点が管理発行トークンの所有者から作って RouteDeps へ渡す、
// realm の API の公開スコープと失効用の組み込みクライアントである。
type BuiltinResource = tokenusecases.BuiltinResource

// RouteDeps は OAuth2 の経路が共通基盤とほかのモジュールから受け取る依存である。
type RouteDeps struct {
	support.Deps
	*support.Authenticator
	ApplicationGate           oauth2http.ApplicationGate
	ClientDisplayNameResolver oauth2http.ClientDisplayNameResolver
	KeyStore                  signingports.KeyStore
	TenantRepo                tenantports.TenantRepository
	UserRepo                  userports.UserRepository
	JWKResolver               *tokens_jose.JWKResolver
	JobEnqueuer               jobsports.Enqueuer
	Notifier                  notificationports.Notifier
	AgentRepo                 agentports.AgentRepository
	RevocationEpochRepo       ssports.AgentRevocationEpochRepository
	// WorkloadVerifier は token exchange の外部ワークロードのトークンを検証する。nil は拒否する。
	WorkloadVerifier    oauthports.WorkloadTokenVerifier
	ManagedTokenRevoker tokenusecases.ManagedTokenRevoker
	BuiltinResource     BuiltinResource
	AttrSchemaRepo      userports.TenantUserAttributeSchemaRepository
	QuotaRepo           tenantports.QuotaRepository

	AuthnResolver authdomain.AuthenticationContextResolver
	LoginSessions oauth2http.LoginSessions
	SignIn        oauth2http.SignInSteps

	ApplicationSignInPolicyRepo policyports.SignInPolicyRepository
	DefaultSignInPolicyRepo     policyports.DefaultSignInPolicyRepository
}

// RegisterRoutes は OAuth 2.0 と OIDC のプロトコル、認可トランザクションの画面、
// クライアントと同意の管理 API の経路を登録する。
func (m Module) RegisterRoutes(g *echo.Group, d RouteDeps) {
	oauth2http.RegisterRoutes(g, m.httpDeps(d))
}

// SignInContinuation は、Authentication の対話のログインが認証を終えたあとに、認可要求を
// 続ける操作を返す。経路と同じ依存で組み立てる。
func (m Module) SignInContinuation(d RouteDeps) signinports.PendingAuthorizations {
	return oauth2http.NewSignInContinuation(m.httpDeps(d))
}

// ResourceAccessTokens は、管理 API と account API の Bearer 認証へ、/introspect と同じ
// 失効の判定と DPoP の検証を渡す検証器を返す。
func (m Module) ResourceAccessTokens(
	agents agentports.AgentRepository, epochs ssports.AgentRevocationEpochRepository,
) support.AccessTokenVerifier {
	return oauth2http.ResourceAccessTokens{
		Introspector: m.TokenIntrospector,
		Revocation: tokenusecases.IntrospectDeps{
			AccessTokenDenylist: m.AccessTokenDenylist,
			AgentRepo:           agents,
			RevocationEpochRepo: epochs,
		},
		DpopReplayStore: m.DpopReplayStore,
	}
}

func (m Module) httpDeps(d RouteDeps) oauth2http.Deps {
	return oauth2http.Deps{
		Deps:                        d.Deps,
		Authenticator:               d.Authenticator,
		ApplicationGate:             d.ApplicationGate,
		AuthzDetailTypeRepo:         m.AuthzDetailTypeRepo,
		McpResourceServerRepo:       m.McpResourceServerRepo,
		ClientRepo:                  m.ClientRepo,
		ConsentRepo:                 m.ConsentRepo,
		ClientDisplayNameResolver:   d.ClientDisplayNameResolver,
		KeyStore:                    d.KeyStore,
		TenantRepo:                  d.TenantRepo,
		PARStore:                    m.PARStore,
		RequestStore:                m.RequestStore,
		UserRepo:                    d.UserRepo,
		CodeStore:                   m.CodeStore,
		JWKResolver:                 d.JWKResolver,
		ClientAssertionReplayStore:  m.ClientAssertionReplayStore,
		ClientSessionStore:          m.ClientSessionStore,
		LogoutNotificationStore:     m.LogoutNotificationStore,
		JobEnqueuer:                 d.JobEnqueuer,
		DeviceCodeStore:             m.DeviceCodeStore,
		ApprovalRequestStore:        m.ApprovalRequestStore,
		Notifier:                    d.Notifier,
		DpopReplayStore:             m.DpopReplayStore,
		RefreshStore:                m.RefreshStore,
		TokenIssuer:                 m.TokenIssuer,
		AgentRepo:                   d.AgentRepo,
		RevocationEpochRepo:         d.RevocationEpochRepo,
		WorkloadVerifier:            d.WorkloadVerifier,
		TokenIntrospector:           m.TokenIntrospector,
		IDTokenHintVerifier:         m.IDTokenHintVerifier,
		AccessTokenDenylist:         m.AccessTokenDenylist,
		ManagedTokenRevoker:         d.ManagedTokenRevoker,
		BuiltinResource:             d.BuiltinResource,
		AttrSchemaRepo:              d.AttrSchemaRepo,
		Authorizer:                  m.Authorizer,
		QuotaRepo:                   d.QuotaRepo,
		AuthnResolver:               d.AuthnResolver,
		LoginSessions:               d.LoginSessions,
		SignIn:                      d.SignIn,
		ApplicationSignInPolicyRepo: d.ApplicationSignInPolicyRepo,
		DefaultSignInPolicyRepo:     d.DefaultSignInPolicyRepo,
	}
}
