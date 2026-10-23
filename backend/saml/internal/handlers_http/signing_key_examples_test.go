package handlers_http_test

// SigningKeys が宣言する XML フェデレーション署名資格情報の具体例を、SAML の SSO と
// メタデータ、WS-Federation のメタデータ、署名鍵の管理 API から観測する。
//
// 鍵の用途とテナントの分離は、発行された Assertion を SP と同じ手順で検証して読む。
// 鍵の選び方を実装の中から読むのではなく、どの証明書で検証が通り、どれで通らないかで
// 「どの鍵で署名されたか」を決める。

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/signingkeys"

	"github.com/ambi/idmagic/backend/authentication"

	"github.com/beevik/etree"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const exampleCSRFToken = "csrf-token-for-signing-key-examples"

// newFederationStack は SAML と WS-Federation を配線し、default テナントに SP を 1 つ置く。
func newFederationStack(t *testing.T) (*stack.Stack, string) {
	t.Helper()
	s := stack.New(t, stack.WithAuthorizationCodeFlow(), stack.WithSaml(), stack.WithWsFederation())
	if err := s.SamlSPs.Save(t.Context(), &samldomain.SamlServiceProvider{
		TenantID:      tenancydomain.DefaultTenantID,
		EntityID:      "https://sp.example.com",
		ACSURLs:       []string{"https://sp.example.com/acs"},
		SignAssertion: true,
		ClaimPolicy: claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{
			Format: samldomain.SamlNameIDFormatPersistent, SourceAttribute: "user_id",
		}},
	}); err != nil {
		t.Fatal(err)
	}
	return s, newSession(t, s, stack.UserID)
}

func newSession(t *testing.T, s *stack.Stack, sub string) string {
	t.Helper()
	authn, err := s.Sessions.Create(s.RealmContext(t, tenancydomain.DefaultRealm), sub, []string{"pwd"}, time.Now().UTC())
	if err != nil || authn == nil {
		t.Fatalf("create session for %s: %v", sub, err)
	}
	return authn.SessionID
}

func serve(t *testing.T, s *stack.Stack, request *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, request)
	return recorder
}

func getFrom(t *testing.T, s *stack.Stack, path string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := serve(t, s, httptest.NewRequestWithContext(t.Context(), http.MethodGet, stack.Issuer+path, http.NoBody))
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
	}
	return recorder
}

// issuedAssertion はセッションを持つ利用者として SP 起点の SSO を 1 回通し、署名済み Assertion を返す。
func issuedAssertion(t *testing.T, s *stack.Stack, sessionID string) *etree.Element {
	t.Helper()
	samlRequest := authnRequestRedirect(t, "https://sp.example.com", "https://sp.example.com/acs")
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		stack.Issuer+"/realms/default/saml/sso?SAMLRequest="+url.QueryEscape(samlRequest), http.NoBody)
	request.AddCookie(&http.Cookie{Name: authentication.SessionCookie, Value: sessionID})
	recorder := serve(t, s, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("sso status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	document := etree.NewDocument()
	if err := document.ReadFromBytes(samlResponseFromPostForm(t, recorder.Body.String())); err != nil {
		t.Fatalf("parse SAMLResponse: %v", err)
	}
	assertion := document.FindElement("//Assertion")
	if assertion == nil || assertion.FindElement("./Signature") == nil {
		t.Fatalf("signed assertion missing: %s", recorder.Body.String())
	}
	return assertion
}

// activeXMLCertificate は、テナントのデフォルトスコープで現在有効な XmlFederationSigning 鍵の証明書を返す。
func activeXMLCertificate(t *testing.T, s *stack.Stack) *x509.Certificate {
	t.Helper()
	signer, err := signingkeys.Module{KeyStore: s.KeyStore}.FederationSigner().Resolve(s.RealmContext(t, tenancydomain.DefaultRealm))
	if err != nil {
		t.Fatalf("resolve federation signer: %v", err)
	}
	return signer.Certificate()
}

func encoded(certificate *x509.Certificate) string {
	return base64.StdEncoding.EncodeToString(certificate.Raw)
}

//spec:covers EX-SIGNINGKEYS-005-01: default テナントの SSO が発行した Assertion は、そのテナントで現在有効な XmlFederationSigning 鍵の証明書で検証でき、acme テナントが公開する XML 証明書でも、default テナントの JWT 署名鍵を包んだ証明書でも検証できない。
func TestSamlAssertionIsSignedOnlyByTheIssuingTenantsXmlFederationKey(t *testing.T) {
	s, sessionID := newFederationStack(t)
	assertion := issuedAssertion(t, s, sessionID)

	if err := validateAgainst(assertion, activeXMLCertificate(t, s)); err != nil {
		t.Fatalf("default テナントの XML 証明書で検証できない: %v", err)
	}

	otherTenant := certificateFromPEMResponse(t, getFrom(t, s, "/realms/"+stack.OtherRealm+"/saml/signing-certificate.pem").Body.Bytes())
	if err := validateAgainst(assertion, otherTenant); err == nil {
		t.Fatal("acme テナントの XML 証明書で default テナントの Assertion が検証できてしまう")
	}

	jwtKey, err := s.KeyStore.GetActiveKey(s.RealmContext(t, tenancydomain.DefaultRealm))
	if err != nil {
		t.Fatal(err)
	}
	jwtSigner, ok := jwtKey.PrivateKey.(crypto.Signer)
	if !ok {
		t.Fatalf("JWT signing key %T cannot sign", jwtKey.PrivateKey)
	}
	// 信頼していない JWT 用の鍵を、製品の証明書生成器に依存せず SP 側の証明書へ包む。
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature}
	der, err := x509.CreateCertificate(rand.Reader, template, template, jwtSigner.Public(), jwtSigner)
	if err != nil {
		t.Fatal(err)
	}
	jwtCertificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateAgainst(assertion, jwtCertificate); err == nil {
		t.Fatal("default テナントの JWT 署名鍵で Assertion が検証できてしまう")
	}
}

//spec:covers EX-SIGNINGKEYS-006-01: 管理者が usage=XmlFederationSigning でローテートすると、次の Assertion は K2 で署名されて K1 では検証できず、猶予期間中の SAML と WS-Federation のメタデータは K1 と K2 の証明書を公開し、K1 の expires_at を過ぎると両メタデータから K1 が消えて K2 が残る。
func TestAdministratorRotatesTheXmlFederationKeyWithoutBreakingTrust(t *testing.T) {
	s, sessionID := newFederationStack(t)
	k1 := activeXMLCertificate(t, s)
	metadataPaths := []string{"/realms/default/saml/metadata", "/realms/default/federationmetadata/2007-06/federationmetadata.xml"}
	for _, path := range metadataPaths {
		if !publishedCertificates(t, getFrom(t, s, path).Body.Bytes())[encoded(k1)] {
			t.Fatalf("%s がローテーション前の K1 を公開していない", path)
		}
	}

	rotate := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, stack.Issuer+"/realms/default/api/admin/v1/keys/rotate", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", stack.Issuer)
		r.Header.Set(support.CSRFHeader, exampleCSRFToken)
		r.AddCookie(&http.Cookie{Name: authentication.SessionCookie, Value: newSession(t, s, stack.AdminUserID)})
		r.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: exampleCSRFToken})
		return serve(t, s, r)
	}
	prepared := rotate(`{"usage":"XmlFederationSigning"}`)
	if prepared.Code != http.StatusOK {
		t.Fatalf("prepare: %d %s", prepared.Code, prepared.Body.String())
	}
	var result struct {
		Next struct {
			Kid string `json:"kid"`
		} `json:"next"`
	}
	if err := json.Unmarshal(prepared.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !activeXMLCertificate(t, s).Equal(k1) {
		t.Fatal("preparation switched the XML signature")
	}
	if r := rotate(`{"usage":"XmlFederationSigning","action":"activate","trust_updated":true,"kid":"` + result.Next.Kid + `"}`); r.Code != http.StatusOK {
		t.Fatalf("activate: %d %s", r.Code, r.Body.String())
	}

	k2 := activeXMLCertificate(t, s)
	if k2.Equal(k1) {
		t.Fatal("ローテーション後も XML フェデレーション鍵が K1 のままである")
	}
	assertion := issuedAssertion(t, s, sessionID)
	if err := validateAgainst(assertion, k2); err != nil {
		t.Fatalf("ローテーション後の Assertion が K2 で検証できない: %v", err)
	}
	if err := validateAgainst(assertion, k1); err == nil {
		t.Fatal("ローテーション後の Assertion が K1 で検証できてしまう")
	}
	for _, path := range metadataPaths {
		published := publishedCertificates(t, getFrom(t, s, path).Body.Bytes())
		if !published[encoded(k1)] || !published[encoded(k2)] {
			t.Fatalf("%s が猶予期間中に K1 と K2 の両方を公開していない", path)
		}
	}

	// 猶予期間の終了は K1 の expires_at を過去へ動かして表す。メモリの鍵ストアは保持する鍵を
	// そのまま返すので、この書き換えは次の読み出しに効く。
	keys, err := s.KeyStore.GetAllKeys(signingports.WithKeyUsage(s.RealmContext(t, tenancydomain.DefaultRealm), signingdomain.KeyUsageXMLFederationSigning))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		if !key.Active {
			elapsed := time.Now().UTC().Add(-time.Second)
			key.ExpiresAt = &elapsed
		}
	}
	for _, path := range metadataPaths {
		published := publishedCertificates(t, getFrom(t, s, path).Body.Bytes())
		if published[encoded(k1)] || !published[encoded(k2)] {
			t.Fatalf("%s が猶予期間の後も K1 を公開しているか、K2 を落としている", path)
		}
	}
}
