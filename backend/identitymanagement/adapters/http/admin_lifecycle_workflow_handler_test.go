package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/identitymanagement"
	idmmemory "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/memory"
	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/adapters/http/server"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

func newAdminLifecycleWorkflowHandler(t *testing.T) *echo.Echo {
	t.Helper()
	userRepo := idmmemory.NewUserRepository()
	workflowRepo := idmmemory.NewLifecycleWorkflowRepository()
	groupRepo := idmmemory.NewGroupRepository()
	now := time.Now().UTC()
	userRepo.Seed(&idmdomain.User{
		ID: "admin", PreferredUsername: "admin", PasswordHash: "unused",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	userRepo.Seed(&idmdomain.User{
		ID: "alice", PreferredUsername: "alice", PasswordHash: "unused",
		Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now,
	})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps:          support.Deps{Issuer: "http://idp.test"},
		AuthnResolver: authusecases.DemoHeaderResolver{},
		IdentityManagement: identitymanagement.Module{
			UserRepo: userRepo, GroupRepo: groupRepo, LifecycleWorkflowRepo: workflowRepo,
		},
	})
	return e
}

// wi-222: the dry-run endpoint must report the target User's actual current
// state, not the hard-coded "would_change" the handler used to return for
// every action regardless of whether the User already satisfied it.
func TestAdminLifecycleWorkflowDryRunReflectsActualUserState(t *testing.T) {
	e := newAdminLifecycleWorkflowHandler(t)
	csrf, cookie := adminCSRF(t, e)

	group := adminJSONRequest(t, e, http.MethodPost, "/api/admin/groups", csrf, cookie, map[string]any{"name": "engineering"})
	if group.Code != http.StatusCreated {
		t.Fatalf("create group status=%d body=%s", group.Code, group.Body.String())
	}
	var createdGroup struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(group.Body.Bytes(), &createdGroup); err != nil {
		t.Fatal(err)
	}
	if add := adminJSONRequest(t, e, http.MethodPost, "/api/admin/groups/"+createdGroup.ID+"/members/alice", csrf, cookie, nil); add.Code != http.StatusNoContent {
		t.Fatalf("add member status=%d body=%s", add.Code, add.Body.String())
	}

	create := adminJSONRequest(t, e, http.MethodPost, "/api/admin/lifecycle_workflows", csrf, cookie, map[string]any{
		"name":    "Joiner",
		"trigger": map[string]any{"kind": "user_created"},
		"actions": []map[string]any{{"kind": "add_group_member", "group_id": createdGroup.ID}},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create workflow status=%d body=%s", create.Code, create.Body.String())
	}
	var createdWorkflow struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &createdWorkflow); err != nil {
		t.Fatal(err)
	}

	dryRun := adminJSONRequest(t, e, http.MethodPost, "/api/admin/lifecycle_workflows/"+createdWorkflow.ID+"/dry_run", csrf, cookie, map[string]any{"target_user_id": "alice"})
	if dryRun.Code != http.StatusOK {
		t.Fatalf("dry_run status=%d body=%s", dryRun.Code, dryRun.Body.String())
	}
	var result struct {
		Steps []struct {
			ActionKind  string `json:"action_kind"`
			WouldChange string `json:"would_change"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(dryRun.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Steps) != 1 || result.Steps[0].WouldChange != "no_op" {
		t.Fatalf("dry_run steps = %#v, want a single no_op step (alice is already a member)", result.Steps)
	}

	// alice's membership must be untouched by the dry-run.
	groupsRequest := httptest.NewRequest(http.MethodGet, "/api/admin/users/alice/groups", http.NoBody)
	groupsRequest.Header.Set("X-Demo-Sub", "admin")
	groupsResponse := httptest.NewRecorder()
	e.ServeHTTP(groupsResponse, groupsRequest)
	if groupsResponse.Code != http.StatusOK {
		t.Fatalf("user groups status=%d body=%s", groupsResponse.Code, groupsResponse.Body.String())
	}
	var view struct {
		Groups []struct {
			ID string `json:"id"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(groupsResponse.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if len(view.Groups) != 1 {
		t.Fatalf("dry-run must not mutate membership: groups=%#v", view.Groups)
	}

	missingUser := adminJSONRequest(t, e, http.MethodPost, "/api/admin/lifecycle_workflows/"+createdWorkflow.ID+"/dry_run", csrf, cookie, map[string]any{"target_user_id": "no-such-user"})
	if missingUser.Code != http.StatusBadRequest {
		t.Fatalf("dry_run for missing user status=%d body=%s", missingUser.Code, missingUser.Body.String())
	}
}
