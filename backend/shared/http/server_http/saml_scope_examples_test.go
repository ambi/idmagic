package server_http

// docs/contexts/saml/scenarios.feature.md の REQ-SAML-005 が宣言する具体例を、
// 実際に発行した API アクセストークンで管理 API を叩いて観測する。
//
// スコープの判定関数だけを呼ぶテストは、判定が正しくても配線されていない実装を素通りさせる。
// ここでは Register が組み立てた経路をそのまま通し、拒否のたびに保存先を読み直して、
// 拒否が防いだ効果まで確かめる。スタックは api_token_standards_test.go の `apiTokenStack`
// を使う。トークンの発行と検証の組み立ては製品と同じでなければ、入口を通したという事実
// そのものが根拠にならない。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	samlScopeSPPath     = "/api/admin/v1/saml/service-providers"
	samlScopeSPEntityID = "https://scoped-sp.example.com"
)

func samlScopeSPBody() string {
	return `{"entity_id":"` + samlScopeSPEntityID + `","acs_urls":["` + samlScopeSPEntityID + `/acs"],` +
		`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",` +
		`"source_attribute":"user_id"}}}`
}

// samlRequest は 1 本のトークンで SAML 管理 API を 1 回叩く。
func (s *apiTokenStack) samlRequest(method, token, body, query string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, apiTokenIssuer+"/realms/"+tenancydomain.DefaultRealm+samlScopeSPPath+query, reader)
	request.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return s.do(request)
}

// storedServiceProviders は保存先を読み直して entityID の一覧を返す。応答だけを見るテストは、
// 拒否を書いてから保存する実装を見分けられない。
func (s *apiTokenStack) storedServiceProviders(t *testing.T) []string {
	t.Helper()
	sps, err := s.samlSPs.ListAll(s.realmContext(t, tenancydomain.DefaultRealm), s.tenantID(t, tenancydomain.DefaultRealm))
	if err != nil {
		t.Fatalf("list service providers: %v", err)
	}
	out := make([]string, 0, len(sps))
	for _, sp := range sps {
		out = append(out, sp.EntityID)
	}
	return out
}

// EX-SAML-005-01: `saml:read` はサービスプロバイダーの参照だけを許し、`saml:write` は登録と
// 削除だけを許す。
//
// 「だけ」の観測には 4 通り要る。read で参照が通ること、read で登録が通らないこと、
// write で登録と削除が通ること、write で参照が通らないことである。通る側だけを読むと、
// どのスコープでも全部通す実装と区別できない。通らない側だけを読むと、何も通さない実装と
// 区別できない。
func TestSamlServiceProviderOperationsFollowTheGranularScopes(t *testing.T) {
	stack := newApiTokenStack(t)
	read, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeSamlRead)
	write, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeSamlWrite)

	// saml:read は参照へ届く。
	if recorder := stack.samlRequest(http.MethodGet, read, "", ""); recorder.Code != http.StatusOK {
		t.Fatalf("saml:read の参照 status=%d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	// saml:read は登録へ届かない。
	if recorder := stack.samlRequest(http.MethodPost, read, samlScopeSPBody(), ""); recorder.Code != http.StatusForbidden {
		t.Fatalf("saml:read の登録 status=%d body=%s, want 403", recorder.Code, recorder.Body.String())
	}

	// saml:write は登録へ届き、保存される。
	created := stack.samlRequest(http.MethodPost, write, samlScopeSPBody(), "")
	if created.Code != http.StatusCreated {
		t.Fatalf("saml:write の登録 status=%d body=%s, want 201", created.Code, created.Body.String())
	}
	if stored := stack.storedServiceProviders(t); len(stored) != 1 || stored[0] != samlScopeSPEntityID {
		t.Fatalf("登録が保存されていない: %v", stored)
	}
	// saml:write は参照へ届かない。参照を配らずに変更だけを配ることが、粒度の意味である。
	if recorder := stack.samlRequest(http.MethodGet, write, "", ""); recorder.Code != http.StatusForbidden {
		t.Fatalf("saml:write の参照 status=%d body=%s, want 403", recorder.Code, recorder.Body.String())
	}
	// saml:write は削除へ届き、保存先から消える。
	deleted := stack.samlRequest(http.MethodDelete, write, "", samlScopeDeleteQuery())
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("saml:write の削除 status=%d body=%s, want 204", deleted.Code, deleted.Body.String())
	}
	if stored := stack.storedServiceProviders(t); len(stored) != 0 {
		t.Fatalf("削除したのに保存先へ残っている: %v", stored)
	}
}

// samlScopeDeleteQuery は削除に要る entity_id クエリを返す。entityID は URI なので、
// path param ではなくクエリで渡す契約になっている。
func samlScopeDeleteQuery() string {
	return "?entity_id=" + samlScopeSPEntityID
}

// EX-SAML-005-02: `saml:read` だけで変更操作をリクエストすると、操作は拒否される。
//
// 契約 `RegisterSamlServiceProvider` と `DeleteSamlServiceProvider` は 403 の本文として
// `InsufficientScopeError` と `AccessDeniedError` を宣言している。スコープ不足はそのうちの
// 前者になるので、状態コードに加えて型と `WWW-Authenticate` の要求スコープ名まで読む。
// そして登録と削除のどちらについても、拒否が保存先を変えていないことを読み直す。
// 応答だけを見ると、403 を書いてから保存する実装を見分けられない。
func TestSamlReadScopeCannotChangeServiceProviders(t *testing.T) {
	stack := newApiTokenStack(t)
	read, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeSamlRead)
	write, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeSamlWrite)

	// 登録は拒否され、何も保存されない。
	refusedCreate := stack.samlRequest(http.MethodPost, read, samlScopeSPBody(), "")
	assertInsufficientScope(t, refusedCreate, "saml:write")
	if stored := stack.storedServiceProviders(t); len(stored) != 0 {
		t.Fatalf("拒否されたのに登録されている: %v", stored)
	}

	// 削除も拒否され、既存の登録は残る。先に saml:write で 1 件用意しておく。
	if recorder := stack.samlRequest(http.MethodPost, write, samlScopeSPBody(), ""); recorder.Code != http.StatusCreated {
		t.Fatalf("前提の登録 status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	refusedDelete := stack.samlRequest(http.MethodDelete, read, "", samlScopeDeleteQuery())
	assertInsufficientScope(t, refusedDelete, "saml:write")
	if stored := stack.storedServiceProviders(t); len(stored) != 1 {
		t.Fatalf("拒否されたのに削除されている: %v", stored)
	}

	// 対照: 同じ削除でも saml:write なら通る。これが無いと、削除を配線していないスタックでも
	// 上の拒否はそのまま緑になる。
	if recorder := stack.samlRequest(http.MethodDelete, write, "", samlScopeDeleteQuery()); recorder.Code != http.StatusNoContent {
		t.Fatalf("対照の削除 status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func assertInsufficientScope(t *testing.T, recorder *httptest.ResponseRecorder, required string) {
	t.Helper()
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", recorder.Code, recorder.Body.String())
	}
	var problem struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("problem body %s: %v", recorder.Body.String(), err)
	}
	if problem.Type != "urn:idmagic:error:insufficient_scope" {
		t.Fatalf("problem type=%q, want insufficient_scope", problem.Type)
	}
	if got := recorder.Header().Get("WWW-Authenticate"); !strings.Contains(got, `scope="`+required+`"`) {
		t.Fatalf("WWW-Authenticate=%q, want the %q scope", got, required)
	}
}
