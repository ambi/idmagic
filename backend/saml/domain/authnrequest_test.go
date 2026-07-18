package domain_test

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
)

const sampleAuthnRequest = `<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ` +
	`xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_req-1" Version="2.0" ` +
	`IssueInstant="2026-07-18T12:00:00Z" ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" ` +
	`Destination="https://idp.example.com/saml/sso" ` +
	`AssertionConsumerServiceURL="https://sp.example.com/acs" ForceAuthn="true" IsPassive="true">` +
	`<saml:Issuer>https://sp.example.com</saml:Issuer>` +
	`<samlp:NameIDPolicy Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"/>` +
	`</samlp:AuthnRequest>`

func sampleServiceProvider() samldomain.SamlServiceProvider {
	return samldomain.SamlServiceProvider{
		TenantID: "acme",
		EntityID: "https://sp.example.com",
		ACSURLs:  []string{"https://sp.example.com/acs", "https://sp.example.com/acs2"},
		ClaimPolicy: claimdomain.ClaimMappingPolicy{
			NameID: claimdomain.NameIdConfiguration{Format: samldomain.SamlNameIDFormatPersistent},
		},
	}
}

func TestEncodeDecodeRedirectRoundTrip(t *testing.T) {
	encoded, err := samldomain.EncodeRedirect([]byte(sampleAuthnRequest))
	if err != nil {
		t.Fatalf("encode redirect: %v", err)
	}
	decoded, err := samldomain.DecodeRedirect(encoded)
	if err != nil {
		t.Fatalf("decode redirect: %v", err)
	}
	if string(decoded) != sampleAuthnRequest {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", decoded, sampleAuthnRequest)
	}
}

func TestDecodeRedirectRejectsInvalidBase64(t *testing.T) {
	if _, err := samldomain.DecodeRedirect("not base64!!!"); err == nil {
		t.Fatal("expected base64 decode error")
	}
}

func TestParseAuthnRequestExtractsFields(t *testing.T) {
	req, err := samldomain.ParseAuthnRequest([]byte(sampleAuthnRequest))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if req.ID != "_req-1" {
		t.Errorf("ID=%q", req.ID)
	}
	if req.Issuer != "https://sp.example.com" {
		t.Errorf("Issuer=%q", req.Issuer)
	}
	if req.Version != "2.0" {
		t.Errorf("Version=%q", req.Version)
	}
	if got, want := req.IssueInstant, time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("IssueInstant=%s, want %s", got, want)
	}
	if req.ACSURL != "https://sp.example.com/acs" {
		t.Errorf("ACSURL=%q", req.ACSURL)
	}
	if req.Destination != "https://idp.example.com/saml/sso" {
		t.Errorf("Destination=%q", req.Destination)
	}
	if req.NameIDFormat != samldomain.SamlNameIDFormatEmailAddress {
		t.Errorf("NameIDFormat=%q", req.NameIDFormat)
	}
	if !req.ForceAuthn {
		t.Error("ForceAuthn=false, want true")
	}
	if !req.IsPassive {
		t.Error("IsPassive=false, want true")
	}
	if req.ProtocolBinding != samldomain.SamlBindingHTTPPOST {
		t.Errorf("ProtocolBinding=%q", req.ProtocolBinding)
	}
}

func TestParseAuthnRequestRejectsMissingVersionOrIssueInstant(t *testing.T) {
	for _, tc := range []struct {
		name string
		xml  string
	}{
		{
			name: "version",
			xml:  strings.Replace(sampleAuthnRequest, ` Version="2.0"`, "", 1),
		},
		{
			name: "issue instant",
			xml:  strings.Replace(sampleAuthnRequest, ` IssueInstant="2026-07-18T12:00:00Z"`, "", 1),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := samldomain.ParseAuthnRequest([]byte(tc.xml)); err == nil {
				t.Fatal("expected request to be rejected")
			}
		})
	}
}

func TestValidateSignInAtRejectsUnsupportedAuthnRequestSemantics(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		req  samldomain.AuthnRequest
	}{
		{name: "wrong version", req: samldomain.AuthnRequest{ID: "_x", Issuer: "https://sp.example.com", Version: "1.1", IssueInstant: now}},
		{name: "too old", req: samldomain.AuthnRequest{ID: "_x", Issuer: "https://sp.example.com", Version: "2.0", IssueInstant: now.Add(-11 * time.Minute)}},
		{name: "too far in future", req: samldomain.AuthnRequest{ID: "_x", Issuer: "https://sp.example.com", Version: "2.0", IssueInstant: now.Add(31 * time.Second)}},
		{name: "unsupported response binding", req: samldomain.AuthnRequest{ID: "_x", Issuer: "https://sp.example.com", Version: "2.0", IssueInstant: now, ProtocolBinding: samldomain.SamlBindingHTTPRedirect}},
		{name: "acs url and index", req: samldomain.AuthnRequest{ID: "_x", Issuer: "https://sp.example.com", Version: "2.0", IssueInstant: now, ACSURL: "https://sp.example.com/acs", ACSIndex: 0, ACSIndexSpecified: true}},
		{name: "acs index unsupported", req: samldomain.AuthnRequest{ID: "_x", Issuer: "https://sp.example.com", Version: "2.0", IssueInstant: now, ACSIndex: 0, ACSIndexSpecified: true}},
		{name: "unsupported name id", req: samldomain.AuthnRequest{ID: "_x", Issuer: "https://sp.example.com", Version: "2.0", IssueInstant: now, NameIDFormat: "urn:example:unsupported"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := samldomain.ValidateSignInAt(tc.req, sampleServiceProvider(), "https://idp.example.com/saml/sso", now); err == nil {
				t.Fatal("expected unsupported request semantics to be rejected")
			}
		})
	}
}

func TestValidateSignInAtAcceptsSupportedSemantics(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	req := samldomain.AuthnRequest{
		ID: "_x", Issuer: "https://sp.example.com", Version: "2.0", IssueInstant: now.Add(-time.Minute),
		ProtocolBinding: samldomain.SamlBindingHTTPPOST, NameIDFormat: samldomain.SamlNameIDFormatEmailAddress,
	}
	if _, err := samldomain.ValidateSignInAt(req, sampleServiceProvider(), "https://idp.example.com/saml/sso", now); err != nil {
		t.Fatalf("validate supported semantics: %v", err)
	}
}

func TestParseAuthnRequestRejectsNonAuthnRequestRoot(t *testing.T) {
	xml := `<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"/>`
	if _, err := samldomain.ParseAuthnRequest([]byte(xml)); err == nil {
		t.Fatal("expected non-AuthnRequest root to be rejected")
	}
}

func TestParseAuthnRequestRejectsMissingIssuer(t *testing.T) {
	xml := `<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ID="_x" Version="2.0"/>`
	if _, err := samldomain.ParseAuthnRequest([]byte(xml)); err == nil {
		t.Fatal("expected missing Issuer to be rejected")
	}
}

func TestParseAuthnRequestRejectsMissingID(t *testing.T) {
	xml := `<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ` +
		`xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" Version="2.0">` +
		`<saml:Issuer>https://sp.example.com</saml:Issuer></samlp:AuthnRequest>`
	if _, err := samldomain.ParseAuthnRequest([]byte(xml)); err == nil {
		t.Fatal("expected missing ID to be rejected")
	}
}

func TestValidateSignInResolvesRequestedACS(t *testing.T) {
	req, err := samldomain.ParseAuthnRequest([]byte(sampleAuthnRequest))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := samldomain.ValidateSignInAt(req, sampleServiceProvider(), "https://idp.example.com/saml/sso", time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if out.ACSURL != "https://sp.example.com/acs" {
		t.Errorf("ACSURL=%q", out.ACSURL)
	}
	if out.InResponseTo != "_req-1" {
		t.Errorf("InResponseTo=%q", out.InResponseTo)
	}
	// 要求の NameIDPolicy が SP 既定より優先される。
	if out.NameIDFormat != samldomain.SamlNameIDFormatEmailAddress {
		t.Errorf("NameIDFormat=%q", out.NameIDFormat)
	}
}

func TestValidateSignInRejectsIssuerMismatch(t *testing.T) {
	req := samldomain.AuthnRequest{ID: "_x", Issuer: "https://evil.example.com"}
	if _, err := samldomain.ValidateSignIn(req, sampleServiceProvider(), "https://idp.example.com/saml/sso"); err == nil {
		t.Fatal("expected issuer mismatch to be rejected")
	}
}

func TestValidateSignInRejectsUnregisteredACS(t *testing.T) {
	req := samldomain.AuthnRequest{
		ID:     "_x",
		Issuer: "https://sp.example.com",
		ACSURL: "https://evil.example.com/acs",
	}
	if _, err := samldomain.ValidateSignIn(req, sampleServiceProvider(), "https://idp.example.com/saml/sso"); err == nil {
		t.Fatal("expected unregistered ACS URL to be rejected (open redirect)")
	}
}

func TestValidateSignInRejectsDestinationMismatch(t *testing.T) {
	req := samldomain.AuthnRequest{
		ID:          "_x",
		Issuer:      "https://sp.example.com",
		Destination: "https://other-idp.example.com/saml/sso",
	}
	if _, err := samldomain.ValidateSignIn(req, sampleServiceProvider(), "https://idp.example.com/saml/sso"); err == nil {
		t.Fatal("expected mismatched Destination to be rejected")
	}
}

func TestValidateRequestSignatureRequiresCertificate(t *testing.T) {
	sp := sampleServiceProvider()
	sp.WantAuthnRequestsSigned = true
	if err := samldomain.ValidateRequestSignature(samldomain.BindingRedirect, []byte(sampleAuthnRequest), "SAMLRequest=x", sp); err == nil {
		t.Fatal("expected missing signing certificate to be rejected")
	}
}

func TestValidateSignInFallsBackToDefaultACS(t *testing.T) {
	req := samldomain.AuthnRequest{ID: "_x", Issuer: "https://sp.example.com", Version: "2.0", IssueInstant: time.Now().UTC()}
	out, err := samldomain.ValidateSignIn(req, sampleServiceProvider(), "https://idp.example.com/saml/sso")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if out.ACSURL != "https://sp.example.com/acs" {
		t.Errorf("ACSURL=%q, want default", out.ACSURL)
	}
	// 要求が NameID format を持たないので SP 既定が用いられる。
	if out.NameIDFormat != samldomain.SamlNameIDFormatPersistent {
		t.Errorf("NameIDFormat=%q, want SP default", out.NameIDFormat)
	}
}

func TestValidateSignInUnspecifiedFormatUsesSPDefault(t *testing.T) {
	req := samldomain.AuthnRequest{
		ID:           "_x",
		Issuer:       "https://sp.example.com",
		Version:      "2.0",
		IssueInstant: time.Now().UTC(),
		NameIDFormat: samldomain.SamlNameIDFormatUnspecified,
	}
	out, err := samldomain.ValidateSignIn(req, sampleServiceProvider(), "https://idp.example.com/saml/sso")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if out.NameIDFormat != samldomain.SamlNameIDFormatPersistent {
		t.Errorf("NameIDFormat=%q, want SP default", out.NameIDFormat)
	}
}

func TestRequiresFreshAuth(t *testing.T) {
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	if samldomain.RequiresFreshAuth(false, now.Add(-time.Hour), now) {
		t.Fatal("ForceAuthn=false should not require fresh authentication")
	}
	if samldomain.RequiresFreshAuth(true, now.Add(-10*time.Second), now) {
		t.Fatal("recent authentication should satisfy ForceAuthn")
	}
	if !samldomain.RequiresFreshAuth(true, now.Add(-time.Minute), now) {
		t.Fatal("stale authentication should require fresh authentication")
	}
}

func TestDecodePostRejectsOversizedRequest(t *testing.T) {
	// base64 で 256KiB 超の生 XML を作って上限を超えさせる。
	oversized := strings.Repeat("A", 256*1024+1)
	encoded := base64.StdEncoding.EncodeToString([]byte(oversized))
	if _, err := samldomain.DecodePost(encoded); err == nil {
		t.Fatal("expected oversized POST request to be rejected")
	}
}
