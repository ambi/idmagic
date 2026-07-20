package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

func newAdminGroupHandler(t *testing.T) (*echo.Echo, *groupmemory.GroupRepository) {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	groupRepo := groupmemory.NewGroupRepository()
	now := time.Now().UTC()
	userRepo.Seed(&userdomain.User{
		ID: "admin", PreferredUsername: "admin", PasswordHash: "unused",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	userRepo.Seed(&userdomain.User{
		ID: "alice", PreferredUsername: "alice", PasswordHash: "unused",
		Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{Issuer: "http://idp.test"}, UserRepo: userRepo, GroupRepo: groupRepo,
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	return e, groupRepo
}

func TestAdminGroupAPIRequiresAdminRole(t *testing.T) {
	e, _ := newAdminGroupHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/groups", http.NoBody)
	request.Header.Set("X-Demo-Sub", "alice")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAdminGroupAPICreateAddMemberAndEffectiveRoles(t *testing.T) {
	e, _ := newAdminGroupHandler(t)
	csrf, cookie := adminCSRF(t, e)

	create := adminJSONRequest(t, e, http.MethodPost, "/api/admin/groups", csrf, cookie, map[string]any{
		"name": "engineering", "roles": []string{"catalog:read"},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	// 名前一意性: 409
	conflict := adminJSONRequest(t, e, http.MethodPost, "/api/admin/groups", csrf, cookie, map[string]any{"name": "engineering"})
	if conflict.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", conflict.Code, conflict.Body.String())
	}

	add := adminJSONRequest(t, e, http.MethodPost, "/api/admin/groups/"+created.ID+"/members/alice", csrf, cookie, nil)
	if add.Code != http.StatusNoContent {
		t.Fatalf("add member status=%d body=%s", add.Code, add.Body.String())
	}
	// 冪等な再追加も 204
	if again := adminJSONRequest(t, e, http.MethodPost, "/api/admin/groups/"+created.ID+"/members/alice", csrf, cookie, nil); again.Code != http.StatusNoContent {
		t.Fatalf("idempotent add status=%d", again.Code)
	}

	groupsResp := httptest.NewRequest(http.MethodGet, "/api/admin/users/alice/groups", http.NoBody)
	groupsResp.Header.Set("X-Demo-Sub", "admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, groupsResp)
	if rec.Code != http.StatusOK {
		t.Fatalf("user groups status=%d body=%s", rec.Code, rec.Body.String())
	}
	var view struct {
		EffectiveRoles []string `json:"effective_roles"`
		GroupRoles     []string `json:"group_roles"`
		DirectRoles    []string `json:"direct_roles"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if len(view.EffectiveRoles) != 1 || view.EffectiveRoles[0] != "catalog:read" {
		t.Fatalf("effective roles=%v", view.EffectiveRoles)
	}
	if len(view.DirectRoles) != 0 {
		t.Fatalf("direct roles=%v", view.DirectRoles)
	}
}

// alice をグループ経由で admin にすると admin API を通過できる (effective roles)。
func TestGroupDerivedAdminRolePassesRBAC(t *testing.T) {
	e, groupRepo := newAdminGroupHandler(t)
	ctx := httptest.NewRequest(http.MethodGet, "/", http.NoBody).Context()
	group := &groupdomain.Group{ID: "group_admins", TenantID: tenancydomain.DefaultTenantID, Name: "admins", Roles: []string{"admin"}, CreatedAt: time.Now().UTC()}
	if err := groupRepo.Save(ctx, group); err != nil {
		t.Fatal(err)
	}
	if _, err := groupRepo.AddMember(ctx, &groupdomain.GroupMember{GroupID: group.ID, UserID: "alice", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/admin/groups", http.NoBody)
	request.Header.Set("X-Demo-Sub", "alice")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("group-derived admin denied: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestDynamicGroupRulePreviewEnableAndManualMembershipRejection(t *testing.T) {
	e, _ := newAdminGroupHandler(t)
	csrf, cookie := adminCSRF(t, e)
	create := adminJSONRequest(t, e, http.MethodPost, "/api/admin/groups", csrf, cookie, map[string]any{
		"name": "alice-only", "membership_type": "dynamic",
		"dynamic_rule": map[string]any{"expression": `user.preferred_username == "alice"`},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	preview := adminJSONRequest(t, e, http.MethodPost, "/api/admin/groups/"+created.ID+"/dynamic-rule/preview", csrf, cookie, map[string]any{
		"expression": `user.preferred_username == "alice"`, "user_ids": []string{"alice"},
	})
	if preview.Code != http.StatusOK || !containsJSON(preview.Body.Bytes(), `"matched":true`) {
		t.Fatalf("preview status=%d body=%s", preview.Code, preview.Body.String())
	}
	enable := adminJSONRequest(t, e, http.MethodPost, "/api/admin/groups/"+created.ID+"/dynamic-rule/enable", csrf, cookie, nil)
	if enable.Code != http.StatusOK {
		t.Fatalf("enable status=%d body=%s", enable.Code, enable.Body.String())
	}
	detailRequest := httptest.NewRequest(http.MethodGet, "/api/admin/groups/"+created.ID, http.NoBody)
	detailRequest.Header.Set("X-Demo-Sub", "admin")
	detail := httptest.NewRecorder()
	e.ServeHTTP(detail, detailRequest)
	if detail.Code != http.StatusOK || !containsJSON(detail.Body.Bytes(), `"preferred_username":"alice"`) {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}
	manual := adminJSONRequest(t, e, http.MethodPost, "/api/admin/groups/"+created.ID+"/members/admin", csrf, cookie, nil)
	if manual.Code != http.StatusConflict {
		t.Fatalf("manual membership status=%d body=%s", manual.Code, manual.Body.String())
	}
}

func containsJSON(body []byte, fragment string) bool {
	return strings.Contains(string(body), fragment)
}
