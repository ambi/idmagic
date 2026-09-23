package requests_wstrust_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	claimusecases "github.com/ambi/idmagic/backend/claimmapping/usecases"
	wstrust "github.com/ambi/idmagic/backend/wsfederation/requests_wstrust"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"
)

// RSTR は署名済みの assertion を包んで運ぶだけであり、中身へ手を入れてはならない。
// 包む側が assertion の内部に空白を足すと、正規化した形が変わって署名が壊れる。そうなると
// RP は RSTR を受け取っても assertion を検証できない。外形だけを読むテストは、この破損を
// 見分けられないので、RP と同じ手順で RSTR から assertion を取り出して署名まで検証する。
func TestBuildRSTR_CarriesTheSignedAssertionVerifiably(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "wstrust signing"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := samltoken.NewSigner(certificate, key)
	if err != nil {
		t.Fatal(err)
	}

	for _, version := range []samltoken.SAMLVersion{samltoken.SAML11, samltoken.SAML20} {
		now := time.Now().UTC()
		signed, _, err := samltoken.BuildSignedAssertion(samltoken.AssertionInput{
			Version: version, Issuer: "https://idp.example/realms/default", Audience: "urn:rp",
			Recipient: "urn:rp", IssueInstant: now, NotBefore: now.Add(-time.Minute),
			NotOnOrAfter: now.Add(5 * time.Minute), AuthnInstant: now,
			Result: claimusecases.ClaimIssuanceResult{
				NameIDFormat: "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent", NameIDValue: "user-1",
				Claims: []claimdomain.IssuedClaim{{ClaimType: "http://schemas.xmlsoap.org/claims/UPN", Values: []string{"alice"}}},
			},
		}, signer)
		if err != nil {
			t.Fatalf("sign assertion: %v", err)
		}
		out, err := wstrust.BuildRSTR(signed, "urn:uuid:m1", "urn:rp", "", now, now.Add(5*time.Minute))
		if err != nil {
			t.Fatalf("BuildRSTR: %v", err)
		}

		document := etree.NewDocument()
		if err := document.ReadFromBytes(out); err != nil {
			t.Fatalf("parse RSTR: %v", err)
		}
		assertion := document.FindElement("//t:RequestedSecurityToken/Assertion")
		if assertion == nil {
			t.Fatalf("RSTR does not carry the assertion: %s", out)
		}
		validation := dsig.NewDefaultValidationContext(&dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{certificate}})
		validation.IdAttribute = "ID"
		if assertion.SelectAttrValue("AssertionID", "") != "" {
			validation.IdAttribute = "AssertionID"
		}
		if _, err := validation.Validate(assertion); err != nil {
			t.Fatalf("SAML version %v: the assertion carried in the RSTR does not validate: %v", version, err)
		}
	}
}
