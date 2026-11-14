package server_http_test

import (
	"net/http"
	"net/url"
	"testing"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

//spec:covers REQ-OAUTH2-005, REQ-OAUTH2-035, REQ-OAUTH2-014: 公開登録した confidential の非 PKCE フローと、管理 API による必須化をトークン取得まで確認する。
func TestRegisteredConfidentialClientUsesPKCEPolicy(t *testing.T) {
	f := newPublicSetupPath(t)
	f.user(t, "pkce-policy-user")
	clientID, key := f.keyClient(t, "openid profile", "authorization_code")
	withoutPKCE := map[string]string{"code_challenge": "", "code_challenge_method": "", "nonce": "pkce-policy-nonce", "prompt": "login consent"}
	code, _ := f.login(t, "pkce-policy-user", clientID, "openid profile", withoutPKCE)
	tokens := f.signedForm(t, key, clientID, "/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {setupRedirect},
	}, http.StatusOK)
	setupString(t, tokens, "access_token")
	setupString(t, tokens, "id_token")
	metadata := f.request(t, http.MethodGet, "/.well-known/openid-configuration", nil, nil, http.StatusOK)
	if metadata["require_pkce"] != false {
		t.Fatalf("discovery require_pkce=%v, want false", metadata["require_pkce"])
	}
	client := f.request(t, http.MethodGet, "/api/admin/v1/clients/"+clientID, nil, f.admin, http.StatusOK)
	if client["require_pkce"] != false {
		t.Fatalf("registered require_pkce=%v, want false", client["require_pkce"])
	}
	updated := f.request(t, http.MethodPatch, "/api/admin/v1/clients/"+clientID, map[string]any{"require_pkce": true}, f.admin, http.StatusOK)
	if updated["require_pkce"] != true {
		t.Fatalf("updated require_pkce=%v, want true", updated["require_pkce"])
	}
	unchanged := f.request(t, http.MethodPatch, "/api/admin/v1/clients/"+clientID, map[string]any{"client_name": "PKCE policy"}, f.admin, http.StatusOK)
	if unchanged["require_pkce"] != true {
		t.Fatalf("omitted update cleared PKCE requirement: %v", unchanged)
	}
	query := stack.AuthorizationQuery(setupVerifier, map[string]string{
		"client_id": clientID, "redirect_uri": setupRedirect, "code_challenge": "", "code_challenge_method": "",
	})
	refused := f.s.Browser(t, "default").Authorize(t, query)
	defer refused.Body.Close()
	if refused.StatusCode != http.StatusBadRequest {
		t.Fatalf("authorization without required PKCE status=%d, want 400", refused.StatusCode)
	}
	code, _ = f.login(t, "pkce-policy-user", clientID, "openid profile", map[string]string{"prompt": "login consent"})
	tokens = f.signedForm(t, key, clientID, "/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {setupRedirect}, "code_verifier": {setupVerifier},
	}, http.StatusOK)
	setupString(t, tokens, "access_token")
	f.request(t, http.MethodPatch, "/api/admin/v1/clients/"+clientID, map[string]any{"require_pkce": false}, f.admin, http.StatusOK)
	code, _ = f.login(t, "pkce-policy-user", clientID, "openid profile", withoutPKCE)
	tokens = f.signedForm(t, key, clientID, "/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {setupRedirect},
	}, http.StatusOK)
	setupString(t, tokens, "access_token")
}

//spec:covers REQ-OAUTH2-016, REQ-OAUTH2-035, REQ-OAUTH2-005: 登録と管理 API の設定を保存し、public の明示 false でも PKCE の省略を拒否する。
func TestClientPKCESettingsAreReturnedByRegistration(t *testing.T) {
	f := newPublicSetupPath(t)
	for _, tt := range []struct {
		name, path, clientType string
		required               *bool
	}{
		{name: "dynamic default", path: "/register", clientType: "confidential"},
		{name: "dynamic required", path: "/register", clientType: "confidential", required: new(true)},
		{name: "admin required", path: "/api/admin/v1/clients", clientType: "confidential", required: new(true)},
		{name: "public false", path: "/register", clientType: "public", required: new(false)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]any{"client_type": tt.clientType, "redirect_uris": []string{setupRedirect}, "grant_types": []string{"authorization_code"}, "scope": "openid profile"}
			if tt.required != nil {
				body["require_pkce"] = *tt.required
			}
			response := f.request(t, http.MethodPost, tt.path, body, f.admin, http.StatusCreated)
			client := response
			if tt.path != "/register" {
				client = response["client"].(map[string]any)
			}
			wantRequired := tt.required != nil && *tt.required
			if client["require_pkce"] != wantRequired {
				t.Fatalf("registered require_pkce=%v, want %v", client["require_pkce"], wantRequired)
			}
			clientID := setupString(t, client, "client_id")
			stored := f.request(t, http.MethodGet, "/api/admin/v1/clients/"+clientID, nil, f.admin, http.StatusOK)
			if stored["require_pkce"] != wantRequired {
				t.Fatalf("stored require_pkce=%v, want %v", stored["require_pkce"], wantRequired)
			}
			if wantRequired || tt.clientType == "public" {
				query := stack.AuthorizationQuery(setupVerifier, map[string]string{"client_id": clientID, "redirect_uri": setupRedirect, "code_challenge": "", "code_challenge_method": ""})
				response := f.s.Browser(t, "default").Authorize(t, query)
				defer response.Body.Close()
				if response.StatusCode != http.StatusBadRequest {
					t.Fatalf("authorization without required PKCE status=%d", response.StatusCode)
				}
			}
		})
	}
}
