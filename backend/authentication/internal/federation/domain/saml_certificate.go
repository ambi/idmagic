package domain

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

var errSAMLSigningCertificate = errors.New("a SAML signing certificate must be one PEM certificate or the base64 DER of one certificate")

// ParseSAMLSigningCertificate は、上流の SAML IdP の署名の証明書を一つ読む。
// 値は PEM の CERTIFICATE のブロック一つか、メタデータの X509Certificate と同じ見出しのない DER の base64 である。
// PEM の前後の文字（openssl x509 が前に出す subject= の行など）は pem.Decode と同じく読み飛ばし、二つ目のブロックは拒否する。
// base64 は空白と改行を含んでよい（XML Signature の X509Data は改行で折り返した値を許す）。
// 有効期間は判定しない。
func ParseSAMLSigningCertificate(text string) (*x509.Certificate, error) {
	var der []byte
	if strings.Contains(text, "-----BEGIN") {
		block, rest := pem.Decode([]byte(strings.TrimSpace(text)))
		if block == nil || block.Type != "CERTIFICATE" || bytes.Contains(rest, []byte("-----BEGIN")) {
			return nil, errSAMLSigningCertificate
		}
		der = block.Bytes
	} else {
		decoded, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(text), ""))
		if err != nil || len(decoded) == 0 {
			return nil, errSAMLSigningCertificate
		}
		der = decoded
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errSAMLSigningCertificate, err)
	}
	return certificate, nil
}

// NormalizeSAMLSigningCertificates は、各値を ParseSAMLSigningCertificate で読み、同じ DER を正規の PEM で包んだ写しを返す。
// 一つでも読めない値があれば、何番目かを示すエラーを返す。
func NormalizeSAMLSigningCertificates(values []string) ([]string, error) {
	if values == nil {
		return nil, nil
	}
	normalized := make([]string, 0, len(values))
	for i, value := range values {
		certificate, err := ParseSAMLSigningCertificate(value)
		if err != nil {
			return nil, fmt.Errorf("SAML signing certificate %d: %w", i+1, err)
		}
		normalized = append(normalized, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw})))
	}
	return normalized, nil
}
