package handlers_http_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/beevik/etree"
	"github.com/labstack/echo/v5"
)

func samlContractJSON(t *testing.T, rec *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()
	if rec.Code != status || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("status=%d Content-Type=%q body=%s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body == nil {
		t.Fatalf("JSON object=%v err=%v", body, err)
	}
	return body
}

func samlContractProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status || rec.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("status=%d Content-Type=%q body=%s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["type"] != "urn:idmagic:error:"+code || body["status"] != float64(status) {
		t.Fatalf("Problem Details=%v err=%v", body, err)
	}
}

func samlContractTime(t *testing.T, body map[string]any, key string) {
	t.Helper()
	value, ok := body[key].(string)
	if parsed, err := time.Parse(time.RFC3339Nano, value); !ok || err != nil || parsed.IsZero() {
		t.Fatalf("%s=%v, want RFC3339 timestamp", key, body[key])
	}
}

func samlContractProfile(t *testing.T, body map[string]any, name, mode string) string {
	t.Helper()
	profile, ok := body["profile"].(map[string]any)
	if !ok || profile["tenant_id"] != tenancydomain.DefaultTenantID || profile["name"] != name || profile["mode"] != mode || profile["is_default"] != false {
		t.Fatalf("profile=%v", profile)
	}
	id, ok := profile["profile_id"].(string)
	if !ok || id == "" || id == "default" {
		t.Fatalf("profile_id=%v", profile["profile_id"])
	}
	samlContractTime(t, profile, "created_at")
	samlContractTime(t, profile, "updated_at")
	base := "https://idp.example/realms/default/saml/idp/" + id
	for key, want := range map[string]string{
		"entity_id": base, "metadata_url": base + "/metadata", "sso_url": base + "/sso",
		"slo_url": base + "/slo", "signing_certificate_url": base + "/signing-certificate.pem",
	} {
		if body[key] != want {
			t.Fatalf("%s=%v, want %q", key, body[key], want)
		}
	}
	if fingerprint, ok := body["signing_certificate_fingerprint_sha256"].(string); !ok || len(strings.Split(fingerprint, ":")) != 32 {
		t.Fatalf("certificate fingerprint=%v", body["signing_certificate_fingerprint_sha256"])
	}
	if count, ok := body["service_provider_count"].(float64); !ok || count < 0 {
		t.Fatalf("service_provider_count=%v", body["service_provider_count"])
	}
	return id
}

//verify:operation CreateSamlIdentityProviderProfile: 作成、更新、削除の状態コードと JSON 応答、保存結果
//verify:operation UpdateSamlIdentityProviderProfile: 作成、更新、削除の状態コードと JSON 応答、保存結果
//verify:operation DeleteSamlIdentityProviderProfile: 作成、更新、削除の状態コードと JSON 応答、保存結果
func TestSamlProfileManagementHTTPContracts(t *testing.T) {
	e := newAdminServer(t)
	const path = "/api/admin/v1/saml/idp-profiles"
	created := samlContractJSON(t, doAdminJSON(e, http.MethodPost, path, `{"name":"Partner","mode":"shared"}`), http.StatusCreated)
	id := samlContractProfile(t, created, "Partner", "shared")
	if created["service_provider_count"] != float64(0) {
		t.Fatalf("new profile=%v", created)
	}
	updated := samlContractJSON(t, doAdminJSON(e, http.MethodPut, path+"/"+id, `{"name":"Dedicated partner","mode":"dedicated"}`), http.StatusOK)
	if got := samlContractProfile(t, updated, "Dedicated partner", "dedicated"); got != id || updated["profile"].(map[string]any)["created_at"] != created["profile"].(map[string]any)["created_at"] {
		t.Fatalf("updated profile=%v", updated)
	}
	listed := samlContractJSON(t, get(e, path), http.StatusOK)
	profiles, ok := listed["profiles"].([]any)
	if !ok || len(profiles) != 2 {
		t.Fatalf("persisted profiles=%v", profiles)
	}
	if got := samlContractProfile(t, profiles[1].(map[string]any), "Dedicated partner", "dedicated"); got != id {
		t.Fatalf("persisted profile_id=%q, want %q", got, id)
	}
	deleted := doAdminJSON(e, http.MethodDelete, path+"/"+id, "")
	if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	remaining := samlContractJSON(t, get(e, path), http.StatusOK)["profiles"].([]any)
	if len(remaining) != 1 || remaining[0].(map[string]any)["profile"].(map[string]any)["profile_id"] != "default" {
		t.Fatalf("profiles after delete=%v", remaining)
	}
	samlContractProblem(t, doAdminJSON(e, http.MethodDelete, path+"/default", ""), http.StatusConflict, "profile_in_use")
	samlContractProblem(t, doAdminJSON(e, http.MethodPut, path+"/missing", `{"name":"Missing","mode":"shared"}`), http.StatusNotFound, "not_found")
	samlContractProblem(t, doAdminJSON(e, http.MethodPost, path, `{"name":"Bad","mode":"invalid"}`), http.StatusBadRequest, "invalid_request")
}

//verify:operation RegisterSamlServiceProvider: 201 の JSON 応答と登録した Service Provider
func TestSamlServiceProviderRegistrationHTTPContract(t *testing.T) {
	e := newAdminServer(t)
	const path = "/api/admin/v1/saml/service-providers"
	const body = `{"entity_id":"https://sp.example.com","display_name":"Partner","acs_urls":["https://sp.example.com/acs"],"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`
	created := samlContractJSON(t, doAdminJSON(e, http.MethodPost, path, body), http.StatusCreated)
	for key, want := range map[string]any{
		"tenant_id": tenancydomain.DefaultTenantID, "idp_profile_id": "default", "entity_id": "https://sp.example.com",
		"display_name": "Partner", "sign_assertion": true, "sign_response": false,
	} {
		if created[key] != want {
			t.Fatalf("%s=%v, want %v", key, created[key], want)
		}
	}
	if value, present := created["want_authn_requests_signed"]; present && value != false {
		t.Fatalf("want_authn_requests_signed=%v", value)
	}
	if !reflect.DeepEqual(created["acs_urls"], []any{"https://sp.example.com/acs"}) {
		t.Fatalf("acs_urls=%v", created["acs_urls"])
	}
	policy, ok := created["claim_policy"].(map[string]any)
	if !ok || !reflect.DeepEqual(policy["name_id"], map[string]any{"format": samldomain.SamlNameIDFormatPersistent, "source_attribute": "user_id"}) {
		t.Fatalf("claim policy=%v", policy)
	}
	samlContractTime(t, created, "created_at")
	samlContractTime(t, created, "updated_at")
	updated := samlContractJSON(t, doAdminJSON(e, http.MethodPost, path, strings.Replace(body, `"Partner"`, `"Renamed"`, 1)), http.StatusOK)
	if updated["display_name"] != "Renamed" || updated["created_at"] != created["created_at"] || updated["idp_profile_id"] != created["idp_profile_id"] {
		t.Fatalf("updated SP=%v", updated)
	}
	samlContractTime(t, updated, "updated_at")
	listed := samlContractJSON(t, get(e, path), http.StatusOK)["service_providers"]
	if !reflect.DeepEqual(listed, []any{updated}) {
		t.Fatalf("persisted SPs=%v", listed)
	}
	samlContractProblem(t, doAdminJSON(e, http.MethodPost, path, `{"entity_id":"https://bad.example"}`), http.StatusBadRequest, "invalid_request")
}

//verify:operation CreateSamlIdentityProviderProfile: 非管理者への 403 Problem Details
//verify:operation UpdateSamlIdentityProviderProfile: 非管理者への 403 Problem Details
//verify:operation DeleteSamlIdentityProviderProfile: 非管理者への 403 Problem Details
//verify:operation RegisterSamlServiceProvider: 非管理者への 403 Problem Details
func TestSamlManagementHTTPContractsRefuseNonAdmin(t *testing.T) {
	e, _, repo := newServerWithRepository(t, &authdomain.AuthenticationContext{UserID: "user-1"})
	for _, test := range []struct{ method, path string }{
		{http.MethodPost, "/api/admin/v1/saml/idp-profiles"},
		{http.MethodPut, "/api/admin/v1/saml/idp-profiles/default"},
		{http.MethodDelete, "/api/admin/v1/saml/idp-profiles/default"},
		{http.MethodPost, "/api/admin/v1/saml/service-providers"},
	} {
		t.Run(test.method+test.path, func(t *testing.T) {
			samlContractProblem(t, doAdminJSON(e, test.method, test.path, `{}`), http.StatusForbidden, "access_denied")
		})
	}
	sp, err := repo.FindByEntityID(context.Background(), tenancydomain.DefaultTenantID, "https://sp.example.com")
	if err != nil || sp == nil || sp.DisplayName != "" {
		t.Fatalf("SP after refusal=%v err=%v", sp, err)
	}
}

func samlContractRequest(e *echo.Echo, method, path string, values url.Values) *httptest.ResponseRecorder {
	if method == http.MethodGet {
		return get(e, path+"?"+values.Encode())
	}
	req := httptest.NewRequest(method, defaultRealmPath(path), strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func samlContractEncoded(t *testing.T, method, xml string) string {
	t.Helper()
	if method == http.MethodPost {
		return base64.StdEncoding.EncodeToString([]byte(xml))
	}
	encoded, err := samldomain.EncodeRedirect([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

//verify:operation SamlSingleSignOnDefault: 各 binding と realm 経路の SAML 応答、署名、セッションおよび拒否
//verify:operation PostSamlSingleSignOnDefault: 各 binding と realm 経路の SAML 応答、署名、セッションおよび拒否
//verify:operation SamlSingleSignOn: 各 binding と realm 経路の SAML 応答、署名、セッションおよび拒否
//verify:operation PostSamlSingleSignOn: 各 binding と realm 経路の SAML 応答、署名、セッションおよび拒否
func TestSamlSSOBindingHTTPContracts(t *testing.T) {
	for _, profile := range []string{"default", refusalProfileA} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			t.Run(profile+method, func(t *testing.T) {
				e, events := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix()})
				path := "/saml/sso"
				entityID := "https://idp.example/realms/default"
				certificatePath := "/saml/signing-certificate.pem"
				if profile != "default" {
					e, events = newProfileBoundServer(t, true)
					path = "/saml/idp/" + profile + "/sso"
					entityID += "/saml/idp/" + profile
					certificatePath = "/saml/idp/" + profile + "/signing-certificate.pem"
				}
				xml := buildAuthnRequestXML(authnRequestOptions{destination: "https://idp.example/realms/default" + path, acsURL: refusalSPACSURL})
				requestDocument := etree.NewDocument()
				if err := requestDocument.ReadFromString(xml); err != nil {
					t.Fatal(err)
				}
				requestID := requestDocument.Root().SelectAttrValue("ID", "")
				values := url.Values{"SAMLRequest": {samlContractEncoded(t, method, xml)}, "RelayState": {"contract-relay"}}
				rec := samlContractRequest(e, method, path, values)
				if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") || rec.Header().Get("Cache-Control") != "no-store" {
					t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
				}
				if !strings.Contains(rec.Body.String(), `action="`+refusalSPACSURL+`"`) || !strings.Contains(rec.Body.String(), `name="RelayState" value="contract-relay"`) {
					t.Fatalf("POST form=%s", rec.Body.String())
				}
				document := etree.NewDocument()
				if err := document.ReadFromBytes(samlResponseFromPostForm(t, rec.Body.String())); err != nil {
					t.Fatal(err)
				}
				root := document.Root()
				if root.SelectAttrValue("Destination", "") != refusalSPACSURL || root.SelectAttrValue("InResponseTo", "") != requestID {
					t.Fatalf("SAMLResponse=%s", rec.Body.String())
				}
				if issuer := root.SelectElement("Issuer"); issuer == nil || issuer.Text() != entityID {
					t.Fatalf("issuer=%v, want %q", issuer, entityID)
				}
				assertion := document.FindElement("//Assertion")
				if assertion == nil {
					t.Fatal("Assertion missing")
				}
				if err := validateAgainst(assertion, profileCertificate(t, e, certificatePath)); err != nil {
					t.Fatalf("signature validation: %v", err)
				}
				if !hasEvent(*events, "SamlSignInIssued") {
					t.Fatal("SamlSignInIssued missing")
				}
				refused := samlContractRequest(e, method, path, url.Values{"SAMLRequest": {"invalid"}})
				if refused.Code != http.StatusBadRequest || !strings.HasPrefix(refused.Header().Get("Content-Type"), "text/plain") || refused.Body.Len() == 0 {
					t.Fatalf("invalid request status=%d headers=%v body=%s", refused.Code, refused.Header(), refused.Body.String())
				}
				assertNoAssertionIssued(t, refused)
			})
		}
	}
}

//verify:operation SamlSingleLogoutDefault: 各 binding の LogoutResponse の署名、返送先、セッション失効および拒否
//verify:operation PostSamlSingleLogoutDefault: 各 binding の LogoutResponse の署名、返送先、セッション失効および拒否
//verify:operation SamlSingleLogout: 各 binding の LogoutResponse の署名、返送先、セッション失効および拒否
//verify:operation PostSamlSingleLogout: 各 binding の LogoutResponse の署名、返送先、セッション失効および拒否
//spec:covers REQ-SAML-009: GET と POST の LogoutResponse はプロファイルの公開証明書で署名を検証できる。
func TestSamlSLOBindingHTTPContracts(t *testing.T) {
	for _, profile := range []string{"default", refusalProfileA} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			t.Run(profile+method, func(t *testing.T) {
				e, events, repo := newServerWithRepository(t, nil)
				path := "/saml/slo"
				certificatePath := "/saml/signing-certificate.pem"
				if profile != "default" {
					e, events, repo = newProfileBoundServerWith(t, true)
					path = "/saml/idp/" + profile + "/slo"
					certificatePath = "/saml/idp/" + profile + "/signing-certificate.pem"
				}
				sp, err := repo.FindByEntityID(context.Background(), tenancydomain.DefaultTenantID, refusalSPEntityID)
				if err != nil || sp == nil {
					t.Fatalf("SP=%v err=%v", sp, err)
				}
				sp.SLOURL = "https://sp.example.com/logout"
				if err := repo.Save(context.Background(), sp); err != nil {
					t.Fatal(err)
				}
				xml := `<samlp:LogoutRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_logout-contract" Version="2.0" Destination="https://idp.example/realms/default` + path + `"><saml:Issuer>` + refusalSPEntityID + `</saml:Issuer><saml:NameID>user-1</saml:NameID></samlp:LogoutRequest>`
				rec := samlContractRequest(e, method, path, url.Values{"SAMLRequest": {samlContractEncoded(t, method, xml)}, "RelayState": {"logout-relay"}})
				if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
					t.Fatalf("logout status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
				}
				location, err := url.Parse(rec.Header().Get("Location"))
				if err != nil || location.Scheme != "https" || location.Host != "sp.example.com" || location.Path != "/logout" || location.Query().Get("RelayState") != "logout-relay" {
					t.Fatalf("logout Location=%v err=%v", location, err)
				}
				response, err := samldomain.DecodeRedirect(location.Query().Get("SAMLResponse"))
				if err != nil {
					t.Fatal(err)
				}
				document := etree.NewDocument()
				if err := document.ReadFromBytes(response); err != nil {
					t.Fatal(err)
				}
				root := document.Root()
				if root.Tag != "LogoutResponse" || root.SelectAttrValue("InResponseTo", "") != "_logout-contract" || root.SelectAttrValue("Destination", "") != sp.SLOURL {
					t.Fatalf("LogoutResponse=%s", response)
				}
				if err := validateAgainst(root, profileCertificate(t, e, certificatePath)); err != nil {
					t.Fatalf("logout signature: %v", err)
				}
				if !hasEvent(*events, "SamlLogout") {
					t.Fatal("SamlLogout missing")
				}
				refused := samlContractRequest(e, method, path, url.Values{"SAMLRequest": {"invalid"}})
				if refused.Code != http.StatusBadRequest || !strings.HasPrefix(refused.Header().Get("Content-Type"), "text/plain") || refused.Body.Len() == 0 || refused.Header().Get("Location") != "" {
					t.Fatalf("invalid logout status=%d headers=%v body=%s", refused.Code, refused.Header(), refused.Body.String())
				}
				local := samlContractRequest(e, method, path, url.Values{"entityID": {"https://unregistered.example"}})
				if local.Code != http.StatusOK || !strings.HasPrefix(local.Header().Get("Content-Type"), "text/plain") || local.Body.String() != "signed out" || local.Header().Get("Location") != "" {
					t.Fatalf("local logout status=%d headers=%v body=%s", local.Code, local.Header(), local.Body.String())
				}
			})
		}
	}
}
