package handlers_http_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"

	httpdeps "github.com/ambi/idmagic/backend/authentication/internal/deps_http"
	federationmemory "github.com/ambi/idmagic/backend/authentication/internal/federation/db_memory"
	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
	federationhttp "github.com/ambi/idmagic/backend/authentication/internal/federation/handlers_http"
	oidcprotocol "github.com/ambi/idmagic/backend/authentication/internal/federation/protocol_oidc"
	federationusecases "github.com/ambi/idmagic/backend/authentication/internal/federation/usecases"
	authusecases "github.com/ambi/idmagic/backend/authentication/internal/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"

	"github.com/labstack/echo/v5"
)

type driverStub struct{}

func (driverStub) Start(
	_ federationdomain.IdentityProviderConnection,
	_ federationdomain.FederatedLoginAttempt,
	_ string,
	_ time.Time,
) (string, error) {
	return "https://idp.example/authorize", nil
}

func (driverStub) Complete(
	context.Context,
	federationdomain.IdentityProviderConnection,
	federationdomain.FederatedLoginAttempt,
	string,
	string,
	time.Time,
) (federationdomain.NormalizedClaims, error) {
	return federationdomain.NormalizedClaims{}, nil
}

func TestPublicProviderDiscoveryOmitsTrustAndSecret(t *testing.T) {
	e, repos := newServer(t)
	now := time.Now().UTC()
	if err := repos.Connections.Save(testing_tenant.Default(context.Background()), &federationdomain.IdentityProviderConnection{
		ID: "oidc", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Workforce",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionActive,
		Issuer: "https://idp.example", ClientID: "client", SecretReference: "env:SECRET",
		AuthorizationEndpoint: "https://idp.example/auth",
		TokenEndpoint:         "https://idp.example/token", JWKSURI: "https://idp.example/jwks",
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: federationdomain.LinkingNone, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/auth/federation/providers", http.NoBody)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `"display_name":"Workforce"`) ||
		strings.Contains(body, "SECRET") || strings.Contains(body, "token_endpoint") {
		t.Fatalf("unsafe discovery response=%s", body)
	}
}

func TestStartFederatedLoginRedirectsOnlyThroughSavedProvider(t *testing.T) {
	e, repos := newServer(t)
	now := time.Now().UTC()
	connection := &federationdomain.IdentityProviderConnection{
		ID: "oidc", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Workforce",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionActive,
		Issuer: "https://idp.example", ClientID: "client",
		AuthorizationEndpoint: "https://idp.example/auth",
		TokenEndpoint:         "https://idp.example/token", JWKSURI: "https://idp.example/jwks",
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: federationdomain.LinkingNone, CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Connections.Save(testing_tenant.Default(context.Background()), connection); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/auth/federation/start?provider_id=oidc", http.NoBody)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "https://idp.example/authorize" {
		t.Fatalf("status=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
}

// newAdminServer wires the same routes as newServer plus a working admin auth stack
// (DemoHeaderResolver keyed by X-Demo-Sub, and a matching CSRF cookie/header pair) so admin
// CRUD/lifecycle/test handlers can be exercised end-to-end. oidcClient may be nil.
func newAdminServer(t *testing.T, oidcClient *oidcprotocol.Client, configure ...func(*federationhttp.Deps, *echo.Echo)) (*echo.Echo, federationmemory.Repositories) {
	t.Helper()
	repos := federationmemory.NewRepositories()
	users := idmanagement.NewMemoryModule().UserRepo
	now := time.Now().UTC()
	if err := users.Save(testing_tenant.Default(context.Background()), &userdomain.User{
		ID: "admin-1", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	deps := federationhttp.Deps{
		Broker: federationusecases.BrokerDeps{
			Connections: repos.Connections, Identities: repos.Identities, Attempts: repos.Attempts,
			Drivers: map[federationdomain.Protocol]federationusecases.ProtocolDriver{
				federationdomain.ProtocolOIDC: driverStub{},
			},
		},
		Auth: httpdeps.Deps{
			Deps: support.Deps{Issuer: "https://idp.test"},
			Authenticator: &support.Authenticator{
				Principals: idmanagement.Module{UserRepo: users}.Principals(),
				Sessions:   httpdeps.SessionAuthentications{Resolver: authusecases.DemoHeaderResolver{}},
			},
		},
		OIDC: oidcClient,
	}
	for _, apply := range configure {
		apply(&deps, e)
	}
	federationhttp.RegisterRoutes(e.Group("", testing_tenant.ResolveDefault), deps)
	return e, repos
}

const adminCSRFToken = "test-csrf-token"

func adminRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("X-Demo-Sub", "admin-1")
	request.Header.Set("Origin", "https://idp.test")
	request.Header.Set("X-Csrf-Token", adminCSRFToken)
	request.AddCookie(&http.Cookie{Name: "idmagic_csrf", Value: adminCSRFToken})
	return request
}

// RED (interface: CreateIdentityProviderConnection): a newly created connection
// starts Disabled, not the removed Draft status.
func TestCreateAdminInitialStatusIsDisabled(t *testing.T) {
	e, _ := newAdminServer(t, nil)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, adminRequest(t, http.MethodPost, "/api/admin/v1/identity-providers", map[string]any{
		"display_name": "Google", "protocol": "oidc", "issuer": "https://accounts.example.com",
		"client_id": "client-1", "authorization_endpoint": "https://accounts.example.com/authorize",
		"token_endpoint": "https://accounts.example.com/token", "jwks_uri": "https://accounts.example.com/jwks",
		"claim_mapping":  map[string]string{"subject": "sub", "username": "email"},
		"linking_policy": "none",
	}))
	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var created federationdomain.IdentityProviderConnection
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Status != federationdomain.ConnectionDisabled {
		t.Fatalf("status=%q, want disabled", created.Status)
	}
}

// samlCertificateDER は、保存の検査が解析する自己署名の証明書の DER である。
func samlCertificateDER(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "SAML IdP"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return der
}

// copiedFromMetadata は、メタデータの X509Certificate のように 64 文字で折り返した、見出しのない DER の base64 である。
func copiedFromMetadata(der []byte) string {
	encoded := base64.StdEncoding.EncodeToString(der)
	var out strings.Builder
	for len(encoded) > 64 {
		out.WriteString(encoded[:64] + "\n")
		encoded = encoded[64:]
	}
	out.WriteString(encoded)
	return out.String()
}

func samlConnectionBody(certificates ...string) map[string]any {
	return map[string]any{
		"display_name": "Entra ID", "protocol": "saml", "issuer": "https://sts.windows.net/tenant/",
		"saml_entity_id": "https://sts.windows.net/tenant/", "saml_sso_url": "https://login.microsoftonline.com/tenant/saml2",
		"saml_signing_certificates": certificates,
		"claim_mapping":             map[string]string{"subject": "NameID", "username": "email"},
		"linking_policy":            "none",
	}
}

// 期待する PEM は、送った DER を標準の pem パッケージで包んだ値であり、製品の正規化の関数から計算しない。
//
//spec:covers REQ-AUTHENTICATION-025: 見出しのない DER の base64 で送った SAML の署名の証明書を同じ DER の正規の PEM で保存して返し、解析できない値を 400 と invalid_request で拒否して保存しないこと
func TestCreateAdminNormalizesSAMLSigningCertificatesCopiedFromMetadata(t *testing.T) {
	e, repos := newAdminServer(t, nil)
	der := samlCertificateDER(t)
	want := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))

	refused := httptest.NewRecorder()
	e.ServeHTTP(refused, adminRequest(t, http.MethodPost, "/api/admin/v1/identity-providers",
		samlConnectionBody(copiedFromMetadata(der), "MIIC not a certificate")))
	if refused.Code != http.StatusBadRequest || !strings.Contains(refused.Body.String(), "invalid_request") {
		t.Fatalf("status=%d body=%s, want 400 invalid_request", refused.Code, refused.Body.String())
	}
	if stored, err := repos.Connections.List(testing_tenant.Default(context.Background()), tenancydomain.DefaultTenantID); err != nil || len(stored) != 0 {
		t.Fatalf("stored=%+v err=%v after a refused create, want nothing saved", stored, err)
	}

	created := httptest.NewRecorder()
	e.ServeHTTP(created, adminRequest(t, http.MethodPost, "/api/admin/v1/identity-providers",
		samlConnectionBody(copiedFromMetadata(der))))
	if created.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", created.Code, created.Body.String())
	}
	var returned federationdomain.IdentityProviderConnection
	if err := json.Unmarshal(created.Body.Bytes(), &returned); err != nil {
		t.Fatal(err)
	}
	if len(returned.SAMLSigningCertificates) != 1 || returned.SAMLSigningCertificates[0] != want {
		t.Fatalf("returned certificates=%q, want %q", returned.SAMLSigningCertificates, want)
	}
	stored, err := repos.Connections.Find(testing_tenant.Default(context.Background()), tenancydomain.DefaultTenantID, returned.ID)
	if err != nil || stored == nil {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
	if len(stored.SAMLSigningCertificates) != 1 || stored.SAMLSigningCertificates[0] != want {
		t.Fatalf("stored certificates=%q, want %q", stored.SAMLSigningCertificates, want)
	}
}

// 更新も作成と同じ規則で揃える。同じ証明書を形を変えて送り直しても信頼の根拠の変更ではないので Active を保ち、
// 解析できない値は拒否して保存済みの接続を変えない。
//
//spec:covers REQ-AUTHENTICATION-025: 更新でも SAML の署名の証明書を正規の PEM に揃え、解析できない値を 400 と invalid_request で拒否して保存しないこと
func TestUpdateAdminNormalizesSAMLSigningCertificatesBeforeComparingTrust(t *testing.T) {
	e, repos := newAdminServer(t, nil)
	der := samlCertificateDER(t)
	canonical := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	now := time.Now().UTC()
	connection := &federationdomain.IdentityProviderConnection{
		ID: "entra", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Entra ID",
		Protocol: federationdomain.ProtocolSAML, Status: federationdomain.ConnectionActive,
		Issuer: "https://sts.windows.net/tenant/", SAMLEntityID: "https://sts.windows.net/tenant/",
		SAMLSSOURL: "https://login.microsoftonline.com/tenant/saml2", SAMLSigningCertificates: []string{canonical},
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "NameID", Username: "email"},
		LinkingPolicy: federationdomain.LinkingNone, CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Connections.Save(testing_tenant.Default(context.Background()), connection); err != nil {
		t.Fatal(err)
	}

	refused := httptest.NewRecorder()
	e.ServeHTTP(refused, adminRequest(t, http.MethodPut, "/api/admin/v1/identity-providers/entra", samlConnectionBody("not a certificate")))
	if refused.Code != http.StatusBadRequest || !strings.Contains(refused.Body.String(), "invalid_request") {
		t.Fatalf("status=%d body=%s, want 400 invalid_request", refused.Code, refused.Body.String())
	}
	stored, err := repos.Connections.Find(testing_tenant.Default(context.Background()), tenancydomain.DefaultTenantID, "entra")
	if err != nil || stored == nil || len(stored.SAMLSigningCertificates) != 1 || stored.SAMLSigningCertificates[0] != canonical {
		t.Fatalf("stored=%+v err=%v after a refused update, want the saved certificate kept", stored, err)
	}

	updated := httptest.NewRecorder()
	e.ServeHTTP(updated, adminRequest(t, http.MethodPut, "/api/admin/v1/identity-providers/entra", samlConnectionBody(copiedFromMetadata(der))))
	var returned federationdomain.IdentityProviderConnection
	if err := json.Unmarshal(updated.Body.Bytes(), &returned); err != nil || updated.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s err=%v", updated.Code, updated.Body.String(), err)
	}
	if returned.Status != federationdomain.ConnectionActive {
		t.Fatalf("status=%q after re-sending the same certificate as base64, want active preserved", returned.Status)
	}
	if len(returned.SAMLSigningCertificates) != 1 || returned.SAMLSigningCertificates[0] != canonical {
		t.Fatalf("returned certificates=%q, want %q", returned.SAMLSigningCertificates, canonical)
	}
}

// RED (interface: UpdateIdentityProviderConnection, §編集時の自動デグレード):
// updating only display_name keeps an Active connection Active; changing the issuer degrades it.
func TestUpdateAdminDegradesOnlyOnTrustSourceChange(t *testing.T) {
	e, repos := newAdminServer(t, nil)
	now := time.Now().UTC()
	connection := &federationdomain.IdentityProviderConnection{
		ID: "google", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Google",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionActive,
		Issuer: "https://accounts.example.com", ClientID: "client-1",
		AuthorizationEndpoint: "https://accounts.example.com/authorize",
		TokenEndpoint:         "https://accounts.example.com/token", JWKSURI: "https://accounts.example.com/jwks",
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: federationdomain.LinkingNone,
		CreatedAt:     now, UpdatedAt: now,
	}
	if err := repos.Connections.Save(testing_tenant.Default(context.Background()), connection); err != nil {
		t.Fatal(err)
	}
	body := map[string]any{
		"display_name": "Google Workspace", "protocol": "oidc", "issuer": connection.Issuer,
		"client_id": connection.ClientID, "authorization_endpoint": connection.AuthorizationEndpoint,
		"token_endpoint": connection.TokenEndpoint, "jwks_uri": connection.JWKSURI,
		"claim_mapping": map[string]string{"subject": "sub", "username": "email"}, "linking_policy": "none",
	}
	response := httptest.NewRecorder()
	e.ServeHTTP(response, adminRequest(t, http.MethodPut, "/api/admin/v1/identity-providers/google", body))
	var updated federationdomain.IdentityProviderConnection
	if err := json.Unmarshal(response.Body.Bytes(), &updated); err != nil {
		t.Fatalf("status=%d body=%s err=%v", response.Code, response.Body.String(), err)
	}
	if updated.Status != federationdomain.ConnectionActive {
		t.Fatalf("status=%q after display_name-only change, want active preserved", updated.Status)
	}

	body["issuer"] = "https://other.example.com"
	response = httptest.NewRecorder()
	e.ServeHTTP(response, adminRequest(t, http.MethodPut, "/api/admin/v1/identity-providers/google", body))
	if err := json.Unmarshal(response.Body.Bytes(), &updated); err != nil {
		t.Fatalf("status=%d body=%s err=%v", response.Code, response.Body.String(), err)
	}
	if updated.Status != federationdomain.ConnectionDisabled {
		t.Fatalf("status=%q after issuer change, want disabled", updated.Status)
	}
}

// RED (interface: DeleteIdentityProviderConnection): deletion no longer requires
// Disabled first; an Active connection can be deleted directly.
func TestDeleteAdminSucceedsForActiveConnection(t *testing.T) {
	e, repos := newAdminServer(t, nil)
	now := time.Now().UTC()
	if err := repos.Connections.Save(testing_tenant.Default(context.Background()), &federationdomain.IdentityProviderConnection{
		ID: "google", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Google",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionActive,
		Issuer: "https://accounts.example.com", ClientID: "client-1",
		AuthorizationEndpoint: "https://accounts.example.com/authorize",
		TokenEndpoint:         "https://accounts.example.com/token", JWKSURI: "https://accounts.example.com/jwks",
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: federationdomain.LinkingNone,
		CreatedAt:     now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	e.ServeHTTP(response, adminRequest(t, http.MethodDelete, "/api/admin/v1/identity-providers/google", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s, want 204 without a status guard", response.Code, response.Body.String())
	}
}

//spec:covers EX-AUTHENTICATION-037-02: 連携が残る接続の削除要求に 409 connection_in_use を返し、接続と連携を保存先に残すこと。連携を解除した後の同じ要求が 204 になることで、拒否が連携によるものだと示す。
func TestDeleteAdminRefusesAConnectionWithLinkedIdentities(t *testing.T) {
	e, repos := newAdminServer(t, nil)
	now := time.Now().UTC()
	if err := repos.Connections.Save(testing_tenant.Default(context.Background()), &federationdomain.IdentityProviderConnection{
		ID: "google", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Google",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionActive,
		Issuer: "https://accounts.example.com", ClientID: "client-1",
		AuthorizationEndpoint: "https://accounts.example.com/authorize",
		TokenEndpoint:         "https://accounts.example.com/token", JWKSURI: "https://accounts.example.com/jwks",
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: federationdomain.LinkingNone,
		CreatedAt:     now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := repos.Identities.Create(testing_tenant.Default(context.Background()), &federationdomain.FederatedIdentity{
		TenantID: tenancydomain.DefaultTenantID, ProviderID: "google", ExternalSubject: "external-1",
		LocalUserID: "user-1", LinkedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	refused := httptest.NewRecorder()
	e.ServeHTTP(refused, adminRequest(t, http.MethodDelete, "/api/admin/v1/identity-providers/google", nil))
	if refused.Code != http.StatusConflict || !strings.Contains(refused.Body.String(), "urn:idmagic:error:connection_in_use") {
		t.Fatalf("status=%d body=%s, want 409 connection_in_use", refused.Code, refused.Body.String())
	}
	if connection, err := repos.Connections.Find(testing_tenant.Default(context.Background()), tenancydomain.DefaultTenantID, "google"); err != nil || connection == nil {
		t.Fatalf("the refused connection is gone: connection=%v err=%v", connection, err)
	}
	if identity, err := repos.Identities.FindByUserProvider(testing_tenant.Default(context.Background()), tenancydomain.DefaultTenantID, "google", "user-1"); err != nil || identity == nil {
		t.Fatalf("the link is gone: identity=%v err=%v", identity, err)
	}

	if err := repos.Identities.Delete(testing_tenant.Default(context.Background()), tenancydomain.DefaultTenantID, "google", "user-1"); err != nil {
		t.Fatal(err)
	}
	deleted := httptest.NewRecorder()
	e.ServeHTTP(deleted, adminRequest(t, http.MethodDelete, "/api/admin/v1/identity-providers/google", nil))
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("after unlinking status=%d body=%s, want 204", deleted.Code, deleted.Body.String())
	}
}

// RED (interface: TestIdentityProviderConnection): the test action returns a
// structured success/failure result reflecting real reachability, not a canned "valid" string.
func TestTestAdminReportsStructuredReachabilityResult(t *testing.T) {
	oidcClient := oidcprotocol.Client{HTTPClient: &http.Client{Transport: fakeRoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() == "https://accounts.example.com/jwks" {
			return nil, context.DeadlineExceeded
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
	})}}
	e, repos := newAdminServer(t, &oidcClient)
	now := time.Now().UTC()
	if err := repos.Connections.Save(testing_tenant.Default(context.Background()), &federationdomain.IdentityProviderConnection{
		ID: "google", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Google",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionDisabled,
		Issuer: "https://accounts.example.com", ClientID: "client-1",
		AuthorizationEndpoint: "https://accounts.example.com/authorize",
		TokenEndpoint:         "https://accounts.example.com/token", JWKSURI: "https://accounts.example.com/jwks",
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: federationdomain.LinkingNone,
		CreatedAt:     now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	e.ServeHTTP(response, adminRequest(t, http.MethodPost, "/api/admin/v1/identity-providers/google/test", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Result struct {
			Success  bool     `json:"success"`
			Failures []string `json:"failures"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Result.Success || len(body.Result.Failures) == 0 {
		t.Fatalf("result=%+v, want success=false with a JWKS unreachability failure", body.Result)
	}
}

type fakeRoundTripper func(*http.Request) (*http.Response, error)

func (f fakeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newServer(t *testing.T) (*echo.Echo, federationmemory.Repositories) {
	t.Helper()
	repos := federationmemory.NewRepositories()
	e := echo.New()
	federationhttp.RegisterRoutes(e.Group("", testing_tenant.ResolveDefault), federationhttp.Deps{
		Broker: federationusecases.BrokerDeps{
			Connections: repos.Connections, Identities: repos.Identities, Attempts: repos.Attempts,
			Drivers: map[federationdomain.Protocol]federationusecases.ProtocolDriver{
				federationdomain.ProtocolOIDC: driverStub{},
			},
		},
	})
	return e, repos
}
