package domain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"
)

func testCertificateDER(tb testing.TB) []byte {
	tb.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	now := time.Date(2026, 11, 29, 0, 0, 0, 0, time.UTC)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "SAML IdP"},
		NotBefore: now, NotAfter: now.Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		tb.Fatal(err)
	}
	return der
}

// wrapLines は base64 を、メタデータの整形のように width 文字ごとの行へ分ける。
func wrapLines(text string, width int, separator string) string {
	var out strings.Builder
	for len(text) > width {
		out.WriteString(text[:width] + separator)
		text = text[width:]
	}
	out.WriteString(text)
	return out.String()
}

//spec:covers REQ-AUTHENTICATION-025: PEM と、メタデータの X509Certificate と同じ見出しのない DER の base64 を、同じ DER の正規の PEM に揃えること
func TestNormalizeSAMLSigningCertificates_KeepsTheSameDER(t *testing.T) {
	der := testCertificateDER(t)
	encoded := base64.StdEncoding.EncodeToString(der)
	canonical := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	for name, value := range map[string]string{
		"canonical PEM":             canonical,
		"PEM with CRLF and padding": "\r\n  " + strings.ReplaceAll(canonical, "\n", "\r\n") + "\t\n",
		"PEM on one body line":      "-----BEGIN CERTIFICATE-----\n" + encoded + "\n-----END CERTIFICATE-----",
		"PEM after openssl subject": "subject=CN=SAML IdP\nissuer=CN=SAML IdP\n" + canonical,
		"PEM then text":             canonical + "trailing note\n",
		"base64 on one line":        encoded,
		"base64 wrapped by LF":      "\n" + wrapLines(encoded, 76, "\n") + "\n",
		"base64 wrapped by CRLF":    wrapLines(encoded, 64, "\r\n"),
		"base64 with spaces":        "  " + wrapLines(encoded, 40, " \t ") + "  ",
	} {
		t.Run(name, func(t *testing.T) {
			normalized, err := NormalizeSAMLSigningCertificates([]string{value})
			if err != nil {
				t.Fatalf("NormalizeSAMLSigningCertificates: %v", err)
			}
			if len(normalized) != 1 || normalized[0] != canonical {
				t.Fatalf("normalized=%q, want %q", normalized, canonical)
			}
			block, rest := pem.Decode([]byte(normalized[0]))
			if block == nil || len(rest) != 0 || !bytes.Equal(block.Bytes, der) {
				t.Fatal("the normalized PEM does not carry the same DER")
			}
		})
	}
}

//spec:covers REQ-AUTHENTICATION-025: 一つの証明書として解析できない署名の証明書を拒否すること
func TestNormalizeSAMLSigningCertificates_RejectsValuesThatAreNotOneCertificate(t *testing.T) {
	der := testCertificateDER(t)
	encoded := base64.StdEncoding.EncodeToString(der)
	canonical := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	for name, value := range map[string]string{
		"empty":                 "",
		"blank":                 " \n\t ",
		"not base64":            "not a certificate",
		"base64 of non-DER":     base64.StdEncoding.EncodeToString([]byte("not a certificate")),
		"DER with trailing":     base64.StdEncoding.EncodeToString(append(append([]byte(nil), der...), 0)),
		"URL-safe alphabet":     strings.NewReplacer("+", "-", "/", "_").Replace(encoded) + "-_",
		"truncated base64":      encoded[:len(encoded)-3],
		"two PEM blocks":        canonical + canonical,
		"PEM then a key":        canonical + string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte{1}})),
		"PEM of another type":   string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})),
		"PEM with broken body":  "-----BEGIN CERTIFICATE-----\n!!!!\n-----END CERTIFICATE-----\n",
		"two base64 DER values": encoded + "\n\n" + encoded,
	} {
		t.Run(name, func(t *testing.T) {
			if normalized, err := NormalizeSAMLSigningCertificates([]string{canonical, value}); err == nil {
				t.Fatalf("normalized=%q, want an error", normalized)
			}
		})
	}
}

func FuzzParseSAMLSigningCertificate(f *testing.F) {
	der := testCertificateDER(f)
	f.Add(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})))
	f.Add(wrapLines(base64.StdEncoding.EncodeToString(der), 64, "\n"))
	f.Add("-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n")
	f.Add("")
	f.Fuzz(func(t *testing.T, text string) {
		certificate, err := ParseSAMLSigningCertificate(text)
		if err != nil {
			return
		}
		// 受け付けた値は、正規形に揃えても同じ DER として読み直せる。
		normalized, err := NormalizeSAMLSigningCertificates([]string{text})
		if err != nil {
			t.Fatalf("parsed but not normalized: %v", err)
		}
		again, err := ParseSAMLSigningCertificate(normalized[0])
		if err != nil || !bytes.Equal(again.Raw, certificate.Raw) {
			t.Fatalf("the normalized value does not parse to the same DER: %v", err)
		}
	})
}
