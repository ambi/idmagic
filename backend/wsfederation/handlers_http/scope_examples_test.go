package handlers_http_test

// docs/domain/ws-federation/scenarios.feature.md の REQ-WSFEDERATION-001 が宣言する具体例を、
// 実際に発行した API アクセストークンで管理 API を叩いて観測する。
//
// スコープの判定関数だけを呼ぶテストは、判定が正しくても配線されていない実装を素通りさせる。
// ここでは testing_stack が Register と同じ組み立てで建てた経路を通し、拒否のたびに
// 保存先を読み直して、拒否が防いだ効果まで確かめる。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	wsfedScopeRPPath    = "/api/admin/v1/wsfed/relying-parties"
	wsfedScopeEntraPath = "/api/admin/v1/wsfed/entra-federation"
	wsfedScopeWtrealm   = "urn:idmagic:scoped-rp"
	wsfedScopeEntraRP   = "urn:idmagic:entra:contoso.com"
)

func wsfedScopeRPBody() string {
	return `{"wtrealm":"` + wsfedScopeWtrealm + `","reply_urls":["https://scoped-rp.example/wsfed"],` +
		`"claim_policy":{"name_id":{"format":"urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",` +
		`"source_attribute":"user_id"}}}`
}

func wsfedScopeEntraBody() string {
	return `{"domain":"contoso.com","source_anchor_attribute":"object_guid"}`
}

// newWsFedScopeStack は API アクセストークンと WS-Federation の管理 API を配線したスタックを建てる。
//
// Entra フェデレーションの構成は、テナントの全利用者が sourceAnchor の属性を一意に持つことを
// 前提に検証する。属性を持たない利用者がいると、スコープの判定を通った要求も 400 で止まり、
// 「write が構成へ届く」ことを観測できない。
func newWsFedScopeStack(t *testing.T) *stack.Stack {
	t.Helper()
	s := stack.New(t, stack.WithApiTokens(), stack.WithWsFederation())
	for i, id := range []string{stack.AdminUserID, stack.UserID} {
		user, err := s.Users.FindBySub(s.RealmContext(t, tenancydomain.DefaultRealm), id)
		if err != nil || user == nil {
			t.Fatalf("user %s: %v %v", id, user, err)
		}
		guid := []string{"6f9619ff-8b86-d011-b42d-00c04fc964ff", "7a9619ff-8b86-d011-b42d-00c04fc964ff"}[i]
		user.Attributes = map[string]userdomain.AttributeValue{
			"object_guid": {Type: idmdomain.AttributeTypeString, String: &guid},
		}
		s.Users.Seed(user)
	}
	return s
}

// wsfedAdminRequest は 1 本のトークンで WS-Federation の管理 API を 1 回叩く。
func wsfedAdminRequest(s *stack.Stack, method, path, token, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, stack.Issuer+"/realms/"+tenancydomain.DefaultRealm+path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, request)
	return recorder
}

// storedWtrealms は保存先を読み直して、default テナントの RP の wtrealm を返す。
// 応答だけを見るテストは、拒否を書いてから保存する実装を見分けられない。
func storedWtrealms(t *testing.T, s *stack.Stack) []string {
	t.Helper()
	rps, err := s.WsFedRPs.ListAll(s.RealmContext(t, tenancydomain.DefaultRealm), s.TenantID(t, tenancydomain.DefaultRealm))
	if err != nil {
		t.Fatalf("list relying parties: %v", err)
	}
	out := make([]string, 0, len(rps))
	for _, rp := range rps {
		out = append(out, rp.Wtrealm)
	}
	return out
}

func assertWsFedInsufficientScope(t *testing.T, recorder *httptest.ResponseRecorder, required string) {
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

// 「だけ」の観測には、通る側と通らない側の両方が要る。通る側だけを読むと、どのスコープでも
// 全部通す実装と区別できない。通らない側だけを読むと、何も通さない実装と区別できない。
// そこで read について参照が通り登録が通らないこと、write について RP の登録と削除、Entra の
// 構成が通り参照が通らないことを読む。通った変更はそのたびに保存先で読み直す。
//
//spec:covers EX-WSFEDERATION-001-01: wsfed:read は RP の一覧だけに届いて登録に届かず、wsfed:write は RP の登録と削除と Entra フェデレーションの構成に届いてそれぞれが保存先に反映され、一覧には届かないこと。
func TestWsFedAdminOperationsFollowTheGranularScopes(t *testing.T) {
	s := newWsFedScopeStack(t)
	read, _ := s.IssueApiToken(t, tenancydomain.DefaultRealm, apitokendomain.ScopeWsFedRead)
	write, _ := s.IssueApiToken(t, tenancydomain.DefaultRealm, apitokendomain.ScopeWsFedWrite)
	realm := tenancydomain.DefaultRealm

	listed := wsfedAdminRequest(s, http.MethodGet, wsfedScopeRPPath, read, "")
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), stack.WsFedRealm) {
		t.Fatalf("wsfed:read の一覧 status=%d body=%s, want 200 with the seeded RP", listed.Code, listed.Body.String())
	}
	if recorder := wsfedAdminRequest(s, http.MethodPost, wsfedScopeRPPath, read, wsfedScopeRPBody()); recorder.Code != http.StatusForbidden {
		t.Fatalf("wsfed:read の登録 status=%d body=%s, want 403", recorder.Code, recorder.Body.String())
	}

	if recorder := wsfedAdminRequest(s, http.MethodPost, wsfedScopeRPPath, write, wsfedScopeRPBody()); recorder.Code != http.StatusCreated {
		t.Fatalf("wsfed:write の登録 status=%d body=%s, want 201", recorder.Code, recorder.Body.String())
	}
	if stored := storedWtrealms(t, s); !slices.Contains(stored, wsfedScopeWtrealm) {
		t.Fatalf("登録が保存されていない: %v", stored)
	}
	if recorder := wsfedAdminRequest(s, http.MethodPost, wsfedScopeEntraPath, write, wsfedScopeEntraBody()); recorder.Code != http.StatusCreated {
		t.Fatalf("wsfed:write の Entra 構成 status=%d body=%s, want 201", recorder.Code, recorder.Body.String())
	}
	entra, err := s.WsFedRPs.FindByWtrealm(s.RealmContext(t, realm), s.TenantID(t, realm), wsfedScopeEntraRP)
	if err != nil || entra == nil || entra.EntraProfile == nil || entra.EntraProfile.Domain != "contoso.com" {
		t.Fatalf("Entra の構成が保存されていない: rp=%+v err=%v", entra, err)
	}
	// 参照を配らずに変更だけを配ることが、粒度の意味である。
	if recorder := wsfedAdminRequest(s, http.MethodGet, wsfedScopeRPPath, write, ""); recorder.Code != http.StatusForbidden {
		t.Fatalf("wsfed:write の一覧 status=%d body=%s, want 403", recorder.Code, recorder.Body.String())
	}
	deleted := wsfedAdminRequest(s, http.MethodDelete, wsfedScopeRPPath+"?wtrealm="+wsfedScopeWtrealm, write, "")
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("wsfed:write の削除 status=%d body=%s, want 204", deleted.Code, deleted.Body.String())
	}
	if stored := storedWtrealms(t, s); slices.Contains(stored, wsfedScopeWtrealm) {
		t.Fatalf("削除したのに保存先へ残っている: %v", stored)
	}
}

// 具体例は拒否を AccessDeniedError と呼ぶ。契約の 403 は InsufficientScopeError と
// AccessDeniedError の双方を宣言していて、スコープ不足はそのうちの前者になるので、状態コードに
// 加えて型と WWW-Authenticate の要求スコープ名まで読む。変更操作の 3 つそれぞれについて、拒否が
// 保存先を変えていないことを読み直す。応答だけを見ると、403 を書いてから保存する実装を見分けられない。
//
//spec:covers EX-WSFEDERATION-001-02: wsfed:read だけのトークンによる RP の登録と削除と Entra フェデレーションの構成が、wsfed:write を求める 403 insufficient_scope で拒否され、保存先の RP が増えも減りもしないこと。同じ削除が wsfed:write なら通ることを対照にする。
func TestWsFedReadScopeCannotChangeTrustSettings(t *testing.T) {
	s := newWsFedScopeStack(t)
	read, _ := s.IssueApiToken(t, tenancydomain.DefaultRealm, apitokendomain.ScopeWsFedRead)
	write, _ := s.IssueApiToken(t, tenancydomain.DefaultRealm, apitokendomain.ScopeWsFedWrite)
	before := storedWtrealms(t, s)

	assertWsFedInsufficientScope(t, wsfedAdminRequest(s, http.MethodPost, wsfedScopeRPPath, read, wsfedScopeRPBody()), "wsfed:write")
	assertWsFedInsufficientScope(t, wsfedAdminRequest(s, http.MethodPost, wsfedScopeEntraPath, read, wsfedScopeEntraBody()), "wsfed:write")
	assertWsFedInsufficientScope(t, wsfedAdminRequest(s, http.MethodDelete, wsfedScopeRPPath+"?wtrealm="+stack.WsFedRealm, read, ""), "wsfed:write")
	if after := storedWtrealms(t, s); !slices.Equal(after, before) {
		t.Fatalf("拒否されたのに保存先が変わった: before=%v after=%v", before, after)
	}

	// 対照: 同じ削除でも wsfed:write なら通る。これが無いと、削除を配線していないスタックでも
	// 上の拒否はそのまま緑になる。
	if recorder := wsfedAdminRequest(s, http.MethodDelete, wsfedScopeRPPath+"?wtrealm="+stack.WsFedRealm, write, ""); recorder.Code != http.StatusNoContent {
		t.Fatalf("対照の削除 status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if stored := storedWtrealms(t, s); slices.Contains(stored, stack.WsFedRealm) {
		t.Fatalf("対照の削除が保存先に反映されていない: %v", stored)
	}
}
