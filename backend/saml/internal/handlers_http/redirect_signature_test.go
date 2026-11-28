package handlers_http_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
)

// SAML 2.0 Bindings 3.4.4.1 は、HTTP-Redirect の署名を
// `SAMLRequest=値&RelayState=値&SigAlg=値`（値は URL エンコードのまま）に対して作り、
// 受け手は受け取った元の符号化で検証しなければならないとする。ここでは IdMagic と違う規則で
// 符号化する SP を、テストが自分で書いた符号化で表す。製品の関数から作ると、同じ思い込みを
// 要求と実装が共有して食い違いが見えなくなる。

// otherSPEncoding は、16 進を小文字で書き（.NET の HttpUtility.UrlEncode）、空白を `%20` にし
// （RFC 3986 の percent-encoding）、予約されていない `-` と `.` まで符号化する SP を表す。
// IdMagic の url.QueryEscape とはどの綴りも一致しない。
func otherSPEncoding(s string) string {
	escaped := url.QueryEscape(s)
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	escaped = strings.ReplaceAll(escaped, "-", "%2d")
	escaped = strings.ReplaceAll(escaped, ".", "%2e")
	var b strings.Builder
	for i := 0; i < len(escaped); i++ {
		if escaped[i] == '%' && i+2 < len(escaped) {
			b.WriteString("%" + strings.ToLower(escaped[i+1:i+3]))
			i += 2
			continue
		}
		b.WriteByte(escaped[i])
	}
	return b.String()
}

type redirectSigningSP struct {
	key     *rsa.PrivateKey
	certPEM string
}

func newRedirectSigningSP(t *testing.T) redirectSigningSP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(4),
		Subject:      pkix.Name{CommonName: "redirect signing sp"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("cert: %v", err)
	}
	return redirectSigningSP{key: key, certPEM: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))}
}

// signedRedirectQuery は SAML メッセージを DEFLATE と base64 で包み、encode で符号化した値を
// 3.4.4.1 の順に連結して署名したクエリを返す。
func (sp redirectSigningSP) signedRedirectQuery(t *testing.T, message, relayState string, encode func(string) string) string {
	t.Helper()
	encoded, err := samldomain.EncodeRedirect([]byte(message))
	if err != nil {
		t.Fatalf("encode redirect: %v", err)
	}
	signed := "SAMLRequest=" + encode(encoded) +
		"&RelayState=" + encode(relayState) +
		"&SigAlg=" + encode("http://www.w3.org/2001/04/xmldsig-more#rsa-sha256")
	sum := sha256.Sum256([]byte(signed))
	signature, err := rsa.SignPKCS1v15(rand.Reader, sp.key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed + "&Signature=" + encode(base64.StdEncoding.EncodeToString(signature))
}

// respellFirstEscape は、最初の英字を含む `%xx` の 16 進を大文字に書き換える。
// デコード後の値は変わらないが、SP が署名した文字列ではなくなる。
func respellFirstEscape(t *testing.T, query string) string {
	t.Helper()
	for i := 0; i+2 < len(query); i++ {
		if query[i] != '%' {
			continue
		}
		hex := query[i+1 : i+3]
		if upper := strings.ToUpper(hex); upper != hex {
			return query[:i+1] + upper + query[i+3:]
		}
	}
	t.Fatalf("英字を含む %%xx がない: %s", query)
	return ""
}

// IdMagic と違う規則で符号化した SP の正しい署名を、HTTP の入口が受け取ったままのクエリで
// 検証することを、SSO と SLO の両方で観測する。HTTP の入口がクエリを正規化して渡しても、
// ドメインが符号化し直しても、この要求は拒否される。対照として、デコード後が同じになる
// 綴りへ一文字変えた要求が拒否されることも読む。これが無いと、署名を見ない実装と区別できない。
//
//spec:covers SAML2BINDINGS-REDIRECT-SIGNATURE: 16 進の大文字と小文字、空白の `+` と `%20`、予約されていない文字を符号化するかどうかが SP ごとに違っても、正しい署名を受け入れる
func TestSamlVerifiesRedirectSignaturesOverTheReceivedEncoding(t *testing.T) {
	sp := newRedirectSigningSP(t)
	const relayState = "return-to /app.html?step=a b"

	t.Run("AuthnRequest", func(t *testing.T) {
		e := newServerRequiringSignedRequests(t, sp.certPEM, "")
		query := sp.signedRedirectQuery(t, buildAuthnRequestXML(authnRequestOptions{acsURL: "https://sp.example.com/acs"}), relayState, otherSPEncoding)
		if recorder := get(e, "/saml/sso?"+query); !issuedAnAssertion(t, recorder) {
			t.Fatalf("正しく署名した AuthnRequest に Assertion が返らない: status=%d body=%s", recorder.Code, recorder.Body.String())
		}

		respelled := sp.signedRedirectQuery(t, buildAuthnRequestXML(authnRequestOptions{acsURL: "https://sp.example.com/acs"}), relayState, otherSPEncoding)
		recorder := get(e, "/saml/sso?"+respellFirstEscape(t, respelled))
		if issuedAnAssertion(t, recorder) {
			t.Fatalf("署名の後に符号化を変えた AuthnRequest に Assertion が返った: %s", recorder.Body.String())
		}
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", recorder.Code)
		}
	})

	t.Run("LogoutRequest", func(t *testing.T) {
		const sloURL = "https://sp.example.com/saml/slo"
		e := newServerRequiringSignedRequests(t, sp.certPEM, sloURL)
		logoutRequest := `<samlp:LogoutRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ` +
			`xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_logout-encoding" Version="2.0" ` +
			`IssueInstant="` + time.Now().UTC().Format(time.RFC3339) + `" ` +
			`Destination="https://idp.example/realms/default/saml/slo">` +
			`<saml:Issuer>https://sp.example.com</saml:Issuer>` +
			`<saml:NameID>user-1</saml:NameID></samlp:LogoutRequest>`
		query := sp.signedRedirectQuery(t, logoutRequest, relayState, otherSPEncoding)

		recorder := get(e, "/saml/slo?"+query)
		if recorder.Code != http.StatusSeeOther || !strings.HasPrefix(recorder.Header().Get("Location"), sloURL+"?SAMLResponse=") {
			t.Fatalf("正しく署名した LogoutRequest に LogoutResponse が返らない: status=%d Location=%q body=%s",
				recorder.Code, recorder.Header().Get("Location"), recorder.Body.String())
		}

		recorder = get(e, "/saml/slo?"+respellFirstEscape(t, query))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("署名の後に符号化を変えた LogoutRequest: status=%d Location=%q, want 400",
				recorder.Code, recorder.Header().Get("Location"))
		}
	})
}
