package cimd_http

import (
	"os"
	"slices"
	"testing"

	clientdomain "github.com/ambi/idmagic/backend/oauth2/internal/client/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// 公開のクライアントが client_id として公開している Client ID Metadata Document を実際に取った記録を再生する。
// 期待値は draft-ietf-oauth-client-id-metadata-document-02 と RFC 7591 の文面から書く。
// 文書の client_id は取得元の URL と一致し（§4.1）、client_name と redirect_uris を持ち、
// 認可サーバーは理解できないメタデータを無視する（RFC 7591 §2）。

func replayedFetch(t *testing.T, file, clientIDURL string) *clientdomain.OAuth2Client {
	t.Helper()
	data, err := os.ReadFile("testdata/" + file)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := testing_httprecord.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	server := testing_httprecord.Replay{}.Serve(t, rec.Exchange(t, "metadata-document"))
	client, err := newFetcherWithClient(testing_httprecord.RoutedClient(server)).Fetch(t.Context(), clientIDURL)
	if err != nil {
		t.Fatalf("Fetch(%s): %v", clientIDURL, err)
	}
	return client
}

type wantPublicClient struct {
	clientID, clientName string
	redirectURIs         []string
	grantTypes           []spec.GrantType
}

// requirePublicClient は、文書が token_endpoint_auth_method に none を宣言した公開クライアントとして、
// 文書の client_id、client_name、redirect_uris、grant_types のまま解決されたことを確かめる。
func requirePublicClient(t *testing.T, client *clientdomain.OAuth2Client, want wantPublicClient) {
	t.Helper()
	if client.ClientID != want.clientID {
		t.Errorf("ClientID = %q, want %q", client.ClientID, want.clientID)
	}
	if client.ClientName == nil || *client.ClientName != want.clientName {
		t.Errorf("ClientName = %v, want %q", client.ClientName, want.clientName)
	}
	if !slices.Equal(client.RedirectURIs, want.redirectURIs) {
		t.Errorf("RedirectURIs = %q, want %q", client.RedirectURIs, want.redirectURIs)
	}
	if !slices.Equal(client.GrantTypes, want.grantTypes) {
		t.Errorf("GrantTypes = %q, want %q", client.GrantTypes, want.grantTypes)
	}
	if client.ClientType != spec.ClientPublic || client.TokenEndpointAuthMethod != "none" {
		t.Errorf("client type = %s with %s, want a public client with none", client.ClientType, client.TokenEndpointAuthMethod)
	}
}

// VS Code の文書は、草案が定めない application_type、logo_uri、client_uri と、device_code の付与、
// ポートと末尾のスラッシュを持つループバックのリダイレクト URI を持つ。
//
//spec:covers CIMD02-STRUCTURE: 実在のクライアントの文書を、草案にないメタデータがあっても解決する。
//spec:covers CIMD02-CLIENT-ID-MATCH: 実在の文書の client_id は取得元 URL と一致する。
//spec:covers EX-OAUTH2-017-01: クライアント所有のメタデータ URL の取得と検証に成功し、その client_id を名乗るクライアントが組み立てられる。
func TestFetcherResolvesRecordedVSCodeDocument(t *testing.T) {
	const clientIDURL = "https://vscode.dev/oauth/client-metadata.json"
	requirePublicClient(t, replayedFetch(t, "vscode.json", clientIDURL), wantPublicClient{
		clientID:     clientIDURL,
		clientName:   "Visual Studio Code",
		redirectURIs: []string{"http://127.0.0.1:33418/", "https://vscode.dev/redirect"},
		grantTypes:   []spec.GrantType{"authorization_code", "refresh_token", "urn:ietf:params:oauth:grant-type:device_code"},
	})
}

// Claude Code の文書は、ポートのない localhost と 127.0.0.1 のリダイレクト URI を持つ。
//
//spec:covers CIMD02-STRUCTURE: 実在のクライアントの文書を解決する。
func TestFetcherResolvesRecordedClaudeCodeDocument(t *testing.T) {
	const clientIDURL = "https://claude.ai/oauth/claude-code-client-metadata"
	requirePublicClient(t, replayedFetch(t, "claude-code.json", clientIDURL), wantPublicClient{
		clientID:     clientIDURL,
		clientName:   "Claude Code",
		redirectURIs: []string{"http://localhost/callback", "http://127.0.0.1/callback"},
		grantTypes:   []spec.GrantType{"authorization_code", "refresh_token"},
	})
}

// Claude の文書は、RFC 7523 の jwt-bearer の付与を持つ。
//
//spec:covers CIMD02-STRUCTURE: 実在のクライアントの文書を解決する。
func TestFetcherResolvesRecordedClaudeDocument(t *testing.T) {
	const clientIDURL = "https://claude.ai/oauth/mcp-oauth-client-metadata"
	requirePublicClient(t, replayedFetch(t, "claude.json", clientIDURL), wantPublicClient{
		clientID:     clientIDURL,
		clientName:   "Claude",
		redirectURIs: []string{"https://claude.ai/api/mcp/auth_callback"},
		grantTypes:   []spec.GrantType{"authorization_code", "refresh_token", "urn:ietf:params:oauth:grant-type:jwt-bearer"},
	})
}
