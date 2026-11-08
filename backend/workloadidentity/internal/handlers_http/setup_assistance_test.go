package handlers_http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/idmanagement"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
	"github.com/ambi/idmagic/backend/workloadidentity"
	workloadhttp "github.com/ambi/idmagic/backend/workloadidentity/internal/handlers_http"
	"github.com/labstack/echo/v5"
)

//spec:covers REQ-WORKLOADIDENTITY-010, REQ-WORKLOADIDENTITY-015: 候補の返送と、非管理者や不正なブラウザー要求による外部取得の拒否を独立に観測する。
//verify:operation DiscoverWorkloadIssuerMetadata: 発行者を取得ポートへ渡し、検証済みの issuer と jwks_uri を 200 本文で返す。
func TestIssuerMetadataHTTPContractAndRefusalEffects(t *testing.T) {
	idm := idmanagement.NewMemoryModule()
	now := time.Now().UTC()
	for _, id := range []string{"admin", "member"} {
		roles := []string{}
		if id == "admin" {
			roles = []string{"admin"}
		}
		if err := idm.UserRepo.Save(testing_tenant.Default(t.Context()), &userdomain.User{ID: id, PreferredUsername: id, Roles: roles, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	module := workloadidentity.NewMemoryModule()
	e := echo.New()
	calls := []string{}
	workloadhttp.RegisterRoutes(e.Group("", testing_tenant.ResolveDefault), workloadhttp.Deps{
		Issuer: "https://idp.test", Authenticator: &support.Authenticator{Principals: idm.Principals(), Sessions: authentication.Module{AuthnResolver: authentication.DemoHeaderResolver{}}.SessionAuthentications()},
		TrustBundleRepo: module.TrustBundleRepo, BindingRepo: module.BindingRepo, Emit: func(spec.DomainEvent) { t.Fatal("metadata emitted an event") },
	}, func(_ context.Context, issuer string) (string, error) {
		calls = append(calls, issuer)
		return "https://keys.example/jwks", nil
	})
	for _, tc := range []struct {
		sub, origin, csrf, body string
		status, calls           int
	}{
		{"admin", "https://idp.test", "csrf", `{"issuer":"https://issuer.example"}`, 200, 1},
		{"member", "https://idp.test", "csrf", `{"issuer":"https://issuer.example"}`, 403, 1},
		{"admin", "https://other.test", "csrf", `{"issuer":"https://issuer.example"}`, 403, 1},
		{"admin", "https://idp.test", "wrong", `{"issuer":"https://issuer.example"}`, 403, 1},
		{"admin", "https://idp.test", "csrf", `{`, 400, 1},
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/workload-identity/issuer-metadata", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Demo-Sub", tc.sub)
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("X-Csrf-Token", tc.csrf)
		req.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "csrf"})
		response := httptest.NewRecorder()
		e.ServeHTTP(response, req)
		body := workloadContractJSON(t, response, tc.status)
		if tc.status == 200 && !reflect.DeepEqual(body, map[string]any{"issuer": "https://issuer.example", "jwks_uri": "https://keys.example/jwks"}) {
			t.Fatalf("metadata=%v", body)
		}
		if len(calls) != tc.calls {
			t.Fatalf("outgoing fetches=%v, want %d", calls, tc.calls)
		}
	}
	if calls[0] != "https://issuer.example" {
		t.Fatalf("issuer=%s", calls[0])
	}
	bundles, err := module.TrustBundleRepo.ListAll(t.Context(), domain.DefaultTenantID)
	if err != nil || len(bundles) != 0 {
		t.Fatalf("metadata saved bundles=%v err=%v", bundles, err)
	}
}

//spec:covers REQ-WORKLOADIDENTITY-013, REQ-WORKLOADIDENTITY-014: 管理 API から登録した関連付けを同じ ID のまま更新し、保存後の照合で観測する。
//verify:operation UpdateAgentWorkloadBinding: PATCH の 200 本文と一覧に更新したパターンが残り、ID と状態が保たれる。
//verify:operation PreviewAgentWorkloadBindings: POST の主体と候補パターンに対し、一致、候補の一致、重なり一覧を返す。
func TestBindingSetupAssistanceHTTPContracts(t *testing.T) {
	events := []spec.DomainEvent{}
	send := workloadContractRequests(t, func(e spec.DomainEvent) { events = append(events, e) })
	bundle := workloadContractJSON(t, send("admin", http.MethodPost, workloadBundlesPath, workloadBundleInput), http.StatusCreated)
	path := workloadBundlesPath + "/" + bundle["id"].(string) + "/bindings"
	created := workloadContractJSON(t, send("admin", http.MethodPost, path, `{"subject_pattern":"spiffe://example.org/worker-*","agent_id":"agent-1"}`), http.StatusCreated)
	preview := workloadContractJSON(t, send("admin", http.MethodPost, path+"/preview", `{"subject":"spiffe://example.org/worker-a","subject_pattern":"spiffe://example.org/worker-a"}`), http.StatusOK)
	if !reflect.DeepEqual(preview["matches"], []any{created}) || !reflect.DeepEqual(preview["possible_overlaps"], []any{created}) || preview["candidate_matches"] != true {
		t.Fatalf("preview=%v, want the saved binding and candidate match", preview)
	}
	bindingPath := "/realms/default/api/admin/v1/workload-identity/bindings/" + created["id"].(string)
	updated := workloadContractJSON(t, send("admin", http.MethodPatch, bindingPath, `{"subject_pattern":"spiffe://example.org/new-*"}`), http.StatusOK)
	if updated["id"] != created["id"] || updated["status"] != "enabled" || updated["subject_pattern"] != "spiffe://example.org/new-*" || updated["created_at"] != created["created_at"] || updated["updated_at"] == nil {
		t.Fatalf("updated=%v, original=%v", updated, created)
	}
	updates := 0
	for _, event := range events {
		if event.EventType() == "AgentWorkloadBindingUpdated" {
			updates++
		}
	}
	if updates != 1 {
		t.Fatalf("update events=%d, want 1 through the assembled router", updates)
	}
	listing := workloadContractJSON(t, send("admin", http.MethodGet, path, ""), http.StatusOK)
	if !reflect.DeepEqual(listing["bindings"], []any{updated}) {
		t.Fatalf("saved=%v, want %v", listing, updated)
	}
	for _, tc := range []struct {
		subject string
		count   int
	}{{"spiffe://example.org/worker-a", 0}, {"spiffe://example.org/new-a", 1}, {"spiffe://example.org/new-a/child", 0}} {
		got := workloadContractJSON(t, send("admin", http.MethodPost, path+"/preview", `{"subject":"`+tc.subject+`"}`), http.StatusOK)
		if len(got["matches"].([]any)) != tc.count {
			t.Fatalf("subject=%s result=%v", tc.subject, got)
		}
	}
}

//spec:covers REQ-WORKLOADIDENTITY-010, REQ-WORKLOADIDENTITY-013, REQ-WORKLOADIDENTITY-014, REQ-WORKLOADIDENTITY-015: 拒否された支援操作は保存済みの設定を変えない。
//verify:operation DiscoverWorkloadIssuerMetadata: 内部アドレスへの取得を 400 workload_issuer_metadata_invalid で拒否し、信頼設定を作らない。
func TestSetupAssistanceRefusals(t *testing.T) {
	send := workloadContractRequests(t)
	bundle := workloadContractJSON(t, send("admin", http.MethodPost, workloadBundlesPath, workloadBundleInput), http.StatusCreated)
	path := workloadBundlesPath + "/" + bundle["id"].(string) + "/bindings"
	created := workloadContractJSON(t, send("admin", http.MethodPost, path, `{"subject_pattern":"original","agent_id":"agent-1"}`), http.StatusCreated)
	bindingPath := "/realms/default/api/admin/v1/workload-identity/bindings/" + created["id"].(string)
	metadataPath := "/realms/default/api/admin/v1/workload-identity/issuer-metadata"
	for _, tc := range []struct {
		sub, method, path, body, code string
		status                        int
	}{
		{"alice", http.MethodPatch, bindingPath, `{"subject_pattern":"changed"}`, "access_denied", 403},
		{"alice", http.MethodPost, path + "/preview", `{"subject":"original"}`, "access_denied", 403},
		{"alice", http.MethodPost, metadataPath, `{"issuer":"https://127.0.0.1"}`, "access_denied", 403},
		{"admin", http.MethodPatch, bindingPath, `{"agent_id":"unknown"}`, "agent_workload_binding_agent_not_found", 422},
		{"admin", http.MethodPatch, bindingPath, `{"subject_pattern":"["}`, "invalid_request", 400},
		{"admin", http.MethodPost, path + "/preview", `{}`, "invalid_request", 400},
		{"admin", http.MethodPost, metadataPath, `{"issuer":"https://127.0.0.1"}`, "workload_issuer_metadata_invalid", 400},
	} {
		problem := workloadContractJSON(t, send(tc.sub, tc.method, tc.path, tc.body), tc.status)
		if problem["type"] != "urn:idmagic:error:"+tc.code {
			t.Fatalf("problem=%v, want %s", problem, tc.code)
		}
	}
	listing := workloadContractJSON(t, send("admin", http.MethodGet, path, ""), http.StatusOK)
	if !reflect.DeepEqual(listing["bindings"], []any{created}) {
		t.Fatalf("refused changes saved: %v", listing)
	}
	bundles := workloadContractJSON(t, send("admin", http.MethodGet, workloadBundlesPath, ""), http.StatusOK)
	if !reflect.DeepEqual(bundles["trust_bundles"], []any{bundle}) {
		t.Fatalf("metadata saved configuration: %v", bundles)
	}
}
