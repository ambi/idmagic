package deps_http

import (
	"context"
	"net/http"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// SessionAuthentications は、共有の HTTP 支援の認証へ、ログインセッションによる認証を渡す。
// Resolver がなければ Cookie は未認証のまま、Sessions がなければセッションを終わらせない。
type SessionAuthentications struct {
	Resolver authdomain.AuthenticationContextResolver
	Sessions *sessionusecases.SessionManager
}

// ResolveSession は、セッションの Cookie から認証の文脈を解決する。
func (s SessionAuthentications) ResolveSession(ctx context.Context, header http.Header) (support.Authentication, error) {
	if s.Resolver == nil {
		return nil, nil //nolint:nilnil // 解決器がなければ Cookie は未認証のまま。
	}
	authn, err := s.Resolver.Resolve(ctx, authdomain.HTTPHeadersAdapter{H: header})
	if err != nil {
		return nil, err
	}
	if authn == nil {
		// nil の *AuthenticationContext を包むと、呼び出し元の nil 判定を素通りする。
		return nil, nil //nolint:nilnil // 有効なセッションがないことを表す。
	}
	return authn, nil
}

// EndSession は、主体が無効になったセッションを失効させる。
func (s SessionAuthentications) EndSession(ctx context.Context, sessionID string) error {
	if s.Sessions == nil {
		return nil
	}
	return s.Sessions.Store.Revoke(ctx, sessionID, spec.SessionEndOther, time.Now().UTC())
}
