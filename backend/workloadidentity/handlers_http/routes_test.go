package handlers_http_test

// WorkloadIdentity 管理 API のエラー契約: 解析できたが業務規則に反する入力
// (name の欠落など) は 422 の RFC 9457 Problem Details で返す。

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/workloadidentity"
	workloadmemory "github.com/ambi/idmagic/backend/workloadidentity/db_memory"

	"github.com/labstack/echo/v5"
)

func newWorkloadIdentityHandler(t *testing.T) *echo.Echo {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	now := time.Now().UTC()
	userRepo.Seed(&userdomain.User{
		ID: "admin", PreferredUsername: "admin", PasswordHash: "unused",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	// 管理者でない主体でも同じ経路を通せるようにしておく (REQ-WORKLOADIDENTITY-010)。
	userRepo.Seed(&userdomain.User{
		ID: "alice", PreferredUsername: "alice", PasswordHash: "unused",
		CreatedAt: now, UpdatedAt: now,
	})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:        "http://idp.test",
		AuthnResolver: authusecases.DemoHeaderResolver{},
		IdManagement:  idmanagement.Module{UserRepo: userRepo},
		WorkloadIdentity: workloadidentity.Module{
			TrustBundleRepo: workloadmemory.NewWorkloadTrustBundleRepository(),
			BindingRepo:     workloadmemory.NewAgentWorkloadBindingRepository(),
		},
	})
	return e
}

func workloadAdminCSRF(t *testing.T, e *echo.Echo) (string, *http.Cookie) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/realms/default/api/auth/account", http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("account status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	cookies := response.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("csrf cookie missing")
	}
	return body.CSRFToken, cookies[0]
}

func TestRegisterTrustBundleRejectsMissingName(t *testing.T) {
	e := newWorkloadIdentityHandler(t)
	csrf, cookie := workloadAdminCSRF(t, e)

	payload, err := json.Marshal(map[string]any{
		"trust_domain":       "issuer.example",
		"issuer":             "https://issuer.example",
		"accepted_audiences": []string{"https://idmagic.example"},
		"jwks_uri":           "https://issuer.example/jwks",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost,
		"/realms/default/api/admin/v1/workload-identity/trust-bundles", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrf)
	request.Header.Set("X-Demo-Sub", "admin")
	request.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, request)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != support.ProblemContentType {
		t.Fatalf("Content-Type=%q, want %q", contentType, support.ProblemContentType)
	}
	var problem support.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("unmarshal body: %v (body=%s)", err, rec.Body.String())
	}
	if problem.Type != "urn:idmagic:error:workload_trust_bundle_name_required" {
		t.Errorf("type=%q, want urn:idmagic:error:workload_trust_bundle_name_required", problem.Type)
	}
}

// EX-WORKLOADIDENTITY-010-01: 認証済みだが `admin` ロールを持たない "alice" の登録要求は
// 拒否され、信頼設定は作成されない。拒否応答と、拒否が防いだ効果（一覧が空のまま）の双方を
// 固定する。
//
// 管理者なら受理される本文をそのまま送っているので、拒否の理由は本文の不備ではなく
// ロールだけに由来する。素通りすれば、テナントの利用者なら誰でも発行者を信頼設定へ足せる。
//
// REQ-WORKLOADIDENTITY-010: 信頼設定の登録は管理者に限られる。管理者なら受理される
// 本文をそのまま送って拒否させ、信頼設定が 1 件も増えていないことを読み直す。
func TestRegisterTrustBundleRejectsNonAdmin(t *testing.T) {
	e := newWorkloadIdentityHandler(t)
	csrf, cookie := workloadAdminCSRF(t, e)

	payload, err := json.Marshal(map[string]any{
		"name":               "prod-cluster",
		"trust_domain":       "issuer.example",
		"issuer":             "https://issuer.example",
		"accepted_audiences": []string{"https://idmagic.example"},
		"jwks_uri":           "https://issuer.example/jwks",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost,
		"/realms/default/api/admin/v1/workload-identity/trust-bundles", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrf)
	request.Header.Set("X-Demo-Sub", "alice")
	request.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, request)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", rec.Code, rec.Body.String())
	}

	listed := httptest.NewRequest(http.MethodGet,
		"/realms/default/api/admin/v1/workload-identity/trust-bundles", http.NoBody)
	listed.Header.Set("X-Demo-Sub", "admin")
	listing := httptest.NewRecorder()
	e.ServeHTTP(listing, listed)
	if listing.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listing.Code, listing.Body.String())
	}
	var view struct {
		TrustBundles []struct {
			ID string `json:"id"`
		} `json:"trust_bundles"`
	}
	if err := json.Unmarshal(listing.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if len(view.TrustBundles) != 0 {
		t.Fatalf("trust bundles = %#v, want the refused registration to have left none behind", view.TrustBundles)
	}
}

// REQ-WORKLOADIDENTITY-008: 管理 API で保存した信頼設定の状態変更を、同じ公開読取経路で観測する。
func TestAdminWorkloadTrustBundleLifecycle(t *testing.T) {
	e := newWorkloadIdentityHandler(t)
	csrf, cookie := workloadAdminCSRF(t, e)
	adminRequest := func(method, path string, body []byte) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, path, bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "http://idp.test")
		request.Header.Set("X-Csrf-Token", csrf)
		request.Header.Set("X-Demo-Sub", "admin")
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)
		return response
	}

	payload, err := json.Marshal(map[string]any{
		"name": "prod-cluster", "trust_domain": "issuer.example", "issuer": "https://issuer.example",
		"accepted_audiences": []string{"https://idmagic.example"}, "jwks_uri": "https://issuer.example/jwks",
	})
	if err != nil {
		t.Fatal(err)
	}
	created := adminRequest(http.MethodPost, "/realms/default/api/admin/v1/workload-identity/trust-bundles", payload)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var bundle struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.ID == "" || bundle.Status != "enabled" {
		t.Fatalf("created bundle=%+v", bundle)
	}

	disabled := adminRequest(http.MethodPost, "/realms/default/api/admin/v1/workload-identity/trust-bundles/"+bundle.ID+"/disable", nil)
	if disabled.Code != http.StatusNoContent {
		t.Fatalf("disable status=%d body=%s", disabled.Code, disabled.Body.String())
	}
	get := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/workload-identity/trust-bundles/"+bundle.ID, http.NoBody)
	get.Header.Set("X-Demo-Sub", "admin")
	got := httptest.NewRecorder()
	e.ServeHTTP(got, get)
	if got.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", got.Code, got.Body.String())
	}
	if err := json.Unmarshal(got.Body.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.Status != "disabled" {
		t.Fatalf("status=%q, want disabled", bundle.Status)
	}
}

// EX-WORKLOADIDENTITY-010-02: 登録以外の管理操作 — 信頼設定の更新、無効化、再有効化、削除、
// JWKS の再取得、関連付けの作成、無効化、再有効化、削除 — も "alice" には 1 つも通らない。
// 具体例が 9 つの操作を並べているので、観測も 9 つ要る。
//
// EX-WORKLOADIDENTITY-010-01 が登録だけを見ているので、登録の経路にだけ認可を付けた実装は
// そこでは緑になる。管理 API に操作を足すたびに認可の付け忘れが起きうるのはこちら側なので、
// 経路を列挙して全数を見る。
//
// 関連付けの操作には実在しない id を使う。認可が資源の解決より先に効くなら 403 が返り、
// あとに効くなら 404 が返るので、403 を要求すること自体が順序を固定している。
// 拒否が防いだ効果は、admin で読み直した信頼設定が登録直後のままであることで観測する。
func TestWorkloadIdentityAdminOperationsRejectNonAdmin(t *testing.T) {
	e := newWorkloadIdentityHandler(t)
	csrf, cookie := workloadAdminCSRF(t, e)

	send := func(sub, method, path string, body []byte) *httptest.ResponseRecorder {
		t.Helper()
		var reader io.Reader = http.NoBody
		if body != nil {
			reader = bytes.NewReader(body)
		}
		request := httptest.NewRequest(method, path, reader)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "http://idp.test")
		request.Header.Set("X-Csrf-Token", csrf)
		request.Header.Set("X-Demo-Sub", sub)
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)
		return response
	}

	registration, err := json.Marshal(map[string]any{
		"name": "prod-cluster", "trust_domain": "issuer.example", "issuer": "https://issuer.example",
		"accepted_audiences": []string{"https://idmagic.example"}, "jwks_uri": "https://issuer.example/jwks",
	})
	if err != nil {
		t.Fatal(err)
	}
	created := send("admin", http.MethodPost, "/realms/default/api/admin/v1/workload-identity/trust-bundles", registration)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var bundle struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}

	bundles := "/realms/default/api/admin/v1/workload-identity/trust-bundles/" + bundle.ID
	bindings := "/realms/default/api/admin/v1/workload-identity/bindings/binding-that-does-not-exist"
	rename, err := json.Marshal(map[string]any{"name": "renamed-by-alice"})
	if err != nil {
		t.Fatal(err)
	}
	create, err := json.Marshal(map[string]any{
		"subject_pattern": "spiffe://example.org/ns/prod/sa/*", "agent_id": "agent-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, op := range []struct {
		name, method, path string
		body               []byte
	}{
		{"信頼設定の更新", http.MethodPatch, bundles, rename},
		{"信頼設定の無効化", http.MethodPost, bundles + "/disable", nil},
		{"信頼設定の再有効化", http.MethodPost, bundles + "/enable", nil},
		{"信頼設定の削除", http.MethodDelete, bundles, nil},
		{"JWKS の再取得", http.MethodPost, bundles + "/refresh", nil},
		{"関連付けの作成", http.MethodPost, bundles + "/bindings", create},
		{"関連付けの無効化", http.MethodPost, bindings + "/disable", nil},
		{"関連付けの再有効化", http.MethodPost, bindings + "/enable", nil},
		{"関連付けの削除", http.MethodDelete, bindings, nil},
	} {
		t.Run(op.name, func(t *testing.T) {
			refused := send("alice", op.method, op.path, op.body)
			if refused.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%s, want 403", refused.Code, refused.Body.String())
			}
		})
	}

	// 拒否が防いだ効果: 9 つの要求のあとも、信頼設定は登録直後のままである。
	after := send("admin", http.MethodGet, bundles, nil)
	if after.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", after.Code, after.Body.String())
	}
	var current struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(after.Body.Bytes(), &current); err != nil {
		t.Fatal(err)
	}
	if current.Name != bundle.Name || current.Status != bundle.Status {
		t.Fatalf("信頼設定が変わった: %+v -> %+v", bundle, current)
	}
}
