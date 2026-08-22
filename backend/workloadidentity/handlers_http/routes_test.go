package handlers_http_test

// WorkloadIdentity 管理 API のエラー契約: 解析できたが業務規則に反する入力
// (name の欠落など) は 422 の RFC 9457 Problem Details で返す。

import (
	"bytes"
	"encoding/json"
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
		Deps:          support.Deps{Issuer: "http://idp.test"},
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
