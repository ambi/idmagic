// Package oauth2 は oauth2 モジュールの DI 組立を自前で持つ。
// 中央 server/routes.go の Deps と bootstrap/deps.go の Dependencies から oauth2 由来
// field を Module へ集約していく。client/consent/認可詳細タイプ分から着手し (wi-173)、
// token/grant (wi-181) が Module へフィールドを追加していく。監査 (audit) の repository は
// wi-146 で独立した audit context の Module へ移設した。
// 全フィールドが揃うまでは oauth2/handlers_http.RegisterRoutes は中央 routes.go から
// 直接呼ばれ続け、Module 自身の Register は持たない。
package oauth2

import (
	"time"

	jobsports "github.com/ambi/idmagic/backend/jobs/ports"

	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	clientusecases "github.com/ambi/idmagic/backend/oauth2/client/usecases"
	consentports "github.com/ambi/idmagic/backend/oauth2/consent/ports"
	consentusecases "github.com/ambi/idmagic/backend/oauth2/consent/usecases"
	logoutdomain "github.com/ambi/idmagic/backend/oauth2/logout/domain"
	logoutports "github.com/ambi/idmagic/backend/oauth2/logout/ports"
	logoutusecases "github.com/ambi/idmagic/backend/oauth2/logout/usecases"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/token/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type JobHandlerDeps struct {
	Notifications     logoutports.LogoutNotificationStore
	TokenSigner       logoutports.LogoutTokenSigner
	BackChannelClient logoutports.BackChannelLogoutClient
	Now               func() time.Time
}

func RegisterJobHandlers(registry jobsports.HandlerRegistrar, deps JobHandlerDeps) {
	registry.Register(logoutdomain.KindBackChannelLogoutDelivery, logoutusecases.BackChannelLogoutHandler(logoutusecases.BackChannelLogoutHandlerDeps{
		Notifications: deps.Notifications, Signer: deps.TokenSigner, Client: deps.BackChannelClient, Now: deps.Now,
	}))
}

// Module は oauth2 context が所有する repository の束。bootstrap は永続化 backend
// (memory / postgres) に応じてこれらを組み立て、Module へ渡すだけでよい。
type Module struct {
	ClientRepo                 oauthports.OAuth2ClientRepository
	ConsentRepo                oauthports.ConsentRepository
	AuthzDetailTypeRepo        oauthports.AuthorizationDetailTypeRepository
	McpResourceServerRepo      oauthports.McpResourceServerRepository
	RequestStore               oauthports.AuthorizationRequestStore
	CodeStore                  oauthports.AuthorizationCodeStore
	PARStore                   oauthports.PARStore
	RefreshStore               oauthports.RefreshTokenStore
	DeviceCodeStore            oauthports.DeviceCodeStore
	ApprovalRequestStore       oauthports.ApprovalRequestStore
	DpopReplayStore            oauthports.DpopReplayStore
	ClientAssertionReplayStore oauthports.ClientAssertionReplayStore
	ClientSessionStore         logoutports.ClientSessionStore
	LogoutNotificationStore    logoutports.LogoutNotificationStore
	AccessTokenDenylist        oauthports.AccessTokenDenylist
	TokenIssuer                oauthports.TokenIssuer
	TokenIntrospector          oauthports.TokenIntrospector
	IDTokenHintVerifier        oauthports.IDTokenHintVerifier
	Authorizer                 oauthports.Authorizer
	EventSink                  oauthports.EventSink
	// WorkloadVerifier verifies external workload attestation tokens for the
	// token-exchange grant ([[wi-54-workload-identity-federation-spiffe]]).
	// nil rejects subject_token_type=JWT URN as unsupported (fail-closed).
	WorkloadVerifier oauthports.WorkloadTokenVerifier
}

// AccountConsents は、利用者本人の同意の一覧と撤回の操作を返す。
func (m Module) AccountConsents(emit func(spec.DomainEvent)) consentports.AccountConsents {
	return consentusecases.AccountConsents{ConsentRepo: m.ConsentRepo, Emit: emit}
}

// SessionTokenRevoker は、ログインセッション単位でトークンを失効させる操作を返す。
func (m Module) SessionTokenRevoker() tokenusecases.SessionTokenRevoker {
	return tokenusecases.SessionTokenRevoker{RefreshStore: m.RefreshStore}
}

// ClientAdministration は、ほかのモジュールの管理画面が使うクライアントの管理操作を返す。
func (m Module) ClientAdministration(emit func(spec.DomainEvent), quota tenantports.QuotaRepository) clientports.ClientAdministration {
	return clientusecases.ClientAdministration{Deps: clientusecases.AdminOAuth2ClientDeps{
		ClientRepo: m.ClientRepo, Emit: emit, QuotaRepo: quota,
	}}
}
