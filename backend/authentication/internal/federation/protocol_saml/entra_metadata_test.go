package protocol_saml

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// copiedFromMetadata は、管理者が IdP のメタデータから接続の画面へ写す値である。
type copiedFromMetadata struct {
	ssoURL       string
	certificates []string
}

// copyFromEntraMetadata は、記録した Entra ID のメタデータの IDPSSODescriptor から、
// HTTP-Redirect の SingleSignOnService の Location と、署名の KeyDescriptor の X509Certificate の文字列を、そのまま取り出す。
func copyFromEntraMetadata(t *testing.T) (copiedFromMetadata, time.Time) {
	t.Helper()
	data, err := os.ReadFile("testdata/entra-federation-metadata.json")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := testing_httprecord.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	retrievedAt, err := time.Parse(time.DateOnly, rec.RetrievedAt)
	if err != nil {
		t.Fatal(err)
	}
	document := etree.NewDocument()
	if err := document.ReadFromBytes(testing_httprecord.BodyBytes(rec.Exchange(t, "federation-metadata").Response.Body)); err != nil {
		t.Fatal(err)
	}
	idp := document.FindElement("//IDPSSODescriptor")
	if idp == nil {
		t.Fatal("the metadata has no IDPSSODescriptor")
	}
	var copied copiedFromMetadata
	for _, sso := range idp.SelectElements("SingleSignOnService") {
		if sso.SelectAttrValue("Binding", "") == "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" {
			copied.ssoURL = sso.SelectAttrValue("Location", "")
		}
	}
	for _, key := range idp.SelectElements("KeyDescriptor") {
		if key.SelectAttrValue("use", "signing") != "signing" {
			continue
		}
		if certificate := key.FindElement(".//X509Certificate"); certificate != nil {
			copied.certificates = append(copied.certificates, certificate.Text())
		}
	}
	if copied.ssoURL == "" || len(copied.certificates) == 0 {
		t.Fatalf("copied %+v, want an HTTP-Redirect SSO URL and signing certificates", copied)
	}
	return copied, retrievedAt.Add(12 * time.Hour)
}

// 記録の証明書が、取得の時点で使える証明書であることを確かめる。
// X509Certificate は XML Signature §4.5.4（X509Data）の定めどおり DER の base64 なので、PEM の見出しで包めば同じ証明書になる。
//
// 管理者が写したままの値（PEM の見出しのない base64）は、まだ確かめない。
// 今は保存が通り、ログインの署名の検証で失敗する。保存と検証の規則を揃える wi-73879 が、写したままの値の検査をここへ足す。
//
//spec:covers REQ-AUTHENTICATION-001: Entra ID の公開の連携メタデータの署名の証明書が、PEM で保存すれば上流の SAML の応答の検証に使える証明書として受け付けられること
func TestSigningCertificates_FromEntraMetadataAreUsableAsPEM(t *testing.T) {
	copied, at := copyFromEntraMetadata(t)
	if !strings.HasPrefix(copied.ssoURL, "https://") {
		t.Fatalf("SSO URL %q copied from the metadata is not HTTPS", copied.ssoURL)
	}
	pems := make([]string, 0, len(copied.certificates))
	for _, certificate := range copied.certificates {
		pems = append(pems, "-----BEGIN CERTIFICATE-----\n"+strings.TrimSpace(certificate)+"\n-----END CERTIFICATE-----\n")
	}
	if failures := ValidateSigningCertificates(pems, at); len(failures) != 0 {
		t.Fatalf("the recorded signing certificates are not usable at retrieval: %v", failures)
	}
}
