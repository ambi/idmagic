package handlers_http

import (
	"context"

	"github.com/ambi/idmagic/backend/apitoken/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

// ResourceAuthenticator は、共有の HTTP 支援の Bearer 認証へ、管理発行の API アクセストークンと
// その有効な記録の照合を渡す。
type ResourceAuthenticator struct {
	Authenticator ports.Authenticator
}

// AuthenticateApiToken は、トークンの有効な記録が持つ利用者、クライアント、粒度スコープを返す。
func (r ResourceAuthenticator) AuthenticateApiToken(ctx context.Context, token string) (support.ApiTokenPrincipal, error) {
	principal, err := r.Authenticator.Authenticate(ctx, token)
	if err != nil {
		return support.ApiTokenPrincipal{}, err
	}
	return support.ApiTokenPrincipal{
		UserID: principal.UserID, ClientID: principal.ClientID, Scopes: principal.Scopes.Strings(),
	}, nil
}
