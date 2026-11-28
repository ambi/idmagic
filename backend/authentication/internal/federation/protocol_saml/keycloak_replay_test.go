package protocol_saml

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"

	federationmemory "github.com/ambi/idmagic/backend/authentication/internal/federation/db_memory"
	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// keycloakSetup は testing_samlrecord が往復の setup に置く値である。
type keycloakSetup struct {
	EntityID           string    `json:"entity_id"`
	SSOURL             string    `json:"sso_url"`
	ACSURL             string    `json:"acs_url"`
	RequestID          string    `json:"request_id"`
	SigningCertificate string    `json:"signing_certificate"`
	ReceivedAt         time.Time `json:"received_at"`
}

// keycloakResponse は記録の一つの応答と、それを受け取ったときの接続とログインの試行である。
type keycloakResponse struct {
	setup      keycloakSetup
	connection federationdomain.IdentityProviderConnection
	attempt    federationdomain.FederatedLoginAttempt
	xml        []byte
}

func loadKeycloakResponse(t *testing.T, name string) keycloakResponse {
	t.Helper()
	data, err := os.ReadFile("testdata/keycloak-saml.json")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := testing_httprecord.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	exchange := rec.Exchange(t, name)
	var setup keycloakSetup
	if err := json.Unmarshal(exchange.Setup, &setup); err != nil {
		t.Fatal(err)
	}
	form, err := url.ParseQuery(string(testing_httprecord.BodyBytes(exchange.Request.Body)))
	if err != nil {
		t.Fatal(err)
	}
	xml, err := base64.StdEncoding.DecodeString(form.Get("SAMLResponse"))
	if err != nil {
		t.Fatal(err)
	}
	return keycloakResponse{
		setup: setup,
		// 管理者が Keycloak の realm の SAML メタデータから写す値で接続を作る。
		// Keycloak のクライアントは email を Basic の属性 email へ対応付けている。
		connection: federationdomain.IdentityProviderConnection{
			ID: "keycloak", TenantID: "tenant", DisplayName: "Keycloak",
			Protocol: federationdomain.ProtocolSAML, Status: federationdomain.ConnectionActive,
			Issuer: setup.EntityID, SAMLSSOURL: setup.SSOURL, SAMLEntityID: setup.EntityID,
			SAMLSigningCertificates: []string{setup.SigningCertificate},
			ClaimMapping:            federationdomain.ClaimMapping{Subject: "NameID", Username: "email", Email: "email"},
			LinkingPolicy:           federationdomain.LinkingNone,
		},
		attempt: federationdomain.FederatedLoginAttempt{RequestID: setup.RequestID},
		xml:     xml,
	}
}

func (r keycloakResponse) validate(xml []byte, now time.Time) (federationdomain.NormalizedClaims, error) {
	return ValidateResponse(
		context.Background(), r.connection, r.attempt,
		base64.StdEncoding.EncodeToString(xml), r.setup.ACSURL, now,
		federationmemory.NewRepositories().Replay,
	)
}

// mutate は応答を XML の木として読み、change を当てて書き戻す。
func (r keycloakResponse) mutate(t *testing.T, change func(root *etree.Element)) []byte {
	t.Helper()
	document := etree.NewDocument()
	if err := document.ReadFromBytes(r.xml); err != nil {
		t.Fatal(err)
	}
	change(document.Root())
	out, err := document.WriteToBytes()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func child(t *testing.T, parent *etree.Element, tag string) *etree.Element {
	t.Helper()
	for _, element := range parent.ChildElements() {
		if element.Tag == tag {
			return element
		}
	}
	t.Fatalf("%s has no %s", parent.Tag, tag)
	return nil
}

// 期待値は Keycloak に作った利用者（username alice、email alice@example.com）と、
// SAML 2.0 Profiles 4.1.3.5 の「Assertion は Assertion または Response への署名で保護する」から決める。
//
//spec:covers REQ-AUTHENTICATION-001: Response、Assertion、その両方に署名した実際の応答から NameID と属性を取り出し、どちらにも署名がない応答を拒否すること
func TestValidateResponse_AcceptsKeycloakSignaturePlacements(t *testing.T) {
	for _, name := range []string{"response-and-assertion-signed", "response-signed", "assertion-signed"} {
		t.Run(name, func(t *testing.T) {
			response := loadKeycloakResponse(t, name)
			claims, err := response.validate(response.xml, response.setup.ReceivedAt)
			if err != nil {
				t.Fatalf("ValidateResponse: %v", err)
			}
			want := federationdomain.NormalizedClaims{Subject: "alice", Username: "alice@example.com", Email: "alice@example.com"}
			if claims != want {
				t.Fatalf("claims = %+v, want %+v", claims, want)
			}
		})
	}
	t.Run("unsigned", func(t *testing.T) {
		response := loadKeycloakResponse(t, "unsigned")
		if _, err := response.validate(response.xml, response.setup.ReceivedAt); err == nil {
			t.Fatal("a response with neither a Response nor an Assertion signature must be rejected")
		}
	})
}

// forgedAssertion は、署名した Assertion から署名を外し、ID と NameID を変えた写しである。
func forgedAssertion(t *testing.T, signed *etree.Element) *etree.Element {
	t.Helper()
	forged := signed.Copy()
	forged.RemoveChild(child(t, forged, "Signature"))
	forged.RemoveAttr("ID")
	forged.CreateAttr("ID", "_forged-assertion")
	child(t, child(t, forged, "Subject"), "NameID").SetText("mallory")
	return forged
}

// 署名の置き場所を緩めた実装が、署名で保護されていない Assertion を読まないことを確かめる。
// 各拒否の対照として、変更を加えずに書き戻した応答が受け付けられることを先に確かめる。
//
//spec:covers REQ-AUTHENTICATION-001: 署名の外にある Assertion、改ざんした値、検証できない Response の署名、二つ目の Assertion を拒否すること
func TestValidateResponse_RejectsAssertionsOutsideTheSignature(t *testing.T) {
	cases := []struct {
		name      string
		recording string
		change    func(t *testing.T, root *etree.Element)
	}{
		{"unsigned assertion next to the signed one", "assertion-signed", func(t *testing.T, root *etree.Element) {
			t.Helper()
			root.AddChild(forgedAssertion(t, child(t, root, "Assertion")))
		}},
		{"unsigned assertion before the signed one", "assertion-signed", func(t *testing.T, root *etree.Element) {
			t.Helper()
			signed := child(t, root, "Assertion")
			root.InsertChildAt(signed.Index(), forgedAssertion(t, signed))
		}},
		{"signed assertion moved into Extensions", "assertion-signed", func(t *testing.T, root *etree.Element) {
			t.Helper()
			signed := child(t, root, "Assertion")
			forged := forgedAssertion(t, signed)
			root.RemoveChild(signed)
			extensions := etree.NewElement("samlp:Extensions")
			extensions.AddChild(signed)
			root.InsertChildAt(child(t, root, "Status").Index(), extensions)
			root.AddChild(forged)
		}},
		{"NameID changed under an Assertion signature", "assertion-signed", func(t *testing.T, root *etree.Element) {
			t.Helper()
			signed := child(t, root, "Assertion")
			child(t, child(t, signed, "Subject"), "NameID").SetText("mallory")
		}},
		{"NameID changed under a Response signature", "response-signed", func(t *testing.T, root *etree.Element) {
			t.Helper()
			child(t, child(t, child(t, root, "Assertion"), "Subject"), "NameID").SetText("mallory")
		}},
		{"Response signature value changed while the Assertion signature stays valid", "response-and-assertion-signed", func(t *testing.T, root *etree.Element) {
			t.Helper()
			value := child(t, child(t, root, "Signature"), "SignatureValue")
			value.SetText(flipFirst(value.Text()))
		}},
		{"Assertion signature value changed", "assertion-signed", func(t *testing.T, root *etree.Element) {
			t.Helper()
			value := child(t, child(t, child(t, root, "Assertion"), "Signature"), "SignatureValue")
			value.SetText(flipFirst(value.Text()))
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := loadKeycloakResponse(t, tc.recording)
			now := response.setup.ReceivedAt
			if _, err := response.validate(response.mutate(t, func(*etree.Element) {}), now); err != nil {
				t.Fatalf("control: the re-serialized recording must be accepted: %v", err)
			}
			mutated := response.mutate(t, func(root *etree.Element) { tc.change(t, root) })
			if claims, err := response.validate(mutated, now); err == nil {
				t.Fatalf("accepted a response with an unprotected assertion: claims=%+v", claims)
			}
		})
	}
}

func flipFirst(base64Text string) string {
	text := strings.TrimSpace(base64Text)
	if strings.HasPrefix(text, "A") {
		return "B" + text[1:]
	}
	return "A" + text[1:]
}

// 記録の応答で、相関、時刻、再送の拒否が署名の置き場所に依らず働くことを確かめる。
// 期待値は SAML 2.0 Profiles 4.1.4.3（Recipient、InResponseTo、NotOnOrAfter、Audience の照合と再送の拒否）から決める。
//
//spec:covers REQ-AUTHENTICATION-001: 実際の応答の宛先、要求 ID、期限、再送の不一致を拒否すること
func TestValidateResponse_RejectsKeycloakResponsesOutsideTheirContext(t *testing.T) {
	for _, name := range []string{"response-signed", "assertion-signed"} {
		t.Run(name, func(t *testing.T) {
			response := loadKeycloakResponse(t, name)
			now := response.setup.ReceivedAt
			encoded := base64.StdEncoding.EncodeToString(response.xml)
			replay := federationmemory.NewRepositories().Replay
			validate := func(acsURL string, attempt federationdomain.FederatedLoginAttempt, at time.Time) error {
				_, err := ValidateResponse(context.Background(), response.connection, attempt, encoded, acsURL, at, replay)
				return err
			}
			if err := validate(response.setup.ACSURL+"/other", response.attempt, now); err == nil {
				t.Error("a response for another ACS URL must be rejected")
			}
			if err := validate(response.setup.ACSURL, federationdomain.FederatedLoginAttempt{RequestID: "_another-request"}, now); err == nil {
				t.Error("a response to another AuthnRequest must be rejected")
			}
			if err := validate(response.setup.ACSURL, response.attempt, now.Add(10*time.Minute)); err == nil {
				t.Error("an expired response must be rejected")
			}
			if err := validate(response.setup.ACSURL, response.attempt, now); err != nil {
				t.Fatalf("first use: %v", err)
			}
			if err := validate(response.setup.ACSURL, response.attempt, now); err == nil {
				t.Error("a replayed response must be rejected")
			}
		})
	}
}
