package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminApplicationLifecycle(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)

	// Create OIDC App
	createOIDC := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "OIDC App", "type": "oidc", "redirect_uris": []string{"https://oidc.example/callback"},
		"client_type": "confidential", "token_endpoint_auth_method": "client_secret_post",
	})
	if createOIDC.Code != http.StatusCreated {
		t.Fatalf("create oidc status=%d body=%s", createOIDC.Code, createOIDC.Body.String())
	}
	var created struct {
		Application struct {
			ApplicationID string `json:"application_id"`
		} `json:"application"`
	}
	if err := json.Unmarshal(createOIDC.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	var createdWire struct {
		Application map[string]any `json:"application"`
	}
	if err := json.Unmarshal(createOIDC.Body.Bytes(), &createdWire); err != nil {
		t.Fatal(err)
	}
	if _, exists := createdWire.Application["bindings"]; exists {
		t.Fatal("legacy bindings field must not be exposed")
	}
	protocol, ok := createdWire.Application["protocol"].(map[string]any)
	if !ok || protocol["type"] != "oidc" {
		t.Fatalf("single protocol projection missing: %#v", createdWire.Application["protocol"])
	}
	appID := created.Application.ApplicationID

	// Create WebLink App
	createWeblink := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "WebLink App", "type": "weblink", "launch_url": "https://weblink.example",
	})
	if createWeblink.Code != http.StatusCreated {
		t.Fatalf("create weblink status=%d body=%s", createWeblink.Code, createWeblink.Body.String())
	}

	// Create WS-Fed App
	createWsFed := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "WsFed App", "type": "wsfed", "wtrealm": "urn:wsfed:app", "reply_urls": []string{"https://wsfed.example/reply"},
	})
	if createWsFed.Code != http.StatusCreated {
		t.Fatalf("create wsfed status=%d body=%s", createWsFed.Code, createWsFed.Body.String())
	}

	// Create SAML App
	createSAML := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "SAML App", "type": "saml", "entity_id": "https://saml.example", "acs_urls": []string{"https://saml.example/acs"},
	})
	if createSAML.Code != http.StatusCreated {
		t.Fatalf("create saml status=%d body=%s", createSAML.Code, createSAML.Body.String())
	}

	// Create Service App
	createService := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "Service App", "type": "service",
	})
	if createService.Code != http.StatusCreated {
		t.Fatalf("create service status=%d body=%s", createService.Code, createService.Body.String())
	}

	// OIDC Bad Request (no redirect_uris)
	createOIDCBad := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "OIDC App Bad", "type": "oidc",
	})
	if createOIDCBad.Code != http.StatusBadRequest {
		t.Fatalf("create oidc bad status=%d body=%s", createOIDCBad.Code, createOIDCBad.Body.String())
	}

	// WSFed Bad Request (no wtrealm)
	createWsFedBad := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "WsFed App Bad", "type": "wsfed",
	})
	if createWsFedBad.Code != http.StatusBadRequest {
		t.Fatalf("create wsfed bad status=%d body=%s", createWsFedBad.Code, createWsFedBad.Body.String())
	}

	// SAML Bad Request (no entity_id)
	createSAMLBad := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "SAML App Bad", "type": "saml",
	})
	if createSAMLBad.Code != http.StatusBadRequest {
		t.Fatalf("create saml bad status=%d body=%s", createSAMLBad.Code, createSAMLBad.Body.String())
	}

	// Invalid Type Bad Request
	createBadType := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "Bad Type App", "type": "unknown",
	})
	if createBadType.Code != http.StatusBadRequest {
		t.Fatalf("create bad type status=%d body=%s", createBadType.Code, createBadType.Body.String())
	}

	// Get (Normal)
	getApp := adminJSON(t, e, http.MethodGet, "/api/admin/applications/"+appID, csrf, cookie, nil)
	if getApp.Code != http.StatusOK {
		t.Fatalf("get app status=%d body=%s", getApp.Code, getApp.Body.String())
	}

	// Get (Not Found)
	getAppNotFound := adminJSON(t, e, http.MethodGet, "/api/admin/applications/non-existent-id", csrf, cookie, nil)
	if getAppNotFound.Code != http.StatusNotFound {
		t.Fatalf("get app not found status=%d body=%s", getAppNotFound.Code, getAppNotFound.Body.String())
	}

	// List
	list := adminJSON(t, e, http.MethodGet, "/api/admin/applications", csrf, cookie, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	var listResp struct {
		Applications []map[string]any `json:"applications"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &listResp); err != nil {
		t.Fatal(err)
	}

	// Update (Normal)
	name := "App A Updated"
	status := "disabled"
	launchURL := "https://a-updated.example"
	update := adminJSON(t, e, http.MethodPatch, "/api/admin/applications/"+appID, csrf, cookie, map[string]any{
		"name": &name, "status": &status, "launch_url": &launchURL,
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", update.Code, update.Body.String())
	}

	// Update (Invalid JSON)
	updateBad := adminJSON(t, e, http.MethodPatch, "/api/admin/applications/"+appID, csrf, cookie, "invalid-json")
	if updateBad.Code != http.StatusBadRequest {
		t.Fatalf("update bad json status=%d body=%s", updateBad.Code, updateBad.Body.String())
	}

	// protocol は作成時に不変であり、旧 attach / detach route は存在しない。
	attach := adminJSON(t, e, http.MethodPost, "/api/admin/applications/"+appID+"/bindings", csrf, cookie, map[string]any{
		"type": "oidc", "client_id": "client-123",
	})
	if attach.Code != http.StatusNotFound {
		t.Fatalf("attach protocol route status=%d body=%s", attach.Code, attach.Body.String())
	}
	detach := adminJSON(t, e, http.MethodDelete, "/api/admin/applications/"+appID+"/bindings/oidc", csrf, cookie, nil)
	if detach.Code != http.StatusNotFound {
		t.Fatalf("detach protocol route status=%d body=%s", detach.Code, detach.Body.String())
	}

	// Assign
	assign := adminJSON(t, e, http.MethodPost, "/api/admin/applications/"+appID+"/assignments", csrf, cookie, map[string]any{
		"subject_type": "user", "subject_id": "regular", "visibility": "visible",
	})
	if assign.Code != http.StatusCreated {
		t.Fatalf("assign status=%d body=%s", assign.Code, assign.Body.String())
	}

	// Assign (Invalid JSON)
	assignBad := adminJSON(t, e, http.MethodPost, "/api/admin/applications/"+appID+"/assignments", csrf, cookie, "invalid-json")
	if assignBad.Code != http.StatusBadRequest {
		t.Fatalf("assign bad json status=%d body=%s", assignBad.Code, assignBad.Body.String())
	}

	// List Assignments
	listAssigns := adminJSON(t, e, http.MethodGet, "/api/admin/applications/"+appID+"/assignments", csrf, cookie, nil)
	if listAssigns.Code != http.StatusOK {
		t.Fatalf("list assignments status=%d body=%s", listAssigns.Code, listAssigns.Body.String())
	}
	var assignsResp struct {
		Assignments []map[string]any `json:"assignments"`
	}
	if err := json.Unmarshal(listAssigns.Body.Bytes(), &assignsResp); err != nil {
		t.Fatal(err)
	}

	// Unassign
	unassign := adminJSON(t, e, http.MethodDelete, "/api/admin/applications/"+appID+"/assignments/user/regular", csrf, cookie, nil)
	if unassign.Code != http.StatusNoContent {
		t.Fatalf("unassign status=%d body=%s", unassign.Code, unassign.Body.String())
	}

	// Delete
	deleteApp := adminJSON(t, e, http.MethodDelete, "/api/admin/applications/"+appID, csrf, cookie, nil)
	if deleteApp.Code != http.StatusNoContent {
		t.Fatalf("delete app status=%d body=%s", deleteApp.Code, deleteApp.Body.String())
	}
}

func TestAdminCategoryLifecycle(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)

	// Create Category
	pos := 1
	create := adminJSON(t, e, http.MethodPost, "/api/admin/application-categories", csrf, cookie, map[string]any{
		"name": "Cat A", "position": &pos,
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create category status=%d body=%s", create.Code, create.Body.String())
	}
	var catCreated struct {
		Category struct {
			CategoryID string `json:"category_id"`
		} `json:"category"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &catCreated); err != nil {
		t.Fatal(err)
	}
	catID := catCreated.Category.CategoryID

	// Create Category (Invalid JSON)
	createBad := adminJSON(t, e, http.MethodPost, "/api/admin/application-categories", csrf, cookie, "invalid-json")
	if createBad.Code != http.StatusBadRequest {
		t.Fatalf("create category bad json status=%d body=%s", createBad.Code, createBad.Body.String())
	}

	// List Categories
	list := adminJSON(t, e, http.MethodGet, "/api/admin/application-categories", csrf, cookie, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list categories status=%d body=%s", list.Code, list.Body.String())
	}

	// Update Category
	pos2 := 2
	update := adminJSON(t, e, http.MethodPatch, "/api/admin/application-categories/"+catID, csrf, cookie, map[string]any{
		"name": "Cat A Updated", "position": &pos2,
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update category status=%d body=%s", update.Code, update.Body.String())
	}

	// Update Category (Invalid JSON)
	updateBad := adminJSON(t, e, http.MethodPatch, "/api/admin/application-categories/"+catID, csrf, cookie, "invalid-json")
	if updateBad.Code != http.StatusBadRequest {
		t.Fatalf("update category bad json status=%d body=%s", updateBad.Code, updateBad.Body.String())
	}

	// Create App and set categories
	appID := createAndAssignWeblink(t, e, csrf, cookie, "App B")
	setCat := adminJSON(t, e, http.MethodPut, "/api/admin/applications/"+appID+"/categories", csrf, cookie, map[string]any{
		"category_ids": []string{catID},
	})
	if setCat.Code != http.StatusOK {
		t.Fatalf("set application categories status=%d body=%s", setCat.Code, setCat.Body.String())
	}

	// Set categories (Invalid JSON)
	setCatBad := adminJSON(t, e, http.MethodPut, "/api/admin/applications/"+appID+"/categories", csrf, cookie, "invalid-json")
	if setCatBad.Code != http.StatusBadRequest {
		t.Fatalf("set categories bad json status=%d body=%s", setCatBad.Code, setCatBad.Body.String())
	}

	// Delete Category
	deleteCat := adminJSON(t, e, http.MethodDelete, "/api/admin/application-categories/"+catID, csrf, cookie, nil)
	if deleteCat.Code != http.StatusNoContent {
		t.Fatalf("delete category status=%d body=%s", deleteCat.Code, deleteCat.Body.String())
	}
}

func TestAccountApplicationUnauthorized(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)

	// GET /api/account/applications without X-Demo-Sub
	request := httptest.NewRequest(http.MethodGet, "/api/account/applications", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, request)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	// GET /api/account/applications/order without X-Demo-Sub
	request2 := httptest.NewRequest(http.MethodGet, "/api/account/applications/order", http.NoBody)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, request2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec2.Code)
	}

	// PUT /api/account/applications/order without X-Demo-Sub -> This will fail on CSRF validation first (403), because no CSRF cookie/header is sent.
	request3 := httptest.NewRequest(http.MethodPut, "/api/account/applications/order", http.NoBody)
	rec3 := httptest.NewRecorder()
	e.ServeHTTP(rec3, request3)
	if rec3.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec3.Code)
	}

	// PUT /api/account/applications/order with CSRF but without X-Demo-Sub -> This should pass CSRF but fail authentication (401).
	request4 := httptest.NewRequest(http.MethodPut, "/api/account/applications/order", http.NoBody)
	request4.Header.Set("X-Csrf-Token", csrf)
	request4.Header.Set("Origin", "http://idp.test")
	request4.AddCookie(cookie)
	rec4 := httptest.NewRecorder()
	e.ServeHTTP(rec4, request4)
	if rec4.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec4.Code)
	}

	// PUT /api/account/applications/order with bad json
	request5 := httptest.NewRequest(http.MethodPut, "/api/account/applications/order", http.NoBody)
	request5.Header.Set("X-Demo-Sub", "admin")
	request5.Header.Set("Content-Type", "application/json")
	request5.Header.Set("Origin", "http://idp.test")
	request5.Header.Set("X-Csrf-Token", csrf)
	request5.AddCookie(cookie)
	rec5 := httptest.NewRecorder()
	e.ServeHTTP(rec5, request5)
	if rec5.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec5.Code)
	}
}

func TestAdminProtocolConfigLifecycle(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)

	// OIDC App Config Update
	createOIDC := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "OIDC App", "type": "oidc", "redirect_uris": []string{"https://oidc.example/callback"},
		"client_type": "confidential", "token_endpoint_auth_method": "client_secret_post",
	})
	var oidcApp struct {
		Application struct {
			ApplicationID string `json:"application_id"`
		} `json:"application"`
	}
	_ = json.Unmarshal(createOIDC.Body.Bytes(), &oidcApp)
	oidcID := oidcApp.Application.ApplicationID

	// Update OIDC Config
	updateOIDC := adminJSON(t, e, http.MethodPatch, "/api/admin/applications/"+oidcID+"/oidc", csrf, cookie, map[string]any{
		"redirect_uris": []string{"https://oidc-updated.example/callback"},
	})
	if updateOIDC.Code != http.StatusNoContent {
		t.Fatalf("update oidc config status=%d body=%s", updateOIDC.Code, updateOIDC.Body.String())
	}

	// Update OIDC (Invalid JSON)
	updateOIDCBad := adminJSON(t, e, http.MethodPatch, "/api/admin/applications/"+oidcID+"/oidc", csrf, cookie, "invalid-json")
	if updateOIDCBad.Code != http.StatusBadRequest {
		t.Fatalf("update oidc config bad json status=%d body=%s", updateOIDCBad.Code, updateOIDCBad.Body.String())
	}

	// WS-Fed App Config Update
	createWsFed := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "WsFed App", "type": "wsfed", "wtrealm": "urn:wsfed:app", "reply_urls": []string{"https://wsfed.example/reply"},
	})
	var wsfedApp struct {
		Application struct {
			ApplicationID string `json:"application_id"`
		} `json:"application"`
	}
	_ = json.Unmarshal(createWsFed.Body.Bytes(), &wsfedApp)
	wsfedID := wsfedApp.Application.ApplicationID

	// Update WS-Fed Config
	updateWsFed := adminJSON(t, e, http.MethodPatch, "/api/admin/applications/"+wsfedID+"/wsfed", csrf, cookie, map[string]any{
		"reply_urls": []string{"https://wsfed-updated.example/reply"},
	})
	if updateWsFed.Code != http.StatusNoContent {
		t.Fatalf("update wsfed config status=%d body=%s", updateWsFed.Code, updateWsFed.Body.String())
	}

	// Update WS-Fed (Invalid JSON)
	updateWsFedBad := adminJSON(t, e, http.MethodPatch, "/api/admin/applications/"+wsfedID+"/wsfed", csrf, cookie, "invalid-json")
	if updateWsFedBad.Code != http.StatusBadRequest {
		t.Fatalf("update wsfed config bad json status=%d body=%s", updateWsFedBad.Code, updateWsFedBad.Body.String())
	}

	// SAML App Config Update
	createSAML := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "SAML App", "type": "saml", "entity_id": "https://saml.example", "acs_urls": []string{"https://saml.example/acs"},
	})
	var samlApp struct {
		Application struct {
			ApplicationID string `json:"application_id"`
		} `json:"application"`
	}
	_ = json.Unmarshal(createSAML.Body.Bytes(), &samlApp)
	samlID := samlApp.Application.ApplicationID

	// Update SAML Config
	updateSAML := adminJSON(t, e, http.MethodPatch, "/api/admin/applications/"+samlID+"/saml", csrf, cookie, map[string]any{
		"acs_urls": []string{"https://saml-updated.example/acs"},
	})
	if updateSAML.Code != http.StatusNoContent {
		t.Fatalf("update saml config status=%d body=%s", updateSAML.Code, updateSAML.Body.String())
	}

	// Update SAML (Invalid JSON)
	updateSAMLBad := adminJSON(t, e, http.MethodPatch, "/api/admin/applications/"+samlID+"/saml", csrf, cookie, "invalid-json")
	if updateSAMLBad.Code != http.StatusBadRequest {
		t.Fatalf("update saml config bad json status=%d body=%s", updateSAMLBad.Code, updateSAMLBad.Body.String())
	}

	// Update OIDC Config (No Binding)
	createWeblink := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "WebLink App", "type": "weblink", "launch_url": "https://weblink.example",
	})
	var weblinkApp struct {
		Application struct {
			ApplicationID string `json:"application_id"`
		} `json:"application"`
	}
	_ = json.Unmarshal(createWeblink.Body.Bytes(), &weblinkApp)
	weblinkID := weblinkApp.Application.ApplicationID

	updateWeblinkOIDC := adminJSON(t, e, http.MethodPatch, "/api/admin/applications/"+weblinkID+"/oidc", csrf, cookie, map[string]any{
		"redirect_uris": []string{"https://oidc-updated.example/callback"},
	})
	if updateWeblinkOIDC.Code != http.StatusBadRequest {
		t.Fatalf("update weblink oidc config status=%d body=%s", updateWeblinkOIDC.Code, updateWeblinkOIDC.Body.String())
	}
}

func TestAdminApplicationExtraErrors(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)

	// Create Category with empty name -> 400
	badCat := adminJSON(t, e, http.MethodPost, "/api/admin/application-categories", csrf, cookie, map[string]any{"name": ""})
	if badCat.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", badCat.Code)
	}

	// Update non-existent category -> 404
	badUpdateCat := adminJSON(t, e, http.MethodPatch, "/api/admin/application-categories/non-existent", csrf, cookie, map[string]any{"name": "New Name"})
	if badUpdateCat.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", badUpdateCat.Code)
	}

	// Delete non-existent category -> 404
	badDeleteCat := adminJSON(t, e, http.MethodDelete, "/api/admin/application-categories/non-existent", csrf, cookie, nil)
	if badDeleteCat.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", badDeleteCat.Code)
	}

	// Set categories on non-existent app -> 404
	badSetCat := adminJSON(t, e, http.MethodPut, "/api/admin/applications/non-existent/categories", csrf, cookie, map[string]any{"category_ids": []string{"some-cat"}})
	if badSetCat.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", badSetCat.Code)
	}

	// Upload icon without file -> 400
	badUpload := adminJSON(t, e, http.MethodPost, "/api/admin/applications/some-app/icon", csrf, cookie, nil)
	if badUpload.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", badUpload.Code)
	}

	// Delete icon on non-existent app -> 404
	badDeleteIcon := adminJSON(t, e, http.MethodDelete, "/api/admin/applications/non-existent/icon", csrf, cookie, nil)
	if badDeleteIcon.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", badDeleteIcon.Code)
	}

	// Detach binding on non-existent app -> 404
	badDetach := adminJSON(t, e, http.MethodDelete, "/api/admin/applications/non-existent/bindings/oidc", csrf, cookie, nil)
	if badDetach.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", badDetach.Code)
	}

	// List assignments on non-existent app -> 404
	badListAssign := adminJSON(t, e, http.MethodGet, "/api/admin/applications/non-existent/assignments", csrf, cookie, nil)
	if badListAssign.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", badListAssign.Code)
	}

	// Get sign-in policy on non-existent app -> 404
	badGetPolicy := adminJSON(t, e, http.MethodGet, "/api/admin/applications/non-existent/sign-in-policy", csrf, cookie, nil)
	if badGetPolicy.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", badGetPolicy.Code)
	}

	// Update sign-in policy on non-existent app -> 404
	badUpdatePolicy := adminJSON(t, e, http.MethodPut, "/api/admin/applications/non-existent/sign-in-policy", csrf, cookie, map[string]any{"rules": []any{}})
	if badUpdatePolicy.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", badUpdatePolicy.Code)
	}
}
