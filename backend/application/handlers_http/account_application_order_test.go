package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func createAndAssignWeblink(t *testing.T, e *echo.Echo, csrf string, cookie *http.Cookie, name string) string {
	t.Helper()
	create := adminJSON(t, e, http.MethodPost, "/api/admin/v1/applications", csrf, cookie, map[string]any{
		"name": name, "type": "weblink", "launch_url": "https://" + name + ".example",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create %s status=%d body=%s", name, create.Code, create.Body.String())
	}
	var created struct {
		Application struct {
			ID string `json:"id"`
		} `json:"application"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	assign := adminJSON(t, e, http.MethodPost,
		"/api/admin/v1/applications/"+created.Application.ID+"/assignments",
		csrf, cookie, map[string]any{"subject_type": "user", "subject_id": "admin"})
	if assign.Code != http.StatusCreated {
		t.Fatalf("assign %s status=%d body=%s", name, assign.Code, assign.Body.String())
	}
	return created.Application.ID
}

func portalAppNames(t *testing.T, e *echo.Echo) []string {
	t.Helper()
	apps := myApplications(t, e, "admin")
	names := make([]string, len(apps))
	for i, a := range apps {
		names[i], _ = a["name"].(string)
	}
	return names
}

//verify:operation ListMyApplications: 200 JSON の applications は保存した手動順の二つの ID、名称、kind、URL、カテゴリーを返す。
//verify:operation GetMyApplicationOrder: 200 JSON は保存した二つの ID を同じ順で返す。
//verify:operation ReorderMyApplications: 200 JSON は指定した二つの ID を返し、ポータルと再読が一致し、未割り当ては 400 の Problem Details で拒否する。
func TestAccountApplicationReorderAndDefaultSort(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)

	idAlpha := createAndAssignWeblink(t, e, csrf, cookie, "Alpha")
	idBeta := createAndAssignWeblink(t, e, csrf, cookie, "Beta")

	// 既定は name 昇順。
	if got := portalAppNames(t, e); len(got) != 2 || got[0] != "Alpha" || got[1] != "Beta" {
		t.Fatalf("default sort want [Alpha Beta], got %v", got)
	}

	// 手動順 [Beta, Alpha] を保存。
	reorder := adminJSON(t, e, http.MethodPut, "/api/account/v1/applications/order", csrf, cookie, map[string]any{
		"application_ids": []string{idBeta, idAlpha},
	})
	if reorder.Code != http.StatusOK {
		t.Fatalf("reorder status=%d body=%s", reorder.Code, reorder.Body.String())
	}
	reorderBody := applicationContractJSON(t, reorder, http.StatusOK)
	ids, _ := reorderBody["application_ids"].([]any)
	if len(ids) != 2 || ids[0] != idBeta || ids[1] != idAlpha {
		t.Fatalf("reorder body=%v", reorderBody)
	}
	portal := applicationContractJSON(t, getAs(e, "admin", "/realms/default/api/account/v1/applications"), http.StatusOK)
	items, _ := portal["applications"].([]any)
	if len(items) != 2 {
		t.Fatalf("portal=%v", portal)
	}
	for i, want := range []struct{ id, name string }{{idBeta, "Beta"}, {idAlpha, "Alpha"}} {
		app, _ := items[i].(map[string]any)
		if app["id"] != want.id || app["name"] != want.name || app["kind"] != "weblink" || app["launch_url"] != "https://"+want.name+".example" {
			t.Fatalf("portal application=%v", app)
		}
		if categories, ok := app["category_ids"].([]any); !ok || len(categories) != 0 {
			t.Fatalf("portal category_ids=%v", app)
		}
	}
	if categories, ok := portal["categories"].([]any); !ok || len(categories) != 0 {
		t.Fatalf("portal categories=%v", portal)
	}

	// ポータル一覧が手動順を反映する。
	if got := portalAppNames(t, e); len(got) != 2 || got[0] != "Beta" || got[1] != "Alpha" {
		t.Fatalf("after reorder want [Beta Alpha], got %v", got)
	}

	// GET order が保存済み順を返す。
	getOrder := httptest.NewRequest(http.MethodGet, "/realms/default/api/account/v1/applications/order", http.NoBody)
	getOrder.Header.Set("X-Demo-Sub", "admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, getOrder)
	if rec.Code != http.StatusOK {
		t.Fatalf("get order status=%d", rec.Code)
	}
	applicationContractJSON(t, rec, http.StatusOK)
	var order struct {
		ApplicationIDs []string `json:"application_ids"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &order); err != nil {
		t.Fatal(err)
	}
	if len(order.ApplicationIDs) != 2 || order.ApplicationIDs[0] != idBeta || order.ApplicationIDs[1] != idAlpha {
		t.Fatalf("get order mismatch: %v", order.ApplicationIDs)
	}

	// 割当外の id を含む順序は 400 で拒否する。
	bad := adminJSON(t, e, http.MethodPut, "/api/account/v1/applications/order", csrf, cookie, map[string]any{
		"application_ids": []string{"not-assigned"},
	})
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("unassigned reorder want 400, got %d body=%s", bad.Code, bad.Body.String())
	}
	assertApplicationProblem(t, bad, http.StatusBadRequest, "invalid_request")
}
