package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

// doScimJSON issues a bearer-authenticated request with a JSON body and
// decodes the response, mirroring doScimGet for POST/PUT/PATCH.
func doScimJSON(t *testing.T, e *echo.Echo, method, tokenStr, path string, body map[string]any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	req.Header.Set("Content-Type", "application/scim+json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	var respBody map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &respBody)
	return rec, respBody
}

func patchOp(op, path string, value any) map[string]any {
	return map[string]any{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]any{
			{"op": op, "path": path, "value": value},
		},
	}
}

// interfaces.CreateScimUser: userName は必須、id は server-assigned、
// meta が一貫して返る (RFC7643-CORE-RESOURCES adoption:partial)。
func TestScimCreateUserResourceContract(t *testing.T) {
	ctx := context.Background()
	e, usecasesInst := newScimTestHarness()
	tokenStr, _, err := usecasesInst.GenerateToken(ctx, tenancydomain.DefaultTenantID, "Integration", 30)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("userName is required", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%v", rec.Code, body)
		}
		if body["scimType"] != "invalidValue" {
			t.Errorf("scimType = %v, want invalidValue", body["scimType"])
		}
	})

	t.Run("client-supplied id is ignored", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
			"userName": "alice@example.com",
			"id":       "client-chosen-id",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%v", rec.Code, body)
		}
		if body["id"] == "client-chosen-id" {
			t.Error("expected server-assigned id, client-supplied id was honored")
		}
		meta, ok := body["meta"].(map[string]any)
		if !ok {
			t.Fatal("expected meta object in response")
		}
		if meta["resourceType"] != "User" || meta["created"] == "" || meta["lastModified"] == "" || meta["location"] == "" {
			t.Errorf("expected complete meta, got %+v", meta)
		}
	})

	t.Run("duplicate userName is a 409 uniqueness conflict", func(t *testing.T) {
		body := map[string]any{"userName": "dupuser@example.com"}
		rec1, _ := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", body)
		if rec1.Code != http.StatusCreated {
			t.Fatalf("expected first create to succeed, got %d", rec1.Code)
		}
		rec2, body2 := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", body)
		if rec2.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d body=%v", rec2.Code, body2)
		}
		if body2["scimType"] != "uniqueness" {
			t.Errorf("scimType = %v, want uniqueness", body2["scimType"])
		}
	})
}

// interfaces.UpdateScimUser: PUT は完全置換 (省略した属性は既定値にリセット)。
func TestScimUpdateUserFullReplace(t *testing.T) {
	ctx := context.Background()
	e, usecasesInst := newScimTestHarness()
	tokenStr, _, err := usecasesInst.GenerateToken(ctx, tenancydomain.DefaultTenantID, "Integration", 30)
	if err != nil {
		t.Fatal(err)
	}

	createRec, created := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
		"userName": "bjensen@example.com",
		"name":     map[string]any{"givenName": "Barbara", "familyName": "Jensen"},
		"active":   false,
	})
	if createRec.Code != http.StatusCreated {
		t.Fatalf("setup create failed: %d", createRec.Code)
	}
	scimID := created["id"].(string)

	t.Run("missing userName is invalidValue", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPut, tokenStr, "/scim/v2/Users/"+scimID, map[string]any{})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%v", rec.Code, body)
		}
		if body["scimType"] != "invalidValue" {
			t.Errorf("scimType = %v, want invalidValue", body["scimType"])
		}
	})

	t.Run("omitted mutable attributes reset to defaults", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPut, tokenStr, "/scim/v2/Users/"+scimID, map[string]any{
			"userName": "bjensen@example.com",
			"id":       "attempted-override",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", rec.Code, body)
		}
		if body["id"] != scimID {
			t.Errorf("id changed via PUT body, got %v want %v", body["id"], scimID)
		}
		name, _ := body["name"].(map[string]any)
		if name["givenName"] != "" || name["familyName"] != "" {
			t.Errorf("expected name fields reset to empty, got %+v", name)
		}
		if body["active"] != true {
			t.Errorf("expected active reset to default true, got %v", body["active"])
		}
	})
}

// interfaces.PatchScimUser: RFC7644-PATCH allowlist と mutability/invalidPath/invalidValue。
func TestScimPatchUserResourceContract(t *testing.T) {
	ctx := context.Background()
	e, usecasesInst := newScimTestHarness()
	tokenStr, _, err := usecasesInst.GenerateToken(ctx, tenancydomain.DefaultTenantID, "Integration", 30)
	if err != nil {
		t.Fatal(err)
	}
	createRec, created := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
		"userName": "carlos@example.com",
	})
	if createRec.Code != http.StatusCreated {
		t.Fatalf("setup create failed: %d", createRec.Code)
	}
	scimID := created["id"].(string)

	t.Run("supported path replace succeeds", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID, patchOp("replace", "name.givenName", "Carlos"))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", rec.Code, body)
		}
		name, _ := body["name"].(map[string]any)
		if name["givenName"] != "Carlos" {
			t.Errorf("givenName = %v, want Carlos", name["givenName"])
		}
	})

	t.Run("unknown path is invalidPath", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID, patchOp("replace", "nickName", "x"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%v", rec.Code, body)
		}
		if body["scimType"] != "invalidPath" {
			t.Errorf("scimType = %v, want invalidPath", body["scimType"])
		}
	})

	t.Run("readOnly path is mutability error", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID, patchOp("replace", "id", "new-id"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%v", rec.Code, body)
		}
		if body["scimType"] != "mutability" {
			t.Errorf("scimType = %v, want mutability", body["scimType"])
		}
	})

	t.Run("unsupported op is invalidValue", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID, patchOp("delete", "active", true))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%v", rec.Code, body)
		}
		if body["scimType"] != "invalidValue" {
			t.Errorf("scimType = %v, want invalidValue", body["scimType"])
		}
	})
}

// interfaces.CreateScimGroup / UpdateScimGroup / PatchScimGroup: displayName
// 必須、id は server-assigned、解決できない member は invalidValue。
func TestScimGroupResourceContract(t *testing.T) {
	ctx := context.Background()
	e, usecasesInst := newScimTestHarness()
	tokenStr, _, err := usecasesInst.GenerateToken(ctx, tenancydomain.DefaultTenantID, "Integration", 30)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("displayName is required", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Groups", map[string]any{})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%v", rec.Code, body)
		}
		if body["scimType"] != "invalidValue" {
			t.Errorf("scimType = %v, want invalidValue", body["scimType"])
		}
	})

	t.Run("unresolvable member is rejected and group is not created", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Groups", map[string]any{
			"displayName": "Ghosts",
			"members":     []map[string]any{{"value": "does-not-exist"}},
		})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%v", rec.Code, body)
		}
		if body["scimType"] != "invalidValue" {
			t.Errorf("scimType = %v, want invalidValue", body["scimType"])
		}

		listRec, listBody := doScimGet(t, e, tokenStr, "/scim/v2/Groups?filter="+url.QueryEscape(`displayName eq "Ghosts"`))
		_ = listRec
		if int(listBody["totalResults"].(float64)) != 0 {
			t.Errorf("expected no group created, totalResults = %v", listBody["totalResults"])
		}
	})

	t.Run("duplicate displayName is a 409 uniqueness conflict", func(t *testing.T) {
		body := map[string]any{"displayName": "Duplicates"}
		rec1, _ := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Groups", body)
		if rec1.Code != http.StatusCreated {
			t.Fatalf("expected first create to succeed, got %d", rec1.Code)
		}
		rec2, body2 := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Groups", body)
		if rec2.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d body=%v", rec2.Code, body2)
		}
		if body2["scimType"] != "uniqueness" {
			t.Errorf("scimType = %v, want uniqueness", body2["scimType"])
		}
	})

	t.Run("client-supplied id is ignored and meta is complete", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Groups", map[string]any{
			"displayName": "RealTeam",
			"id":          "client-chosen-id",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%v", rec.Code, body)
		}
		if body["id"] == "client-chosen-id" {
			t.Error("expected server-assigned id, client-supplied id was honored")
		}
		meta, _ := body["meta"].(map[string]any)
		if meta["location"] == "" || meta["resourceType"] != "Group" {
			t.Errorf("expected complete meta, got %+v", meta)
		}
	})

	t.Run("PUT replaces members fully, omitted members clears all", func(t *testing.T) {
		userRec, user := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{"userName": "member1@example.com"})
		if userRec.Code != http.StatusCreated {
			t.Fatalf("setup user create failed: %d", userRec.Code)
		}
		userScimID := user["id"].(string)

		groupRec, group := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Groups", map[string]any{
			"displayName": "ReplaceTest",
			"members":     []map[string]any{{"value": userScimID}},
		})
		if groupRec.Code != http.StatusCreated {
			t.Fatalf("setup group create failed: %d", groupRec.Code)
		}
		groupScimID := group["id"].(string)

		putRec, putBody := doScimJSON(t, e, http.MethodPut, tokenStr, "/scim/v2/Groups/"+groupScimID, map[string]any{
			"displayName": "ReplaceTest",
		})
		if putRec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", putRec.Code, putBody)
		}
		members, _ := putBody["members"].([]any)
		if len(members) != 0 {
			t.Errorf("expected members cleared by omission, got %v", members)
		}
	})

	t.Run("PATCH add with unresolvable member is invalidValue", func(t *testing.T) {
		groupRec, group := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Groups", map[string]any{"displayName": "PatchTest"})
		if groupRec.Code != http.StatusCreated {
			t.Fatalf("setup group create failed: %d", groupRec.Code)
		}
		groupScimID := group["id"].(string)

		rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Groups/"+groupScimID,
			patchOp("add", "members", []map[string]any{{"value": "does-not-exist"}}))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%v", rec.Code, body)
		}
		if body["scimType"] != "invalidValue" {
			t.Errorf("scimType = %v, want invalidValue", body["scimType"])
		}
	})

	t.Run("PATCH unknown path is invalidPath", func(t *testing.T) {
		groupRec, group := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Groups", map[string]any{"displayName": "PatchPathTest"})
		if groupRec.Code != http.StatusCreated {
			t.Fatalf("setup group create failed: %d", groupRec.Code)
		}
		groupScimID := group["id"].(string)

		rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Groups/"+groupScimID, patchOp("replace", "description", "x"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%v", rec.Code, body)
		}
		if body["scimType"] != "invalidPath" {
			t.Errorf("scimType = %v, want invalidPath", body["scimType"])
		}
	})
}

// interfaces.GetScimSchemas: 空配列ではなく User/Group の実属性を返す。
func TestScimGetSchemasReturnsRealAttributes(t *testing.T) {
	ctx := context.Background()
	e, usecasesInst := newScimTestHarness()
	tokenStr, _, err := usecasesInst.GenerateToken(ctx, tenancydomain.DefaultTenantID, "Integration", 30)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/scim/v2/Schemas", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var schemas []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &schemas); err != nil {
		t.Fatalf("failed to decode schemas: %v", err)
	}
	if len(schemas) < 2 {
		t.Fatalf("expected at least 2 schemas (User, Group), got %d", len(schemas))
	}
	for _, schema := range schemas {
		attrs, _ := schema["attributes"].([]any)
		if len(attrs) == 0 {
			t.Errorf("schema %v has empty attributes", schema["id"])
		}
	}
}
