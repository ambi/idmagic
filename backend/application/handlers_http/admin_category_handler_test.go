package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"testing"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

// カテゴリ名の欠落と未知カテゴリの割当は、generic な "invalid_request" ではなく
// それぞれ固有の code を持ち、業務規則違反として 422 を返す
// (spec/contexts/application/main.tsp の CreateApplicationCategoryError422 /
// SetApplicationCategoriesError422)。
func TestCreateApplicationCategory_EmptyNameYieldsDistinguishableCode(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)

	rec := adminJSON(t, e, http.MethodPost, "/api/admin/v1/application-categories", csrf, cookie, map[string]any{"name": "  "})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != support.ProblemContentType {
		t.Fatalf("Content-Type = %q, want %q", ct, support.ProblemContentType)
	}
	var problem support.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("unmarshal body: %v (body=%s)", err, rec.Body.String())
	}
	if problem.Type != "urn:idmagic:error:category_name_required" {
		t.Errorf("type = %q, want %q", problem.Type, "urn:idmagic:error:category_name_required")
	}
	if problem.Status != http.StatusUnprocessableEntity {
		t.Errorf("status field = %d, want %d", problem.Status, http.StatusUnprocessableEntity)
	}
}

func TestSetApplicationCategories_UnknownCategoryYieldsDistinguishableCode(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	appID := createAndAssignWeblink(t, e, csrf, cookie, "Payroll")

	rec := adminJSON(t, e, http.MethodPut, "/api/admin/v1/applications/"+appID+"/categories", csrf, cookie, map[string]any{
		"category_ids": []string{"does-not-exist"},
	})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d (body=%s)", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != support.ProblemContentType {
		t.Fatalf("Content-Type = %q, want %q", ct, support.ProblemContentType)
	}
	var problem support.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("unmarshal body: %v (body=%s)", err, rec.Body.String())
	}
	if problem.Type != "urn:idmagic:error:unknown_category" {
		t.Errorf("type = %q, want %q", problem.Type, "urn:idmagic:error:unknown_category")
	}
}
