package tokens_saml

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
)

// selfSignedCert は署名検証のためのテスト用 X.509 証明書と鍵を生成する。
func selfSignedCert(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "idmagic federation signing"},
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
	return cert, key
}

func validateSignature(signedXML []byte, cert *x509.Certificate) error {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(signedXML); err != nil {
		return err
	}
	store := &dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{cert}}
	vctx := dsig.NewDefaultValidationContext(store)
	vctx.IdAttribute = idAttribute
	_, err := vctx.Validate(doc.Root())
	return err
}

func sampleInput() AssertionInput {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	return AssertionInput{
		Issuer:       "https://idp.example.com/realms/contoso",
		Audience:     "urn:federation:MicrosoftOnline",
		Recipient:    "https://login.microsoftonline.com/login.srf",
		IssueInstant: now,
		NotBefore:    now.Add(-1 * time.Minute),
		NotOnOrAfter: now.Add(5 * time.Minute),
		AuthnInstant: now,
		NameIDFormat: "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",
		NameIDValue:  "AAECAwQFBgc=",
		Attributes: []Attribute{
			{Name: "http://schemas.xmlsoap.org/claims/UPN", Values: []string{"alice@contoso.com"}},
			{Name: "http://schemas.xmlsoap.org/claims/Group", Values: []string{"admins", "users"}},
		},
	}
}

//spec:covers REQ-WSFEDERATION-002: Kerberos の認証事実を SAML 1.1 と 2.0 の署名済み Assertion に保持する。
func TestKerberosAuthenticationContextInBothSignedVersions(t *testing.T) {
	cert, key := selfSignedCert(t)
	signer, err := NewSigner(cert, key)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		version                                SAMLVersion
		idAttribute, element, attribute, value string
	}{
		{SAML11, idAttribute11, "//AuthenticationStatement", "AuthenticationMethod", "urn:ietf:rfc:1510"},
		{SAML20, idAttribute, "//AuthnContextClassRef", "", "urn:oasis:names:tc:SAML:2.0:ac:classes:Kerberos"},
	} {
		input := sampleInput()
		input.Version, input.AuthnContext = tc.version, AuthnContextKerberos
		signed, _, err := IssueSignedAssertion(input, signer)
		if err != nil {
			t.Fatal(err)
		}
		doc := etree.NewDocument()
		if err := doc.ReadFromBytes(signed); err != nil {
			t.Fatal(err)
		}
		statement := doc.FindElement(tc.element)
		if statement == nil {
			t.Fatalf("authentication method missing: %s", tc.element)
		}
		method := statement.Text()
		if tc.attribute != "" {
			method = statement.SelectAttrValue(tc.attribute, "")
		}
		if method != tc.value {
			t.Fatalf("method=%q want=%q", method, tc.value)
		}
		validation := dsig.NewDefaultValidationContext(&dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{cert}})
		validation.IdAttribute = tc.idAttribute
		if _, err := validation.Validate(doc.Root()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestIssueSignedAssertion_RoundTrip(t *testing.T) {
	cert, key := selfSignedCert(t)
	signer, err := NewSigner(cert, key)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}

	signedXML, id, err := IssueSignedAssertion(sampleInput(), signer)
	if err != nil {
		t.Fatalf("issue signed assertion: %v", err)
	}
	if id == "" || id[0] != '_' {
		t.Fatalf("assertion id %q must be an NCName starting with '_'", id)
	}

	if err := validateSignature(signedXML, cert); err != nil {
		t.Fatalf("signature did not validate: %v", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(signedXML); err != nil {
		t.Fatalf("parse signed xml: %v", err)
	}
	if got := doc.FindElement("//NameID"); got == nil || got.Text() != "AAECAwQFBgc=" {
		t.Fatalf("NameID not found or wrong value: %+v", got)
	}
	if got := doc.FindElement("//Signature"); got == nil {
		t.Fatal("enveloped signature missing")
	}
	upn := false
	for _, attr := range doc.FindElements("//Attribute") {
		if attr.SelectAttrValue("Name", "") == "http://schemas.xmlsoap.org/claims/UPN" {
			if v := attr.FindElement("AttributeValue"); v != nil && v.Text() == "alice@contoso.com" {
				upn = true
			}
		}
	}
	if !upn {
		t.Fatal("UPN attribute value not found in assertion")
	}
}

func TestSignedAssertion_TamperDetected(t *testing.T) {
	cert, key := selfSignedCert(t)
	signer, err := NewSigner(cert, key)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	signedXML, _, err := IssueSignedAssertion(sampleInput(), signer)
	if err != nil {
		t.Fatalf("issue signed assertion: %v", err)
	}

	// 署名後に NameID を改竄すると検証が失敗しなければならない。
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(signedXML); err != nil {
		t.Fatalf("parse signed xml: %v", err)
	}
	doc.FindElement("//NameID").SetText("attacker@evil.example")
	tampered, err := doc.WriteToBytes()
	if err != nil {
		t.Fatalf("serialize tampered xml: %v", err)
	}
	if err := validateSignature(tampered, cert); err == nil {
		t.Fatal("tampered assertion validated; expected signature failure")
	}
}

func TestIssueSignedAssertion_SAML11RoundTrip(t *testing.T) {
	cert, key := selfSignedCert(t)
	signer, err := NewSigner(cert, key)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}

	in := sampleInput()
	in.Version = SAML11
	in.AuthnContext = AuthnContextPassword
	signedXML, id, err := IssueSignedAssertion(in, signer)
	if err != nil {
		t.Fatalf("issue signed assertion: %v", err)
	}
	if id == "" || id[0] != '_' {
		t.Fatalf("assertion id %q must be an NCName starting with '_'", id)
	}

	// SAML 1.1 は AssertionID を ID 参照属性とする。
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(signedXML); err != nil {
		t.Fatalf("parse signed xml: %v", err)
	}
	store := &dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{cert}}
	vctx := dsig.NewDefaultValidationContext(store)
	vctx.IdAttribute = idAttribute11
	if _, err := vctx.Validate(doc.Root()); err != nil {
		t.Fatalf("SAML 1.1 signature did not validate: %v", err)
	}

	root := doc.Root()
	if root.SelectAttrValue("MajorVersion", "") != "1" || root.SelectAttrValue("MinorVersion", "") != "1" {
		t.Fatalf("expected MajorVersion/MinorVersion 1/1, got %s/%s",
			root.SelectAttrValue("MajorVersion", ""), root.SelectAttrValue("MinorVersion", ""))
	}
	if root.SelectAttrValue("Issuer", "") != in.Issuer {
		t.Fatalf("Issuer attribute = %q, want %q", root.SelectAttrValue("Issuer", ""), in.Issuer)
	}
	if got := doc.FindElement("//NameIdentifier"); got == nil || got.Text() != "AAECAwQFBgc=" {
		t.Fatalf("SAML 1.1 NameIdentifier missing or wrong: %+v", got)
	}
	if got := doc.FindElement("//AuthenticationStatement"); got == nil ||
		got.SelectAttrValue("AuthenticationMethod", "") != authnMethodPassword {
		t.Fatalf("AuthenticationMethod missing or not password: %+v", got)
	}
	// claim 型 URI は AD FS 流に namespace/name へ分割される。
	upn := false
	for _, attr := range doc.FindElements("//Attribute") {
		if attr.SelectAttrValue("AttributeName", "") == "UPN" &&
			attr.SelectAttrValue("AttributeNamespace", "") == "http://schemas.xmlsoap.org/claims" {
			if v := attr.FindElement("AttributeValue"); v != nil && v.Text() == "alice@contoso.com" {
				upn = true
			}
		}
	}
	if !upn {
		t.Fatal("UPN attribute (split namespace/name) not found in SAML 1.1 assertion")
	}
}

// HTTP 境界の観測 (handlers_http) は RP の既定 token type である SAML 1.1 しか通らないので、SAML 2.0 の
// 枝だけを保持者証明へ変えても気づけない。両方の版で、Subject を持つすべての位置の確認方法が Bearer で
// あることを見る。
//
//spec:covers WSTRUST13-ISSUE-BEARER: 発行する assertion が版によらず Bearer で確認されることを固定する。
func TestBuildAssertion_ConfirmsTheSubjectAsBearerInBothVersions(t *testing.T) {
	versions := map[SAMLVersion]struct {
		path   string
		method string
	}{
		SAML20: {path: ".//SubjectConfirmation", method: bearerConfirmation20},
		SAML11: {path: ".//SubjectConfirmation/ConfirmationMethod", method: bearerConfirmation11},
	}
	for version, want := range versions {
		in := sampleInput()
		in.Version = version
		assertion, _, err := BuildAssertion(in)
		if err != nil {
			t.Fatalf("build assertion: %v", err)
		}
		found := assertion.FindElements(want.path)
		if len(found) == 0 {
			doc := etree.NewDocument()
			doc.SetRoot(assertion.Copy())
			xml, _ := doc.WriteToString()
			t.Fatalf("the assertion states no subject confirmation: %s", xml)
		}
		for _, element := range found {
			// SAML 2.0 は属性、SAML 1.1 は子要素の本文で確認方法を述べる。
			got := element.SelectAttrValue("Method", element.Text())
			if got != want.method {
				t.Fatalf("subject confirmation = %q, want %q", got, want.method)
			}
		}
	}
}

// Entra は SAML 1.1 では AuthenticationMethod、SAML 2.0 では AuthnContextClassRef の multipleauthn で
// IdP の MFA を受け入れる。HTTP 境界は Entra の RP の SAML 1.1 しか通らないので、両方の版をここで見る。
//
//spec:covers ENTRA-MFA-AUTHENTICATION-METHOD: 第二要素を含む認証を、SAML 1.1 の AuthenticationMethod と SAML 2.0 の AuthnContextClassRef の multipleauthn にすること。
func TestBuildAssertion_StatesMultipleAuthnInBothVersions(t *testing.T) {
	const multipleAuthn = "http://schemas.microsoft.com/claims/multipleauthn"
	for version, read := range map[SAMLVersion]func(*etree.Element) string{
		SAML20: func(a *etree.Element) string {
			if ref := a.FindElement("./AuthnStatement/AuthnContext/AuthnContextClassRef"); ref != nil {
				return ref.Text()
			}
			return ""
		},
		SAML11: func(a *etree.Element) string {
			if statement := a.FindElement("./AuthenticationStatement"); statement != nil {
				return statement.SelectAttrValue("AuthenticationMethod", "")
			}
			return ""
		},
	} {
		in := sampleInput()
		in.Version = version
		in.AuthnContext = AuthnContextMultipleAuthn
		assertion, _, err := BuildAssertion(in)
		if err != nil {
			t.Fatalf("build assertion: %v", err)
		}
		if got := read(assertion); got != multipleAuthn {
			t.Fatalf("version %v authentication method = %q, want %q", version, got, multipleAuthn)
		}
	}
}

func TestBuildAssertion_InputValidation(t *testing.T) {
	base := sampleInput()
	bad := map[string]func(*AssertionInput){
		"missing issuer":      func(in *AssertionInput) { in.Issuer = "" },
		"missing audience":    func(in *AssertionInput) { in.Audience = "" },
		"missing nameid fmt":  func(in *AssertionInput) { in.NameIDFormat = "" },
		"missing nameid val":  func(in *AssertionInput) { in.NameIDValue = "" },
		"invalid time window": func(in *AssertionInput) { in.NotOnOrAfter = in.NotBefore },
	}
	for name, mutate := range bad {
		t.Run(name, func(t *testing.T) {
			in := base
			in.Attributes = append([]Attribute(nil), base.Attributes...)
			mutate(&in)
			if _, _, err := BuildAssertion(in); err == nil {
				t.Fatalf("%s: expected error, got nil", name)
			}
		})
	}
}
