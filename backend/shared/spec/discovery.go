package spec

import (
	"fmt"
	"slices"
)

type discoveryEndpoint struct {
	Field         string
	OperationName string
}

var discoveryEndpoints = []discoveryEndpoint{
	{"authorization_endpoint", "Authorize"},
	{"token_endpoint", "Token"},
	{"userinfo_endpoint", "UserInfo"},
	{"jwks_uri", "GetJwks"},
	{"introspection_endpoint", "Introspect"},
	{"revocation_endpoint", "Revoke"},
	{"pushed_authorization_request_endpoint", "PushAuthorizationRequest"},
	{"device_authorization_endpoint", "DeviceAuthorization"},
	{"backchannel_authentication_endpoint", "BackchannelAuthenticate"},
	{"registration_endpoint", "RegisterClient"},
	{"end_session_endpoint", "EndSession"},
}

var discoveryScopes = []string{
	"openid", "profile", "email", "phone", "address", "custom_attribute", "offline_access",
	"idmagic.admin", "idmagic.account", "account:read", "account:write", "account:mfa:write",
	"account:sessions:write", "account:consents:write", "account:password:write",
}

var discoveryClaims = []string{
	"sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "acr", "amr", "azp",
	"name", "given_name", "family_name", "preferred_username", "email", "email_verified", "updated_at",
}

// BuildDiscoveryDocument derives protocol metadata from the TypeSpec-generated
// operation catalog and runtime protocol capabilities.
func (c *RuntimeContract) BuildDiscoveryDocument(issuer string) (map[string]any, error) {
	doc := map[string]any{"issuer": issuer}
	for _, endpoint := range discoveryEndpoints {
		operation, ok := c.Operation(endpoint.OperationName)
		if !ok {
			return nil, fmt.Errorf("operation %s not found", endpoint.OperationName)
		}
		doc[endpoint.Field] = issuer + operation.Path
	}

	doc["response_types_supported"] = []string{"code"}
	doc["response_modes_supported"] = []string{"query", "form_post"}
	doc["grant_types_supported"] = []string{
		string(GrantAuthorizationCode), string(GrantRefreshToken), string(GrantClientCredentials),
		string(GrantDeviceCode), string(GrantTokenExchange), string(GrantCiba),
	}
	// CIBA Core section 4: only poll mode is implemented and user_code is unsupported.
	doc["backchannel_token_delivery_modes_supported"] = []string{"poll"}
	doc["backchannel_user_code_parameter_supported"] = false
	doc["id_token_signing_alg_values_supported"] = []string{"PS256", "ES256"}
	doc["token_endpoint_auth_signing_alg_values_supported"] = []string{"PS256", "ES256"}
	doc["code_challenge_methods_supported"] = []string{string(CodeChallengeMethodS256)}
	doc["dpop_signing_alg_values_supported"] = []string{"PS256", "ES256"}
	doc["token_endpoint_auth_methods_supported"] = []string{
		"client_secret_basic", "client_secret_post", "private_key_jwt", "tls_client_auth",
	}
	doc["scopes_supported"] = slices.Clone(discoveryScopes)
	doc["subject_types_supported"] = []string{"public"}
	doc["introspection_endpoint_auth_methods_supported"] = []string{"client_secret_basic", "private_key_jwt", "tls_client_auth"}
	doc["revocation_endpoint_auth_methods_supported"] = []string{"client_secret_basic", "private_key_jwt", "tls_client_auth", "none"}
	doc["require_pushed_authorization_requests"] = false
	doc["require_pkce"] = true
	doc["tls_client_certificate_bound_access_tokens"] = true
	doc["client_id_metadata_document_supported"] = true
	doc["authorization_response_iss_parameter_supported"] = true
	doc["claims_supported"] = slices.Clone(discoveryClaims)
	doc["acr_values_supported"] = []string{"urn:idmagic:acr:pwd", "urn:idmagic:acr:mfa"}
	doc["service_documentation"] = issuer + "/docs"
	doc["ui_locales_supported"] = []string{"en", "ja"}
	return doc, nil
}
