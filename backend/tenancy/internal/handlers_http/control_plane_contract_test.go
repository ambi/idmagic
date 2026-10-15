package handlers_http_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/tenancy/domain"
	memory "github.com/ambi/idmagic/backend/tenancy/internal/db_memory"
)

func assertTenantSummary(t *testing.T, body map[string]any, id, realm, name, status, style string) {
	t.Helper()
	if body["id"] != id || body["realm"] != realm || body["display_name"] != name || body["status"] != status || body["endpoint_style"] != style {
		t.Fatalf("tenant=%v, want %s %s %s %s %s", body, id, realm, name, status, style)
	}
	created, _ := body["created_at"].(string)
	if at, err := time.Parse(time.RFC3339Nano, created); err != nil || at.IsZero() {
		t.Fatalf("tenant created_at=%q: %v", created, err)
	}
}

//spec:covers REQ-TENANCY-025, REQ-TENANCY-045, REQ-TENANCY-044: 作成、参照、更新の本文は採番した ID とテナントの状態を一貫して返す。
//spec:covers REQ-TENANCY-027, REQ-TENANCY-011: 状態変更は空の 204 を返し、テナントの再読に反映される。
//spec:covers REQ-TENANCY-012: クォータの更新は保存した上書きを 200 で返す。
//verify:operation CreateTenant: 作成は 201 のサーバー発行 ID、realm、表示名、active、path と作成時刻を返す。
//verify:operation GetTenant: GET は 200 の同じ ID、realm、表示名、状態、所在の形と作成時刻を返す。
//verify:operation UpdateTenant: PATCH は 200 の変更した表示名と同じ ID を返し、再読で保存を確認する。
//verify:operation DisableTenant: POST は空の 204 を返し、GET が disabled と disabled_at を返す。
//verify:operation EnableTenant: POST は空の 204 を返し、GET が active に戻り disabled_at を省く。
//verify:operation SetTenantEndpointStyle: PUT は空の 204 を返し、GET が切り替えた subdomain を返す。
//verify:operation UpdateTenantQuota: PUT は 200 JSON に保存した users と groups の上書きを返す。
func TestControlPlaneTenantHTTPContracts(t *testing.T) {
	s := newRulesServer(t, systemAdmin(), rulesOptions{tenantBaseDomain: "idp.test"})
	created := s.send(t, http.MethodPost, tenantsPath, map[string]string{"realm": "contract", "display_name": "Contract"})
	requireStatus(t, created, http.StatusCreated)
	body := decodeJSON(t, created)
	id, _ := body["id"].(string)
	if id == "" {
		t.Fatal("created tenant missing id")
	}
	assertTenantSummary(t, body, id, "contract", "Contract", "active", "path")
	path := tenantsPath + "/contract"
	read := func(name, status, style string) map[string]any {
		t.Helper()
		rec := s.get(t, path)
		requireStatus(t, rec, http.StatusOK)
		got := decodeJSON(t, rec)
		assertTenantSummary(t, got, id, "contract", name, status, style)
		return got
	}
	read("Contract", "active", "path")
	updated := s.send(t, http.MethodPatch, path, map[string]string{"display_name": "Changed"})
	requireStatus(t, updated, http.StatusOK)
	assertTenantSummary(t, decodeJSON(t, updated), id, "contract", "Changed", "active", "path")
	read("Changed", "active", "path")
	quota := s.send(t, http.MethodPut, path+"/quota", map[string]int{"users": 120, "groups": 8})
	requireStatus(t, quota, http.StatusOK)
	quotaBody := decodeJSON(t, quota)
	if quotaBody["users"] != float64(120) || quotaBody["groups"] != float64(8) {
		t.Fatalf("quota=%v", quotaBody)
	}
	for _, change := range []string{"disable", "enable"} {
		rec := s.send(t, http.MethodPost, path+"/"+change, nil)
		if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
			t.Fatalf("%s status=%d body=%s", change, rec.Code, rec.Body.String())
		}
		status := "active"
		if change == "disable" {
			status = "disabled"
		}
		got := read("Changed", status, "path")
		if change == "disable" {
			at, _ := got["disabled_at"].(string)
			if value, err := time.Parse(time.RFC3339Nano, at); err != nil || value.IsZero() {
				t.Fatalf("disabled_at=%q: %v", at, err)
			}
		} else if _, present := got["disabled_at"]; present {
			t.Fatalf("enabled tenant retains disabled_at: %v", got)
		}
	}
	style := s.send(t, http.MethodPut, path+"/endpoint-style", map[string]string{"endpoint_style": "subdomain"})
	if style.Code != http.StatusNoContent || style.Body.Len() != 0 {
		t.Fatalf("endpoint style status=%d body=%s", style.Code, style.Body.String())
	}
	read("Changed", "active", "subdomain")
}

//spec:covers REQ-TENANCY-012: System 管理者でない主体のクォータ更新は拒否し、保存済みの上限とイベントを変えない。
//verify:operation CreateTenant: 通常のテナント管理者の作成は 403 access_denied の Problem Details で拒否する。
//verify:operation GetTenant: 通常のテナント管理者の制御面の取得は 403 access_denied の Problem Details で拒否する。
//verify:operation UpdateTenant: 通常のテナント管理者の制御面の更新は 403 access_denied の Problem Details で拒否する。
//verify:operation DisableTenant: 通常のテナント管理者の無効化は 403 access_denied の Problem Details で拒否する。
//verify:operation EnableTenant: 通常のテナント管理者の再開は 403 access_denied の Problem Details で拒否する。
//verify:operation SetTenantEndpointStyle: 通常のテナント管理者の所在変更は 403 access_denied の Problem Details で拒否する。
//verify:operation UpdateTenantQuota: 通常のテナント管理者の上限変更は 403 access_denied の Problem Details で拒否する。
func TestControlPlaneTenantContractsRefuseTenantAdmin(t *testing.T) {
	quotas := memory.NewQuotaRepository()
	users := 200
	if err := quotas.SetQuota(context.Background(), "acme", &domain.TenantQuota{Users: &users}); err != nil {
		t.Fatal(err)
	}
	s := newRulesServer(t, settingsActor("admin", domain.DefaultTenantID, []string{"admin"}), rulesOptions{quotas: quotas})
	for _, request := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, tenantsPath, map[string]string{"realm": "forbidden", "display_name": "Forbidden"}},
		{http.MethodGet, tenantsPath + "/acme", nil},
		{http.MethodPatch, tenantsPath + "/acme", map[string]string{"display_name": "Forbidden"}},
		{http.MethodPost, tenantsPath + "/acme/disable", nil},
		{http.MethodPost, tenantsPath + "/acme/enable", nil},
		{http.MethodPut, tenantsPath + "/acme/endpoint-style", map[string]string{"endpoint_style": "path"}},
		{http.MethodPut, tenantsPath + "/acme/quota", map[string]int{"users": 1}},
	} {
		assertTenancyProblem(t, s.send(t, request.method, request.path, request.body), http.StatusForbidden, "access_denied")
	}
	quota, err := quotas.GetQuota(context.Background(), "acme")
	if err != nil || quota.Users == nil || *quota.Users != 200 || quota.Groups != nil {
		t.Fatalf("refused update changed quota: %+v, error=%v", quota, err)
	}
	if len(*s.events) != 0 {
		t.Fatalf("refused operations emitted events: %v", *s.events)
	}
}
