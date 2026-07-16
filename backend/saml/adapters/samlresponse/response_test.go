package samlresponse_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	claimusecases "github.com/ambi/idmagic/backend/claimmapping/usecases"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"

	"github.com/ambi/idmagic/backend/saml/adapters/samlresponse"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/wsfederation/adapters/samltoken"
)

func newSigner(t *testing.T) (*samltoken.Signer, *x509.Certificate) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "idmagic saml signing"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(1 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	signer, err := samltoken.NewSigner(cert, key)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	return signer, cert
}

func sampleAssertion(t *testing.T) *etree.Element {
	t.Helper()
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	assertion, _, err := samltoken.BuildAssertion(samltoken.AssertionInput{
		Version:      samltoken.SAML20,
		Issuer:       "https://idp.example.com/saml",
		Audience:     "https://sp.example.com",
		Recipient:    "https://sp.example.com/acs",
		InResponseTo: "_req-1",
		IssueInstant: now,
		NotBefore:    now.Add(-1 * time.Minute),
		NotOnOrAfter: now.Add(5 * time.Minute),
		AuthnInstant: now,
		Result: claimusecases.ClaimIssuanceResult{
			NameIDFormat: samldomain.SamlNameIDFormatPersistent,
			NameIDValue:  "alice",
			Claims: []claimdomain.IssuedClaim{
				{ClaimType: "email", Values: []string{"alice@example.com"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("build assertion: %v", err)
	}
	return assertion
}

func TestBuildResponseWrapsAssertion(t *testing.T) {
	out, err := samlresponse.BuildResponse(samlresponse.ResponseInput{
		Issuer:       "https://idp.example.com/saml",
		Destination:  "https://sp.example.com/acs",
		InResponseTo: "_req-1",
		IssueInstant: time.Now(),
		Assertion:    sampleAssertion(t),
	}, nil)
	if err != nil {
		t.Fatalf("build response: %v", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(out); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	root := doc.Root()
	if root == nil || root.Tag != "Response" {
		t.Fatalf("root=%v", root)
	}
	if got := root.SelectAttrValue("InResponseTo", ""); got != "_req-1" {
		t.Errorf("InResponseTo=%q", got)
	}
	if got := root.SelectAttrValue("Destination", ""); got != "https://sp.example.com/acs" {
		t.Errorf("Destination=%q", got)
	}
	if doc.FindElement("//Status/StatusCode") == nil {
		t.Error("StatusCode missing")
	}
	if doc.FindElement("//Assertion") == nil {
		t.Error("Assertion missing")
	}
}

func TestBuildResponseSignsResponse(t *testing.T) {
	signer, cert := newSigner(t)
	out, err := samlresponse.BuildResponse(samlresponse.ResponseInput{
		Issuer:       "https://idp.example.com/saml",
		Destination:  "https://sp.example.com/acs",
		IssueInstant: time.Now(),
		Assertion:    sampleAssertion(t),
		SignResponse: true,
	}, signer)
	if err != nil {
		t.Fatalf("build response: %v", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(out); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if doc.FindElement("//Signature") == nil {
		t.Fatal("Response signature missing")
	}

	store := &dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{cert}}
	vctx := dsig.NewDefaultValidationContext(store)
	vctx.IdAttribute = "ID"
	if _, err := vctx.Validate(doc.Root()); err != nil {
		t.Fatalf("response signature did not validate: %v", err)
	}
}

func TestBuildResponseRequiresSignerWhenSigning(t *testing.T) {
	_, err := samlresponse.BuildResponse(samlresponse.ResponseInput{
		Issuer:       "https://idp.example.com/saml",
		Destination:  "https://sp.example.com/acs",
		IssueInstant: time.Now(),
		Assertion:    sampleAssertion(t),
		SignResponse: true,
	}, nil)
	if err == nil {
		t.Fatal("expected missing signer to be rejected")
	}
}

func TestBuildResponseValidatesInput(t *testing.T) {
	assertion := sampleAssertion(t)
	cases := []samlresponse.ResponseInput{
		{Destination: "https://sp.example.com/acs", Assertion: assertion},
		{Issuer: "https://idp.example.com/saml", Assertion: assertion},
		{Issuer: "https://idp.example.com/saml", Destination: "https://sp.example.com/acs"},
	}
	for i, in := range cases {
		if _, err := samlresponse.BuildResponse(in, nil); err == nil {
			t.Errorf("case %d: expected validation error", i)
		}
	}
}

func TestEncodePostForm(t *testing.T) {
	responseXML := []byte(`<samlp:Response/>`)
	out, err := samlresponse.EncodePostForm(responseXML, "https://sp.example.com/acs", "state-123")
	if err != nil {
		t.Fatalf("encode post form: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(responseXML)
	if !bytes.Contains(out, []byte(encoded)) {
		t.Error("form does not contain base64 SAMLResponse")
	}
	if !bytes.Contains(out, []byte("state-123")) {
		t.Error("form does not contain RelayState")
	}
	if !bytes.Contains(out, []byte(`action="https://sp.example.com/acs"`)) {
		t.Error("form does not POST to ACS URL")
	}
	// 自動送信は inline event handler ではなく固定の <script> で行う。その内容は
	// CSP hash で許可される (support.AutoSubmitScript, ADR-076)。
	if bytes.Contains(out, []byte("onload=")) {
		t.Error("form must not use an inline onload handler under strict CSP")
	}
	if !bytes.Contains(out, []byte("<script>"+support.AutoSubmitScript+"</script>")) {
		t.Errorf("form does not contain the pinned submit script: %s", out)
	}
}

func TestEncodePostFormRequiresACS(t *testing.T) {
	if _, err := samlresponse.EncodePostForm([]byte("<x/>"), "", ""); err == nil {
		t.Fatal("expected missing ACS URL to be rejected")
	}
}
