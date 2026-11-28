package domain_test

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

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

// recordedRegistration は、記録を取ったときに SP が名乗った entityID と ACS である。
// 管理者はこの値で SP を登録する。値は testdata/redirect_signatures/record/ の各スクリプトの SP の設定から写した。
// 記録の XML から写すと、解析の誤りが登録の側にも入って打ち消し合うためである。
var recordedRegistration = map[string]struct{ entityID, acsURL string }{
	// shibboleth.sh の ApplicationDefaults entityID と、/Shibboleth.sso の既定の SAML2/POST の ACS（localhost で起動）。
	"shibboleth.json": {"https://sp.example.com/shibboleth", "http://localhost/Shibboleth.sso/SAML2/POST"},
	// simplesamlphp.sh の entityID と setAssertionConsumerServiceURL。
	"simplesamlphp.json": {"https://sp.example.com", "https://sp.example.com/acs"},
	// sustainsys.sh の Issuer と AssertionConsumerServiceUrl。
	"sustainsys.json": {"https://sp.example.com", "https://sp.example.com/acs"},
}

// 実物の SP の AuthnRequest を、記録の SP の設定どおりに登録した SP に対して確かめる。
// SAML 2.0 Core 3.2.2 の InResponseTo は要求の ID を往復させ、3.4.1 の AssertionConsumerServiceURL が
// 登録した ACS と一致するときはその ACS に、3.4.1.1 の NameIDPolicy の Format があればその形式で応える。
// Format のない要求には SP の既定の形式を使う。
//
//spec:covers SAML2BINDINGS-REDIRECT-SIGNATURE: 署名を要求する SP から HTTP-Redirect で受けた AuthnRequest は、記録の SP の設定で登録した SP への要求として検証を通り、ACS、InResponseTo、NameID の形式を確定すること
func TestValidateSignInAcceptsRecordedServiceProviders(t *testing.T) {
	const destination = "https://idp.example/realms/default/saml/sso"
	wantFormat := map[string]string{
		// SimpleSAMLphp の記録の NameIDPolicy は transient を求める。ほかの二つは Format を持たない。
		"simplesamlphp.json": samldomain.SamlNameIDFormatTransient,
	}
	seen := 0
	for file, registration := range recordedRegistration {
		raw, err := os.ReadFile(filepath.Join("testdata", "redirect_signatures", file))
		if err != nil {
			t.Fatal(err)
		}
		var recording redirectRecording
		if err := json.Unmarshal(raw, &recording); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		for _, request := range recording.Requests {
			if request.Message != "AuthnRequest" {
				continue
			}
			seen++
			t.Run(file, func(t *testing.T) {
				location, err := url.Parse(request.Location)
				if err != nil {
					t.Fatal(err)
				}
				xml, err := samldomain.DecodeRedirect(location.Query().Get("SAMLRequest"))
				if err != nil {
					t.Fatal(err)
				}
				req, err := samldomain.ParseAuthnRequest(xml)
				if err != nil {
					t.Fatalf("ParseAuthnRequest: %v", err)
				}
				sp := sampleServiceProvider()
				sp.EntityID = registration.entityID
				sp.ACSURLs = []string{registration.acsURL}
				// 記録の時刻は記録を取ったときのものなので、検証の時刻を要求の発行の直後に置く。
				validated, err := samldomain.ValidateSignIn(req, sp, destination, req.IssueInstant.Add(5*time.Second))
				if err != nil {
					t.Fatalf("%s の AuthnRequest が拒否された: %v\n%s", recording.ServiceProvider, err, xml)
				}
				if validated.ACSURL != registration.acsURL {
					t.Errorf("ACS = %q, want the registered %q", validated.ACSURL, registration.acsURL)
				}
				if validated.InResponseTo != req.ID || req.ID == "" {
					t.Errorf("InResponseTo = %q, want the request ID %q", validated.InResponseTo, req.ID)
				}
				want := wantFormat[file]
				if want == "" {
					want = sp.ClaimPolicy.NameID.Format
				}
				if validated.NameIDFormat != want {
					t.Errorf("NameID format = %q, want %q", validated.NameIDFormat, want)
				}
			})
		}
	}
	if seen != len(recordedRegistration) {
		t.Fatalf("AuthnRequest の記録が %d 件、SP の登録値が %d 件で、一致しない", seen, len(recordedRegistration))
	}
}
