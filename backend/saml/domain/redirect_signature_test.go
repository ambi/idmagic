package domain_test

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
	"net/url"
	"strings"
	"testing"
	"time"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
)

// この節の期待値は SAML 2.0 Bindings 3.4.4.1 の文面から書く。署名する文字列は
// `SAMLRequest=値&RelayState=値&SigAlg=値`（RelayState は現れるときだけ）であり、値は
// クエリに載せた URL エンコードのままである。URL エンコードは一意でないので、受け手は
// 受け取った元の値で検証しなければならず、デコードして符号化し直してはならない。
// 符号化の規則は製品の関数から作らず、SP の実装が使う規則をテストが自分で書く。

const rsaSHA256 = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"

// redirectEncoding は SP が URL エンコードに使う規則の一つである。
type redirectEncoding struct {
	name   string
	encode func(string) string
}

// redirectEncodings は SP の実装に見られる、互いに違う正当な符号化の規則である。
var redirectEncodings = []redirectEncoding{
	// Go の url.QueryEscape、Java の URLEncoder、PHP の urlencode と同じく、大文字の 16 進と空白の `+`。
	{"uppercase hex with plus", url.QueryEscape},
	// .NET の HttpUtility.UrlEncode は 16 進を小文字で書く。
	{"lowercase hex", func(s string) string { return lowerHex(url.QueryEscape(s)) }},
	// RFC 3986 の percent-encoding で書く実装は、空白を `%20` にする。
	{"space as %20", func(s string) string { return strings.ReplaceAll(url.QueryEscape(s), "+", "%20") }},
	// 予約されていない文字まで符号化しても、デコード後の値は変わらない。
	{"unreserved characters encoded", encodeEveryByte},
}

func lowerHex(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			b.WriteByte('%')
			b.WriteString(strings.ToLower(s[i+1 : i+3]))
			i += 2
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func encodeEveryByte(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			b.WriteByte(c)
			continue
		}
		b.WriteString("%" + strings.ToUpper(hexByte(c)))
	}
	return b.String()
}

func hexByte(c byte) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[c>>4], digits[c&0x0f]})
}

// redirectSigner は SP の署名鍵と、IdP に登録する証明書を持つ。
type redirectSigner struct {
	key     *rsa.PrivateKey
	certPEM string
}

func newRedirectSigner(t *testing.T) redirectSigner {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "redirect signing sp"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return redirectSigner{key: key, certPEM: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))}
}

func (s redirectSigner) serviceProvider() samldomain.SamlServiceProvider {
	sp := sampleServiceProvider()
	sp.WantAuthnRequestsSigned = true
	sp.AuthnRequestSigningCertificatePEM = s.certPEM
	return sp
}

// signedQuery は redirectSAMLRequest を載せ、3.4.4.1 の順に、encode で符号化した値を連結して
// 署名し、Signature を足したクエリを返す。relayState が nil なら RelayState を載せない。
func (s redirectSigner) signedQuery(t *testing.T, encode func(string) string, relayState *string) (signed, query string) {
	t.Helper()
	signed = "SAMLRequest=" + encode(redirectSAMLRequest)
	if relayState != nil {
		signed += "&RelayState=" + encode(*relayState)
	}
	signed += "&SigAlg=" + encode(rsaSHA256)
	sum := sha256.Sum256([]byte(signed))
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed, signed + "&Signature=" + encode(base64.StdEncoding.EncodeToString(sig))
}

// 値の文字は、符号化の規則ごとに綴りが変わるものを選ぶ。`/`、`+`、`=` は base64 に、
// 空白、`?`、`-`、`.` は RelayState に現れる。
const (
	redirectSAMLRequest = "fZFLT8MwEITv/AqL7/uR0JRqZf/P+Ab/+K/3n/i+h=="
)

func redirectRelayState() *string {
	relay := "https://sp.example.com/app-1/start.html?next=a b"
	return &relay
}

func validateRedirect(query string, sp samldomain.SamlServiceProvider) error {
	return samldomain.ValidateRequestSignature(samldomain.BindingRedirect, nil, query, sp)
}

//spec:covers SAML2BINDINGS-REDIRECT-SIGNATURE: 受け取ったクエリの `SAMLRequest`、`RelayState`（クエリに現れたときだけ）、`SigAlg` の値を URL エンコードのまま
func TestValidateRequestSignatureVerifiesTheReceivedEncoding(t *testing.T) {
	signer := newRedirectSigner(t)
	sp := signer.serviceProvider()

	for _, encoding := range redirectEncodings {
		t.Run(encoding.name, func(t *testing.T) {
			_, query := signer.signedQuery(t, encoding.encode, redirectRelayState())
			if err := validateRedirect(query, sp); err != nil {
				t.Fatalf("正しい署名が拒否された: %v\nquery=%s", err, query)
			}

			// デコード後が同じになる別の綴りへ一文字変えると、署名した文字列ではなくなる。
			respelled := respellOneEscape(t, query)
			if err := validateRedirect(respelled, sp); err == nil {
				t.Fatalf("署名した後に符号化を変えた要求が受理された: %s", respelled)
			}
		})
	}

	t.Run("RelayState absent", func(t *testing.T) {
		_, query := signer.signedQuery(t, url.QueryEscape, nil)
		if err := validateRedirect(query, sp); err != nil {
			t.Fatalf("RelayState のない正しい署名が拒否された: %v", err)
		}
	})

	t.Run("RelayState present with an empty value", func(t *testing.T) {
		empty := ""
		_, query := signer.signedQuery(t, url.QueryEscape, &empty)
		if err := validateRedirect(query, sp); err != nil {
			t.Fatalf("空の RelayState を含めた正しい署名が拒否された: %v", err)
		}
	})

	t.Run("parameters outside the signature are ignored", func(t *testing.T) {
		_, query := signer.signedQuery(t, url.QueryEscape, redirectRelayState())
		for _, extended := range []string{"foo=bar&" + query, query + "&foo=bar"} {
			if err := validateRedirect(extended, sp); err != nil {
				t.Fatalf("署名の対象でない引数のある要求が拒否された: %v\nquery=%s", err, extended)
			}
		}
	})

	t.Run("signature order differs from the query order", func(t *testing.T) {
		signed, query := signer.signedQuery(t, url.QueryEscape, redirectRelayState())
		signature := strings.TrimPrefix(query, signed+"&")
		parts := strings.Split(signed, "&")
		reordered := signature + "&" + parts[2] + "&" + parts[1] + "&" + parts[0]
		if err := validateRedirect(reordered, sp); err != nil {
			t.Fatalf("引数の並びが違う正しい署名が拒否された: %v\nquery=%s", err, reordered)
		}
	})
}

// respellOneEscape は、署名の対象の範囲にある最初の `%XX` の 16 進の大文字と小文字を入れ替える。
func respellOneEscape(t *testing.T, query string) string {
	t.Helper()
	signedEnd := strings.Index(query, "&Signature=")
	i := strings.Index(query[:signedEnd], "%")
	for i >= 0 {
		hex := query[i+1 : i+3]
		flipped := strings.ToLower(hex)
		if flipped == hex {
			flipped = strings.ToUpper(hex)
		}
		if flipped != hex {
			return query[:i+1] + flipped + query[i+3:]
		}
		next := strings.Index(query[i+1:signedEnd], "%")
		if next < 0 {
			break
		}
		i += 1 + next
	}
	t.Fatalf("署名の対象に英字を含む %%XX がない: %s", query)
	return ""
}

// 重複した引数の一方で署名を検証し、もう一方を処理に使うと、署名していない値を処理する。
//
//spec:covers SAML2BINDINGS-REDIRECT-SIGNATURE: 名前をデコードして `SAMLRequest`、`RelayState`、`SigAlg`、`Signature` のいずれかが二度以上現れる要求は拒否する
func TestValidateRequestSignatureRejectsRepeatedSignedParameters(t *testing.T) {
	signer := newRedirectSigner(t)
	sp := signer.serviceProvider()
	_, query := signer.signedQuery(t, url.QueryEscape, redirectRelayState())
	if err := validateRedirect(query, sp); err != nil {
		t.Fatalf("対照の正しい署名が拒否された: %v", err)
	}

	repeated := map[string]string{
		"SAMLRequest after":             query + "&SAMLRequest=other",
		"SAMLRequest before":            "SAMLRequest=other&" + query,
		"SAMLRequest with same value":   query + "&SAMLRequest=" + url.QueryEscape(redirectSAMLRequest),
		"SAMLRequest with encoded name": "SAML%52equest=other&" + query,
		"RelayState":                    query + "&RelayState=other",
		"RelayState with empty value":   query + "&RelayState=",
		"SigAlg":                        query + "&SigAlg=" + url.QueryEscape(rsaSHA256),
		"Signature":                     query + "&Signature=AAAA",
	}
	for name, input := range repeated {
		t.Run(name, func(t *testing.T) {
			if err := validateRedirect(input, sp); err == nil {
				t.Fatalf("署名の対象の引数が二度現れる要求が受理された: %s", input)
			}
		})
	}
}
