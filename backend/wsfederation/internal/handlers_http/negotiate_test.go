package handlers_http_test

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	kerberosfixture "github.com/ambi/idmagic/backend/authentication/testing_kerberos"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"
)

//spec:covers REQ-WSFEDERATION-002: Kerberos の認証事実を WIA の要求と署名済み Assertion の認証方式へ渡す。
func TestPassiveWIAIssuesKerberosAssertion(t *testing.T) {
	e, events, signer := newServerWithSigner(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"wia"}})
	response := get(e, "/wsfed?wa=wsignin1.0&wtrealm=urn:idmagic:demo-rp&wauth=urn:federation:authentication:windows")
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromString(wresultFromPassiveForm(t, response.Body.String())); err != nil {
		t.Fatal(err)
	}
	assertion := doc.FindElement("//Assertion")
	if assertion == nil {
		t.Fatal("assertion missing")
	}
	statement := assertion.FindElement("./AuthenticationStatement")
	if statement == nil || statement.SelectAttrValue("AuthenticationMethod", "") != "urn:ietf:rfc:1510" {
		t.Fatal("assertion does not report Kerberos")
	}
	validation := dsig.NewDefaultValidationContext(&dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{signer.Certificate()}})
	validation.IdAttribute = idAttributeForAssertion(assertion)
	if _, err := validation.Validate(assertion); err != nil {
		t.Fatalf("signature invalid: %v", err)
	}
	if !hasEvent(*events, "WsFedSignInIssued") {
		t.Fatal("issuance event missing")
	}
}

//spec:covers REQ-WSFEDERATION-002, REQ-AUTHENTICATION-042, REQ-AUTHENTICATION-043: 正式なルーターの WIA チャレンジから、実物チケット、Cookie と署名済みフォームまでを接続する。
//verify:operation WsFederationSignIn: WIA の 401 と WWW-Authenticate、認証後の 303 Location と Cookie、発行時の 200 のフォームを観測する。
func TestPassiveWIAAuthenticatesThroughConfiguredRouter(t *testing.T) {
	for _, name := range []string{"no-session", "password-session"} {
		t.Run(name, func(t *testing.T) { assertPassiveKerberosAuthentication(t, name == "password-session") })
	}
}

func assertPassiveKerberosAuthentication(t *testing.T, passwordSession bool) {
	t.Helper()
	f := kerberosfixture.New(t)
	f.Config.Bindings[0].UserID = "kerberos-user"
	path := f.WriteConfiguration(t, tenancydomain.DefaultTenantID, false)
	var initialCookies []*http.Cookie
	e, events, signer := newServerWithSigner(t, nil, func(deps *httpadapter.Deps) {
		deps.Issuer = "https://sso.example.com"
		user := &userdomain.User{
			ID: "kerberos-user", TenantID: tenancydomain.DefaultTenantID,
			PreferredUsername: "kerberos-user", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := deps.UserRepo.Save(context.Background(), user); err != nil {
			t.Fatal(err)
		}
		deps.Authentication = authentication.NewMemoryModule(deps.UserRepo)
		deps.Authentication.SessionManager = deps.Authentication.NewSessionManager()
		deps.Authentication.AuthnResolver = deps.Authentication.SessionManager
		if _, err := deps.Authentication.ConfigureNegotiate(path, deps.UserRepo); err != nil {
			t.Fatal(err)
		}
		if passwordSession {
			session, err := deps.Authentication.SessionManager.Create(testing_tenant.Default(context.Background()), user.ID, []string{"pwd"}, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			initialCookies = []*http.Cookie{{Name: "idmagic_session", Value: session.SessionID}}
		}
	})
	if passwordSession {
		probe := httptest.NewRequest(http.MethodGet, defaultRealmPath("/wsfed?wa=wsignin1.0&wtrealm=urn:idmagic:demo-rp"), http.NoBody)
		probe.AddCookie(initialCookies[0])
		response := httptest.NewRecorder()
		e.ServeHTTP(response, probe)
		if response.Code != 200 {
			t.Fatalf("initial password session was not recognized: %d %s", response.Code, response.Body.String())
		}
	}
	target := defaultRealmPath("/wsfed?wa=wsignin1.0&wtrealm=urn:idmagic:demo-rp&wauth=urn:federation:authentication:windows")
	request := func(cookies []*http.Cookie, token []byte) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, target, http.NoBody)
		for _, cookie := range cookies {
			r.AddCookie(cookie)
		}
		if token != nil {
			r.Header.Set("Authorization", "Negotiate "+base64.StdEncoding.EncodeToString(token))
		}
		response := httptest.NewRecorder()
		e.ServeHTTP(response, r)
		return response
	}
	challenge := request(initialCookies, nil)
	if challenge.Code != 401 || challenge.Header().Get("WWW-Authenticate") != "Negotiate" {
		t.Fatalf("challenge=%d %s", challenge.Code, challenge.Body.String())
	}
	token := f.Token(t, "alice", f.Config.Realm, time.Now().Add(time.Hour), f.Certificate)
	completed := request(challenge.Result().Cookies(), token)
	if completed.Code != 303 || completed.Header().Get("Location") != target {
		t.Fatalf("completed=%d %s location=%s", completed.Code, completed.Body.String(), completed.Header().Get("Location"))
	}
	issued := request(completed.Result().Cookies(), nil)
	if issued.Code != 200 {
		t.Fatalf("issued=%d %s", issued.Code, issued.Body.String())
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromString(wresultFromPassiveForm(t, issued.Body.String())); err != nil {
		t.Fatal(err)
	}
	assertion := doc.FindElement("//Assertion")
	if assertion == nil || assertion.FindElement("./AuthenticationStatement").SelectAttrValue("AuthenticationMethod", "") != "urn:ietf:rfc:1510" {
		t.Fatal("the issued assertion did not report Kerberos")
	}
	validation := dsig.NewDefaultValidationContext(&dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{signer.Certificate()}})
	validation.IdAttribute = idAttributeForAssertion(assertion)
	if _, err := validation.Validate(assertion); err != nil {
		t.Fatalf("invalid assertion signature: %v", err)
	}
	if !hasEvent(*events, "NegotiateAuthSucceeded") || !hasEvent(*events, "WsFedSignInIssued") {
		t.Fatalf("events=%v", *events)
	}
}

//spec:covers REQ-AUTHENTICATION-043, REQ-WSFEDERATION-002: 手動の Kerberos だけを設定したテナントでも、通常の WS-Fed 要求はフォームへ進める。
func TestConfiguredKerberosPreservesNormalPassiveLogin(t *testing.T) {
	f := kerberosfixture.New(t)
	path := f.WriteConfiguration(t, tenancydomain.DefaultTenantID, false)
	e, _, _ := newServerWithSigner(t, nil, func(deps *httpadapter.Deps) {
		deps.Issuer = "https://sso.example.com"
		deps.Authentication = authentication.NewMemoryModule(deps.UserRepo)
		deps.Authentication.SessionManager = deps.Authentication.NewSessionManager()
		deps.Authentication.AuthnResolver = deps.Authentication.SessionManager
		if _, err := deps.Authentication.ConfigureNegotiate(path, deps.UserRepo); err != nil {
			t.Fatal(err)
		}
	})
	response := get(e, "/wsfed?wa=wsignin1.0&wtrealm=urn:idmagic:demo-rp")
	if response.Code != 303 || !strings.Contains(response.Header().Get("Location"), "/login?return_to=") || response.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("response=%d location=%s", response.Code, response.Header().Get("Location"))
	}
}
