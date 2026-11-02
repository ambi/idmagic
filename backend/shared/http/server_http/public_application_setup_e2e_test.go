package server_http_test

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"
)

//spec:covers REQ-APPLICATION-011: 公開作成した OIDC、SAML、WS-Federation の Application は割り当て後にサインオンでき、解除後は認可コードもアサーションも返さない。
func TestPublicApplicationSetupEnforcesAssignments(t *testing.T) {
	for _, protocol := range []string{"oidc", "saml", "wsfed"} {
		t.Run(protocol, func(t *testing.T) {
			f := newPublicSetupPath(t, stack.WithApplicationApi(), stack.WithSaml(), stack.WithWsFederation())
			userID := f.user(t, "assigned-user")
			input := map[string]any{"name": "Public " + protocol, "type": protocol}
			const entity = "https://public-sp.example"
			const acs = "https://public-sp.example/acs"
			const realm = "urn:public:wsfed"
			const reply = "https://public-rp.example/reply"
			switch protocol {
			case "oidc":
				input["redirect_uris"], input["scope"] = []string{setupRedirect}, "openid profile"
			case "saml":
				input["entity_id"], input["acs_urls"] = entity, []string{acs}
			case "wsfed":
				input["wtrealm"], input["reply_urls"] = realm, []string{reply}
			}
			created := f.request(t, http.MethodPost, "/api/admin/v1/applications", input, f.admin, http.StatusCreated)
			application, ok := created["application"].(map[string]any)
			if !ok {
				t.Fatalf("creation has no application: %v", created)
			}
			appID := setupString(t, application, "id")
			f.request(t, http.MethodPost, "/api/admin/v1/applications/"+appID+"/assignments",
				map[string]any{"subject_type": "user", "subject_id": userID}, f.admin, http.StatusCreated)
			unassign := func() {
				f.request(t, http.MethodDelete, "/api/admin/v1/applications/"+appID+"/assignments/user/"+userID,
					nil, f.admin, http.StatusNoContent)
			}
			if protocol == "oidc" {
				clientID := setupString(t, created, "client_id")
				browser := f.s.Browser(t, "default")
				query := stack.AuthorizationQuery(setupVerifier, map[string]string{"client_id": clientID, "redirect_uri": setupRedirect})
				_ = browser.Authorize(t, query).Body.Close()
				browser.SignIn(t, "assigned-user", setupPassword)
				redirect, err := url.Parse(browser.Consent(t, "allow"))
				if err != nil || redirect.Query().Get("code") == "" {
					t.Fatalf("OIDC redirect=%v error=%v", redirect, err)
				}
				form := url.Values{
					"grant_type": {"authorization_code"}, "code": {redirect.Query().Get("code")},
					"redirect_uri": {setupRedirect}, "code_verifier": {setupVerifier},
				}
				req := httptest.NewRequest(http.MethodPost, stack.Issuer+"/realms/default/token", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.SetBasicAuth(clientID, setupString(t, created, "client_secret"))
				rec := httptest.NewRecorder()
				f.s.Echo.ServeHTTP(rec, req)
				var issued map[string]any
				if err := json.Unmarshal(rec.Body.Bytes(), &issued); err != nil || rec.Code != http.StatusOK {
					t.Fatalf("OIDC token status=%d body=%s error=%v", rec.Code, rec.Body.String(), err)
				}
				claims := unverifiedClaims(t, setupString(t, issued, "id_token"))
				if claims["sub"] != userID || claims["aud"] != clientID {
					t.Fatalf("OIDC identity=%v, want user %s and client %s", claims, userID, clientID)
				}
				unassign()
				response := browser.Authorize(t, query)
				_ = response.Body.Close()
				refused, err := url.Parse(response.Header.Get("Location"))
				if err != nil || refused.Query().Get("error") != "access_denied" || refused.Query().Get("code") != "" {
					t.Fatalf("unassigned OIDC status=%d redirect=%v error=%v", response.StatusCode, refused, err)
				}
				return
			}
			loginClient, _ := f.keyClient(t, "openid profile", "authorization_code")
			_, cookie := f.login(t, "assigned-user", loginClient, "openid profile")
			path, field, destination := "/saml/sso?entityID="+url.QueryEscape(entity), "SAMLResponse", acs
			metadataPath := "/saml/metadata"
			if protocol == "wsfed" {
				path, field, destination = "/wsfed?wa=wsignin1.0&wtrealm="+url.QueryEscape(realm)+"&wreply="+url.QueryEscape(reply), "wresult", reply
				metadataPath = "/federationmetadata/2007-06/federationmetadata.xml"
			}
			response := publicSignOn(t, f, path, cookie)
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), destination) {
				t.Fatalf("%s sign-on status=%d body=%s", protocol, response.Code, response.Body.String())
			}
			value := publicHiddenField(t, response.Body.String(), field)
			if protocol == "saml" {
				raw, err := base64.StdEncoding.DecodeString(value)
				if err != nil {
					t.Fatal(err)
				}
				value = string(raw)
			}
			document := etree.NewDocument()
			if err := document.ReadFromString(value); err != nil {
				t.Fatal(err)
			}
			assertion := document.FindElement("//Assertion")
			if assertion == nil {
				t.Fatalf("%s did not return an assertion: %s", protocol, value)
			}
			subject := assertion.FindElement(".//NameID")
			if subject == nil {
				subject = assertion.FindElement(".//NameIdentifier")
			}
			if subject == nil || subject.Text() != userID {
				t.Fatalf("%s assertion does not identify registered user %s: %s", protocol, userID, value)
			}
			metadata := publicSignOn(t, f, metadataPath, nil)
			publicVerifyAssertion(t, assertion, metadata.Body.String())
			unassign()
			denied := publicSignOn(t, f, path, cookie)
			if denied.Code != http.StatusForbidden || strings.Contains(denied.Body.String(), `name="`+field+`"`) {
				t.Fatalf("unassigned %s status=%d, want 403 without %s", protocol, denied.Code, field)
			}
		})
	}
}

func publicSignOn(t *testing.T, f *publicSetupPath, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, stack.Issuer+"/realms/default"+path, http.NoBody)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(rec, req)
	return rec
}

func publicHiddenField(t *testing.T, body, field string) string {
	t.Helper()
	_, after, ok := strings.Cut(body, `name="`+field+`" value="`)
	if !ok {
		t.Fatalf("form does not contain %s: %s", field, body)
	}
	value, _, ok := strings.Cut(after, `"`)
	if !ok {
		t.Fatalf("unterminated %s field", field)
	}
	return html.UnescapeString(value)
}

func publicVerifyAssertion(t *testing.T, assertion *etree.Element, metadata string) {
	t.Helper()
	document := etree.NewDocument()
	if err := document.ReadFromString(metadata); err != nil {
		t.Fatal(err)
	}
	element := document.FindElement("//X509Certificate")
	if element == nil {
		t.Fatalf("metadata has no signing certificate: %s", metadata)
	}
	der, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(element.Text()), ""))
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	validation := dsig.NewDefaultValidationContext(&dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{certificate}})
	if assertion.SelectAttr("AssertionID") != nil {
		validation.IdAttribute = "AssertionID"
	}
	if _, err := validation.Validate(assertion); err != nil {
		t.Fatalf("returned assertion does not verify with the published certificate: %v", err)
	}
}
