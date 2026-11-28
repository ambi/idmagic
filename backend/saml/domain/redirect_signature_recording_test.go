package domain_test

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
)

// redirectRecording は、実物の SP が送った HTTP-Redirect のリダイレクトの URL の記録である。
// 取得日、SP の版、取得方法は記録ごとに残し、取得に使ったスクリプトは testdata の record/ にある。
type redirectRecording struct {
	RetrievedAt     string `json:"retrieved_at"`
	ServiceProvider string `json:"service_provider"`
	Method          string `json:"method"`
	CertificatePEM  string `json:"certificate_pem"`
	Requests        []struct {
		Message  string `json:"message"`
		Location string `json:"location"`
	} `json:"requests"`
}

// 実物の SP は SP ごとの規則で URL エンコードして署名する。SimpleSAMLphp は PHP の urlencode で
// `~` を `%7E` にし、Sustainsys.Saml2 は Uri.EscapeDataString、Shibboleth SP は xmltooling の
// 符号化を使う。受け取ったクエリのまま検証すれば、どの記録も記録の証明書で検証を通る。
// 記録の値を一文字だけ別の綴りにした要求が拒否されることも併せて読み、検証が署名を見ていることを確かめる。
//
//spec:covers SAML2BINDINGS-REDIRECT-SIGNATURE: 署名を要求する SP から HTTP-Redirect で受けた AuthnRequest と LogoutRequest は
func TestValidateRequestSignatureAcceptsRecordedServiceProviders(t *testing.T) {
	paths, err := filepath.Glob("testdata/redirect_signatures/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) < 2 {
		t.Fatalf("符号化の規則が違う SP の記録が二つ以上要る: %v", paths)
	}
	messages := map[string]bool{}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var recording redirectRecording
		if err := json.Unmarshal(raw, &recording); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if recording.RetrievedAt == "" || recording.ServiceProvider == "" || recording.Method == "" {
			t.Fatalf("%s: 記録には取得日、SP の版、取得方法が要る", path)
		}
		sp := sampleServiceProvider()
		sp.WantAuthnRequestsSigned = true
		sp.AuthnRequestSigningCertificatePEM = recording.CertificatePEM

		for _, request := range recording.Requests {
			messages[request.Message] = true
			t.Run(recording.ServiceProvider+"/"+request.Message, func(t *testing.T) {
				location, err := url.Parse(request.Location)
				if err != nil {
					t.Fatalf("parse location: %v", err)
				}
				assertRecordedMessage(t, request.Message, location.Query().Get("SAMLRequest"))

				if err := validateRedirect(location.RawQuery, sp); err != nil {
					t.Fatalf("%s が署名した要求が拒否された: %v\nquery=%s", recording.ServiceProvider, err, location.RawQuery)
				}
				respelled := respellOneEscape(t, location.RawQuery)
				if err := validateRedirect(respelled, sp); err == nil {
					t.Fatalf("署名の後に符号化を変えた要求が受理された: %s", respelled)
				}
			})
		}
	}
	for _, message := range []string{"AuthnRequest", "LogoutRequest"} {
		if !messages[message] {
			t.Errorf("%s の記録がない", message)
		}
	}
}

// assertRecordedMessage は、記録の SAMLRequest が記録の名乗るメッセージであることを確かめる。
func assertRecordedMessage(t *testing.T, message, samlRequest string) {
	t.Helper()
	xml, err := samldomain.DecodeRedirect(samlRequest)
	if err != nil {
		t.Fatalf("decode SAMLRequest: %v", err)
	}
	switch message {
	case "AuthnRequest":
		_, err = samldomain.ParseAuthnRequest(xml)
	case "LogoutRequest":
		_, err = samldomain.ParseLogoutRequest(xml)
	default:
		t.Fatalf("記録のメッセージ %q を知らない", message)
	}
	if err != nil {
		t.Fatalf("SAMLRequest が %s として読めない: %v\n%s", message, err, xml)
	}
}
