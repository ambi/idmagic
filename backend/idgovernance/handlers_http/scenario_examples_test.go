package handlers_http_test

// IdGovernance の管理 API の具体例を、製品と同じ `Register` の組み立てで観測する。
// 拒否の具体例は応答だけでなく、拒否が防いだ効果（保存された定義、WorkflowRun、Job）を
// 保存先から読み直して確かめる。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/idgovernance"
	igmemory "github.com/ambi/idmagic/backend/idgovernance/db_memory"
	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	igusecases "github.com/ambi/idmagic/backend/idgovernance/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/jobs"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	acmeTenant    = "acme"
	defaultAdmin  = "admin"
	acmeAdmin     = "acme-admin"
	nonAdminAlice = "alice"
)

// governanceStack は管理 API と、拒否が防いだ効果を読み直す保存先を持つ。
type governanceStack struct {
	e         *echo.Echo
	users     *usermemory.UserRepository
	groups    *groupmemory.GroupRepository
	apps      *appmemory.ApplicationRepository
	workflows *igmemory.LifecycleWorkflowRepository
	runs      *igmemory.LifecycleWorkflowRunRepository
	jobs      *jobsmemory.JobRepository
}

func newGovernanceStack(t *testing.T) *governanceStack {
	t.Helper()
	now := time.Now().UTC()
	tenants := tenancymemory.NewTenantRepository()
	for _, tenant := range []*tenancydomain.Tenant{
		{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: now},
		{ID: acmeTenant, Realm: acmeTenant, DisplayName: "Acme", Status: tenancydomain.TenantStatusActive, CreatedAt: now},
	} {
		if err := tenants.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	s := &governanceStack{
		users: usermemory.NewUserRepository(), groups: groupmemory.NewGroupRepository(),
		apps: appmemory.NewApplicationRepository(), workflows: igmemory.NewLifecycleWorkflowRepository(),
		runs: igmemory.NewLifecycleWorkflowRunRepository(), jobs: jobsmemory.NewJobRepository(),
	}
	schemas := usermemory.NewTenantUserAttributeSchemaRepository()
	if err := schemas.Save(context.Background(), &userdomain.TenantUserAttributeSchema{
		TenantID: acmeTenant, CreatedAt: now, UpdatedAt: now,
		Attributes: []userdomain.UserAttributeDef{{Key: "badge_color", Type: idmdomain.AttributeTypeString}},
	}); err != nil {
		t.Fatal(err)
	}
	for _, user := range []*userdomain.User{
		{ID: defaultAdmin, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin", Roles: []string{"admin"}},
		{ID: nonAdminAlice, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice", Roles: []string{"member"}},
		{ID: acmeAdmin, TenantID: acmeTenant, PreferredUsername: "acme-admin", Roles: []string{"admin"}},
	} {
		user.PasswordHash, user.CreatedAt, user.UpdatedAt = "unused", now, now
		user.Lifecycle = userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}
		s.users.Seed(user)
	}
	s.e = echo.New()
	httpadapter.Register(s.e, httpadapter.Deps{
		Issuer:        "http://idp.test",
		AuthnResolver: authusecases.DemoHeaderResolver{},
		TenantRepo:    tenants,
		Tenancy:       tenancy.Module{TenantRepo: tenants, AttrSchemaRepo: schemas},
		IdManagement:  idmanagement.Module{UserRepo: s.users, GroupRepo: s.groups},
		Application:   application.Module{Repo: s.apps, AssignmentRepo: appmemory.NewApplicationAssignmentRepository()},
		Jobs:          jobs.Module{Repo: s.jobs},
		IdGovernance:  idgovernance.Module{LifecycleWorkflowRepo: s.workflows, LifecycleWorkflowRunRepo: s.runs},
	})
	return s
}

func (s *governanceStack) seedGroup(t *testing.T, tenantID, id string, membership groupdomain.GroupMembershipType) {
	t.Helper()
	now := time.Now().UTC()
	if err := s.groups.Save(context.Background(), &groupdomain.Group{ID: id, TenantID: tenantID, Name: id, MembershipType: membership, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
}

// call は realm の管理 API を sub として呼ぶ。状態を変える要求には、その主体の
// CSRF トークンと cookie を付ける。
func (s *governanceStack) call(t *testing.T, realm, sub, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, "/realms/"+realm+path, bytes.NewReader(payload))
	request.Header.Set("X-Demo-Sub", sub)
	if method != http.MethodGet {
		csrf, cookie := s.csrf(t, realm, sub)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "http://idp.test")
		request.Header.Set("X-Csrf-Token", csrf)
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	s.e.ServeHTTP(response, request)
	return response
}

func (s *governanceStack) csrf(t *testing.T, realm, sub string) (string, *http.Cookie) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/realms/"+realm+"/api/auth/account", http.NoBody)
	request.Header.Set("X-Demo-Sub", sub)
	response := httptest.NewRecorder()
	s.e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("account %s/%s status=%d body=%s", realm, sub, response.Code, response.Body.String())
	}
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	cookies := response.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("csrf cookie missing")
	}
	return body.CSRFToken, cookies[0]
}

// workflowView は管理 API が返すワークフローの表現のうち、具体例が観測する部分である。
type workflowView struct {
	ID              string                    `json:"id"`
	Name            string                    `json:"name"`
	Status          string                    `json:"status"`
	CurrentRevision int64                     `json:"current_revision"`
	EnabledRevision *int64                    `json:"enabled_revision"`
	Trigger         igdomain.WorkflowTrigger  `json:"trigger"`
	Actions         []igdomain.WorkflowAction `json:"actions"`
}

func decodeWorkflow(t *testing.T, response *httptest.ResponseRecorder) workflowView {
	t.Helper()
	var view workflowView
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode workflow %s: %v", response.Body.String(), err)
	}
	return view
}

func (s *governanceStack) createWorkflow(t *testing.T, realm, sub string, body map[string]any) workflowView {
	t.Helper()
	response := s.call(t, realm, sub, http.MethodPost, "/api/admin/v1/lifecycle-workflows", body)
	if response.Code != http.StatusCreated {
		t.Fatalf("create workflow status=%d body=%s", response.Code, response.Body.String())
	}
	return decodeWorkflow(t, response)
}

// assertProblem は応答が名指した状態と RFC 9457 の type を持つことを確かめる。
func assertProblem(t *testing.T, response *httptest.ResponseRecorder, status int, problemType string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status=%d body=%s, want %d", response.Code, response.Body.String(), status)
	}
	var problem support.Problem
	if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem %s: %v", response.Body.String(), err)
	}
	if problem.Type != "urn:idmagic:error:"+problemType {
		t.Fatalf("problem type=%q, want urn:idmagic:error:%s", problem.Type, problemType)
	}
}

func (s *governanceStack) workflowCount(t *testing.T, tenantID string) int {
	t.Helper()
	all, err := s.workflows.List(context.Background(), tenantID)
	if err != nil {
		t.Fatal(err)
	}
	return len(all)
}

//spec:covers EX-IDGOVERNANCE-001-01: 作成 API が、選んだトリガーと並べ替えた順序のアクションをリビジョン 1 の draft として返し、保存先にも同じ順序で残ることを固定する。
func TestCreateReturnsTheSelectedDefinitionAsRevisionOneDraft(t *testing.T) {
	s := newGovernanceStack(t)
	s.seedGroup(t, acmeTenant, "engineering", groupdomain.GroupMembershipManual)
	created := s.createWorkflow(t, acmeTenant, acmeAdmin, map[string]any{
		"name":    "Joiner",
		"trigger": map[string]any{"kind": "user_attributes_changed", "watched_attributes": []string{"department"}},
		"actions": []map[string]any{
			{"kind": "send_email", "template_key": "welcome"},
			{"kind": "add_group_member", "group_id": "engineering"},
		},
	})
	if created.Status != "draft" || created.CurrentRevision != 1 || created.EnabledRevision != nil {
		t.Fatalf("created = %+v, want revision 1 draft without enabled_revision", created)
	}
	if created.Trigger.Kind != igdomain.WorkflowTriggerUserAttributesChanged || len(created.Trigger.WatchedAttributes) != 1 {
		t.Fatalf("trigger = %+v, want the selected user_attributes_changed trigger", created.Trigger)
	}
	if len(created.Actions) != 2 || created.Actions[0].Kind != igdomain.WorkflowActionSendEmail || created.Actions[1].Kind != igdomain.WorkflowActionAddGroupMember {
		t.Fatalf("actions = %+v, want send_email then add_group_member", created.Actions)
	}
	stored, err := s.workflows.FindRevision(context.Background(), acmeTenant, created.ID, 1)
	if err != nil || stored == nil || stored.Actions[0].Kind != igdomain.WorkflowActionSendEmail {
		t.Fatalf("stored revision = %+v, %v", stored, err)
	}
}

//spec:covers EX-IDGOVERNANCE-001-02: 参照先のグループが無いアクションは、作成が InvalidRequestError で拒否され、ワークフローが 1 つも保存されないことを固定する。
func TestCreateRefusesAnActionWithoutAnExistingReference(t *testing.T) {
	s := newGovernanceStack(t)
	response := s.call(t, acmeTenant, acmeAdmin, http.MethodPost, "/api/admin/v1/lifecycle-workflows", map[string]any{
		"name":    "Joiner",
		"trigger": map[string]any{"kind": "user_created"},
		"actions": []map[string]any{{"kind": "add_group_member", "group_id": "no-such-group"}},
	})
	assertProblem(t, response, http.StatusBadRequest, "invalid_request")
	if count := s.workflowCount(t, acmeTenant); count != 0 {
		t.Fatalf("workflows = %d, want the refused create to leave none", count)
	}
}

//spec:covers EX-IDGOVERNANCE-002-01: 更新 API が current_revision を 1 つ増やし、変更した定義が一覧と詳細（編集フォームが読む API）の双方に現れることを固定する。
func TestUpdateRaisesTheRevisionAndIsReflectedInListAndDetail(t *testing.T) {
	s := newGovernanceStack(t)
	created := s.createWorkflow(t, acmeTenant, acmeAdmin, map[string]any{
		"name": "Leaver", "trigger": map[string]any{"kind": "user_created"},
		"actions": []map[string]any{{"kind": "disable_user"}},
	})
	updated := s.call(t, acmeTenant, acmeAdmin, http.MethodPut, "/api/admin/v1/lifecycle-workflows/"+created.ID, map[string]any{
		"expected_revision": 1, "name": "Leaver",
		"trigger": map[string]any{"kind": "user_created"},
		"actions": []map[string]any{{"kind": "disable_user"}, {"kind": "send_email", "template_key": "bye"}},
	})
	if updated.Code != http.StatusOK || decodeWorkflow(t, updated).CurrentRevision != 2 {
		t.Fatalf("update status=%d body=%s, want current_revision 2", updated.Code, updated.Body.String())
	}
	detail := decodeWorkflow(t, s.call(t, acmeTenant, acmeAdmin, http.MethodGet, "/api/admin/v1/lifecycle-workflows/"+created.ID, nil))
	if detail.CurrentRevision != 2 || len(detail.Actions) != 2 {
		t.Fatalf("detail = %+v, want the revision 2 definition", detail)
	}
	list := s.call(t, acmeTenant, acmeAdmin, http.MethodGet, "/api/admin/v1/lifecycle-workflows", nil)
	var listed struct {
		Workflows []workflowView `json:"workflows"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &listed); err != nil || len(listed.Workflows) != 1 || listed.Workflows[0].CurrentRevision != 2 || len(listed.Workflows[0].Actions) != 2 {
		t.Fatalf("list = %s, want the revision 2 definition", list.Body.String())
	}
}

//spec:covers EX-IDGOVERNANCE-006-01: add_group_member に動的グループを指定した保存は InvalidRequestError で拒否され、revision が増えないことを固定する。
func TestSavingAnAddGroupMemberActionOnADynamicGroupIsRefused(t *testing.T) {
	s := newGovernanceStack(t)
	s.seedGroup(t, acmeTenant, "all-engineers", groupdomain.GroupMembershipDynamic)
	created := s.createWorkflow(t, acmeTenant, acmeAdmin, map[string]any{
		"name": "Joiner", "trigger": map[string]any{"kind": "user_created"},
		"actions": []map[string]any{{"kind": "disable_user"}},
	})
	response := s.call(t, acmeTenant, acmeAdmin, http.MethodPut, "/api/admin/v1/lifecycle-workflows/"+created.ID, map[string]any{
		"expected_revision": 1, "name": "Joiner",
		"trigger": map[string]any{"kind": "user_created"},
		"actions": []map[string]any{{"kind": "add_group_member", "group_id": "all-engineers"}},
	})
	assertProblem(t, response, http.StatusBadRequest, "invalid_request")
	stored, err := s.workflows.Find(context.Background(), acmeTenant, created.ID)
	if err != nil || stored.CurrentRevision != 1 {
		t.Fatalf("stored = %+v, %v; want the refused save to leave revision 1", stored, err)
	}
}

//spec:covers EX-IDGOVERNANCE-007-01: 属性スキーマに無いフィルターのフィールドは保存で、別テナントのグループを参照する revision は有効化で、それぞれ InvalidRequestError になり、定義は draft のまま revision も増えないことを固定する。
func TestUnknownFieldsAndForeignGroupsCannotBeSavedOrEnabled(t *testing.T) {
	s := newGovernanceStack(t)
	s.seedGroup(t, tenancydomain.DefaultTenantID, "default-group", groupdomain.GroupMembershipManual)
	created := s.createWorkflow(t, acmeTenant, acmeAdmin, map[string]any{
		"name": "Joiner",
		"trigger": map[string]any{"kind": "user_created", "filters": []map[string]any{
			{"field": "badge_color", "operator": "eq", "value": "blue"},
		}},
		"actions": []map[string]any{{"kind": "disable_user"}},
	})

	unknown := s.call(t, acmeTenant, acmeAdmin, http.MethodPut, "/api/admin/v1/lifecycle-workflows/"+created.ID, map[string]any{
		"expected_revision": 1, "name": "Joiner",
		"trigger": map[string]any{"kind": "user_created", "filters": []map[string]any{
			{"field": "favorite_color", "operator": "eq", "value": "blue"},
		}},
		"actions": []map[string]any{{"kind": "disable_user"}},
	})
	assertProblem(t, unknown, http.StatusBadRequest, "invalid_request")

	// 保存の検証を経ずに入った revision でも、有効化の検証が拒否することを確かめるため、
	// 別テナントのグループを指す revision を保存先へ直接置く。
	now := time.Now().UTC()
	if err := s.workflows.SaveRevision(context.Background(), &igdomain.LifecycleWorkflowRevision{
		WorkflowID: created.ID, TenantID: acmeTenant, Revision: 2, CreatedAt: now,
		Trigger: igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserCreated},
		Actions: []igdomain.WorkflowAction{{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: "default-group"}},
	}); err != nil {
		t.Fatal(err)
	}
	stored, err := s.workflows.Find(context.Background(), acmeTenant, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored.CurrentRevision = 2
	if err := s.workflows.Save(context.Background(), stored); err != nil {
		t.Fatal(err)
	}
	enabled := s.call(t, acmeTenant, acmeAdmin, http.MethodPost, "/api/admin/v1/lifecycle-workflows/"+created.ID+"/enable", map[string]any{"expected_revision": 2})
	assertProblem(t, enabled, http.StatusBadRequest, "invalid_request")
	after, err := s.workflows.Find(context.Background(), acmeTenant, created.ID)
	if err != nil || after.Status != igdomain.LifecycleWorkflowDraft || after.EnabledRevision != nil || after.CurrentRevision != 2 {
		t.Fatalf("workflow after refusals = %+v, %v; want an unchanged draft", after, err)
	}
}

//spec:covers EX-IDGOVERNANCE-011-02: 無効化済みワークフローの WorkflowRun の再試行は InvalidRequestError で拒否され、WorkflowRun は失敗のまま残り、Job も作られないことを固定する。
func TestRetryingARunOfADisabledWorkflowStartsNothing(t *testing.T) {
	s := newGovernanceStack(t)
	created := s.createWorkflow(t, acmeTenant, acmeAdmin, map[string]any{
		"name": "Leaver", "trigger": map[string]any{"kind": "user_created"},
		"actions": []map[string]any{{"kind": "disable_user"}},
	})
	if response := s.call(t, acmeTenant, acmeAdmin, http.MethodPost, "/api/admin/v1/lifecycle-workflows/"+created.ID+"/enable", map[string]any{"expected_revision": 1}); response.Code != http.StatusOK {
		t.Fatalf("enable status=%d body=%s", response.Code, response.Body.String())
	}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: acmeTenant, WorkflowID: created.ID, Revision: 1, SourceOccurrenceID: "occurrence-1", TargetUserID: acmeAdmin, TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{{Kind: igdomain.WorkflowActionDisableUser}}, Status: igdomain.WorkflowRunQueued, TriggeredAt: time.Now().UTC()}
	if _, err := s.runs.SaveRun(context.Background(), run, []igdomain.WorkflowStep{{RunID: run.ID, Action: run.Actions[0], Outcome: igdomain.WorkflowStepPending}}); err != nil {
		t.Fatal(err)
	}
	if err := s.runs.CompleteRun(context.Background(), acmeTenant, run.ID, igdomain.WorkflowRunFailed, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if response := s.call(t, acmeTenant, acmeAdmin, http.MethodPost, "/api/admin/v1/lifecycle-workflows/"+created.ID+"/disable", map[string]any{"expected_revision": 1}); response.Code != http.StatusOK {
		t.Fatalf("disable status=%d body=%s", response.Code, response.Body.String())
	}

	retried := s.call(t, acmeTenant, acmeAdmin, http.MethodPost, "/api/admin/v1/lifecycle-workflow-runs/"+run.ID+"/retry", map[string]any{})
	assertProblem(t, retried, http.StatusBadRequest, "invalid_request")
	after, err := s.runs.FindRun(context.Background(), acmeTenant, run.ID)
	if err != nil || after.Status != igdomain.WorkflowRunFailed || after.JobID != nil {
		t.Fatalf("run after the refused retry = %+v, %v; want it failed without a job", after, err)
	}
	if queued, err := s.jobs.ListByTenantAndKinds(context.Background(), acmeTenant, []jobsdomain.JobKind{igusecases.LifecycleWorkflowRunJobKind}, 0); err != nil || len(queued) != 0 {
		t.Fatalf("jobs = %d, %v; want none", len(queued), err)
	}
}

//spec:covers EX-IDGOVERNANCE-012-01: 別テナントのワークフローは存在しないもの（InvalidRequestError）として扱われて名前も漏れず、別テナントの group_id を参照する保存は、同じ id のグループが自テナントに無い限り拒否され、別テナントのグループへフォールバックしないことを固定する。
func TestWorkflowsAndResourcesDoNotCrossTheTenantBoundary(t *testing.T) {
	s := newGovernanceStack(t)
	s.seedGroup(t, tenancydomain.DefaultTenantID, "default-group", groupdomain.GroupMembershipManual)
	foreign := s.createWorkflow(t, tenancydomain.DefaultRealm, defaultAdmin, map[string]any{
		"name": "Default only workflow", "trigger": map[string]any{"kind": "user_created"},
		"actions": []map[string]any{{"kind": "add_group_member", "group_id": "default-group"}},
	})

	read := s.call(t, acmeTenant, acmeAdmin, http.MethodGet, "/api/admin/v1/lifecycle-workflows/"+foreign.ID, nil)
	assertProblem(t, read, http.StatusBadRequest, "invalid_request")
	if strings.Contains(read.Body.String(), "Default only workflow") {
		t.Fatalf("body = %s, want no trace of the other tenant's workflow", read.Body.String())
	}

	saved := s.call(t, acmeTenant, acmeAdmin, http.MethodPost, "/api/admin/v1/lifecycle-workflows", map[string]any{
		"name": "Borrowing", "trigger": map[string]any{"kind": "user_created"},
		"actions": []map[string]any{{"kind": "add_group_member", "group_id": "default-group"}},
	})
	assertProblem(t, saved, http.StatusBadRequest, "invalid_request")
	if count := s.workflowCount(t, acmeTenant); count != 0 {
		t.Fatalf("acme workflows = %d, want none", count)
	}
}

//spec:covers EX-IDGOVERNANCE-014-01: admin ロールを持たない主体の作成、更新、有効化、無効化、削除、プレビュー、再試行はすべて AccessDeniedError になり、定義、WorkflowRun、Job のいずれも変わらないことを固定する。
func TestNonAdminCannotChangeWorkflowsOrStartRuns(t *testing.T) {
	s := newGovernanceStack(t)
	realm := tenancydomain.DefaultRealm
	tenantID := tenancydomain.DefaultTenantID
	target := s.createWorkflow(t, realm, defaultAdmin, map[string]any{
		"name": "Leaver", "trigger": map[string]any{"kind": "user_created"},
		"actions": []map[string]any{{"kind": "disable_user"}},
	})
	if response := s.call(t, realm, defaultAdmin, http.MethodPost, "/api/admin/v1/lifecycle-workflows/"+target.ID+"/enable", map[string]any{"expected_revision": 1}); response.Code != http.StatusOK {
		t.Fatalf("enable status=%d body=%s", response.Code, response.Body.String())
	}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: tenantID, WorkflowID: target.ID, Revision: 1, SourceOccurrenceID: "occurrence-1", TargetUserID: nonAdminAlice, TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{{Kind: igdomain.WorkflowActionDisableUser}}, Status: igdomain.WorkflowRunQueued, TriggeredAt: time.Now().UTC()}
	if _, err := s.runs.SaveRun(context.Background(), run, []igdomain.WorkflowStep{{RunID: run.ID, Action: run.Actions[0], Outcome: igdomain.WorkflowStepPending}}); err != nil {
		t.Fatal(err)
	}
	if err := s.runs.CompleteRun(context.Background(), tenantID, run.ID, igdomain.WorkflowRunFailed, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	definition := map[string]any{
		"expected_revision": 1, "name": "Changed",
		"trigger": map[string]any{"kind": "user_created"},
		"actions": []map[string]any{{"kind": "enable_user"}},
	}
	for _, request := range []struct {
		name, method, path string
		body               any
	}{
		{"create", http.MethodPost, "/api/admin/v1/lifecycle-workflows", definition},
		{"update", http.MethodPut, "/api/admin/v1/lifecycle-workflows/" + target.ID, definition},
		{"enable", http.MethodPost, "/api/admin/v1/lifecycle-workflows/" + target.ID + "/enable", map[string]any{"expected_revision": 1}},
		{"disable", http.MethodPost, "/api/admin/v1/lifecycle-workflows/" + target.ID + "/disable", map[string]any{"expected_revision": 1}},
		{"delete", http.MethodDelete, "/api/admin/v1/lifecycle-workflows/" + target.ID, map[string]any{"expected_revision": 1}},
		{"dry-run", http.MethodPost, "/api/admin/v1/lifecycle-workflows/" + target.ID + "/dry-run", map[string]any{"target_user_id": nonAdminAlice}},
		{"retry", http.MethodPost, "/api/admin/v1/lifecycle-workflow-runs/" + run.ID + "/retry", map[string]any{}},
	} {
		t.Run(request.name, func(t *testing.T) {
			assertProblem(t, s.call(t, realm, nonAdminAlice, request.method, request.path, request.body), http.StatusForbidden, "access_denied")
		})
	}

	all, err := s.workflows.List(context.Background(), tenantID)
	if err != nil || len(all) != 1 || all[0].Name != "Leaver" || all[0].Status != igdomain.LifecycleWorkflowEnabled || all[0].CurrentRevision != 1 {
		t.Fatalf("workflows = %+v, %v; want only the untouched enabled Leaver", all, err)
	}
	after, err := s.runs.FindRun(context.Background(), tenantID, run.ID)
	if err != nil || after.Status != igdomain.WorkflowRunFailed || after.JobID != nil {
		t.Fatalf("run = %+v, %v; want it failed without a job", after, err)
	}
	if queued, err := s.jobs.ListByTenantAndKinds(context.Background(), tenantID, []jobsdomain.JobKind{igusecases.LifecycleWorkflowRunJobKind}, 0); err != nil || len(queued) != 0 {
		t.Fatalf("jobs = %d, %v; want none", len(queued), err)
	}
}

//spec:covers EX-IDGOVERNANCE-014-02: admin ロールを持たない主体の一覧と詳細の要求は AccessDeniedError になり、応答にワークフローの名前が含まれないことを固定する。
func TestNonAdminCannotReadWorkflows(t *testing.T) {
	s := newGovernanceStack(t)
	target := s.createWorkflow(t, tenancydomain.DefaultRealm, defaultAdmin, map[string]any{
		"name": "Secret leaver", "trigger": map[string]any{"kind": "user_created"},
		"actions": []map[string]any{{"kind": "disable_user"}},
	})
	for _, path := range []string{"/api/admin/v1/lifecycle-workflows", "/api/admin/v1/lifecycle-workflows/" + target.ID} {
		response := s.call(t, tenancydomain.DefaultRealm, nonAdminAlice, http.MethodGet, path, nil)
		assertProblem(t, response, http.StatusForbidden, "access_denied")
		if strings.Contains(response.Body.String(), "Secret leaver") {
			t.Fatalf("%s body = %s, want no workflow in the refusal", path, response.Body.String())
		}
	}
}
