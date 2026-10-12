package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const workloadBundlesPath = "/realms/default/api/admin/v1/workload-identity/trust-bundles"

// インライン JWKS により、正式な配線を使いながら外部ネットワークに依存しない。
const workloadBundleInput = `{"name":"cluster","trust_domain":"issuer.example","issuer":"https://issuer.example","accepted_audiences":["idmagic"],"jwks":{"keys":[{"kid":"workload-key","kty":"RSA","n":"AQAB","e":"AQAB"}]}}`

func workloadContractRequests(t *testing.T) func(sub, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	e := newWorkloadIdentityHandler(t)
	csrf, cookie := workloadAdminCSRF(t, e)
	return func(sub, method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "http://idp.test")
		request.Header.Set("X-Csrf-Token", csrf)
		request.Header.Set("X-Demo-Sub", sub)
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)
		return response
	}
}

func workloadContractJSON(t *testing.T, response *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status=%d body=%s, want %d", response.Code, response.Body.String(), status)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func assertWorkloadBundle(t *testing.T, body map[string]any, name, status string) string {
	t.Helper()
	want := map[string]any{
		"tenant_id": tenancydomain.DefaultTenantID, "name": name, "status": status,
		"trust_domain": "issuer.example", "issuer": "https://issuer.example",
		"has_inline_jwks": true, "accepted_audiences": []any{"idmagic"},
		"max_subject_token_ttl_seconds": float64(3600),
	}
	for key, value := range want {
		if !reflect.DeepEqual(body[key], value) {
			t.Errorf("%s=%v, want %v", key, body[key], value)
		}
	}
	id, ok := body["id"].(string)
	if !ok || id == "" {
		t.Fatalf("id=%v, want a generated identifier", body["id"])
	}
	created, ok := body["created_at"].(string)
	if _, err := time.Parse(time.RFC3339Nano, created); !ok || err != nil {
		t.Fatalf("created_at=%v, want a timestamp", body["created_at"])
	}
	if _, exposed := body["jwks"]; exposed {
		t.Fatal("bundle response exposes inline JWKS")
	}
	return id
}

func assertWorkloadNoContent(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("status=%d body=%q, want empty 204", response.Code, response.Body.String())
	}
}

//verify:operation RegisterWorkloadTrustBundle: JSON 要求を受けた 201 本文の設定と既定 TTL、enabled を照合する。
//verify:operation GetWorkloadTrustBundle: 登録後の GET の 200 本文が保存した設定と状態を返すことを照合する。
//verify:operation UpdateWorkloadTrustBundle: PATCH の 200 本文は名前だけを変え、発行者とトラストドメインを維持する。
//verify:operation DisableWorkloadTrustBundle: 無効化は空の 204 を返し、GET の本文が disabled になる。
//verify:operation EnableWorkloadTrustBundle: 再有効化は空の 204 を返し、GET の本文が enabled に戻る。
//verify:operation DeleteWorkloadTrustBundle: 削除は空の 204 を返し、一覧から設定が消える。
//verify:operation RefreshWorkloadTrustBundleJWKS: 再取得の 200 本文の reachable、鍵数とキャッシュ時刻を照合し、GET でも保存時刻を確認する。
func TestWorkloadTrustBundleHTTPContracts(t *testing.T) {
	send := workloadContractRequests(t)
	created := workloadContractJSON(t, send("admin", http.MethodPost, workloadBundlesPath, workloadBundleInput), http.StatusCreated)
	id := assertWorkloadBundle(t, created, "cluster", "enabled")
	path := workloadBundlesPath + "/" + id
	got := workloadContractJSON(t, send("admin", http.MethodGet, path, ""), http.StatusOK)
	if assertWorkloadBundle(t, got, "cluster", "enabled") != id {
		t.Fatal("GET changed bundle identifier")
	}
	updated := workloadContractJSON(t, send("admin", http.MethodPatch, path, `{"name":"renamed"}`), http.StatusOK)
	if assertWorkloadBundle(t, updated, "renamed", "enabled") != id {
		t.Fatal("PATCH changed bundle identifier")
	}
	assertWorkloadNoContent(t, send("admin", http.MethodPost, path+"/disable", ""))
	assertWorkloadBundle(t, workloadContractJSON(t, send("admin", http.MethodGet, path, ""), http.StatusOK), "renamed", "disabled")
	assertWorkloadNoContent(t, send("admin", http.MethodPost, path+"/enable", ""))
	assertWorkloadBundle(t, workloadContractJSON(t, send("admin", http.MethodGet, path, ""), http.StatusOK), "renamed", "enabled")
	refresh := workloadContractJSON(t, send("admin", http.MethodPost, path+"/refresh", ""), http.StatusOK)
	if refresh["reachable"] != true || refresh["key_count"] != float64(1) || refresh["jwks_cached_at"] == nil {
		t.Fatalf("refresh=%v, want reachable single key and cache timestamp", refresh)
	}
	got = workloadContractJSON(t, send("admin", http.MethodGet, path, ""), http.StatusOK)
	if got["jwks_cached_at"] != refresh["jwks_cached_at"] {
		t.Fatalf("GET cache timestamp=%v, want %v", got["jwks_cached_at"], refresh["jwks_cached_at"])
	}
	assertWorkloadNoContent(t, send("admin", http.MethodDelete, path, ""))
	listing := workloadContractJSON(t, send("admin", http.MethodGet, workloadBundlesPath, ""), http.StatusOK)
	if !reflect.DeepEqual(listing["trust_bundles"], []any{}) {
		t.Fatalf("deleted bundle remains: %v", listing)
	}
}

//verify:operation CreateAgentWorkloadBinding: 作成要求に対する 201 本文の主体、Agent、所属、enabled と時刻を照合する。
//verify:operation ListAgentWorkloadBindings: GET の 200 本文は作成した関連付けを返し、無効化と再有効化を反映する。
//verify:operation DisableAgentWorkloadBinding: 無効化は空の 204 を返し、一覧の状態を disabled にする。
//verify:operation EnableAgentWorkloadBinding: 再有効化は空の 204 を返し、一覧の状態を enabled に戻す。
//verify:operation DeleteAgentWorkloadBinding: 削除は空の 204 を返し、一覧から対象を除く。
func TestAgentWorkloadBindingHTTPContracts(t *testing.T) {
	send := workloadContractRequests(t)
	created := workloadContractJSON(t, send("admin", http.MethodPost, workloadBundlesPath, workloadBundleInput), http.StatusCreated)
	bundleID := assertWorkloadBundle(t, created, "cluster", "enabled")
	path := workloadBundlesPath + "/" + bundleID + "/bindings"
	binding := workloadContractJSON(t, send("admin", http.MethodPost, path, `{"subject_pattern":"spiffe://example.org/worker","agent_id":"agent-1"}`), http.StatusCreated)
	for key, value := range map[string]any{
		"tenant_id": tenancydomain.DefaultTenantID, "trust_bundle_id": bundleID,
		"subject_pattern": "spiffe://example.org/worker", "agent_id": "agent-1", "status": "enabled",
	} {
		if binding[key] != value {
			t.Errorf("%s=%v, want %v", key, binding[key], value)
		}
	}
	id, ok := binding["id"].(string)
	if !ok || id == "" || binding["created_at"] == nil {
		t.Fatalf("binding lacks identifier or creation timestamp: %v", binding)
	}
	listing := workloadContractJSON(t, send("admin", http.MethodGet, path, ""), http.StatusOK)
	if !reflect.DeepEqual(listing["bindings"], []any{binding}) {
		t.Fatalf("listing=%v, want created binding %v", listing, binding)
	}
	bindingPath := "/realms/default/api/admin/v1/workload-identity/bindings/" + id
	for _, transition := range []struct{ action, status string }{{"disable", "disabled"}, {"enable", "enabled"}} {
		assertWorkloadNoContent(t, send("admin", http.MethodPost, bindingPath+"/"+transition.action, ""))
		listing = workloadContractJSON(t, send("admin", http.MethodGet, path, ""), http.StatusOK)
		rows, ok := listing["bindings"].([]any)
		if !ok || len(rows) != 1 {
			t.Fatalf("listing=%v, want one binding", listing)
		}
		row, ok := rows[0].(map[string]any)
		if !ok || row["id"] != id || row["status"] != transition.status || row["updated_at"] == nil {
			t.Fatalf("binding=%v, want %s after %s", rows[0], transition.status, transition.action)
		}
	}
	assertWorkloadNoContent(t, send("admin", http.MethodDelete, bindingPath, ""))
	listing = workloadContractJSON(t, send("admin", http.MethodGet, path, ""), http.StatusOK)
	if !reflect.DeepEqual(listing["bindings"], []any{}) {
		t.Fatalf("deleted binding remains: %v", listing)
	}
}

//verify:operation GetWorkloadTrustBundle: 非管理者の GET は 403 access_denied の Problem Details を返す。
//verify:operation ListAgentWorkloadBindings: 非管理者の一覧要求は 403 access_denied の Problem Details を返す。
func TestWorkloadIdentityReadsRefuseNonAdmin(t *testing.T) {
	send := workloadContractRequests(t)
	created := workloadContractJSON(t, send("admin", http.MethodPost, workloadBundlesPath, workloadBundleInput), http.StatusCreated)
	path := workloadBundlesPath + "/" + assertWorkloadBundle(t, created, "cluster", "enabled")
	for _, target := range []string{path, path + "/bindings"} {
		response := send("alice", http.MethodGet, target, "")
		body := workloadContractJSON(t, response, http.StatusForbidden)
		if response.Header().Get("Content-Type") != support.ProblemContentType || body["type"] != "urn:idmagic:error:access_denied" || body["status"] != float64(403) {
			t.Fatalf("Content-Type=%q body=%v, want access_denied Problem Details", response.Header().Get("Content-Type"), body)
		}
	}
}

//spec:covers REQ-WORKLOADIDENTITY-008, REQ-WORKLOADIDENTITY-009: 存在しない設定と関連付けへの要求は 404 の Problem Details を返し、設定や関連付けを作らない。
//verify:operation GetWorkloadTrustBundle: 不明な信頼設定の参照は 404 workload_trust_bundle_not_found の本文を返す。
//verify:operation UpdateWorkloadTrustBundle: 不明な信頼設定の更新は 404 workload_trust_bundle_not_found の本文を返す。
//verify:operation DisableWorkloadTrustBundle: 不明な信頼設定の無効化は 404 workload_trust_bundle_not_found の本文を返す。
//verify:operation EnableWorkloadTrustBundle: 不明な信頼設定の再有効化は 404 workload_trust_bundle_not_found の本文を返す。
//verify:operation DeleteWorkloadTrustBundle: 不明な信頼設定の削除は 404 workload_trust_bundle_not_found の本文を返す。
//verify:operation RefreshWorkloadTrustBundleJWKS: 不明な信頼設定の再取得は 404 workload_trust_bundle_not_found の本文を返す。
//verify:operation ListAgentWorkloadBindings: 不明な信頼設定の配下の一覧は 404 workload_trust_bundle_not_found の本文を返す。
//verify:operation CreateAgentWorkloadBinding: 不明な信頼設定への関連付け作成は 404 workload_trust_bundle_not_found の本文を返す。
//verify:operation DisableAgentWorkloadBinding: 不明な関連付けの無効化は 404 agent_workload_binding_not_found の本文を返す。
//verify:operation EnableAgentWorkloadBinding: 不明な関連付けの再有効化は 404 agent_workload_binding_not_found の本文を返す。
//verify:operation DeleteAgentWorkloadBinding: 不明な関連付けの削除は 404 agent_workload_binding_not_found の本文を返す。
func TestWorkloadIdentityMissingResourceContracts(t *testing.T) {
	send := workloadContractRequests(t)
	bundle := workloadBundlesPath + "/missing"
	binding := "/realms/default/api/admin/v1/workload-identity/bindings/missing"
	for _, tc := range []struct{ method, path, body, code string }{
		{http.MethodGet, bundle, "", "workload_trust_bundle_not_found"},
		{http.MethodPatch, bundle, `{"name":"renamed"}`, "workload_trust_bundle_not_found"},
		{http.MethodPost, bundle + "/disable", "", "workload_trust_bundle_not_found"},
		{http.MethodPost, bundle + "/enable", "", "workload_trust_bundle_not_found"},
		{http.MethodDelete, bundle, "", "workload_trust_bundle_not_found"},
		{http.MethodPost, bundle + "/refresh", "", "workload_trust_bundle_not_found"},
		{http.MethodGet, bundle + "/bindings", "", "workload_trust_bundle_not_found"},
		{http.MethodPost, bundle + "/bindings", `{"subject_pattern":"spiffe://example.org/worker","agent_id":"agent-1"}`, "workload_trust_bundle_not_found"},
		{http.MethodPost, binding + "/disable", "", "agent_workload_binding_not_found"},
		{http.MethodPost, binding + "/enable", "", "agent_workload_binding_not_found"},
		{http.MethodDelete, binding, "", "agent_workload_binding_not_found"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			response := send("admin", tc.method, tc.path, tc.body)
			body := workloadContractJSON(t, response, http.StatusNotFound)
			if response.Header().Get("Content-Type") != support.ProblemContentType || body["type"] != "urn:idmagic:error:"+tc.code || body["status"] != float64(404) {
				t.Fatalf("Content-Type=%q body=%v, want 404 %s Problem Details", response.Header().Get("Content-Type"), body, tc.code)
			}
		})
	}
	listing := workloadContractJSON(t, send("admin", http.MethodGet, workloadBundlesPath, ""), http.StatusOK)
	if !reflect.DeepEqual(listing["trust_bundles"], []any{}) {
		t.Fatalf("refused requests created a bundle: %v", listing)
	}
}
