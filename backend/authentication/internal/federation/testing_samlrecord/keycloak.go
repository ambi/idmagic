package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/beevik/etree"
)

// Realm、利用者、クライアントの値は記録に現れる。秘密ではない固定の値である。
const (
	realmName = "idmagic"
	userName  = "alice"
	userEmail = "alice@example.com"
	// ACSURL は IdMagic の上流 SAML の ACS の URL の形（テナントの issuer と /api/auth/federation/saml/callback）に合わせた値である。
	// Keycloak は AuthnRequest の Issuer をクライアント ID として探し、Audience にも同じ値を置く。
	ACSURL = "https://idmagic.example/realms/acme/api/auth/federation/saml/callback"
)

// keycloak は開発モードの Keycloak の管理 REST API とログイン画面を操作する。
type keycloak struct {
	base     string
	client   *http.Client
	token    string
	clientID string // 作った SAML クライアントの内部 ID
}

func (k *keycloak) realmURL() string { return k.base + "/realms/" + realmName }

func (k *keycloak) waitReady(limit time.Duration) error {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		resp, err := get(k.client, k.base+"/realms/master/.well-known/openid-configuration")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return errors.New("keycloak did not become ready")
}

// get と postForm は、記録の道具が送るすべての要求の入口である。道具は一度きりの手作業なので、取り消しの文脈を持たない。
func get(client *http.Client, target string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, target, http.NoBody)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func postForm(client *http.Client, target string, values url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, target, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return client.Do(req)
}

// login は master realm の admin-cli で管理者のアクセストークンを得る。
func (k *keycloak) login(adminPassword string) error {
	resp, err := postForm(k.client, k.base+"/realms/master/protocol/openid-connect/token", url.Values{
		"grant_type": {"password"}, "client_id": {"admin-cli"},
		"username": {"admin"}, "password": {adminPassword},
	})
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil || token.AccessToken == "" {
		return fmt.Errorf("admin token: status %d", resp.StatusCode)
	}
	k.token = token.AccessToken
	return nil
}

func (k *keycloak) serverVersion() (string, error) {
	var info struct {
		SystemInfo struct {
			Version string `json:"version"`
		} `json:"systemInfo"`
	}
	if err := k.admin(http.MethodGet, "/admin/serverinfo", nil, &info); err != nil {
		return "", err
	}
	if info.SystemInfo.Version == "" {
		return "", errors.New("serverinfo has no version")
	}
	return info.SystemInfo.Version, nil
}

// prepareRealm は realm、利用者、SAML クライアントを作る。
func (k *keycloak) prepareRealm(userPassword string) error {
	if err := k.admin(http.MethodPost, "/admin/realms", map[string]any{"realm": realmName, "enabled": true}, nil); err != nil {
		return err
	}
	if err := k.admin(http.MethodPost, "/admin/realms/"+realmName+"/users", map[string]any{
		"username": userName, "email": userEmail, "emailVerified": true,
		"firstName": "Alice", "lastName": "Example", "enabled": true,
		"credentials": []map[string]any{{"type": "password", "value": userPassword, "temporary": false}},
	}, nil); err != nil {
		return err
	}
	if err := k.admin(http.MethodPost, "/admin/realms/"+realmName+"/clients", k.clientRepresentation(placements[0]), nil); err != nil {
		return err
	}
	var clients []struct {
		ID string `json:"id"`
	}
	if err := k.admin(http.MethodGet, "/admin/realms/"+realmName+"/clients?clientId="+url.QueryEscape(ACSURL), nil, &clients); err != nil {
		return err
	}
	if len(clients) != 1 {
		return fmt.Errorf("want one SAML client, got %d", len(clients))
	}
	k.clientID = clients[0].ID
	return nil
}

func (k *keycloak) clientRepresentation(p placement) map[string]any {
	return map[string]any{
		"clientId": ACSURL, "protocol": "saml", "enabled": true,
		"redirectUris": []string{ACSURL},
		"attributes": map[string]string{
			"saml.server.signature":               fmt.Sprint(p.signResponse),
			"saml.assertion.signature":            fmt.Sprint(p.signAssertion),
			"saml.client.signature":               "false",
			"saml.signature.algorithm":            "RSA_SHA256",
			"saml.force.post.binding":             "true",
			"saml.encrypt":                        "false",
			"saml_name_id_format":                 "username",
			"saml_assertion_consumer_url_post":    ACSURL,
			"saml_single_logout_service_url_post": "",
		},
		"protocolMappers": []map[string]any{{
			"name": "email", "protocol": "saml", "protocolMapper": "saml-user-property-mapper",
			"config": map[string]string{
				"user.attribute": "email", "attribute.name": "email",
				"attribute.nameformat": "Basic", "friendly.name": "email",
			},
		}},
	}
}

// setPlacement はクライアントの署名の置き場所を切り替える。
func (k *keycloak) setPlacement(p placement) error {
	var current map[string]any
	path := "/admin/realms/" + realmName + "/clients/" + k.clientID
	if err := k.admin(http.MethodGet, path, nil, &current); err != nil {
		return err
	}
	attributes, _ := current["attributes"].(map[string]any)
	if attributes == nil {
		return errors.New("client has no attributes")
	}
	attributes["saml.server.signature"] = fmt.Sprint(p.signResponse)
	attributes["saml.assertion.signature"] = fmt.Sprint(p.signAssertion)
	return k.admin(http.MethodPut, path, current, nil)
}

// signingCertificate は realm の SAML IdP メタデータから署名証明書を PEM で返す。管理者が接続に保存する値と同じ出どころである。
func (k *keycloak) signingCertificate() (string, error) {
	resp, err := get(k.client, k.realmURL()+"/protocol/saml/descriptor")
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	document := etree.NewDocument()
	if err := document.ReadFromBytes(body); err != nil {
		return "", fmt.Errorf("parse SAML descriptor: %w", err)
	}
	for _, descriptor := range document.FindElements("//KeyDescriptor") {
		if use := descriptor.SelectAttrValue("use", "signing"); use != "signing" {
			continue
		}
		if certificate := descriptor.FindElement(".//X509Certificate"); certificate != nil {
			return pemCertificate(strings.TrimSpace(certificate.Text())), nil
		}
	}
	return "", errors.New("SAML descriptor has no signing certificate")
}

func pemCertificate(base64DER string) string {
	var out strings.Builder
	out.WriteString("-----BEGIN CERTIFICATE-----\n")
	compact := strings.Join(strings.Fields(base64DER), "")
	for len(compact) > 64 {
		out.WriteString(compact[:64] + "\n")
		compact = compact[64:]
	}
	out.WriteString(compact + "\n-----END CERTIFICATE-----\n")
	return out.String()
}

func (k *keycloak) admin(method, path string, payload, into any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, k.base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+k.token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := k.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: status %d: %s", method, path, resp.StatusCode, data)
	}
	if into != nil {
		return json.Unmarshal(data, into)
	}
	return nil
}
