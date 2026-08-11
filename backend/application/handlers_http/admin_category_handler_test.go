package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// wi-326 T003: admin_category_handler.go used to collapse both conditions into
// the generic "invalid_request" code (background). They now carry
// distinguishable codes matching the new SCL error models
// CategoryNameRequiredError / UnknownCategoryError (spec/contexts/application.yaml).
func TestCreateApplicationCategory_EmptyNameYieldsDistinguishableCode(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)

	rec := adminJSON(t, e, http.MethodPost, "/api/admin/v1/application-categories", csrf, cookie, map[string]any{"name": "  "})

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v (body=%s)", err, rec.Body.String())
	}
	if body.Error != "category_name_required" {
		t.Errorf("error code = %q, want %q", body.Error, "category_name_required")
	}
}

func TestSetApplicationCategories_UnknownCategoryYieldsDistinguishableCode(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	appID := createAndAssignWeblink(t, e, csrf, cookie, "Payroll")

	rec := adminJSON(t, e, http.MethodPut, "/api/admin/v1/applications/"+appID+"/categories", csrf, cookie, map[string]any{
		"category_ids": []string{"does-not-exist"},
	})

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v (body=%s)", err, rec.Body.String())
	}
	if body.Error != "unknown_category" {
		t.Errorf("error code = %q, want %q", body.Error, "unknown_category")
	}
}
