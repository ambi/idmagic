package server_http

// docs/contexts/oauth2/scenarios.feature.md の REQ-OAUTH2-003 が宣言する具体例を、
// 実際に発行した API アクセストークンで OAuth2 の管理 API を叩いて観測する。
//
// 判定関数 `requireAdminApiTokenScope` だけを呼ぶテストは、判定が正しくても配線されて
// いない実装を素通しさせる。ここでは Register が組み立てた経路をそのまま通し、拒否の
// たびに保存先を読み直して、拒否が防いだ効果まで確かめる。スタックは
// api_token_standards_test.go の `apiTokenStack` を使う。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	oauthScopeClientsPath     = "/api/admin/v1/clients"
	oauthScopeDetailTypesPath = "/api/admin/v1/authorization-detail-types"
	oauthScopeMcpPath         = "/api/admin/v1/mcp-resource-servers"

	oauthScopeNewClientName = "scoped-client"
	oauthScopeNewDetailType = "scoped_payment_initiation"
	oauthScopeNewMcpName    = "scoped-mcp"
)

func oauthScopeClientBody() string {
	return `{"client_name":"` + oauthScopeNewClientName + `","client_type":"confidential",` +
		`"redirect_uris":["https://scoped.example.com/callback"],` +
		`"token_endpoint_auth_method":"client_secret_basic"}`
}

func oauthScopeDetailTypeBody() string {
	return `{"type":"` + oauthScopeNewDetailType + `","description":"scoped",` +
		`"display_template":"Scoped payment","schema":{"rules":[{"name":"actions","semantics":"set"}]}}`
}

func oauthScopeMcpBody() string {
	return `{"resource":"https://scoped-mcp.example.com","name":"` + oauthScopeNewMcpName + `"}`
}

// oauthAdminRequest は 1 本のトークンで OAuth2 の管理 API を 1 回叩く。
func (s *apiTokenStack) oauthAdminRequest(method, path, token, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(
		method, apiTokenIssuer+"/realms/"+tenancydomain.DefaultRealm+path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return s.do(request)
}

// storedClientNames などは保存先を読み直して名前の一覧を返す。応答だけを見るテストは、
// 拒否を書いてから保存する実装を見分けられない。
func (s *apiTokenStack) storedClientNames(t *testing.T) []string {
	t.Helper()
	clients, err := s.oauthClients.FindAll(s.realmContext(t, tenancydomain.DefaultRealm), s.tenantID(t, tenancydomain.DefaultRealm))
	if err != nil {
		t.Fatalf("list clients: %v", err)
	}
	names := make([]string, 0, len(clients))
	for _, client := range clients {
		name := client.ClientID
		if client.ClientName != nil {
			name = *client.ClientName
		}
		names = append(names, name)
	}
	return names
}

func (s *apiTokenStack) storedDetailTypes(t *testing.T) []string {
	t.Helper()
	types, err := s.authzDetailTypes.ListAll(s.realmContext(t, tenancydomain.DefaultRealm), s.tenantID(t, tenancydomain.DefaultRealm))
	if err != nil {
		t.Fatalf("list authorization detail types: %v", err)
	}
	names := make([]string, 0, len(types))
	for _, detailType := range types {
		names = append(names, detailType.Type)
	}
	return names
}

func (s *apiTokenStack) storedMcpResourceServers(t *testing.T) []string {
	t.Helper()
	servers, err := s.mcpResourceServers.ListAll(s.realmContext(t, tenancydomain.DefaultRealm), s.tenantID(t, tenancydomain.DefaultRealm))
	if err != nil {
		t.Fatalf("list mcp resource servers: %v", err)
	}
	names := make([]string, 0, len(servers))
	for _, server := range servers {
		names = append(names, server.Name)
	}
	return names
}

// EX-OAUTH2-003-01: `oauth-clients:read` は OAuth2 クライアントの参照だけを、
// `authorization-detail-types:write` は認可詳細タイプの変更だけを、
// `mcp-resource-servers:read` は MCP リソースサーバーの参照だけを許す。
//
// 具体例は `Then` を 3 つ並べていて、resource ごとに別のスコープを名指ししている。
// 「だけ」の観測には resource あたり 2 方向が要る。許す側だけを読むとどのスコープでも
// 全部通す実装と、拒む側だけを読むと何も通さない実装と区別できない。
func TestOAuth2AdminOperationsFollowTheGranularScopes(t *testing.T) {
	stack := newApiTokenStack(t)
	clientsRead, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeOAuthClientsRead)
	detailTypesWrite, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeAuthorizationDetailTypesWrite)
	mcpRead, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeMcpResourceServersRead)

	// oauth-clients:read は参照へ届く。
	if recorder := stack.oauthAdminRequest(
		http.MethodGet, oauthScopeClientsPath, clientsRead, "",
	); recorder.Code != http.StatusOK {
		t.Fatalf("oauth-clients:read の参照 status=%d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	// oauth-clients:read は登録へ届かず、保存先も変わらない。
	before := stack.storedClientNames(t)
	if recorder := stack.oauthAdminRequest(
		http.MethodPost, oauthScopeClientsPath, clientsRead, oauthScopeClientBody(),
	); recorder.Code != http.StatusForbidden {
		t.Fatalf("oauth-clients:read の登録 status=%d body=%s, want 403", recorder.Code, recorder.Body.String())
	}
	if after := stack.storedClientNames(t); len(after) != len(before) {
		t.Fatalf("参照スコープの登録が保存された: before=%v after=%v", before, after)
	}

	// authorization-detail-types:write は変更へ届き、保存される。
	created := stack.oauthAdminRequest(
		http.MethodPost, oauthScopeDetailTypesPath, detailTypesWrite, oauthScopeDetailTypeBody())
	if created.Code != http.StatusCreated {
		t.Fatalf("authorization-detail-types:write の登録 status=%d body=%s, want 201",
			created.Code, created.Body.String())
	}
	if stored := stack.storedDetailTypes(t); len(stored) != 1 || stored[0] != oauthScopeNewDetailType {
		t.Fatalf("登録が保存されていない: %v", stored)
	}
	// authorization-detail-types:write は参照へ届かない。参照を配らずに変更だけを配ることが、
	// 粒度の意味である。
	if recorder := stack.oauthAdminRequest(
		http.MethodGet, oauthScopeDetailTypesPath, detailTypesWrite, "",
	); recorder.Code != http.StatusForbidden {
		t.Fatalf("authorization-detail-types:write の参照 status=%d body=%s, want 403",
			recorder.Code, recorder.Body.String())
	}

	// mcp-resource-servers:read は参照へ届く。
	if recorder := stack.oauthAdminRequest(
		http.MethodGet, oauthScopeMcpPath, mcpRead, "",
	); recorder.Code != http.StatusOK {
		t.Fatalf("mcp-resource-servers:read の参照 status=%d body=%s, want 200",
			recorder.Code, recorder.Body.String())
	}
	// mcp-resource-servers:read は登録へ届かず、保存先も変わらない。
	if recorder := stack.oauthAdminRequest(
		http.MethodPost, oauthScopeMcpPath, mcpRead, oauthScopeMcpBody(),
	); recorder.Code != http.StatusForbidden {
		t.Fatalf("mcp-resource-servers:read の登録 status=%d body=%s, want 403",
			recorder.Code, recorder.Body.String())
	}
	if stored := stack.storedMcpResourceServers(t); len(stored) != 0 {
		t.Fatalf("参照スコープの登録が保存された: %v", stored)
	}
}

// EX-OAUTH2-003-02: `oauth-clients:read` だけで OAuth2 クライアントの変更を要求すると拒否される。
//
// 契約 `CreateAdminOAuth2Client` などは 403 の本文として `InsufficientScopeError` を宣言して
// いる。状態コードに加えて `WWW-Authenticate` が要求スコープ名を名指ししていることまで読み、
// 変更 3 種 (登録、更新、削除) のどれについても保存先が変わっていないことを読み直す。
func TestOAuth2ReadScopeCannotChangeOAuth2Clients(t *testing.T) {
	stack := newApiTokenStack(t)
	clientsRead, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeOAuthClientsRead)
	before := stack.storedClientNames(t)

	for _, attempt := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "登録", method: http.MethodPost, path: oauthScopeClientsPath, body: oauthScopeClientBody()},
		{name: "更新", method: http.MethodPatch, path: oauthScopeClientsPath + "/" + apiTokenRSClientID, body: `{"client_name":"renamed"}`},
		{name: "削除", method: http.MethodDelete, path: oauthScopeClientsPath + "/" + apiTokenRSClientID},
	} {
		recorder := stack.oauthAdminRequest(attempt.method, attempt.path, clientsRead, attempt.body)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("%s status=%d body=%s, want 403", attempt.name, recorder.Code, recorder.Body.String())
		}
		if challenge := recorder.Header().Get("WWW-Authenticate"); !strings.Contains(challenge, "oauth-clients:write") {
			t.Fatalf("%s の WWW-Authenticate が要求スコープを名指ししない: %q", attempt.name, challenge)
		}
	}

	// 3 回の拒否のあとも、クライアントの一覧は 1 件も増減していない。
	if after := stack.storedClientNames(t); len(after) != len(before) {
		t.Fatalf("拒否された変更が保存先を動かした: before=%v after=%v", before, after)
	}
}

// EX-OAUTH2-003-03: 別 resource のスコープで操作を要求すると拒否される。
//
// スコープの粒度は resource で切ってあるので、`authorization-detail-types:write` を持って
// いても OAuth2 クライアントには届かず、`oauth-clients:write` を持っていても認可詳細タイプ
// には届かない。両方向を読むのは、片方向だけでは「この resource は誰にも書けない」実装と
// 区別できないためである。対照として、正しいスコープなら同じ要求が通ることも読む。
func TestOAuth2ScopeOfAnotherResourceCannotReachTheOperation(t *testing.T) {
	stack := newApiTokenStack(t)
	clientsWrite, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeOAuthClientsWrite)
	detailTypesWrite, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeAuthorizationDetailTypesWrite)

	// 認可詳細タイプのスコープは OAuth2 クライアントの登録へ届かない。
	before := stack.storedClientNames(t)
	refused := stack.oauthAdminRequest(
		http.MethodPost, oauthScopeClientsPath, detailTypesWrite, oauthScopeClientBody())
	if refused.Code != http.StatusForbidden {
		t.Fatalf("別 resource のスコープでのクライアント登録 status=%d body=%s, want 403",
			refused.Code, refused.Body.String())
	}
	if after := stack.storedClientNames(t); len(after) != len(before) {
		t.Fatalf("別 resource のスコープでの登録が保存された: before=%v after=%v", before, after)
	}

	// OAuth2 クライアントのスコープは認可詳細タイプの登録へ届かない。
	refused = stack.oauthAdminRequest(
		http.MethodPost, oauthScopeDetailTypesPath, clientsWrite, oauthScopeDetailTypeBody())
	if refused.Code != http.StatusForbidden {
		t.Fatalf("別 resource のスコープでの認可詳細タイプ登録 status=%d body=%s, want 403",
			refused.Code, refused.Body.String())
	}
	if stored := stack.storedDetailTypes(t); len(stored) != 0 {
		t.Fatalf("別 resource のスコープでの登録が保存された: %v", stored)
	}

	// 対照: それぞれ正しいスコープなら同じ要求が通る。拒否したのが resource の食い違いで
	// あって、要求そのものではないと示す。
	if allowed := stack.oauthAdminRequest(
		http.MethodPost, oauthScopeClientsPath, clientsWrite, oauthScopeClientBody(),
	); allowed.Code != http.StatusCreated {
		t.Fatalf("前提が壊れている: oauth-clients:write の登録が status=%d body=%s",
			allowed.Code, allowed.Body.String())
	}
	if allowed := stack.oauthAdminRequest(
		http.MethodPost, oauthScopeDetailTypesPath, detailTypesWrite, oauthScopeDetailTypeBody(),
	); allowed.Code != http.StatusCreated {
		t.Fatalf("前提が壊れている: authorization-detail-types:write の登録が status=%d body=%s",
			allowed.Code, allowed.Body.String())
	}
}
