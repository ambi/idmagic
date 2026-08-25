package domain

import (
	"net/url"
	"testing"
)

// FuzzParseClientIDMetadataDocument は、取得先の URL と一致しない client_id を持つ文書が
// クライアント定義として通らないことを表明する。
//
// この文書は client_id が指す URL から取得したもので、内容は攻撃者が用意できる。
// client_id と取得 URL の一致検査が緩むと、攻撃者は別のクライアントになりすませる。
func FuzzParseClientIDMetadataDocument(f *testing.F) {
	const requestURL = "https://client.example/id"

	f.Add([]byte(`{"client_id":"https://client.example/id","client_name":"c","redirect_uris":["https://client.example/cb"]}`))
	f.Add([]byte(`{"client_id":"https://attacker.example/id","client_name":"c","redirect_uris":["https://x/cb"]}`))
	f.Add([]byte(`{"client_id":"https://client.example/id","client_name":"c","redirect_uris":[]}`))
	f.Add([]byte(`{"client_id":"https://client.example/id","client_name":"c","redirect_uris":["x"],"token_endpoint_auth_method":"private_key_jwt"}`))
	f.Add([]byte(`{`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 64*1024 {
			return
		}
		client, err := ParseClientIDMetadataDocument(raw, requestURL)
		if err != nil {
			if client != nil {
				t.Fatalf("ParseClientIDMetadataDocument returned %+v together with an error", client)
			}
			return
		}
		if client.ClientID != requestURL {
			t.Fatalf("accepted a document whose client_id %q differs from the fetch URL %q",
				client.ClientID, requestURL)
		}
		if len(client.RedirectURIs) == 0 {
			t.Fatalf("accepted a document without redirect_uris: %q", raw)
		}
		if client.TokenEndpointAuthMethod != AuthMethodNone {
			t.Fatalf("accepted a document declaring auth method %q", client.TokenEndpointAuthMethod)
		}
	})
}

// FuzzIsClientIDMetadataDocumentURL は、CIMD 識別子と認めるのが https かつパスを持ち
// userinfo と fragment を持たない URL に限ることを表明する。
// 生文字列に対する接頭辞判定へ退行すると破れる。
func FuzzIsClientIDMetadataDocumentURL(f *testing.F) {
	f.Add("https://client.example/id")
	f.Add("http://client.example/id")
	f.Add("https://client.example")
	f.Add("https://user@client.example/id")
	f.Add("https://client.example/id#frag")
	f.Add("HTTPS://client.example/id")
	f.Add("https:/\\/evil.example/id")

	f.Fuzz(func(t *testing.T, clientID string) {
		if len(clientID) > 8192 {
			return
		}
		if !IsClientIDMetadataDocumentURL(clientID) {
			return
		}
		parsed, err := url.Parse(clientID)
		if err != nil {
			t.Fatalf("accepted a client_id that does not parse: %q", clientID)
		}
		if parsed.Scheme != "https" {
			t.Fatalf("accepted a non-https client_id: %q", clientID)
		}
		if parsed.Hostname() == "" || parsed.Path == "" {
			t.Fatalf("accepted a client_id without host or path: %q", clientID)
		}
		if parsed.User != nil {
			t.Fatalf("accepted a client_id carrying userinfo: %q", clientID)
		}
		if parsed.Fragment != "" {
			t.Fatalf("accepted a client_id carrying a fragment: %q", clientID)
		}
	})
}
