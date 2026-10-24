// Package tokens_saml は、SAML の語彙で与えた NameID、属性、認証コンテキストを、署名済み SAML
// assertion という XML ワイヤ形式に変換する共有ライブラリである。WS-Federation と SAML の両方の
// プロトコルが使い、Entra / AD FS の WS-Federation 既定である SAML 1.1 と、SAML 2.0 の双方を
// 組み立てる。モジュールの型は受け取らず、発行結果と認証方式から SAML の語彙への変換は利用側が持つ。
// XML 署名は goxmldsig に委ね、自前実装しない。enveloped signature・
// exclusive C14N・RSA-SHA256 を用いる。SAML 1.1 と 2.0 で ID 参照属性が異なる (AssertionID / ID)。
package tokens_saml

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"
)

const (
	samlAssertion20NS = "urn:oasis:names:tc:SAML:2.0:assertion"
	samlAssertion11NS = "urn:oasis:names:tc:SAML:1.0:assertion"

	bearerConfirmation20 = "urn:oasis:names:tc:SAML:2.0:cm:bearer"
	bearerConfirmation11 = "urn:oasis:names:tc:SAML:1.0:cm:bearer"

	authnContextUnspec20 = "urn:oasis:names:tc:SAML:2.0:ac:classes:unspecified"
	authnContextPassword = "urn:oasis:names:tc:SAML:2.0:ac:classes:Password"
	authnMethodUnspec11  = "urn:oasis:names:tc:SAML:1.0:am:unspecified"
	authnMethodPassword  = "urn:oasis:names:tc:SAML:1.0:am:password"
	// authnMultipleAuthn は Microsoft の MFA の認証方式で、SAML 1.1 と 2.0 の両方で同じ URI を使う。
	authnMultipleAuthn = "http://schemas.microsoft.com/claims/multipleauthn"

	attrNameFormatURI = "urn:oasis:names:tc:SAML:2.0:attrname-format:uri"
	xmlDateTime       = "2006-01-02T15:04:05Z"

	// ID 参照属性。SAML 2.0 は ID、SAML 1.1 は AssertionID。
	idAttribute   = "ID"
	idAttribute11 = "AssertionID"
)

// SAMLVersion は組み立てる assertion の SAML バージョン。ゼロ値は SAML 2.0。
type SAMLVersion int

const (
	// SAML20 は SAML 2.0 assertion。
	SAML20 SAMLVersion = iota
	// SAML11 は SAML 1.1 assertion (Entra / AD FS WS-Fed 既定)。
	SAML11
)

// AssertionInput は 1 つの SAML assertion を組み立てるための入力。
type AssertionInput struct {
	Version      SAMLVersion  // SAML バージョン (既定 2.0)。
	Issuer       string       // IdP の entityID (issuer)。
	Audience     string       // RP の entityID / wtrealm。
	Recipient    string       // 任意。bearer の Recipient (ACS / wreply)。
	InResponseTo string       // 任意。SP-initiated の AuthnRequest ID (SAML 2.0 のみ)。
	IssueInstant time.Time    // 発行時刻。
	NotBefore    time.Time    // 有効期間の開始。
	NotOnOrAfter time.Time    // 有効期間の終了。
	AuthnInstant time.Time    // 認証時刻。
	AuthnContext AuthnContext // 認証コンテキスト (既定は unspecified)。
	NameIDFormat string       // NameID の Format。
	NameIDValue  string       // NameID の値。
	Attributes   []Attribute  // assertion の属性。
}

// AuthnContext は assertion に載せる認証コンテキストである。ゼロ値は unspecified。
type AuthnContext int

const (
	// AuthnContextUnspecified は認証方式を特定しない。
	AuthnContextUnspecified AuthnContext = iota
	// AuthnContextPassword はパスワードによる認証を表す。
	AuthnContextPassword
	// AuthnContextMultipleAuthn は第二要素を含む認証を表す。Entra はこの値で IdP の MFA を受け入れる。
	AuthnContextMultipleAuthn
	// AuthnContextKerberos は実施した Kerberos 認証を表す。
	AuthnContextKerberos
)

// Attribute は assertion の 1 つの属性である。Name は属性の URI、Values は値の並びである。
type Attribute struct {
	Name   string
	Values []string
}

func (in AssertionInput) validate() error {
	switch {
	case strings.TrimSpace(in.Issuer) == "":
		return fmt.Errorf("samltoken: issuer is required")
	case strings.TrimSpace(in.Audience) == "":
		return fmt.Errorf("samltoken: audience is required")
	case strings.TrimSpace(in.NameIDFormat) == "":
		return fmt.Errorf("samltoken: NameID format is required")
	case strings.TrimSpace(in.NameIDValue) == "":
		return fmt.Errorf("samltoken: NameID value is required")
	case in.NotOnOrAfter.Before(in.NotBefore) || in.NotOnOrAfter.Equal(in.NotBefore):
		return fmt.Errorf("samltoken: NotOnOrAfter must be after NotBefore")
	}
	return nil
}

// idAttributeFor は SAML バージョンに対応する署名 ID 参照属性名を返す。
func idAttributeFor(v SAMLVersion) string {
	if v == SAML11 {
		return idAttribute11
	}
	return idAttribute
}

// BuildAssertion は未署名の SAML assertion を etree 要素として組み立て、その ID を返す。
func BuildAssertion(in AssertionInput) (*etree.Element, string, error) {
	if err := in.validate(); err != nil {
		return nil, "", err
	}
	id, err := newAssertionID()
	if err != nil {
		return nil, "", err
	}
	if in.Version == SAML11 {
		return buildSAML11(in, id), id, nil
	}
	return buildSAML20(in, id), id, nil
}

// buildSAML20 は SAML 2.0 assertion を組み立てる。
func buildSAML20(in AssertionInput, id string) *etree.Element {
	a := etree.NewElement("Assertion")
	a.CreateAttr("xmlns", samlAssertion20NS)
	a.CreateAttr("Version", "2.0")
	a.CreateAttr(idAttribute, id)
	a.CreateAttr("IssueInstant", in.IssueInstant.UTC().Format(xmlDateTime))

	a.CreateElement("Issuer").SetText(in.Issuer)

	subject := a.CreateElement("Subject")
	nameID := subject.CreateElement("NameID")
	nameID.CreateAttr("Format", in.NameIDFormat)
	nameID.SetText(in.NameIDValue)
	confirmation := subject.CreateElement("SubjectConfirmation")
	confirmation.CreateAttr("Method", bearerConfirmation20)
	confirmationData := confirmation.CreateElement("SubjectConfirmationData")
	confirmationData.CreateAttr("NotOnOrAfter", in.NotOnOrAfter.UTC().Format(xmlDateTime))
	if r := strings.TrimSpace(in.Recipient); r != "" {
		confirmationData.CreateAttr("Recipient", r)
	}
	if irt := strings.TrimSpace(in.InResponseTo); irt != "" {
		confirmationData.CreateAttr("InResponseTo", irt)
	}

	conditions := a.CreateElement("Conditions")
	conditions.CreateAttr("NotBefore", in.NotBefore.UTC().Format(xmlDateTime))
	conditions.CreateAttr("NotOnOrAfter", in.NotOnOrAfter.UTC().Format(xmlDateTime))
	conditions.CreateElement("AudienceRestriction").CreateElement("Audience").SetText(in.Audience)

	authn := a.CreateElement("AuthnStatement")
	authn.CreateAttr("AuthnInstant", in.AuthnInstant.UTC().Format(xmlDateTime))
	classRef := authnContextUnspec20
	switch in.AuthnContext {
	case AuthnContextPassword:
		classRef = authnContextPassword
	case AuthnContextMultipleAuthn:
		classRef = authnMultipleAuthn
	case AuthnContextKerberos:
		classRef = "urn:oasis:names:tc:SAML:2.0:ac:classes:Kerberos"
	}
	authn.CreateElement("AuthnContext").CreateElement("AuthnContextClassRef").SetText(classRef)

	if len(in.Attributes) > 0 {
		statement := a.CreateElement("AttributeStatement")
		for _, attribute := range in.Attributes {
			attr := statement.CreateElement("Attribute")
			attr.CreateAttr("Name", attribute.Name)
			attr.CreateAttr("NameFormat", attrNameFormatURI)
			for _, value := range attribute.Values {
				attr.CreateElement("AttributeValue").SetText(value)
			}
		}
	}

	return a
}

// buildSAML11 は SAML 1.1 assertion を組み立てる (Entra / AD FS WS-Fed 互換)。
func buildSAML11(in AssertionInput, id string) *etree.Element {
	a := etree.NewElement("Assertion")
	a.CreateAttr("xmlns", samlAssertion11NS)
	a.CreateAttr("MajorVersion", "1")
	a.CreateAttr("MinorVersion", "1")
	a.CreateAttr(idAttribute11, id)
	a.CreateAttr("Issuer", in.Issuer)
	a.CreateAttr("IssueInstant", in.IssueInstant.UTC().Format(xmlDateTime))

	conditions := a.CreateElement("Conditions")
	conditions.CreateAttr("NotBefore", in.NotBefore.UTC().Format(xmlDateTime))
	conditions.CreateAttr("NotOnOrAfter", in.NotOnOrAfter.UTC().Format(xmlDateTime))
	conditions.CreateElement("AudienceRestrictionCondition").CreateElement("Audience").SetText(in.Audience)

	authn := a.CreateElement("AuthenticationStatement")
	method := authnMethodUnspec11
	switch in.AuthnContext {
	case AuthnContextPassword:
		method = authnMethodPassword
	case AuthnContextMultipleAuthn:
		method = authnMultipleAuthn
	case AuthnContextKerberos:
		method = "urn:ietf:rfc:1510"
	}
	authn.CreateAttr("AuthenticationMethod", method)
	authn.CreateAttr("AuthenticationInstant", in.AuthnInstant.UTC().Format(xmlDateTime))
	addSAML11Subject(authn, in)

	if len(in.Attributes) > 0 {
		statement := a.CreateElement("AttributeStatement")
		addSAML11Subject(statement, in)
		for _, attribute := range in.Attributes {
			ns, name := splitAttributeName(attribute.Name)
			attr := statement.CreateElement("Attribute")
			attr.CreateAttr("AttributeName", name)
			attr.CreateAttr("AttributeNamespace", ns)
			for _, value := range attribute.Values {
				attr.CreateElement("AttributeValue").SetText(value)
			}
		}
	}

	return a
}

// addSAML11Subject は SAML 1.1 の Subject (NameIdentifier + bearer 確認) を親要素に追加する。
func addSAML11Subject(parent *etree.Element, in AssertionInput) {
	subject := parent.CreateElement("Subject")
	nameID := subject.CreateElement("NameIdentifier")
	nameID.CreateAttr("Format", in.NameIDFormat)
	nameID.SetText(in.NameIDValue)
	confirmation := subject.CreateElement("SubjectConfirmation")
	confirmation.CreateElement("ConfirmationMethod").SetText(bearerConfirmation11)
}

// splitAttributeName は属性の URI を AD FS 流の (namespace, name) に分割する。
// 最後の '/' で分け、namespace は手前まで、name は後ろ。'/' が無ければ namespace は空。
func splitAttributeName(uri string) (namespace, name string) {
	if before, after, found := strings.CutLast(uri, "/"); found {
		return before, after
	}
	return "", uri
}

// signerScopeKey は、署名器を解決する鍵のスコープを文脈に載せるキーである。
type signerScopeKey struct{}

// WithSignerScope は、署名器を解決する鍵のスコープ（SAML の IdP プロファイルなど）を文脈に載せる。
// SignerProvider の実装がこのスコープの鍵を選ぶ。
func WithSignerScope(ctx context.Context, scopeID string) context.Context {
	return context.WithValue(ctx, signerScopeKey{}, scopeID)
}

// SignerScope は WithSignerScope で載せた鍵のスコープを返す。載っていなければ ok は false である。
func SignerScope(ctx context.Context) (scopeID string, ok bool) {
	scopeID, ok = ctx.Value(signerScopeKey{}).(string)
	return scopeID, ok
}

// SignerProvider は、要求の文脈に合う署名器と、メタデータに載せる署名証明書を解決する。
type SignerProvider interface {
	Resolve(ctx context.Context) (*Signer, error)
	Certificates(ctx context.Context, now time.Time) ([]*x509.Certificate, error)
	PreparedCertificate(ctx context.Context) (*x509.Certificate, error)
}

// Signer は SAML assertion を enveloped 署名する。SAML 1.1/2.0 で ID 参照属性が
// 異なるため、署名時に対象属性名を受け取り、署名コンテキストを都度構築する。
type Signer struct {
	cert *x509.Certificate
	key  crypto.Signer
}

// 静的な署名器はローテーションを提供しない。
func (s *Signer) PreparedCertificate(context.Context) (*x509.Certificate, error) {
	return nil, nil //nolint:nilnil // 準備済みの証明書がないことを表す。
}

// NewSigner は federation 署名証明書 (X.509) と RSA 秘密鍵から署名器を作る。
func NewSigner(cert *x509.Certificate, key crypto.Signer) (*Signer, error) {
	if cert == nil || key == nil {
		return nil, fmt.Errorf("samltoken: signing certificate and key are required")
	}
	return &Signer{cert: cert, key: key}, nil
}

// Resolve lets a static Signer satisfy SignerProvider in focused adapter tests.
func (s *Signer) Resolve(context.Context) (*Signer, error) {
	if s == nil {
		return nil, fmt.Errorf("samltoken: signer is required")
	}
	return s, nil
}

// Certificates lets a static Signer satisfy SignerProvider.
func (s *Signer) Certificates(context.Context, time.Time) ([]*x509.Certificate, error) {
	if s == nil || s.cert == nil {
		return nil, fmt.Errorf("samltoken: signing certificate is required")
	}
	return []*x509.Certificate{s.cert}, nil
}

// Certificate は federation 署名証明書の読み取り専用コピーを返す。
func (s *Signer) Certificate() *x509.Certificate {
	if s == nil || s.cert == nil {
		return nil
	}
	return s.cert
}

func (s *Signer) signingContext(idAttr string) (*dsig.SigningContext, error) {
	ctx, err := dsig.NewSigningContext(s.key, [][]byte{s.cert.Raw})
	if err != nil {
		return nil, fmt.Errorf("samltoken: new signing context: %w", err)
	}
	ctx.Hash = crypto.SHA256
	ctx.IdAttribute = idAttr
	ctx.Canonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")
	return ctx, nil
}

// Sign は assertion 要素を enveloped 署名し、署名を含む新しい要素を返す。idAttribute は
// 署名対象の ID 参照属性名 (SAML 2.0 は "ID"、SAML 1.1 は "AssertionID")。
func (s *Signer) Sign(assertion *etree.Element, idAttribute string) (*etree.Element, error) {
	ctx, err := s.signingContext(idAttribute)
	if err != nil {
		return nil, err
	}
	signed, err := ctx.SignEnveloped(assertion)
	if err != nil {
		return nil, fmt.Errorf("samltoken: sign enveloped: %w", err)
	}
	return signed, nil
}

// BuildSignedAssertion は assertion を組み立てて enveloped 署名し、署名済みの要素を返す。
// RSTR への埋め込み用に直列化前の要素を返す点が IssueSignedAssertion と異なる。
func BuildSignedAssertion(in AssertionInput, s *Signer) (*etree.Element, string, error) {
	assertion, id, err := BuildAssertion(in)
	if err != nil {
		return nil, "", err
	}
	signed, err := s.Sign(assertion, idAttributeFor(in.Version))
	if err != nil {
		return nil, "", err
	}
	return signed, id, nil
}

// IssueSignedAssertion は assertion を組み立てて署名し、直列化した XML と assertion ID を返す。
func IssueSignedAssertion(in AssertionInput, s *Signer) ([]byte, string, error) {
	assertion, id, err := BuildAssertion(in)
	if err != nil {
		return nil, "", err
	}
	signed, err := s.Sign(assertion, idAttributeFor(in.Version))
	if err != nil {
		return nil, "", err
	}
	doc := etree.NewDocument()
	doc.SetRoot(signed)
	out, err := doc.WriteToBytes()
	if err != nil {
		return nil, "", fmt.Errorf("samltoken: serialize assertion: %w", err)
	}
	return out, id, nil
}

// newAssertionID は SAML ID 属性に適した NCName (先頭非数字) を生成する。
func newAssertionID() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("samltoken: generate assertion id: %w", err)
	}
	return "_" + hex.EncodeToString(buf), nil
}
