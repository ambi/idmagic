// /userinfo
package handlers_http

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/token/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	certificatesMTLS "github.com/ambi/idmagic/backend/shared/security/certificates_mtls"
	tokensJOSE "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

// effectiveUserAttributeDefs はテナントに有効な属性定義 (組み込み + tenant custom)
// を返す。AttrSchemaRepo 未設定時は組み込み定義のみ。
func (d Deps) effectiveUserAttributeDefs(ctx context.Context, tenantID string) ([]userdomain.UserAttributeDef, error) {
	defs := userdomain.BuiltinUserAttributeDefs()
	if d.AttrSchemaRepo == nil {
		return defs, nil
	}
	schema, err := d.AttrSchemaRepo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if schema != nil {
		defs = append(defs, schema.Attributes...)
	}
	return defs, nil
}

func (d Deps) handleUserInfo(c *echo.Context) error {
	auth := c.Request().Header.Get("Authorization")
	dpopHeader := c.Request().Header.Get("DPoP")
	bearer := strings.HasPrefix(auth, "Bearer ")
	dpopAuth := strings.HasPrefix(auth, "DPoP ")
	if !bearer && !dpopAuth {
		return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_token", "Bearer token が必要"))
	}
	var token string
	if bearer {
		token = strings.TrimPrefix(auth, "Bearer ")
	} else {
		token = strings.TrimPrefix(auth, "DPoP ")
	}
	intro, err := d.TokenIntrospector.IntrospectAccessToken(c.Request().Context(), token)
	if err != nil {
		return writeOAuthError(c, err)
	}
	if !intro.Active {
		return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_token", "トークンが無効"))
	}
	if d.AccessTokenDenylist != nil && intro.JTI != "" {
		revoked, err := d.AccessTokenDenylist.IsRevoked(c.Request().Context(), intro.JTI)
		if err != nil {
			return writeOAuthError(c, err)
		}
		if revoked {
			return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_token", "トークンが失効済みです"))
		}
	}
	if intro.SenderConstraint != nil {
		switch intro.SenderConstraint.Type {
		case spec.SenderConstraintMTLS:
			cert, err := certificatesMTLS.ParseClientCertificateHeader(c.Request().Header.Get(clientCertHeader))
			if err != nil || subtle.ConstantTimeCompare(
				[]byte(cert.ThumbprintS256),
				[]byte(intro.SenderConstraint.X5TS256),
			) != 1 {
				return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_token", "mTLS 証明書バインドが一致しません"))
			}
		case spec.SenderConstraintDPoP:
			if dpopHeader == "" || d.DpopReplayStore == nil {
				return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_token", "DPoP proof が必要"))
			}
			r, err := tokensJOSE.VerifyDPoP(
				c.Request().Context(), dpopHeader,
				c.Request().Method, support.RequestHTU(c, d.Issuer),
				d.DpopReplayStore, time.Now().UTC(),
			)
			if err != nil || r == nil || subtle.ConstantTimeCompare(
				[]byte(r.JKT), []byte(intro.SenderConstraint.JKT),
			) != 1 {
				return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_token", "DPoP 鍵バインドが一致しません"))
			}
		}
	}
	res, err := tokenusecases.UserInfo(c.Request().Context(), d.UserRepo, d.Authorizer, tokenusecases.UserInfoInput{
		Scopes: strings.Fields(intro.Scope), Sub: intro.Sub, Active: intro.Active, ClientID: intro.ClientID,
		ResolveAttributeDefs: d.effectiveUserAttributeDefs,
	})
	if err != nil {
		return writeOAuthError(c, err)
	}
	return c.JSON(http.StatusOK, res)
}
