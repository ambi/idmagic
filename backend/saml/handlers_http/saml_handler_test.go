package handlers_http_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/saml"
	samlmemory "github.com/ambi/idmagic/backend/saml/db_memory"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/labstack/echo/v5"
)

type stubResolver struct {
	ctx *authdomain.AuthenticationContext
}

func (s stubResolver) Resolve(context.Context, authdomain.Headers) (*authdomain.AuthenticationContext, error) {
	return s.ctx, nil
}

func devSigner(t *testing.T) *samltoken.Signer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test saml signing"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}
	signer, err := samltoken.NewSigner(cert, key)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	return signer
}

func certPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test sp signing"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("cert: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func newServer(t *testing.T, authn *authdomain.AuthenticationContext) (*echo.Echo, *[]spec.DomainEvent) {
	t.Helper()
	e, captured, _ := newServerWithRepository(t, authn)
	return e, captured
}

func newServerWithRepository(t *testing.T, authn *authdomain.AuthenticationContext) (*echo.Echo, *[]spec.DomainEvent, *samlmemory.SamlServiceProviderRepository) {
	t.Helper()

	captured := &[]spec.DomainEvent{}

	spRepo := samlmemory.NewSamlServiceProviderRepository()
	spRepo.Seed(&samldomain.SamlServiceProvider{
		EntityID:      "https://sp.example.com",
		ACSURLs:       []string{"https://sp.example.com/acs"},
		SignAssertion: true,
		ClaimPolicy: claimdomain.ClaimMappingPolicy{
			NameID: claimdomain.NameIdConfiguration{
				Format:          samldomain.SamlNameIDFormatPersistent,
				SourceAttribute: "user_id",
			},
		},
	})

	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{ID: "user-1", PreferredUsername: "alice"})
	keyStore, err := keys_memory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer:   "https://idp.example",
			Contract: spec.CurrentRuntimeContract(),

			Emit: func(ev spec.DomainEvent) { *captured = append(*captured, ev) },
		}, Saml: saml.Module{SPRepo: spRepo, ReplayStore: samlmemory.NewAuthnRequestReplayStore()},
		UserRepo:         userRepo,
		FederationSigner: samltoken.KeyStoreSignerProvider{KeyStore: keyStore},
		AuthnResolver:    stubResolver{ctx: authn},
	})
	return e, captured, spRepo
}

func hasEvent(events []spec.DomainEvent, eventType string) bool {
	for _, ev := range events {
		if ev.EventType() == eventType {
			return true
		}
	}
	return false
}

func get(e *echo.Echo, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, defaultRealmPath(target), http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// authnRequestRedirect は HTTP-Redirect binding 用の SAMLRequest を組み立てる。
func authnRequestRedirect(t *testing.T, issuer, acsURL string) string {
	t.Helper()
	return authnRequestRedirectWith(t, issuer, acsURL, "https://idp.example/realms/default/saml/sso", false)
}

func authnRequestRedirectWith(t *testing.T, issuer, acsURL, destination string, forceAuthn bool) string {
	t.Helper()
	xml := `<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ` +
		`xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_req-1" Version="2.0" ` +
		`IssueInstant="` + time.Now().UTC().Format(time.RFC3339) + `" ` +
		`Destination="` + destination + `"`
	if forceAuthn {
		xml += ` ForceAuthn="true"`
	}
	if acsURL != "" {
		xml += ` AssertionConsumerServiceURL="` + acsURL + `"`
	}
	xml += `><saml:Issuer>` + issuer + `</saml:Issuer></samlp:AuthnRequest>`
	encoded, err := samldomain.EncodeRedirect([]byte(xml))
	if err != nil {
		t.Fatalf("encode redirect: %v", err)
	}
	return encoded
}

func TestSamlSSO_SPInitiatedAuthenticatedIssuesPostForm(t *testing.T) {
	e, events := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}})

	samlReq := authnRequestRedirect(t, "https://sp.example.com", "https://sp.example.com/acs")
	rec := get(e, "/saml/sso?SAMLRequest="+url.QueryEscape(samlReq)+"&RelayState=state-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `action="https://sp.example.com/acs"`) {
		t.Fatalf("form action missing: %s", body)
	}
	if !strings.Contains(body, `name="SAMLResponse"`) {
		t.Fatal("SAMLResponse hidden input missing")
	}
	if !strings.Contains(body, "state-1") {
		t.Fatal("RelayState not echoed")
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control=%q, want no-store", rec.Header().Get("Cache-Control"))
	}
	if !hasEvent(*events, "SamlSignInIssued") {
		t.Fatal("SamlSignInIssued not emitted")
	}
}

func TestSamlSSO_IdPInitiatedIssuesPostForm(t *testing.T) {
	e, events := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix()})

	rec := get(e, "/saml/sso?entityID="+url.QueryEscape("https://sp.example.com"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `action="https://sp.example.com/acs"`) {
		t.Fatalf("form action missing: %s", rec.Body.String())
	}
	if !hasEvent(*events, "SamlSignInIssued") {
		t.Fatal("SamlSignInIssued not emitted")
	}
}

func TestSamlSSO_UnauthenticatedRedirectsToLogin(t *testing.T) {
	e, _ := newServer(t, nil)

	samlReq := authnRequestRedirect(t, "https://sp.example.com", "")
	rec := get(e, "/saml/sso?SAMLRequest="+url.QueryEscape(samlReq))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want 303", rec.Code)
	}
	loc := rec.Header().Get("Location")
	// redirect 先はテナントの正規ロケーション配下の相対パス (ADR-144)。
	if !strings.HasPrefix(loc, "/realms/default/login") || !strings.Contains(loc, "return_to=") {
		t.Fatalf("Location=%q, want /realms/default/login with return_to", loc)
	}
	decoded, err := url.QueryUnescape(loc[strings.Index(loc, "return_to=")+len("return_to="):])
	if err != nil {
		t.Fatalf("unescape return_to: %v", err)
	}
	if !strings.Contains(decoded, "/saml/sso") {
		t.Fatalf("return_to does not point back to /saml/sso: %q", loc)
	}
}

func TestSamlSSO_ForceAuthnWithStaleSessionRedirectsToLogin(t *testing.T) {
	e, _ := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Add(-10 * time.Minute).Unix(), AMR: []string{"pwd"}})

	samlReq := authnRequestRedirectWith(t, "https://sp.example.com", "", "https://idp.example/realms/default/saml/sso", true)
	rec := get(e, "/saml/sso?SAMLRequest="+url.QueryEscape(samlReq))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); !strings.HasPrefix(loc, "/realms/default/login") {
		t.Fatalf("Location=%q, want /realms/default/login", loc)
	}
}

func TestSamlSSO_UnknownServiceProviderRejected(t *testing.T) {
	e, events := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1"})
	samlReq := authnRequestRedirect(t, "https://evil.example.com", "")
	if rec := get(e, "/saml/sso?SAMLRequest="+url.QueryEscape(samlReq)); rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
	if !hasEvent(*events, "SamlSignInRejected") {
		t.Fatal("SamlSignInRejected not emitted")
	}
}

func TestSamlSSO_DisallowedACSRejected(t *testing.T) {
	e, _ := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1"})
	samlReq := authnRequestRedirect(t, "https://sp.example.com", "https://evil.example.com/steal")
	rec := get(e, "/saml/sso?SAMLRequest="+url.QueryEscape(samlReq))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400 (open redirect prevention)", rec.Code)
	}
}

func TestSamlSSO_DestinationMismatchRejected(t *testing.T) {
	e, _ := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1"})
	samlReq := authnRequestRedirectWith(t, "https://sp.example.com", "", "https://evil-idp.example/saml/sso", false)
	rec := get(e, "/saml/sso?SAMLRequest="+url.QueryEscape(samlReq))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}

func TestSamlSSO_UnsignedRequestRejectedWhenSignatureRequired(t *testing.T) {
	spRepo := samlmemory.NewSamlServiceProviderRepository()
	spRepo.Seed(&samldomain.SamlServiceProvider{
		EntityID:                          "https://sp.example.com",
		ACSURLs:                           []string{"https://sp.example.com/acs"},
		SignAssertion:                     true,
		WantAuthnRequestsSigned:           true,
		AuthnRequestSigningCertificatePEM: certPEM(t),
		ClaimPolicy: claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{
			Format: samldomain.SamlNameIDFormatPersistent, SourceAttribute: "user_id",
		}},
	})
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{ID: "user-1", PreferredUsername: "alice"})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer:   "https://idp.example",
			Contract: spec.CurrentRuntimeContract(),
		}, Saml: saml.Module{SPRepo: spRepo, ReplayStore: samlmemory.NewAuthnRequestReplayStore()},
		UserRepo:         userRepo,
		FederationSigner: devSigner(t),
		AuthnResolver:    stubResolver{ctx: &authdomain.AuthenticationContext{UserID: "user-1", AuthTime: time.Now().Unix()}},
	})
	samlReq := authnRequestRedirect(t, "https://sp.example.com", "https://sp.example.com/acs")
	rec := get(e, "/saml/sso?SAMLRequest="+url.QueryEscape(samlReq))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}

func TestSamlSLO_RedirectsToRegisteredSLOURL(t *testing.T) {
	captured := &[]spec.DomainEvent{}
	spRepo := samlmemory.NewSamlServiceProviderRepository()
	spRepo.Seed(&samldomain.SamlServiceProvider{
		EntityID: "https://sp.example.com",
		ACSURLs:  []string{"https://sp.example.com/acs"},
		SLOURL:   "https://sp.example.com/saml/slo",
		ClaimPolicy: claimdomain.ClaimMappingPolicy{
			NameID: claimdomain.NameIdConfiguration{Format: samldomain.SamlNameIDFormatPersistent, SourceAttribute: "user_id"},
		},
	})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer:   "https://idp.example",
			Contract: spec.CurrentRuntimeContract(),

			Emit: func(ev spec.DomainEvent) { *captured = append(*captured, ev) },
		}, Saml: saml.Module{SPRepo: spRepo, ReplayStore: samlmemory.NewAuthnRequestReplayStore()},
		UserRepo:         usermemory.NewUserRepository(),
		FederationSigner: devSigner(t),
		AuthnResolver:    stubResolver{ctx: nil},
	})

	rec := get(e, "/saml/slo?entityID="+url.QueryEscape("https://sp.example.com")+"&RelayState=s1")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "https://sp.example.com/saml/slo?RelayState=s1" {
		t.Fatalf("Location=%q, want registered SLO URL", loc)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("session cookie not cleared: %q", rec.Header().Get("Set-Cookie"))
	}
	if !hasEvent(*captured, "SamlLogout") {
		t.Fatal("SamlLogout not emitted")
	}
}

func TestSamlSLO_LogoutRequestReturnsLogoutResponse(t *testing.T) {
	captured := &[]spec.DomainEvent{}
	spRepo := samlmemory.NewSamlServiceProviderRepository()
	spRepo.Seed(&samldomain.SamlServiceProvider{
		EntityID: "https://sp.example.com",
		ACSURLs:  []string{"https://sp.example.com/acs"},
		SLOURL:   "https://sp.example.com/saml/slo",
		ClaimPolicy: claimdomain.ClaimMappingPolicy{
			NameID: claimdomain.NameIdConfiguration{Format: samldomain.SamlNameIDFormatPersistent, SourceAttribute: "user_id"},
		},
	})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer:   "https://idp.example",
			Contract: spec.CurrentRuntimeContract(),

			Emit: func(ev spec.DomainEvent) { *captured = append(*captured, ev) },
		}, Saml: saml.Module{SPRepo: spRepo, ReplayStore: samlmemory.NewAuthnRequestReplayStore()},
		UserRepo:         usermemory.NewUserRepository(),
		FederationSigner: devSigner(t),
		AuthnResolver:    stubResolver{ctx: nil},
	})
	xml := `<samlp:LogoutRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ` +
		`xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_logout-1" Version="2.0" ` +
		`Destination="https://idp.example/realms/default/saml/slo">` +
		`<saml:Issuer>https://sp.example.com</saml:Issuer>` +
		`<saml:NameID>user-1</saml:NameID></samlp:LogoutRequest>`
	samlReq, err := samldomain.EncodeRedirect([]byte(xml))
	if err != nil {
		t.Fatalf("encode LogoutRequest: %v", err)
	}
	rec := get(e, "/saml/slo?SAMLRequest="+url.QueryEscape(samlReq)+"&RelayState=s1")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want 303 body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "https://sp.example.com/saml/slo?SAMLResponse=") || !strings.Contains(loc, "RelayState=s1") {
		t.Fatalf("Location=%q, want LogoutResponse redirect", loc)
	}
	if !hasEvent(*captured, "SamlLogout") {
		t.Fatal("SamlLogout not emitted")
	}
}

func TestSamlSLO_UnknownSPReturns200(t *testing.T) {
	e, _ := newServer(t, nil)
	// seed した SP は SLO URL を持たないため、リダイレクトせず 200 を返す (open redirect 防止)。
	rec := get(e, "/saml/slo?entityID="+url.QueryEscape("https://sp.example.com"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
}

func TestSamlMetadata_Published(t *testing.T) {
	e, _ := newServer(t, nil)
	rec := get(e, "/saml/metadata")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"IDPSSODescriptor",
		"SingleSignOnService",
		"X509Certificate",
		"https://idp.example/realms/default/saml/sso",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metadata missing %q:\n%s", want, body)
		}
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/xml") {
		t.Fatalf("Content-Type=%q, want application/xml", ct)
	}
}

func TestSamlSigningCertificateDownloadMatchesMetadata(t *testing.T) {
	e, _ := newServer(t, nil)
	pemResponse := get(e, "/saml/signing-certificate.pem")
	if pemResponse.Code != http.StatusOK {
		t.Fatalf("certificate status=%d body=%s", pemResponse.Code, pemResponse.Body.String())
	}
	block, _ := pem.Decode(pemResponse.Body.Bytes())
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("invalid certificate PEM: %q", pemResponse.Body.String())
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	metadataResponse := get(e, "/saml/metadata")
	if metadataResponse.Code != http.StatusOK {
		t.Fatalf("metadata status=%d body=%s", metadataResponse.Code, metadataResponse.Body.String())
	}
	if want := base64.StdEncoding.EncodeToString(block.Bytes); !strings.Contains(metadataResponse.Body.String(), want) {
		t.Fatal("downloaded certificate is not published in SAML metadata")
	}
}

func TestSamlIDPProfilePublishesIsolatedMetadataAndRejectsCrossProfileRequest(t *testing.T) {
	e, _, repo := newServerWithRepository(t, nil)
	profile := &samldomain.SamlIdentityProviderProfile{
		TenantID: tenancydomain.DefaultTenantID, ProfileID: "partner-a",
		Name: "Partner A", Mode: samldomain.IDPProfileModeDedicated,
	}
	if err := repo.SaveIDPProfile(context.Background(), profile); err != nil {
		t.Fatal(err)
	}

	defaultMetadata := get(e, "/saml/metadata")
	profileMetadata := get(e, "/saml/idp/partner-a/metadata")
	if profileMetadata.Code != http.StatusOK {
		t.Fatalf("profile metadata status=%d body=%s", profileMetadata.Code, profileMetadata.Body.String())
	}
	for _, want := range []string{
		`entityID="https://idp.example/realms/default/saml/idp/partner-a"`,
		"https://idp.example/realms/default/saml/idp/partner-a/sso",
		"https://idp.example/realms/default/saml/idp/partner-a/slo",
	} {
		if !strings.Contains(profileMetadata.Body.String(), want) {
			t.Fatalf("profile metadata missing %q:\n%s", want, profileMetadata.Body.String())
		}
	}
	if defaultMetadata.Body.String() == profileMetadata.Body.String() {
		t.Fatal("default and additional profiles must publish distinct metadata")
	}
	profileCertificate := get(e, "/saml/idp/partner-a/signing-certificate.pem")
	block, _ := pem.Decode(profileCertificate.Body.Bytes())
	if block == nil {
		t.Fatalf("invalid profile certificate: %s", profileCertificate.Body.String())
	}
	if encoded := base64.StdEncoding.EncodeToString(block.Bytes); !strings.Contains(profileMetadata.Body.String(), encoded) {
		t.Fatal("profile certificate is not published in its metadata")
	}
	if rec := get(e, "/saml/idp/missing/metadata"); rec.Code != http.StatusNotFound {
		t.Fatalf("missing profile status=%d body=%s", rec.Code, rec.Body.String())
	}

	request := authnRequestRedirectWith(
		t,
		"https://sp.example.com",
		"https://sp.example.com/acs",
		"https://idp.example/realms/default/saml/idp/partner-a/sso",
		false,
	)
	rec := get(e, "/saml/idp/partner-a/sso?SAMLRequest="+url.QueryEscape(request))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "not assigned") {
		t.Fatalf("cross-profile status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func newAdminServer(t *testing.T) *echo.Echo {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID:                "admin-1",
		TenantID:          tenancydomain.DefaultTenantID,
		PreferredUsername: "admin@example.com",
		Roles:             []string{"admin"},
	})
	keyStore, err := keys_memory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer:   "https://idp.example",
			Contract: spec.CurrentRuntimeContract(),
		}, Saml: saml.Module{SPRepo: samlmemory.NewSamlServiceProviderRepository()},
		UserRepo:         userRepo,
		AuthnResolver:    stubResolver{ctx: &authdomain.AuthenticationContext{UserID: "admin-1"}},
		FederationSigner: samltoken.KeyStoreSignerProvider{KeyStore: keyStore},
	})
	return e
}

func doJSON(e *echo.Echo, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, defaultRealmPath(target), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func doAdminJSON(e *echo.Echo, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, defaultRealmPath(target), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://idp.example")
	req.Header.Set("X-Csrf-Token", "csrf")
	req.Header.Set("Cookie", "idmagic_csrf=csrf")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestAdminIDPProfileCRUDAndCanonicalEndpoints(t *testing.T) {
	e := newAdminServer(t)
	const path = "/api/admin/v1/saml/idp-profiles"
	create := doAdminJSON(e, http.MethodPost, path, `{"name":"Partner trust","mode":"dedicated"}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	body := create.Body.String()
	if !strings.Contains(body, `"mode":"dedicated"`) ||
		!strings.Contains(body, `/saml/idp/`) ||
		!strings.Contains(body, `/metadata`) {
		t.Fatalf("create response missing profile endpoints: %s", body)
	}
	var created struct {
		Profile samldomain.SamlIdentityProviderProfile `json:"profile"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Profile.ProfileID == "" || created.Profile.ProfileID == samldomain.DefaultIDPProfileID {
		t.Fatalf("generated profile ID = %q", created.Profile.ProfileID)
	}
	if rec := get(e, path); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"profile_id":"default"`) ||
		!strings.Contains(rec.Body.String(), created.Profile.ProfileID) {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := doAdminJSON(e, http.MethodDelete, path+"/"+created.Profile.ProfileID, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := doAdminJSON(e, http.MethodDelete, path+"/default", ""); rec.Code != http.StatusConflict {
		t.Fatalf("delete default status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminServiceProvider_CRUD(t *testing.T) {
	e := newAdminServer(t)
	profileCreate := doAdminJSON(e, http.MethodPost, "/api/admin/v1/saml/idp-profiles", `{"name":"Partner trust","mode":"shared"}`)
	if profileCreate.Code != http.StatusCreated {
		t.Fatalf("create profile status=%d body=%s", profileCreate.Code, profileCreate.Body.String())
	}
	var profileResponse struct {
		Profile samldomain.SamlIdentityProviderProfile `json:"profile"`
	}
	if err := json.Unmarshal(profileCreate.Body.Bytes(), &profileResponse); err != nil {
		t.Fatal(err)
	}

	const path = "/api/admin/v1/saml/service-providers"
	body := `{"entity_id":"https://sp.example.com","idp_profile_id":"` + profileResponse.Profile.ProfileID +
		`","acs_urls":["https://sp.example.com/acs"],` +
		`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`

	if rec := doJSON(e, http.MethodPost, path, body); rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}
	updateWithoutProfile := `{"entity_id":"https://sp.example.com","acs_urls":["https://sp.example.com/acs"],` +
		`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`
	if rec := doJSON(e, http.MethodPost, path, updateWithoutProfile); rec.Code != http.StatusOK {
		t.Fatalf("update status=%d, want 200", rec.Code)
	}
	if rec := get(e, path); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), "https://sp.example.com") ||
		!strings.Contains(rec.Body.String(), `"idp_profile_id":"`+profileResponse.Profile.ProfileID+`"`) {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := doJSON(e, http.MethodDelete, path+"?entity_id="+url.QueryEscape("https://sp.example.com"), ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d, want 204", rec.Code)
	}
	if rec := get(e, path); strings.Contains(rec.Body.String(), "https://sp.example.com") {
		t.Fatalf("SP still present after delete: %s", rec.Body.String())
	}
}

func TestAdminServiceProvider_RejectsInvalid(t *testing.T) {
	e := newAdminServer(t)
	// acs_urls 欠落。
	body := `{"entity_id":"https://sp.example.com","claim_policy":{"name_id":{"format":"f","source_attribute":"user_id"}}}`
	if rec := doJSON(e, http.MethodPost, "/api/admin/v1/saml/service-providers", body); rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}

func TestAdminServiceProvider_RejectsUnsupportedSignedAuthnRequests(t *testing.T) {
	e := newAdminServer(t)
	body := `{"entity_id":"https://sp.example.com","acs_urls":["https://sp.example.com/acs"],` +
		`"want_authn_requests_signed":true,` +
		`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent","source_attribute":"user_id"}}}`
	if rec := doJSON(e, http.MethodPost, "/api/admin/v1/saml/service-providers", body); rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}

func TestAdminServiceProvider_ForbiddenForNonAdmin(t *testing.T) {
	e, _ := newServer(t, &authdomain.AuthenticationContext{UserID: "user-1"}) // 非 admin
	if rec := get(e, "/api/admin/v1/saml/service-providers"); rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", rec.Code)
	}
}

// defaultRealmPath は bare path を default テナントの正規ロケーション配下へ移す。
// ADR-144 で bare path はどのテナントの正規ロケーションでもなくなったため、
// テストのリクエスト先も /realms/default 配下でなければ 404 になる。
func defaultRealmPath(path string) string {
	if strings.HasPrefix(path, "/realms/") {
		return path
	}
	return "/realms/default" + path
}
