package realm_http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 応答の本文は、2026-11-12 に公開のエンドポイントから観測した項目の形を、架空のドメインで書いたものである。
const (
	federatedBody = `{"State":3,"UserState":2,"Login":"idmagic-realm-check@contoso.example","NameSpaceType":"Federated",` +
		`"DomainName":"contoso.example","FederationGlobalVersion":-1,` +
		`"AuthURL":"https://idp.example/realms/default/wsfed?username=idmagic-realm-check%40contoso.example&wa=wsignin1.0&wtrealm=urn%3afederation%3aMicrosoftOnline&wctx=",` +
		`"FederationBrandName":"Contoso","CloudInstanceName":"microsoftonline.com","CloudInstanceIssuerUri":"urn:federation:MicrosoftOnline"}`
	managedBody = `{"State":4,"UserState":1,"Login":"idmagic-realm-check@contoso.example","NameSpaceType":"Managed",` +
		`"DomainName":"contoso.example","FederationBrandName":"Contoso","CloudInstanceName":"microsoftonline.com","CloudInstanceIssuerUri":"urn:federation:MicrosoftOnline"}`
)

//spec:covers REQ-WSFEDERATION-001: レルム探索が、固定のローカル部とドメインを符号化した login と json=1 を送り、NameSpaceType と AuthURL を読み、200 でない応答、上限を超える本文、読めない本文、リダイレクトを失敗として返すこと。
func TestRealmDiscoveryReadsTheMicrosoftResponse(t *testing.T) {
	var gotQuery string
	responses := map[string]func(http.ResponseWriter){
		"/federated": func(w http.ResponseWriter) { _, _ = w.Write([]byte(federatedBody)) },
		"/managed":   func(w http.ResponseWriter) { _, _ = w.Write([]byte(managedBody)) },
		"/error":     func(w http.ResponseWriter) { w.WriteHeader(http.StatusServiceUnavailable) },
		"/large": func(w http.ResponseWriter) {
			_, _ = w.Write([]byte(`{"NameSpaceType":"Managed","padding":"` + strings.Repeat("x", maxResponseBytes) + `"}`))
		},
		"/garbled":  func(w http.ResponseWriter) { _, _ = w.Write([]byte(`<html>sign in</html>`)) },
		"/redirect": func(w http.ResponseWriter) { w.Header().Set("Location", "/managed"); w.WriteHeader(http.StatusFound) },
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		responses[r.URL.Path](w)
	}))
	t.Cleanup(server.Close)
	discover := func(path, domain string) (string, string, error) {
		t.Helper()
		client, err := New(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		realm, err := client.Discover(context.Background(), domain)
		return realm.NamespaceType, realm.AuthURL, err
	}

	namespace, authURL, err := discover("/federated", "contoso.example")
	if err != nil || namespace != "Federated" || !strings.HasPrefix(authURL, "https://idp.example/realms/default/wsfed?") {
		t.Fatalf("federated = (%q, %q, %v)", namespace, authURL, err)
	}
	if gotQuery != "json=1&login=idmagic-realm-check%40contoso.example" {
		t.Fatalf("query = %q", gotQuery)
	}
	// ドメインは構成済みの値でも、URL の問い合わせ文字列の区切りを持ち込ませない。
	if _, _, err := discover("/managed", "contoso.example&json=0"); err != nil || gotQuery != "json=1&login=idmagic-realm-check%40contoso.example%26json%3D0" {
		t.Fatalf("escaped query = %q, %v", gotQuery, err)
	}
	if namespace, authURL, err := discover("/managed", "contoso.example"); err != nil || namespace != "Managed" || authURL != "" {
		t.Fatalf("managed = (%q, %q, %v)", namespace, authURL, err)
	}
	for _, path := range []string{"/error", "/large", "/garbled", "/redirect"} {
		if namespace, _, err := discover(path, "contoso.example"); err == nil {
			t.Fatalf("%s = %q, want an error", path, namespace)
		}
	}
}

//spec:covers REQ-WSFEDERATION-001: 探索の URL の設定が、https か loopback の http だけを受け、空なら探索を無効にすること。
func TestNewRealmDiscoveryAcceptsOnlyHTTPSOrLoopback(t *testing.T) {
	for _, raw := range []string{
		"https://login.microsoftonline.com/getuserrealm.srf",
		"http://127.0.0.1:4010/getuserrealm.srf",
		"http://localhost:4010/getuserrealm.srf",
		"http://[::1]:4010/getuserrealm.srf",
	} {
		if _, err := New(raw); err != nil {
			t.Fatalf("New(%q) = %v", raw, err)
		}
	}
	for _, raw := range []string{
		"http://login.microsoftonline.com/getuserrealm.srf",
		"ftp://login.microsoftonline.com/",
		"https://user@login.microsoftonline.com/",
		"https://login.microsoftonline.com/getuserrealm.srf?json=0",
		"login.microsoftonline.com",
	} {
		if _, err := New(raw); err == nil {
			t.Fatalf("New(%q) accepted", raw)
		}
	}
	disabled, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := disabled.Discover(context.Background(), "contoso.example"); err == nil {
		t.Fatal("disabled discovery returned a realm")
	}
}
