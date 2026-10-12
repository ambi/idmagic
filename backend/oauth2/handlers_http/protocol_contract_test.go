package handlers_http_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"

	"github.com/labstack/echo/v5"
)

func oauthContractForm(t *testing.T, e *echo.Echo, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/realms/default"+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func oauthContractError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	body := oauthContractJSON(t, rec, status)
	if body["error"] != code {
		t.Fatalf("OAuth error=%v", body)
	}
	if description, ok := body["error_description"].(string); !ok || description == "" {
		t.Fatalf("error_description=%v", body)
	}
}

//verify:operation Token: token の JSON 応答、内省の active と claims、失効後の inactive
//verify:operation Introspect: token の JSON 応答、内省の active と claims、失効後の inactive
//verify:operation Revoke: token の JSON 応答、内省の active と claims、失効後の inactive
func TestTokenIntrospectionAndRevocationHTTPContracts(t *testing.T) {
	f := newTokenServer(t)
	send := func(path string, values url.Values) *httptest.ResponseRecorder {
		values.Set("client_id", "client-conf")
		values.Set("client_secret", "secret-conf")
		return oauthContractForm(t, f.e, path, values)
	}
	issued := oauthContractJSON(t, send("/token", url.Values{"grant_type": {"client_credentials"}, "scope": {"openid"}}), http.StatusOK)
	token, ok := issued["access_token"].(string)
	if !ok || token == "" || issued["token_type"] != "Bearer" || issued["scope"] != "openid" {
		t.Fatalf("issued=%v", issued)
	}
	if expiry, ok := issued["expires_in"].(float64); !ok || expiry <= 0 {
		t.Fatalf("expires_in=%v", issued)
	}
	if _, present := issued["refresh_token"]; present {
		t.Fatal("client_credentials returned a refresh token")
	}
	active := oauthContractJSON(t, send("/introspect", url.Values{"token": {token}}), http.StatusOK)
	if active["active"] != true || active["client_id"] != "client-conf" || active["sub"] != "client-conf" || active["scope"] != "openid" {
		t.Fatalf("introspection=%v", active)
	}
	for _, key := range []string{"exp", "iat"} {
		if value, ok := active[key].(float64); !ok || value <= 0 {
			t.Fatalf("%s=%v", key, active)
		}
	}
	revoked := send("/revoke", url.Values{"token": {token}, "token_type_hint": {"access_token"}})
	if revoked.Code != http.StatusOK || revoked.Body.Len() != 0 {
		t.Fatalf("revocation status=%d body=%s", revoked.Code, revoked.Body.String())
	}
	inactive := oauthContractJSON(t, send("/introspect", url.Values{"token": {token}}), http.StatusOK)
	if len(inactive) != 1 || inactive["active"] != false {
		t.Fatalf("revoked introspection=%v", inactive)
	}
}

//verify:operation Token: 不正な client secret への 401 invalid_client の JSON 応答
//verify:operation Introspect: 不正な client secret への 401 invalid_client の JSON 応答
//verify:operation Revoke: 不正な client secret への 401 invalid_client の JSON 応答
func TestTokenEndpointsHTTPContractsRefuseClientAuthentication(t *testing.T) {
	f := newTokenServer(t)
	for _, path := range []string{"/token", "/introspect", "/revoke"} {
		oauthContractError(t, oauthContractForm(t, f.e, path, url.Values{
			"client_id": {"client-conf"}, "client_secret": {"wrong-secret"}, "grant_type": {"client_credentials"}, "token": {"token"},
		}), http.StatusUnauthorized, "invalid_client")
	}
}

//verify:operation UserInfo: GET・POST の 200 JSON 応答の sub と preferred_username
//verify:operation PostUserInfo: GET・POST の 200 JSON 応答の sub と preferred_username
func TestUserInfoHTTPContracts(t *testing.T) {
	intro := &fakeIntrospector{result: &oauthports.IntrospectionResult{Active: true, Sub: "user_alice", Scope: "openid profile", ClientID: "contract-client"}}
	e := newUserInfoServer(t, intro, nil)
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req := httptest.NewRequest(method, "/realms/default/userinfo", http.NoBody)
		req.Header.Set("Authorization", "Bearer contract-token")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		body := oauthContractJSON(t, rec, http.StatusOK)
		if body["sub"] != "user_alice" || body["preferred_username"] != "alice" {
			t.Fatalf("%s userinfo=%v", method, body)
		}
	}
}

//verify:operation PushAuthorizationRequest: 201 の request_uri と expires_in、および不正な client secret の 401
func TestPARHTTPContracts(t *testing.T) {
	e := newPARTestServer(t)
	form := url.Values{
		"client_id": {parClientID}, "redirect_uri": {parRedirectURI}, "response_type": {"code"}, "scope": {"openid"},
		"code_challenge": {"abcdef0123456789abcdef0123456789abcdef0123ab"}, "code_challenge_method": {"S256"},
	}
	for _, secret := range []string{parClientSecret, "wrong-secret"} {
		req := httptest.NewRequest(http.MethodPost, "/realms/default/par", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(parClientID, secret)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if secret != parClientSecret {
			oauthContractError(t, rec, http.StatusUnauthorized, "invalid_client")
			continue
		}
		body := oauthContractJSON(t, rec, http.StatusCreated)
		requestURI, ok := body["request_uri"].(string)
		if !ok || !strings.HasPrefix(requestURI, "urn:ietf:params:oauth:request_uri:") {
			t.Fatalf("request_uri=%v", body)
		}
		if expiry, ok := body["expires_in"].(float64); !ok || expiry <= 0 {
			t.Fatalf("expires_in=%v", body)
		}
	}
}
