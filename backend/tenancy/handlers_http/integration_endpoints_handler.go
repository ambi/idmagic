package handlers_http

import (
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type AdminOAuthEndpointSet struct {
	OpenIDConfiguration                string `json:"openid_configuration"`
	OAuthAuthorizationServer           string `json:"oauth_authorization_server"`
	AuthorizationEndpoint              string `json:"authorization_endpoint"`
	TokenEndpoint                      string `json:"token_endpoint"`
	UserInfoEndpoint                   string `json:"userinfo_endpoint"`
	JWKSURI                            string `json:"jwks_uri"`
	RevocationEndpoint                 string `json:"revocation_endpoint"`
	IntrospectionEndpoint              string `json:"introspection_endpoint"`
	EndSessionEndpoint                 string `json:"end_session_endpoint"`
	RegistrationEndpoint               string `json:"registration_endpoint"`
	PushedAuthorizationRequestEndpoint string `json:"pushed_authorization_request_endpoint"`
	DeviceAuthorizationEndpoint        string `json:"device_authorization_endpoint"`
}

type AdminFederationCertificate struct {
	DownloadURL       string    `json:"download_url"`
	FingerprintSHA256 string    `json:"fingerprint_sha256"`
	NotBefore         time.Time `json:"not_before"`
	NotAfter          time.Time `json:"not_after"`
}

type AdminSAMLIdentityProviderEndpointSet struct {
	EntityID           string                     `json:"entity_id"`
	MetadataURL        string                     `json:"metadata_url"`
	SSOURL             string                     `json:"sso_url"`
	SLOURL             string                     `json:"slo_url"`
	SigningCertificate AdminFederationCertificate `json:"signing_certificate"`
}

type AdminWSFederationEndpointSet struct {
	Realm               string `json:"realm"`
	MetadataURL         string `json:"metadata_url"`
	PassiveLogonURL     string `json:"passive_logon_url"`
	ActiveLogonURL      string `json:"active_logon_url"`
	MetadataExchangeURL string `json:"metadata_exchange_url"`
}

type AdminAPIEndpointSet struct {
	ManagementAPIBaseURL string `json:"management_api_base_url"`
	SCIMBaseURL          string `json:"scim_base_url"`
	AccountAPIBaseURL    string `json:"account_api_base_url"`
}

type AdminIntegrationEndpointCatalog struct {
	Issuer       string                               `json:"issuer"`
	OAuth        AdminOAuthEndpointSet                `json:"oauth"`
	SAML         AdminSAMLIdentityProviderEndpointSet `json:"saml"`
	WSFederation AdminWSFederationEndpointSet         `json:"ws_federation"`
	APIs         AdminAPIEndpointSet                  `json:"apis"`
}

func (d Deps) handleGetAdminIntegrationEndpoints(c *echo.Context) error {
	if _, err := d.requireTenantAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.FederationSigner == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "federation_credentials_unavailable", "Federation signing credentials are unavailable.")
	}
	signer, err := d.FederationSigner.Resolve(c.Request().Context())
	if err != nil || signer == nil || signer.Certificate() == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "federation_credentials_unavailable", "Federation signing credentials are unavailable.")
	}
	issuer := strings.TrimRight(support.RequestIssuer(c, d.Issuer), "/")
	return support.NoStoreJSON(c, http.StatusOK, buildAdminIntegrationEndpointCatalog(issuer, signer.Certificate()))
}

func buildAdminIntegrationEndpointCatalog(issuer string, cert *x509.Certificate) AdminIntegrationEndpointCatalog {
	url := func(path string) string { return issuer + path }
	return AdminIntegrationEndpointCatalog{
		Issuer: issuer,
		OAuth: AdminOAuthEndpointSet{
			OpenIDConfiguration:                url("/.well-known/openid-configuration"),
			OAuthAuthorizationServer:           url("/.well-known/oauth-authorization-server"),
			AuthorizationEndpoint:              url("/authorize"),
			TokenEndpoint:                      url("/token"),
			UserInfoEndpoint:                   url("/userinfo"),
			JWKSURI:                            url("/jwks"),
			RevocationEndpoint:                 url("/revoke"),
			IntrospectionEndpoint:              url("/introspect"),
			EndSessionEndpoint:                 url("/end_session"),
			RegistrationEndpoint:               url("/register"),
			PushedAuthorizationRequestEndpoint: url("/par"),
			DeviceAuthorizationEndpoint:        url("/device_authorization"),
		},
		SAML: AdminSAMLIdentityProviderEndpointSet{
			EntityID:    issuer,
			MetadataURL: url("/saml/metadata"),
			SSOURL:      url("/saml/sso"),
			SLOURL:      url("/saml/slo"),
			SigningCertificate: AdminFederationCertificate{
				DownloadURL:       url("/saml/signing-certificate.pem"),
				FingerprintSHA256: certificateFingerprintSHA256(cert),
				NotBefore:         cert.NotBefore.UTC(),
				NotAfter:          cert.NotAfter.UTC(),
			},
		},
		WSFederation: AdminWSFederationEndpointSet{
			Realm:               issuer,
			MetadataURL:         url("/federationmetadata/2007-06/federationmetadata.xml"),
			PassiveLogonURL:     url("/wsfed"),
			ActiveLogonURL:      url("/trust/usernamemixed"),
			MetadataExchangeURL: url("/trust/mex"),
		},
		APIs: AdminAPIEndpointSet{
			ManagementAPIBaseURL: url("/api/admin/v1"),
			SCIMBaseURL:          url("/scim/v2"),
			AccountAPIBaseURL:    url("/api/account/v1"),
		},
	}
}

func certificateFingerprintSHA256(cert *x509.Certificate) string {
	digest := sha256.Sum256(cert.Raw)
	hexDigest := fmt.Sprintf("%X", digest[:])
	parts := make([]string, 0, len(hexDigest)/2)
	for i := 0; i < len(hexDigest); i += 2 {
		parts = append(parts, hexDigest[i:i+2])
	}
	return strings.Join(parts, ":")
}
