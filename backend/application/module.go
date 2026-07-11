// Package application は application bounded context の DI 組立と route 登録を自前で持つ
// (ADR-091, wi-172 パイロット)。中央 server/routes.go の Deps と bootstrap/deps.go の
// Dependencies から application 由来 field を Module 1 個に集約し撤去する。
package application

import (
	apphttp "github.com/ambi/idmagic/backend/application/adapters/http"
	appports "github.com/ambi/idmagic/backend/application/ports"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	samlports "github.com/ambi/idmagic/backend/saml/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	wsfederationports "github.com/ambi/idmagic/backend/wsfederation/ports"

	"github.com/labstack/echo/v5"
)

// Module は application context が所有する repository の束。bootstrap は永続化 backend
// (memory / postgres_valkey) に応じてこれらを組み立て、Module へ渡すだけでよい。
type Module struct {
	Repo                    appports.ApplicationRepository
	IconStore               appports.ApplicationIconStore
	AssignmentRepo          appports.AssignmentRepository
	OrderingRepo            appports.ApplicationOrderingRepository
	CategoryRepo            appports.ApplicationCategoryRepository
	SignInPolicyRepo        appports.SignInPolicyRepository
	DefaultSignInPolicyRepo appports.DefaultSignInPolicyRepository
}

// Gate は Application 割当を fail-closed で判定する published capability を組み立てる
// (SCL context_map: Application publishes ApplicationAssignmentRef)。oauth2 / saml /
// wsfederation の federation 開始経路がこれを消費する。
func (m Module) Gate(groupRepo idmports.GroupRepository, trustedForwardedHops int) *support.ApplicationGate {
	return &support.ApplicationGate{
		ApplicationRepo:             m.Repo,
		ApplicationAssignmentRepo:   m.AssignmentRepo,
		GroupRepo:                   groupRepo,
		ApplicationSignInPolicyRepo: m.SignInPolicyRepo,
		DefaultSignInPolicyRepo:     m.DefaultSignInPolicyRepo,
		GateTrustedForwardedHops:    trustedForwardedHops,
	}
}

// ClientDisplayNames は client_id をアプリ表示名へ解決するリゾルバを組み立てる
// (oauth2 / authentication の監査・表示系が消費する)。
func (m Module) ClientDisplayNames(clientRepo oauthports.OAuth2ClientRepository) *support.ClientDisplayNameResolver {
	return &support.ClientDisplayNameResolver{ClientRepo: clientRepo, ApplicationRepo: m.Repo}
}

// Register は Application カタログの admin / account エンドポイントを登録する。
func (m Module) Register(
	g *echo.Group, deps support.Deps, authenticator *support.Authenticator,
	groupRepo idmports.GroupRepository, userRepo idmports.UserRepository, clientRepo oauthports.OAuth2ClientRepository,
	wsFedRPRepo wsfederationports.WsFedRelyingPartyRepository, samlSPRepo samlports.SamlServiceProviderRepository,
) {
	apphttp.RegisterRoutes(g, apphttp.Deps{
		Deps:                        deps,
		Authenticator:               authenticator,
		ApplicationRepo:             m.Repo,
		ApplicationIconStore:        m.IconStore,
		ApplicationAssignmentRepo:   m.AssignmentRepo,
		ApplicationOrderingRepo:     m.OrderingRepo,
		ApplicationCategoryRepo:     m.CategoryRepo,
		ApplicationSignInPolicyRepo: m.SignInPolicyRepo,
		DefaultSignInPolicyRepo:     m.DefaultSignInPolicyRepo,
		GroupRepo:                   groupRepo,
		UserRepo:                    userRepo,
		ClientRepo:                  clientRepo,
		WsFedRPRepo:                 wsFedRPRepo,
		SamlSPRepo:                  samlSPRepo,
	})
}
