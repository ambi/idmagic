// /userinfo
package handlers_http

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	claimusecases "github.com/ambi/idmagic/backend/claimmapping/usecases"
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
	return claimusecases.ResolveTenantAttributeDefs(ctx, tenantID, d.AttrSchemaRepo)
}

func (d Deps) handleUserInfo(c *echo.Context) error {
	auth := c.Request().Header.Get("Authorization")
	dpopHeader := c.Request().Header.Get("DPoP")
	bearer := strings.HasPrefix(auth, "Bearer ")
	dpopAuth := strings.HasPrefix(auth, "DPoP ")
	if !bearer && !dpopAuth {
		return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_token", "Bearer token is required"))
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
		return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_token", "The token is invalid."))
	}
	if d.AccessTokenDenylist != nil && intro.JTI != "" {
		revoked, err := d.AccessTokenDenylist.IsRevoked(c.Request().Context(), intro.JTI)
		if err != nil {
			return writeOAuthError(c, err)
		}
		if revoked {
			return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_token", "The token has expired."))
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
				return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_token", "mTLS certificate binding mismatch"))
			}
		case spec.SenderConstraintDPoP:
			if dpopHeader == "" || d.DpopReplayStore == nil {
				return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_token", "DPoP proof is required"))
			}
			// ath is checked against the access token string the client presented,
			// not against any post-introspection representation of it.
			r, err := tokensJOSE.VerifyDPoPForResource(
				c.Request().Context(), dpopHeader,
				c.Request().Method, support.RequestHTU(c, d.Issuer), token,
				d.DpopReplayStore, time.Now().UTC(),
			)
			if err != nil || r == nil || subtle.ConstantTimeCompare(
				[]byte(r.JKT), []byte(intro.SenderConstraint.JKT),
			) != 1 {
				return writeOAuthError(c, tokenusecases.NewOAuthError("invalid_token", "DPoP key binding mismatch"))
			}
		}
	}
	var claimPolicy *claimdomain.ClaimMappingPolicy
	if d.ClientRepo != nil {
		client, err := d.ClientRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), intro.ClientID)
		if err != nil {
			return writeOAuthError(c, err)
		}
		if client != nil {
			claimPolicy = client.ClaimPolicy
		}
	}
	res, err := tokenusecases.UserInfo(c.Request().Context(), d.UserRepo, d.Authorizer, tokenusecases.UserInfoInput{
		Scopes: strings.Fields(intro.Scope), Sub: intro.Sub, Active: intro.Active, ClientID: intro.ClientID,
		ResolveAttributeDefs: d.effectiveUserAttributeDefs, ClaimPolicy: claimPolicy,
	})
	if err != nil {
		return writeOAuthError(c, err)
	}
	return c.JSON(http.StatusOK, res)
}
