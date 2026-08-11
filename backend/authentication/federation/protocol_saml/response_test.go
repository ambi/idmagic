package protocol_saml

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"

	federationmemory "github.com/ambi/idmagic/backend/authentication/federation/db_memory"
	"github.com/ambi/idmagic/backend/authentication/federation/domain"
)

func TestBuildAuthnRequestBindsDestinationACSAndRequestID(t *testing.T) {
	connection := testSAMLConnection(t)
	attempt := domain.FederatedLoginAttempt{State: "relay", RequestID: "_request-1"}
	redirect, err := BuildAuthnRequest(connection, attempt, "https://broker.example/callback", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if redirect == "" || redirect[:len(connection.SAMLSSOURL)] != connection.SAMLSSOURL {
		t.Fatalf("redirect=%q", redirect)
	}
}

func TestValidateResponseChecksSignatureCorrelationAudienceAndClaims(t *testing.T) {
	connection, key, certificate := testSAMLConnectionWithKey(t)
	now := time.Now().UTC()
	response := signedResponse(t, key, certificate, now)
	replay := federationmemory.NewRepositories().Replay
	claims, err := ValidateResponse(
		context.Background(), connection,
		domain.FederatedLoginAttempt{RequestID: "_request-1"},
		base64.StdEncoding.EncodeToString(response),
		"https://broker.example/callback", now, replay,
	)
	if err != nil {
		t.Fatalf("ValidateResponse: %v", err)
	}
	if claims.Subject != "external-1" || claims.Username != "user@example.com" || !claims.EmailVerified {
		t.Fatalf("claims=%+v", claims)
	}
	if _, err := ValidateResponse(
		context.Background(), connection,
		domain.FederatedLoginAttempt{RequestID: "_request-1"},
		base64.StdEncoding.EncodeToString(response),
		"https://broker.example/callback", now, replay,
	); err == nil {
		t.Fatal("response replay must be rejected")
	}
}

// RED (interface: TestIdentityProviderConnection, SAML test-connection design):
// a currently-valid signing certificate reports no failures.
func TestValidateSigningCertificatesAcceptsValidCertificate(t *testing.T) {
	now := time.Now().UTC()
	certificatePEM := pemCertificate(t, now.Add(-time.Hour), now.Add(time.Hour))
	if failures := ValidateSigningCertificates([]string{certificatePEM}, now); len(failures) != 0 {
		t.Fatalf("failures=%v, want none", failures)
	}
}

// RED: an expired certificate is reported as a failure, and an unparsable PEM value is too.
func TestValidateSigningCertificatesRejectsExpiredOrUnparsableCertificate(t *testing.T) {
	now := time.Now().UTC()
	expired := pemCertificate(t, now.Add(-2*time.Hour), now.Add(-time.Hour))
	failures := ValidateSigningCertificates([]string{expired, "not a certificate"}, now)
	if len(failures) != 2 {
		t.Fatalf("failures=%v, want 2 (expired + unparsable)", failures)
	}
}

// RED: no configured certificates is itself a failure.
func TestValidateSigningCertificatesRejectsEmptyList(t *testing.T) {
	if failures := ValidateSigningCertificates(nil, time.Now().UTC()); len(failures) != 1 {
		t.Fatalf("failures=%v, want 1", failures)
	}
}

func pemCertificate(t *testing.T, notBefore, notAfter time.Time) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "SAML IdP"},
		NotBefore: notBefore, NotAfter: notAfter, KeyUsage: x509.KeyUsageDigitalSignature,
	}, &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "SAML IdP"},
		NotBefore: notBefore, NotAfter: notAfter, KeyUsage: x509.KeyUsageDigitalSignature,
	}, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func testSAMLConnection(t *testing.T) domain.IdentityProviderConnection {
	t.Helper()
	connection, _, _ := testSAMLConnectionWithKey(t)
	return connection
}

func testSAMLConnectionWithKey(t *testing.T) (domain.IdentityProviderConnection, *rsa.PrivateKey, *x509.Certificate) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	der, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "SAML IdP"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature,
	}, &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "SAML IdP"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature,
	}, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	return domain.IdentityProviderConnection{
		ID: "saml", TenantID: "tenant", DisplayName: "SAML",
		Protocol: domain.ProtocolSAML, Status: domain.ConnectionActive,
		Issuer: "https://idp.example", SAMLSSOURL: "https://idp.example/sso",
		SAMLEntityID: "https://idp.example", SAMLSigningCertificates: []string{certPEM},
		ClaimMapping: domain.ClaimMapping{
			Subject: "NameID", Username: "email", Email: "email",
			EmailVerified: "email_verified", Name: "name",
		},
		LinkingPolicy: domain.LinkingNone,
	}, key, cert
}

func signedResponse(t *testing.T, key *rsa.PrivateKey, cert *x509.Certificate, now time.Time) []byte {
	t.Helper()
	response := etree.NewElement("samlp:Response")
	response.CreateAttr("xmlns:samlp", "urn:oasis:names:tc:SAML:2.0:protocol")
	response.CreateAttr("xmlns:saml", "urn:oasis:names:tc:SAML:2.0:assertion")
	response.CreateAttr("ID", "_response-1")
	response.CreateAttr("Version", "2.0")
	response.CreateAttr("IssueInstant", now.Format(time.RFC3339))
	response.CreateAttr("Destination", "https://broker.example/callback")
	response.CreateAttr("InResponseTo", "_request-1")
	response.CreateElement("saml:Issuer").SetText("https://idp.example")
	status := response.CreateElement("samlp:Status")
	status.CreateElement("samlp:StatusCode").CreateAttr("Value", "urn:oasis:names:tc:SAML:2.0:status:Success")
	assertion := response.CreateElement("saml:Assertion")
	assertion.CreateAttr("ID", "_assertion-1")
	assertion.CreateAttr("Version", "2.0")
	assertion.CreateAttr("IssueInstant", now.Format(time.RFC3339))
	assertion.CreateElement("saml:Issuer").SetText("https://idp.example")
	subject := assertion.CreateElement("saml:Subject")
	subject.CreateElement("saml:NameID").SetText("external-1")
	confirmation := subject.CreateElement("saml:SubjectConfirmation")
	confirmation.CreateAttr("Method", "urn:oasis:names:tc:SAML:2.0:cm:bearer")
	data := confirmation.CreateElement("saml:SubjectConfirmationData")
	data.CreateAttr("Recipient", "https://broker.example/callback")
	data.CreateAttr("InResponseTo", "_request-1")
	data.CreateAttr("NotOnOrAfter", now.Add(5*time.Minute).Format(time.RFC3339))
	conditions := assertion.CreateElement("saml:Conditions")
	conditions.CreateAttr("NotBefore", now.Add(-time.Minute).Format(time.RFC3339))
	conditions.CreateAttr("NotOnOrAfter", now.Add(5*time.Minute).Format(time.RFC3339))
	restriction := conditions.CreateElement("saml:AudienceRestriction")
	restriction.CreateElement("saml:Audience").SetText("https://broker.example/callback")
	statement := assertion.CreateElement("saml:AttributeStatement")
	for name, value := range map[string]string{
		"email": "user@example.com", "email_verified": "true", "name": "User",
	} {
		attribute := statement.CreateElement("saml:Attribute")
		attribute.CreateAttr("Name", name)
		attribute.CreateElement("saml:AttributeValue").SetText(value)
	}
	signingContext, err := dsig.NewSigningContext(key, [][]byte{cert.Raw})
	if err != nil {
		t.Fatal(err)
	}
	signingContext.IdAttribute = "ID"
	signed, err := signingContext.SignEnveloped(response)
	if err != nil {
		t.Fatal(err)
	}
	document := etree.NewDocument()
	document.SetRoot(signed)
	out, err := document.WriteToBytes()
	if err != nil {
		t.Fatal(err)
	}
	return out
}
