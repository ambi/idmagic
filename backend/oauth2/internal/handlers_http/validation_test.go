package handlers_http

import (
	"maps"
	"net/url"
	"testing"
)

func TestParseAuthorizeRequestCoercesMaxAge(t *testing.T) {
	input, err := parseAuthorizeRequest(url.Values{
		"client_id":             {"client"},
		"redirect_uri":          {"https://example.com/callback"},
		"response_type":         {"code"},
		"code_challenge":        {"challenge"},
		"code_challenge_method": {"S256"},
		"max_age":               {"60"},
	})
	if err != nil {
		t.Fatalf("parse authorize request: %v", err)
	}
	if input.MaxAge == nil || *input.MaxAge != 60 {
		t.Fatalf("max_age = %v, want 60", input.MaxAge)
	}
}

//spec:covers REQ-OAUTH2-005: PKCE の省略は許し、明示した空値は有効な PKCE として扱わない。
func TestParseAuthorizeRequestDistinguishesMissingAndEmptyPKCE(t *testing.T) {
	for _, tt := range []struct {
		name      string
		pkce      url.Values
		wantError bool
	}{
		{name: "omitted"},
		{name: "empty challenge", pkce: url.Values{"code_challenge": {""}}, wantError: true},
		{name: "empty method", pkce: url.Values{"code_challenge_method": {""}}, wantError: true},
		{name: "both empty", pkce: url.Values{"code_challenge": {""}, "code_challenge_method": {""}}, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			query := url.Values{"client_id": {"client"}, "response_type": {"code"}}
			maps.Copy(query, tt.pkce)
			_, err := parseAuthorizeRequest(query)
			if (err != nil) != tt.wantError {
				t.Fatalf("parse err=%v, wantError=%v", err, tt.wantError)
			}
		})
	}
}

func TestParseAuthorizeRequestRejectsNegativeMaxAge(t *testing.T) {
	_, err := parseAuthorizeRequest(url.Values{
		"client_id":             {"client"},
		"redirect_uri":          {"https://example.com/callback"},
		"response_type":         {"code"},
		"code_challenge":        {"challenge"},
		"code_challenge_method": {"S256"},
		"max_age":               {"-1"},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

// Contract: AuthorizeParameters は security-sensitive な単一値parameterを重複させない。
func TestParseAuthorizeRequestRejectsDuplicateSecurityParameter(t *testing.T) {
	_, err := parseAuthorizeRequest(url.Values{
		"client_id": {"client", "other"}, "redirect_uri": {"https://example.com/callback"},
		"response_type": {"code"}, "code_challenge": {"challenge"}, "code_challenge_method": {"S256"},
	})
	if err == nil {
		t.Fatal("expected duplicate client_id to be rejected")
	}
}

func TestValidateRegisterClientRequestRejectsUnknownGrant(t *testing.T) {
	request := &registerClientRequest{
		ClientType: "confidential", RedirectURIs: []string{"https://example.com/callback"},
		GrantTypes: []string{"password"}, TokenEndpointAuthMethod: "client_secret_basic", FapiProfile: "none",
	}
	if err := validateRegisterClientRequest(request); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateRegisterClientRequestSupportedGrants(t *testing.T) {
	for _, grant := range []string{
		"authorization_code", "refresh_token", "client_credentials",
		"urn:ietf:params:oauth:grant-type:device_code",
		"urn:ietf:params:oauth:grant-type:token-exchange",
		"urn:openid:params:grant-type:ciba",
		"urn:ietf:params:oauth:grant-type:jwt-bearer",
		"urn:ietf:params:oauth:grant-type:jwt-dpop",
	} {
		t.Run(grant, func(t *testing.T) {
			request := &registerClientRequest{
				ClientType: "confidential", RedirectURIs: []string{"https://example.com/callback"},
				GrantTypes: []string{grant}, TokenEndpointAuthMethod: "client_secret_basic", FapiProfile: "none",
			}
			if err := validateRegisterClientRequest(request); err != nil {
				t.Fatalf("supported grant %q: %v", grant, err)
			}
		})
	}
}

func TestValidateRegisterClientRequestRequiresHttpsJwksURI(t *testing.T) {
	request := &registerClientRequest{
		RedirectURIs: []string{"https://example.com/callback"},
		JwksURI:      new("http://example.com/jwks.json"),
	}
	if err := validateRegisterClientRequest(request); err == nil {
		t.Fatal("expected jwks_uri validation error")
	}
}
