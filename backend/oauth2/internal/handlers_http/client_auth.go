// クライアント認証 (client_secret_basic / client_secret_post / private_key_jwt)
package handlers_http

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/internal/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	certificatesMTLS "github.com/ambi/idmagic/backend/shared/security/certificates_mtls"
	tokensJOSE "github.com/ambi/idmagic/backend/shared/security/tokens_jose"

	"github.com/labstack/echo/v5"
)

const clientCertHeader = "X-Client-Certificate"

const invalidClientDescription = "client authentication failed"

var dummyClientSecretHash = oauthdomain.HashClientSecret("idmagic-invalid-client")

type authedClient struct {
	ID                 string
	MTLSThumbprintS256 string
}

// basicCredentials は Authorization ヘッダーが Basic 方式なら、方式名の後の資格情報を返す。
// RFC 9110 §11.1 に従い、方式名は大文字と小文字を区別せずに照合する。
func basicCredentials(header string) (string, bool) {
	const prefix = "basic "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	return header[len(prefix):], true
}

func (d Deps) authenticateTokenClient(c *echo.Context) (authedClient, error) {
	basicAuth, hasBasic := basicCredentials(c.Request().Header.Get("Authorization"))
	hasSecret := c.Request().PostFormValue("client_secret") != ""
	hasAssertion := c.Request().PostFormValue("client_assertion") != "" ||
		c.Request().PostFormValue("client_assertion_type") != ""
	hasCertificate := c.Request().Header.Get(clientCertHeader) != ""
	methods := 0
	for _, present := range []bool{hasBasic, hasSecret, hasAssertion, hasCertificate} {
		if present {
			methods++
		}
	}
	if methods > 1 {
		return authedClient{}, usecases.NewOAuthError("invalid_request", "Multiple client authentication methods are mixed.")
	}

	// 1. client_assertion (private_key_jwt)
	if hasAssertion {
		const assertionType = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
		if c.Request().PostFormValue("client_assertion_type") != assertionType {
			return authedClient{}, usecases.NewOAuthError("invalid_request", "unsupported client_assertion_type")
		}
		a := c.Request().PostFormValue("client_assertion")
		if a == "" {
			return authedClient{}, usecases.NewOAuthError("invalid_request", "client_assertion is required")
		}
		clientID := c.Request().PostFormValue("client_id")
		if clientID == "" {
			// RFC 7521 §4.2 は client_id を OPTIONAL とする。未検証の sub は登録を探す先にだけ使い、
			// 下の検証が iss == sub == clientID と登録の鍵での署名を確かめる。
			subject, err := tokensJOSE.UnverifiedClientAssertionSubject(a)
			if err != nil {
				return authedClient{}, invalidClientError()
			}
			clientID = subject
		}
		tenantID := support.RequestTenantID(c)
		client, err := d.ClientRepo.FindByID(c.Request().Context(), tenantID, clientID)
		if err != nil || client == nil || client.TokenEndpointAuthMethod != oauthdomain.AuthMethodPrivateKeyJwt {
			return authedClient{}, invalidClientError()
		}
		resolver := d.JWKResolver
		if resolver == nil {
			resolver = tokensJOSE.NewJWKResolver()
		}
		_, err = tokensJOSE.VerifyClientAssertion(
			c.Request().Context(), a, clientID,
			acceptableClientAssertionAudiences(support.RequestIssuer(c, d.Issuer), c.Request()),
			func(ctx context.Context, cid string) ([]map[string]any, error) {
				cl, err := d.ClientRepo.FindByID(ctx, tenantID, cid)
				if err != nil {
					return nil, err
				}
				if cl == nil {
					return nil, invalidClientError()
				}
				return resolver.Resolve(ctx, tokensJOSE.JWKSource{JWKS: cl.JWKS, JwksURI: cl.JwksURI})
			},
			d.ClientAssertionReplayStore, time.Now().UTC(), nil,
		)
		if err != nil {
			return authedClient{}, invalidClientError()
		}
		return authedClient{ID: clientID}, nil
	}

	// 2. tls_client_auth
	if hasCertificate {
		clientID := c.Request().PostFormValue("client_id")
		client, err := d.ClientRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), clientID)
		if err != nil || client == nil ||
			client.TokenEndpointAuthMethod != oauthdomain.AuthMethodTlsClientAuth ||
			client.TlsClientAuthSubjectDN == nil {
			return authedClient{}, invalidClientError()
		}
		cert, err := certificatesMTLS.ParseClientCertificateHeader(c.Request().Header.Get(clientCertHeader))
		if err != nil || !certificatesMTLS.ClientCertSubjectMatches(*client.TlsClientAuthSubjectDN, cert.SubjectDN) {
			return authedClient{}, invalidClientError()
		}
		return authedClient{ID: clientID, MTLSThumbprintS256: cert.ThumbprintS256}, nil
	}

	// 3. client_secret_basic / client_secret_post
	var clientID, secret string
	method := oauthdomain.AuthMethodNone
	switch {
	case hasBasic:
		raw, err := base64.StdEncoding.DecodeString(basicAuth)
		if err != nil {
			return authedClient{}, usecases.NewOAuthError("invalid_client", "failed to decode Basic auth")
		}
		parts := strings.SplitN(string(raw), ":", 2)
		if len(parts) != 2 {
			return authedClient{}, usecases.NewOAuthError("invalid_client", "The Basic authentication format is invalid.")
		}
		clientID, err = url.QueryUnescape(parts[0])
		if err != nil {
			return authedClient{}, usecases.NewOAuthError("invalid_client", "The client_id format is invalid.")
		}
		secret, err = url.QueryUnescape(parts[1])
		if err != nil {
			return authedClient{}, usecases.NewOAuthError("invalid_client", "invalid client_secret format")
		}
		method = oauthdomain.AuthMethodClientSecretBasic
	case hasSecret:
		clientID = c.Request().PostFormValue("client_id")
		secret = c.Request().PostFormValue("client_secret")
		method = oauthdomain.AuthMethodClientSecretPost
	default:
		clientID = c.Request().PostFormValue("client_id")
	}
	client, err := d.ClientRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), clientID)
	if err != nil || client == nil {
		if method != oauthdomain.AuthMethodNone {
			oauthdomain.VerifyClientSecret(secret, dummyClientSecretHash)
		}
		return authedClient{}, invalidClientError()
	}
	if client.TokenEndpointAuthMethod != method {
		if method != oauthdomain.AuthMethodNone {
			hash := dummyClientSecretHash
			if client.ClientSecretHash != nil {
				hash = *client.ClientSecretHash
			}
			oauthdomain.VerifyClientSecret(secret, hash)
		}
		return authedClient{}, invalidClientError()
	}
	if method == oauthdomain.AuthMethodNone {
		return authedClient{ID: clientID}, nil
	}
	credentials, err := d.ClientRepo.ListClientSecretCredentials(c.Request().Context(), client.ClientID)
	if err != nil {
		return authedClient{}, invalidClientError()
	}
	// credential table 未移行の既存 client は legacy hash を読み続ける。全 credential を
	// 照合してから結果を判定し、active credential の数による早期 return を避ける。
	matched := false
	now := time.Now().UTC()
	for _, credential := range credentials {
		verified := oauthdomain.VerifyClientSecret(secret, credential.SecretHash)
		matched = matched || (credential.IsActiveAt(now) && verified)
	}
	if len(credentials) == 0 && client.ClientSecretHash != nil {
		matched = oauthdomain.VerifyClientSecret(secret, *client.ClientSecretHash)
	}
	if !matched {
		return authedClient{}, invalidClientError()
	}
	return authedClient{ID: clientID}, nil
}

func invalidClientError() error {
	return usecases.NewOAuthError("invalid_client", invalidClientDescription)
}

func acceptableClientAssertionAudiences(issuer string, req *http.Request) []string {
	base := strings.TrimSuffix(issuer, "/")
	values := []string{base, base + "/token", base + "/par", base + "/introspect", base + "/revoke"}
	if req != nil {
		values = append(values, base+req.URL.Path)
	}
	return values
}
