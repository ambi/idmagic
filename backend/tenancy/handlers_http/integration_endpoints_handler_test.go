package handlers_http_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantmemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenancyhttp "github.com/ambi/idmagic/backend/tenancy/handlers_http"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/labstack/echo/v5"
)

func integrationEndpointServer(t *testing.T, endpointStyle tenantdomain.TenantEndpointStyle) *echo.Echo {
	t.Helper()
	const tenantID = "11111111-1111-4111-8111-111111111111"
	tenantRepo := tenantmemory.NewTenantRepository()
	if err := tenantRepo.Save(context.Background(), &tenantdomain.Tenant{
		ID: tenantID, Realm: "acme", DisplayName: "Acme", Status: tenantdomain.TenantStatusActive,
		EndpointStyle: endpointStyle, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID: "admin", TenantID: tenantID, PreferredUsername: "admin", Roles: []string{"admin"},
	})
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject:      pkix.Name{CommonName: "Acme federation signing"},
		NotBefore:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:     time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}, &x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject:      pkix.Name{CommonName: "Acme federation signing"},
	}, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := samltoken.NewSigner(cert, key)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:           "https://idp.example",
		TenantBaseDomain: "idp.example",
		Contract:         spec.CurrentRuntimeContract(),
		TenantRepo:       tenantRepo,
		UserRepo:         userRepo,
		AuthnResolver:    stubIntegrationAuthnResolver{userID: "admin"},
		FederationSigner: signer,
	})
	return e
}

type stubIntegrationAuthnResolver struct {
	userID string
}

func (s stubIntegrationAuthnResolver) Resolve(context.Context, authdomain.Headers) (*authdomain.AuthenticationContext, error) {
	return &authdomain.AuthenticationContext{UserID: s.userID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"}}, nil
}

func TestAdminIntegrationEndpointsUseCanonicalPathIssuer(t *testing.T) {
	e := integrationEndpointServer(t, tenantdomain.TenantEndpointStylePath)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/realms/acme/api/admin/v1/integration-endpoints", http.NoBody)
	req.Host = "idp.example"
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got tenancyhttp.AdminIntegrationEndpointCatalog
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	const issuer = "https://idp.example/realms/acme"
	if got.Issuer != issuer ||
		got.OAuth.OpenIDConfiguration != issuer+"/.well-known/openid-configuration" ||
		got.SAML.MetadataURL != issuer+"/saml/metadata" ||
		got.WSFederation.MetadataURL != issuer+"/federationmetadata/2007-06/federationmetadata.xml" ||
		got.APIs.SCIMBaseURL != issuer+"/scim/v2" {
		t.Fatalf("catalog=%+v", got)
	}
	if got.SAML.SigningCertificate.FingerprintSHA256 == "" {
		t.Fatal("certificate fingerprint is required")
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control=%q", cc)
	}
}

func TestAdminIntegrationEndpointsUseCanonicalSubdomainIssuer(t *testing.T) {
	e := integrationEndpointServer(t, tenantdomain.TenantEndpointStyleSubdomain)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/integration-endpoints", http.NoBody)
	req.Host = "acme.idp.example"
	req.Header.Set("X-Forwarded-Proto", "https")
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got tenancyhttp.AdminIntegrationEndpointCatalog
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Issuer != "https://acme.idp.example" {
		t.Fatalf("issuer=%q", got.Issuer)
	}
}
