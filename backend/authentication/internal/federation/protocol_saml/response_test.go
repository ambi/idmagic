package protocol_saml

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
)

func TestBuildAuthnRequestBindsDestinationACSAndRequestID(t *testing.T) {
	connection := testSAMLConnection(t)
	attempt := federationdomain.FederatedLoginAttempt{State: "relay", RequestID: "_request-1"}
	redirect, err := BuildAuthnRequest(connection, attempt, "https://broker.example/callback", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if redirect == "" || redirect[:len(connection.SAMLSSOURL)] != connection.SAMLSSOURL {
		t.Fatalf("redirect=%q", redirect)
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

func testSAMLConnection(t *testing.T) federationdomain.IdentityProviderConnection {
	t.Helper()
	now := time.Now().UTC()
	return federationdomain.IdentityProviderConnection{
		ID: "saml", TenantID: "tenant", DisplayName: "SAML",
		Protocol: federationdomain.ProtocolSAML, Status: federationdomain.ConnectionActive,
		Issuer: "https://idp.example", SAMLSSOURL: "https://idp.example/sso",
		SAMLEntityID:            "https://idp.example",
		SAMLSigningCertificates: []string{pemCertificate(t, now.Add(-time.Hour), now.Add(time.Hour))},
		ClaimMapping:            federationdomain.ClaimMapping{Subject: "NameID", Username: "email"},
		LinkingPolicy:           federationdomain.LinkingNone,
	}
}
