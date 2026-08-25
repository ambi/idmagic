package certificates_mtls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

// fuzzClientCertificateDER は固定のクライアント証明書を 1 度だけ作る。
// 入力ごとに鍵を生成すると探索が進まない。
func fuzzClientCertificateDER(tb testing.TB) []byte {
	tb.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		tb.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(7),
		Subject:      pkix.Name{CommonName: "idmagic mtls fuzz", Organization: []string{"IdMagic"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		tb.Fatalf("create certificate: %v", err)
	}
	return der
}

// FuzzParseClientCertificateHeader は、拒否したヘッダから主体を持ち出さないことを表明する。
// このヘッダの解析結果がクライアント認証の主体を決めるため、error と一緒に
// ParsedClientCertificate を返すと、呼び出し側が err を握り潰した瞬間に未検証の主体を信頼する。
func FuzzParseClientCertificateHeader(f *testing.F) {
	f.Add("")
	f.Add("not-a-certificate")
	f.Add("-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----")
	f.Add("%ZZ")
	f.Add(strings.Repeat("A", 1024))

	f.Fuzz(func(t *testing.T, header string) {
		if len(header) > 256*1024 {
			return
		}
		parsed, err := ParseClientCertificateHeader(header)
		if err != nil {
			if parsed != nil {
				t.Fatalf("ParseClientCertificateHeader returned %+v together with an error", parsed)
			}
			return
		}
		if parsed == nil {
			t.Fatal("ParseClientCertificateHeader returned no error and no certificate")
		}
		// thumbprint は base64url (padding なし) の SHA-256 なので必ず 43 文字。
		if len(parsed.ThumbprintS256) != 43 {
			t.Fatalf("thumbprint %q has %d characters, want 43", parsed.ThumbprintS256, len(parsed.ThumbprintS256))
		}
	})
}

// FuzzClientCertificateEncodingIsStable は、同じ証明書がどの包み方で届いても同じ thumbprint に
// なることを表明する。PEM 経路と base64 DER 経路で違うバイト列をハッシュすると、
// 一方の経路で発行した cnf 束縛がもう一方の経路で一致しなくなる。
func FuzzClientCertificateEncodingIsStable(f *testing.F) {
	f.Add(true, "\n", false)
	f.Add(false, " ", false)
	f.Add(false, "\n", true)
	f.Add(true, "\r\n", true)

	der := fuzzClientCertificateDER(f)
	sum := sha256.Sum256(der)
	want := base64.RawURLEncoding.EncodeToString(sum[:])

	f.Fuzz(func(t *testing.T, usePEM bool, spacer string, urlEncode bool) {
		if len(spacer) > 64 || strings.ContainsAny(spacer, "-=+/") {
			return
		}
		var header string
		if usePEM {
			header = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
		} else {
			// base64 DER 経路は空白を畳んでから復号するので、行の折り方を変えても結果は変わらない。
			encoded := base64.StdEncoding.EncodeToString(der)
			var wrapped strings.Builder
			for i := 0; i < len(encoded); i += 64 {
				wrapped.WriteString(encoded[i:min(i+64, len(encoded))])
				wrapped.WriteString(spacer)
			}
			header = spacer + wrapped.String()
		}
		if urlEncode {
			header = strings.ReplaceAll(header, "\n", "%0A")
		}

		parsed, err := ParseClientCertificateHeader(header)
		if err != nil {
			return
		}
		if parsed.ThumbprintS256 != want {
			t.Fatalf("thumbprint changed with the wrapping: got %q want %q (pem=%v urlEncode=%v spacer=%q)",
				parsed.ThumbprintS256, want, usePEM, urlEncode, spacer)
		}
	})
}

// FuzzClientCertSubjectMatches は、認証に使う比較が反射的かつ対称であることを表明する。
// 正規化が壊れると、同じ主体が自分自身と一致しなくなるか、比較の向きで結果が変わる。
func FuzzClientCertSubjectMatches(f *testing.F) {
	f.Add("CN=a,O=b", "cn=a, o=b")
	f.Add("CN=a", "CN=A")
	f.Add("", "")
	f.Add("CN=a,,O=b", "CN=a,O=b")
	f.Add("CN=a\nO=b", "CN=a,O=b")

	f.Fuzz(func(t *testing.T, expected, presented string) {
		if !ClientCertSubjectMatches(expected, expected) {
			t.Fatalf("a subject did not match itself: %q", expected)
		}
		if ClientCertSubjectMatches(expected, presented) != ClientCertSubjectMatches(presented, expected) {
			t.Fatalf("the comparison is not symmetric: %q vs %q", expected, presented)
		}
	})
}
