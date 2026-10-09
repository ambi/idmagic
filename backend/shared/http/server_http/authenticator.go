package server_http

import (
	apitokenhttp "github.com/ambi/idmagic/backend/apitoken/handlers_http"
	apitokenports "github.com/ambi/idmagic/backend/apitoken/ports"
	authhttpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	idmhttpdeps "github.com/ambi/idmagic/backend/idmanagement/deps_http"
	oauth2http "github.com/ambi/idmagic/backend/oauth2/handlers_http"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/token/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

// newAuthenticator は、管理 API と account API の認証を、各 Context のアダプターを結んで
// 組み立てる。apiTokens が nil なら、管理発行の API アクセストークンを無効として扱う。
func newAuthenticator(d Deps, apiTokens apitokenports.Authenticator) *support.Authenticator {
	authenticator := &support.Authenticator{
		Sessions: authhttpdeps.SessionAuthentications{
			Resolver: d.Authentication.AuthnResolver, Sessions: d.Authentication.SessionManager,
		},
	}
	if d.OAuth2.TokenIntrospector != nil {
		authenticator.AccessTokens = oauth2http.ResourceAccessTokens{
			Introspector: d.OAuth2.TokenIntrospector,
			// admin / account portal の Bearer にも /introspect と同じ失効判定を通す
			// (REQ-OAUTH2-047)。/introspect の配線 (oauth2 handlers_http) と同じ repository 群。
			Revocation: tokenusecases.IntrospectDeps{
				AccessTokenDenylist: d.OAuth2.AccessTokenDenylist,
				AgentRepo:           d.IdManagement.AgentRepo,
				RevocationEpochRepo: d.SharedSignals.RevocationEpochRepo,
			},
			DpopReplayStore: d.OAuth2.DpopReplayStore,
		}
	}
	if apiTokens != nil {
		authenticator.ApiTokens = apitokenhttp.ResourceAuthenticator{Authenticator: apiTokens}
	}
	if d.IdManagement.UserRepo != nil {
		authenticator.Principals = idmhttpdeps.Principals{
			Users: d.IdManagement.UserRepo, Groups: d.IdManagement.GroupRepo,
		}
	}
	return authenticator
}
