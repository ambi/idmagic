package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"

	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
	samlprotocol "github.com/ambi/idmagic/backend/authentication/internal/federation/protocol_saml"
	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// RequestID は記録の AuthnRequest の ID である。応答の InResponseTo はこの値になるので、再生のログインの試行も同じ値を持つ。
const RequestID = "_idmagic-recorded-request"

// RelayState は記録の AuthnRequest の RelayState である。
const RelayState = "recorded-relay-state"

// placement は Keycloak のクライアントの Sign documents（Response）と Sign assertions の組である。
type placement struct {
	name          string
	source        string
	signResponse  bool
	signAssertion bool
}

var placements = []placement{
	{"response-and-assertion-signed", "Sign documents と Sign assertions を両方有効にした応答。", true, true},
	{"response-signed", "Sign documents だけを有効にした応答（Keycloak の新しいクライアントの既定）。", true, false},
	{"assertion-signed", "Sign assertions だけを有効にした応答（Microsoft Entra ID の既定と同じ置き場所）。", false, true},
	{"unsigned", "Sign documents と Sign assertions を両方無効にした応答。", false, false},
}

// Setup は往復の setup に置く値である。再生のテストは同じ形を読み、接続とログインの試行を組み立てる。
type Setup struct {
	EntityID           string    `json:"entity_id"`
	SSOURL             string    `json:"sso_url"`
	ACSURL             string    `json:"acs_url"`
	RequestID          string    `json:"request_id"`
	SigningCertificate string    `json:"signing_certificate"`
	ReceivedAt         time.Time `json:"received_at"`
}

func recordPlacements(k *keycloak, userPassword string) ([]testing_httprecord.Exchange, error) {
	certificate, err := k.signingCertificate()
	if err != nil {
		return nil, err
	}
	connection := federationdomain.IdentityProviderConnection{
		Protocol: federationdomain.ProtocolSAML, Status: federationdomain.ConnectionActive,
		SAMLSSOURL: k.realmURL() + "/protocol/saml", SAMLEntityID: k.realmURL(),
	}
	exchanges := make([]testing_httprecord.Exchange, 0, len(placements))
	for _, p := range placements {
		if err := k.setPlacement(p); err != nil {
			return nil, fmt.Errorf("%s: %w", p.name, err)
		}
		form, receivedAt, err := k.browserLogin(connection, userPassword)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.name, err)
		}
		setup, err := json.Marshal(Setup{
			EntityID: connection.SAMLEntityID, SSOURL: connection.SAMLSSOURL, ACSURL: ACSURL,
			RequestID: RequestID, SigningCertificate: certificate, ReceivedAt: receivedAt,
		})
		if err != nil {
			return nil, err
		}
		body, err := json.Marshal(form.values.Encode())
		if err != nil {
			return nil, err
		}
		acs, err := url.Parse(form.action)
		if err != nil {
			return nil, err
		}
		exchanges = append(exchanges, testing_httprecord.Exchange{
			Name: p.name, Source: p.source, Setup: setup,
			Request: testing_httprecord.RecordedRequest{
				Method: http.MethodPost, Path: acs.Path,
				Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
				Body:    body,
			},
		})
	}
	return exchanges, nil
}

// browserLogin は新しいブラウザーとして AuthnRequest からログインし、Keycloak が ACS へ自動送信させるフォームを返す。
func (k *keycloak) browserLogin(connection federationdomain.IdentityProviderConnection, userPassword string) (htmlForm, time.Time, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return htmlForm{}, time.Time{}, err
	}
	browser := &http.Client{Jar: jar, Timeout: 30 * time.Second}
	start, err := samlprotocol.BuildAuthnRequest(connection,
		federationdomain.FederatedLoginAttempt{State: RelayState, RequestID: RequestID}, ACSURL, time.Now())
	if err != nil {
		return htmlForm{}, time.Time{}, err
	}
	loginPage, err := fetchForm(browser, http.MethodGet, start, nil, "kc-form-login")
	if err != nil {
		return htmlForm{}, time.Time{}, fmt.Errorf("login page: %w", err)
	}
	credentials := url.Values{"username": {userName}, "password": {userPassword}, "credentialId": {""}}
	post, err := fetchForm(browser, http.MethodPost, loginPage.action, credentials, "")
	receivedAt := time.Now().UTC()
	if err != nil {
		return htmlForm{}, time.Time{}, fmt.Errorf("SAML POST form: %w", err)
	}
	if post.values.Get("SAMLResponse") == "" || post.action != ACSURL {
		return htmlForm{}, time.Time{}, fmt.Errorf("unexpected form to %q with fields %v", post.action, keys(post.values))
	}
	return post, receivedAt, nil
}

type htmlForm struct {
	action string
	values url.Values
}

// fetchForm は要求を送り、応答の HTML の最初のフォーム（id を指定すればその id のフォーム）を返す。
func fetchForm(client *http.Client, method, target string, values url.Values, id string) (htmlForm, error) {
	var resp *http.Response
	var err error
	if method == http.MethodPost {
		resp, err = postForm(client, target, values)
	} else {
		resp, err = get(client, target)
	}
	if err != nil {
		return htmlForm{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return htmlForm{}, fmt.Errorf("status %d", resp.StatusCode)
	}
	document, err := html.Parse(resp.Body)
	if err != nil {
		return htmlForm{}, err
	}
	for node := range document.Descendants() {
		if node.Type != html.ElementNode || node.Data != "form" || (id != "" && attr(node, "id") != id) {
			continue
		}
		form := htmlForm{action: attr(node, "action"), values: url.Values{}}
		for input := range node.Descendants() {
			if input.Type == html.ElementNode && input.Data == "input" && attr(input, "name") != "" {
				form.values.Set(attr(input, "name"), attr(input, "value"))
			}
		}
		return form, nil
	}
	return htmlForm{}, errors.New("no matching form in the page")
}

func attr(node *html.Node, name string) string {
	for _, a := range node.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val
		}
	}
	return ""
}

func keys(values url.Values) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	return out
}
