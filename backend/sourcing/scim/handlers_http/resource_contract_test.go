package handlers_http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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
	e, _, apiTokens := newScimTestHarness()
	tokenStr := issueAllScimToken(t, apiTokens)

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

	t.Run("emails project by primary and response is canonical", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
			"userName": "email-priority@example.com",
			"emails": []any{
				map[string]any{"value": "work@example.com", "type": "work"},
				map[string]any{"value": "primary@example.com", "type": "home", "primary": true},
			},
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%v", rec.Code, body)
		}
		emails, _ := body["emails"].([]any)
		if len(emails) != 1 {
			t.Fatalf("expected one canonical email, got %v", body["emails"])
		}
		email, _ := emails[0].(map[string]any)
		if email["value"] != "primary@example.com" || email["type"] != "work" || email["primary"] != true {
			t.Errorf("unexpected canonical email: %v", email)
		}
	})

	t.Run("omitted email is omitted from response", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
			"userName": "without-email@example.com",
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%v", rec.Code, body)
		}
		if _, exists := body["emails"]; exists {
			t.Errorf("expected emails to be omitted, got %v", body["emails"])
		}
	})

	t.Run("invalid emails reject the whole create", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
			"userName": "two-primary@example.com",
			"emails": []any{
				map[string]any{"value": "one@example.com", "primary": true},
				map[string]any{"value": "two@example.com", "primary": true},
			},
		})
		if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidValue" {
			t.Fatalf("expected 400 invalidValue, got %d body=%v", rec.Code, body)
		}
		_, listBody := doScimGet(t, e, tokenStr, "/scim/v2/Users?filter="+url.QueryEscape(`userName eq "two-primary@example.com"`))
		if int(listBody["totalResults"].(float64)) != 0 {
			t.Errorf("expected invalid user not to be created, got %v", listBody)
		}
	})

	for _, attr := range []string{"phoneNumbers", "addresses"} {
		t.Run("unsupported "+attr+" is invalidValue", func(t *testing.T) {
			rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
				"userName": "unsupported-" + attr + "@example.com",
				attr:       []any{},
			})
			if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidValue" {
				t.Fatalf("expected 400 invalidValue, got %d body=%v", rec.Code, body)
			}
		})
	}
}

// REQ-SOURCING-007: enterprise extension の employeeNumber/department/manager を
// CreateScimUser で対応する。
func TestScimCreateUserEnterpriseExtension(t *testing.T) {
	e, _, apiTokens := newScimTestHarness()
	tokenStr := issueAllScimToken(t, apiTokens)
	const enterpriseURN = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"

	t.Run("employeeNumber and department round-trip", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
			"userName": "enterprise1@example.com",
			enterpriseURN: map[string]any{
				"employeeNumber": "701984",
				"department":     "Tour Operations",
			},
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%v", rec.Code, body)
		}
		schemas, _ := body["schemas"].([]any)
		found := false
		for _, s := range schemas {
			if s == enterpriseURN {
				found = true
			}
		}
		if !found {
			t.Errorf("expected schemas to include enterprise extension URN, got %v", schemas)
		}
		ext, ok := body[enterpriseURN].(map[string]any)
		if !ok {
			t.Fatalf("expected enterprise extension object in response, got %v", body[enterpriseURN])
		}
		if ext["employeeNumber"] != "701984" || ext["department"] != "Tour Operations" {
			t.Errorf("unexpected enterprise extension: %+v", ext)
		}

		// GET must round-trip the same values a separate code path (GetUser,
		// not CreateUser) than the create response just asserted above.
		getRec, getBody := doScimGet(t, e, tokenStr, "/scim/v2/Users/"+body["id"].(string))
		if getRec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", getRec.Code, getBody)
		}
		getExt, ok := getBody[enterpriseURN].(map[string]any)
		if !ok || getExt["employeeNumber"] != "701984" || getExt["department"] != "Tour Operations" {
			t.Errorf("GET did not round-trip enterprise extension: %+v", getBody[enterpriseURN])
		}
	})

	t.Run("manager resolves to an existing tenant User and round-trips its scim id", func(t *testing.T) {
		mgrRec, mgr := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{"userName": "manager1@example.com"})
		if mgrRec.Code != http.StatusCreated {
			t.Fatalf("setup manager create failed: %d", mgrRec.Code)
		}
		mgrScimID := mgr["id"].(string)

		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
			"userName":    "report1@example.com",
			enterpriseURN: map[string]any{"manager": map[string]any{"value": mgrScimID}},
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%v", rec.Code, body)
		}
		ext, _ := body[enterpriseURN].(map[string]any)
		manager, _ := ext["manager"].(map[string]any)
		if manager["value"] != mgrScimID {
			t.Errorf("manager.value = %v, want %v", manager["value"], mgrScimID)
		}
	})

	t.Run("manager referencing an unknown scim id is invalidValue and creates nothing", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
			"userName":    "orphan1@example.com",
			enterpriseURN: map[string]any{"manager": map[string]any{"value": "does-not-exist"}},
		})
		if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidValue" {
			t.Fatalf("expected 400 invalidValue, got %d body=%v", rec.Code, body)
		}
		_, listBody := doScimGet(t, e, tokenStr, "/scim/v2/Users?filter="+url.QueryEscape(`userName eq "orphan1@example.com"`))
		if int(listBody["totalResults"].(float64)) != 0 {
			t.Errorf("expected invalid user not to be created, got %v", listBody)
		}
	})

	t.Run("employeeNumber wrong type is invalidValue", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
			"userName":    "badtype1@example.com",
			enterpriseURN: map[string]any{"employeeNumber": 42},
		})
		if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidValue" {
			t.Fatalf("expected 400 invalidValue, got %d body=%v", rec.Code, body)
		}
	})

	t.Run("without enterprise extension attributes, schemas omits the URN", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{"userName": "plain1@example.com"})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%v", rec.Code, body)
		}
		if _, exists := body[enterpriseURN]; exists {
			t.Errorf("expected no enterprise extension object, got %v", body[enterpriseURN])
		}
		schemas, _ := body["schemas"].([]any)
		for _, s := range schemas {
			if s == enterpriseURN {
				t.Errorf("expected schemas to omit enterprise extension URN, got %v", schemas)
			}
		}
	})
}

// REQ-SOURCING-007: PATCH は bare 名と URN 修飾済みパスの両方で enterprise
// extension 属性を対応する。
func TestScimPatchUserEnterpriseExtension(t *testing.T) {
	e, _, apiTokens := newScimTestHarness()
	tokenStr := issueAllScimToken(t, apiTokens)
	const enterpriseURN = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"

	createRec, created := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{"userName": "patchee@example.com"})
	if createRec.Code != http.StatusCreated {
		t.Fatalf("setup create failed: %d", createRec.Code)
	}
	scimID := created["id"].(string)

	t.Run("bare employeeNumber path", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID, patchOp("replace", "employeeNumber", "12345"))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", rec.Code, body)
		}
		ext, _ := body[enterpriseURN].(map[string]any)
		if ext["employeeNumber"] != "12345" {
			t.Errorf("employeeNumber = %v, want 12345", ext["employeeNumber"])
		}
	})

	t.Run("urn-qualified department path", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID,
			patchOp("replace", enterpriseURN+":department", "Engineering"))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", rec.Code, body)
		}
		ext, _ := body[enterpriseURN].(map[string]any)
		if ext["department"] != "Engineering" {
			t.Errorf("department = %v, want Engineering", ext["department"])
		}
	})

	t.Run("manager as bare string value (Entra quirk)", func(t *testing.T) {
		mgrRec, mgr := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{"userName": "manager2@example.com"})
		if mgrRec.Code != http.StatusCreated {
			t.Fatalf("setup manager create failed: %d", mgrRec.Code)
		}
		mgrScimID := mgr["id"].(string)

		rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID, patchOp("replace", "manager", mgrScimID))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", rec.Code, body)
		}
		ext, _ := body[enterpriseURN].(map[string]any)
		manager, _ := ext["manager"].(map[string]any)
		if manager["value"] != mgrScimID {
			t.Errorf("manager.value = %v, want %v", manager["value"], mgrScimID)
		}
	})

	t.Run("remove clears the attribute", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID, patchOp("remove", "employeeNumber", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", rec.Code, body)
		}
		ext, _ := body[enterpriseURN].(map[string]any)
		if _, exists := ext["employeeNumber"]; exists {
			t.Errorf("expected employeeNumber removed, got %v", ext["employeeNumber"])
		}
	})

	t.Run("manager referencing an unknown scim id is invalidValue", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID, patchOp("replace", "manager", "does-not-exist"))
		if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidValue" {
			t.Fatalf("expected 400 invalidValue, got %d body=%v", rec.Code, body)
		}
	})
}

// interfaces.UpdateScimUser: PUT は完全置換 (省略した属性は既定値にリセット)。
func TestScimUpdateUserFullReplace(t *testing.T) {
	e, _, apiTokens := newScimTestHarness()
	tokenStr := issueAllScimToken(t, apiTokens)

	createRec, created := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
		"userName": "bjensen@example.com",
		"name":     map[string]any{"givenName": "Barbara", "familyName": "Jensen"},
		"emails":   []any{map[string]any{"value": "bjensen@example.com", "primary": true}},
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
		if _, exists := body["emails"]; exists {
			t.Errorf("expected omitted emails to clear canonical email, got %v", body["emails"])
		}
	})

	// REQ-SOURCING-007: PUT の完全置換は enterprise extension 属性にも適用される。
	t.Run("omitted enterprise extension resets to defaults", func(t *testing.T) {
		const enterpriseURN = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
		setRec, setBody := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID,
			patchOp("replace", "employeeNumber", "999"))
		if setRec.Code != http.StatusOK {
			t.Fatalf("setup patch failed: %d body=%v", setRec.Code, setBody)
		}

		rec, body := doScimJSON(t, e, http.MethodPut, tokenStr, "/scim/v2/Users/"+scimID, map[string]any{
			"userName": "bjensen@example.com",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", rec.Code, body)
		}
		if _, exists := body[enterpriseURN]; exists {
			t.Errorf("expected enterprise extension cleared by omission, got %v", body[enterpriseURN])
		}
	})
}

// interfaces.PatchScimUser: RFC7644-PATCH allowlist と mutability/invalidPath/invalidValue。
func TestScimPatchUserResourceContract(t *testing.T) {
	e, _, apiTokens := newScimTestHarness()
	tokenStr := issueAllScimToken(t, apiTokens)
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

	t.Run("emails project by work fallback", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID,
			patchOp("replace", "emails", []any{
				map[string]any{"value": "home@example.com", "type": "home"},
				map[string]any{"value": "work@example.com", "type": "WORK"},
			}))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", rec.Code, body)
		}
		emails, _ := body["emails"].([]any)
		email, _ := emails[0].(map[string]any)
		if email["value"] != "work@example.com" || email["type"] != "work" || email["primary"] != true {
			t.Errorf("unexpected canonical email: %v", email)
		}
	})

	t.Run("remove emails clears canonical email", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID,
			patchOp("remove", "emails", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", rec.Code, body)
		}
		if _, exists := body["emails"]; exists {
			t.Errorf("expected emails to be omitted after remove, got %v", body["emails"])
		}
	})

	for _, path := range []string{"phoneNumbers", "addresses"} {
		t.Run("unsupported "+path+" path is invalidPath", func(t *testing.T) {
			rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID,
				patchOp("replace", path, []any{}))
			if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidPath" {
				t.Fatalf("expected 400 invalidPath, got %d body=%v", rec.Code, body)
			}
		})
	}
}

// interfaces.CreateScimGroup / UpdateScimGroup / PatchScimGroup: displayName
// 必須、id は server-assigned、解決できない member は invalidValue。
func TestScimGroupResourceContract(t *testing.T) {
	e, _, apiTokens := newScimTestHarness()
	tokenStr := issueAllScimToken(t, apiTokens)

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

	t.Run("User member type is accepted and returned canonically", func(t *testing.T) {
		userRec, user := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
			"userName": "typed-member@example.com",
		})
		if userRec.Code != http.StatusCreated {
			t.Fatalf("setup user create failed: %d", userRec.Code)
		}
		groupRec, group := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Groups", map[string]any{
			"displayName": "TypedMembers",
			"members": []any{map[string]any{
				"value": user["id"],
				"type":  "uSeR",
			}},
		})
		if groupRec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%v", groupRec.Code, group)
		}
		members, _ := group["members"].([]any)
		member, _ := members[0].(map[string]any)
		if member["type"] != "User" {
			t.Errorf("member type = %v, want User", member["type"])
		}
	})

	t.Run("Group member type is rejected without creating the group", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Groups", map[string]any{
			"displayName": "NestedGroup",
			"members": []any{map[string]any{
				"value": "some-group",
				"type":  "Group",
			}},
		})
		if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidValue" {
			t.Fatalf("expected 400 invalidValue, got %d body=%v", rec.Code, body)
		}
		_, listBody := doScimGet(t, e, tokenStr, "/scim/v2/Groups?filter="+url.QueryEscape(`displayName eq "NestedGroup"`))
		if int(listBody["totalResults"].(float64)) != 0 {
			t.Errorf("expected nested group not to be created, got %v", listBody)
		}
	})
}

// interfaces.GetScimSchemas: 空配列ではなく User/Group の実属性を返す。
func TestScimGetSchemasReturnsRealAttributes(t *testing.T) {
	e, _, apiTokens := newScimTestHarness()
	tokenStr := issueAllScimToken(t, apiTokens)

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

// REQ-SOURCING-007: GetScimSchemas は enterprise extension schema を、
// GetScimResourceTypes は User の schemaExtensions を広告する。
func TestScimEnterpriseExtensionDiscovery(t *testing.T) {
	e, _, apiTokens := newScimTestHarness()
	tokenStr := issueAllScimToken(t, apiTokens)
	const enterpriseURN = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"

	t.Run("Schemas includes the enterprise extension", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/Schemas", http.NoBody)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		var schemas []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &schemas); err != nil {
			t.Fatalf("failed to decode schemas: %v", err)
		}
		found := false
		for _, schema := range schemas {
			if schema["id"] == enterpriseURN {
				found = true
				attrs, _ := schema["attributes"].([]any)
				if len(attrs) == 0 {
					t.Errorf("enterprise extension schema has empty attributes")
				}
			}
		}
		if !found {
			t.Errorf("expected enterprise extension schema in %v", schemas)
		}
	})

	t.Run("ResourceTypes advertises the extension on User", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/scim/v2/ResourceTypes", http.NoBody)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		var types []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &types); err != nil {
			t.Fatalf("failed to decode resource types: %v", err)
		}
		for _, rt := range types {
			if rt["id"] != "User" {
				continue
			}
			exts, _ := rt["schemaExtensions"].([]any)
			found := false
			for _, e := range exts {
				ext, _ := e.(map[string]any)
				if ext["schema"] == enterpriseURN {
					found = true
				}
			}
			if !found {
				t.Errorf("expected User schemaExtensions to include %q, got %v", enterpriseURN, exts)
			}
		}
	})
}
