package server_http

import (
	"github.com/ambi/idmagic/backend/apitoken"
	apitokenports "github.com/ambi/idmagic/backend/apitoken/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

// newAuthenticator は、管理 API と account API の認証を、各モジュールのアダプターを結んで
// 組み立てる。apiTokens が nil なら、管理発行の API アクセストークンを無効として扱う。
func newAuthenticator(d Deps, apiTokens apitokenports.Authenticator) *support.Authenticator {
	authenticator := &support.Authenticator{
		Sessions: d.Authentication.SessionAuthentications(),
	}
	if d.OAuth2.TokenIntrospector != nil {
		// admin / account portal の Bearer にも /introspect と同じ失効判定を通す (REQ-OAUTH2-047)。
		authenticator.AccessTokens = d.OAuth2.ResourceAccessTokens(
			d.IdManagement.AgentRepo, d.SharedSignals.RevocationEpochRepo,
			d.IdManagement.UserRepo, d.SharedSignals.UserRevocationEpochRepo,
		)
	}
	if apiTokens != nil {
		authenticator.ApiTokens = apitoken.ResourceAuthenticator(apiTokens)
	}
	if d.IdManagement.UserRepo != nil {
		authenticator.Principals = d.IdManagement.Principals()
	}
	return authenticator
}
