package server_http_test

// Application の API アクセストークンが、スコープごとに何を通し何を拒むかを HTTP 境界で観測する。
//
// ここに置く判断基準は「use case を直接呼ぶテストでは通らないか」である。
// 粒度スコープの判定はルーティングと認証ミドルウェアに宿っており、use case からは見えない。
//
// 越境したテナントへの提示は application_api_token_tenant_test.go が持つ。

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

// defaultRealm は testing_stack が常に建てる既定テナントの realm である。
const defaultRealm = "default"

// assignedApplicationID と unassignedApplicationID は、seedAssignedApplication が置く 2 つの
// Application である。「自分のものだけが返る」は、返らないものが存在しなければ観測できない。
const (
	assignedApplicationID   = "mine"
	unassignedApplicationID = "not-mine"
)

// seedAssignedApplication は、トークンが固定する利用者へ割り当て済みの Application を 1 つと、
// 同じテナントに割り当てのない Application を 1 つ置く。
func seedAssignedApplication(t *testing.T, s *stack.Stack) {
	t.Helper()
	ctx := s.RealmContext(t, defaultRealm)
	now := time.Now().UTC()
	tenantID := s.TenantID(t, defaultRealm)
	for _, id := range []string{assignedApplicationID, unassignedApplicationID} {
		if err := s.Applications.Save(ctx, &appdomain.Application{
			TenantID: tenantID, ID: id, Name: id,
			Kind: appdomain.ApplicationWeblink, Status: appdomain.ApplicationActive,
			LaunchURL: "https://" + id + ".example", CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("seed application %s: %v", id, err)
		}
	}
	if err := s.ApplicationAssignments.Save(ctx, &appdomain.ApplicationAssignment{
		TenantID: tenantID, ApplicationID: assignedApplicationID,
		SubjectType: appdomain.AssignmentSubjectUser, SubjectID: stack.AdminUserID,
		Visibility: appdomain.AssignmentVisible, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
}

func applicationProblemCode(t *testing.T, body []byte) string {
	t.Helper()
	var problem support.Problem
	if err := json.Unmarshal(body, &problem); err != nil {
		t.Fatalf("decode problem: %v; body=%s", err, body)
	}
	return problem.Type
}

// アプリケーションと保存済みの順序だけを返し、account:write は自分の順序を保存する。
//
//spec:covers REQ-APPLICATION-003, EX-APPLICATION-003-01: account:read は自分に割り当てられた
func TestAccountScopesReadAndWriteOnlyTheCallersOwnApplications(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens(), stack.WithApplicationApi())
	seedAssignedApplication(t, s)
	ctx := s.RealmContext(t, defaultRealm)
	if err := s.ApplicationOrderings.Save(ctx, &appdomain.ApplicationOrdering{
		UserID: stack.AdminUserID, ApplicationIDs: []string{"mine"},
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed ordering: %v", err)
	}

	readOnly, _ := s.IssueApiToken(t, defaultRealm, apitokendomain.ScopeAccountRead)

	listed := applicationApiTokenRequest(t, s, http.MethodGet,
		"/realms/default/api/account/v1/applications", readOnly, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s, want 200", listed.Code, listed.Body.String())
	}
	var listBody struct {
		Applications []struct {
			ID string `json:"id"`
		} `json:"applications"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode list: %v; body=%s", err, listed.Body.String())
	}
	if len(listBody.Applications) != 1 || listBody.Applications[0].ID != "mine" {
		t.Fatalf("list returned %+v, want only the assigned application", listBody.Applications)
	}

	order := applicationApiTokenRequest(t, s, http.MethodGet,
		"/realms/default/api/account/v1/applications/order", readOnly, nil)
	if order.Code != http.StatusOK {
		t.Fatalf("order status=%d body=%s, want 200", order.Code, order.Body.String())
	}
	var orderBody struct {
		ApplicationIDs []string `json:"application_ids"`
	}
	if err := json.Unmarshal(order.Body.Bytes(), &orderBody); err != nil {
		t.Fatalf("decode order: %v; body=%s", err, order.Body.String())
	}
	if len(orderBody.ApplicationIDs) != 1 || orderBody.ApplicationIDs[0] != "mine" {
		t.Fatalf("order returned %v, want the saved order", orderBody.ApplicationIDs)
	}

	writable, _ := s.IssueApiToken(t, defaultRealm, apitokendomain.ScopeAccountWrite)
	saved := applicationApiTokenRequest(t, s, http.MethodPut,
		"/realms/default/api/account/v1/applications/order", writable,
		map[string]any{"application_ids": []string{"mine"}})
	if saved.Code != http.StatusOK && saved.Code != http.StatusNoContent {
		t.Fatalf("save order status=%d body=%s", saved.Code, saved.Body.String())
	}
	// 応答は use case の戻り値から組み立てられる。保存されたかどうかは読み直して見る。
	stored, err := s.ApplicationOrderings.Get(context.Background(), s.TenantID(t, defaultRealm), stack.AdminUserID)
	if err != nil {
		t.Fatalf("read ordering: %v", err)
	}
	if stored == nil || len(stored.ApplicationIDs) != 1 || stored.ApplicationIDs[0] != "mine" {
		t.Fatalf("stored ordering %+v, want the caller's own order", stored)
	}
}

// 順序の保存は 403 で拒否され、保存済みの順序は書き換わらない。
//
//spec:covers REQ-APPLICATION-003, EX-APPLICATION-003-03: account:read だけのトークンによる
func TestAccountReadScopeAloneCannotSaveTheApplicationOrder(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens(), stack.WithApplicationApi())
	seedAssignedApplication(t, s)
	ctx := s.RealmContext(t, defaultRealm)
	if err := s.ApplicationOrderings.Save(ctx, &appdomain.ApplicationOrdering{
		UserID: stack.AdminUserID, ApplicationIDs: []string{"mine"},
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed ordering: %v", err)
	}

	readOnly, _ := s.IssueApiToken(t, defaultRealm, apitokendomain.ScopeAccountRead)
	refused := applicationApiTokenRequest(t, s, http.MethodPut,
		"/realms/default/api/account/v1/applications/order", readOnly,
		map[string]any{"application_ids": []string{"not-mine"}})

	if refused.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", refused.Code, refused.Body.String())
	}
	if got := applicationProblemCode(t, refused.Body.Bytes()); got != "urn:idmagic:error:insufficient_scope" {
		t.Fatalf("problem type=%q, want the insufficient_scope refusal", got)
	}
	stored, err := s.ApplicationOrderings.Get(context.Background(), s.TenantID(t, defaultRealm), stack.AdminUserID)
	if err != nil {
		t.Fatalf("read ordering: %v", err)
	}
	if stored == nil || len(stored.ApplicationIDs) != 1 || stored.ApplicationIDs[0] != "mine" {
		t.Fatalf("ordering changed after refusal: %+v", stored)
	}
}

// applications:write は作成と削除を、settings:read と settings:write はテナントの
// デフォルトサインインポリシーの対応種別だけを通す。
//
//spec:covers REQ-APPLICATION-004, EX-APPLICATION-004-01: applications:read は参照だけを、
func TestApplicationAdminScopesAllowTheOperationsTheyName(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens(), stack.WithApplicationApi())
	seedAssignedApplication(t, s)

	read, _ := s.IssueApiToken(t, defaultRealm, apitokendomain.ScopeApplicationsRead)
	for _, path := range []string{
		"/realms/default/api/admin/v1/applications",
		"/realms/default/api/admin/v1/application-categories",
		"/realms/default/api/admin/v1/applications/mine/assignments",
	} {
		response := applicationApiTokenRequest(t, s, http.MethodGet, path, read, nil)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s, want 200", path, response.Code, response.Body.String())
		}
	}

	write, _ := s.IssueApiToken(t, defaultRealm, apitokendomain.ScopeApplicationsWrite)
	created := applicationApiTokenRequest(t, s, http.MethodPost,
		"/realms/default/api/admin/v1/applications", write,
		map[string]any{"name": "Created", "type": "weblink", "launch_url": "https://created.example"})
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s, want 201", created.Code, created.Body.String())
	}
	var createdBody struct {
		Application struct {
			ID string `json:"id"`
		} `json:"application"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &createdBody); err != nil {
		t.Fatalf("decode create: %v; body=%s", err, created.Body.String())
	}
	// 応答が 201 でも保存されたとは限らない。作成した id を保存先から読み直す。
	saved, err := s.Applications.FindByID(context.Background(), s.TenantID(t, defaultRealm), createdBody.Application.ID)
	if err != nil || saved == nil {
		t.Fatalf("created application not stored: %v", err)
	}

	deleted := applicationApiTokenRequest(t, s, http.MethodDelete,
		"/realms/default/api/admin/v1/applications/"+createdBody.Application.ID, write, nil)
	if deleted.Code != http.StatusNoContent && deleted.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	gone, err := s.Applications.FindByID(context.Background(), s.TenantID(t, defaultRealm), createdBody.Application.ID)
	if err != nil {
		t.Fatalf("read after delete: %v", err)
	}
	if gone != nil {
		t.Fatalf("application still stored after delete: %+v", gone)
	}

	settingsRead, _ := s.IssueApiToken(t, defaultRealm, apitokendomain.ScopeSettingsRead)
	policy := applicationApiTokenRequest(t, s, http.MethodGet,
		"/realms/default/api/admin/v1/default-sign-in-policy", settingsRead, nil)
	if policy.Code != http.StatusOK {
		t.Fatalf("read default policy status=%d body=%s, want 200", policy.Code, policy.Body.String())
	}

	settingsWrite, _ := s.IssueApiToken(t, defaultRealm, apitokendomain.ScopeSettingsWrite)
	updated := applicationApiTokenRequest(t, s, http.MethodPut,
		"/realms/default/api/admin/v1/default-sign-in-policy", settingsWrite,
		map[string]any{"rules": []map[string]any{
			{"name": "MFA", "enabled": true, "required_authn": map[string]any{"strength": "Mfa"}},
		}})
	if updated.Code != http.StatusOK && updated.Code != http.StatusNoContent {
		t.Fatalf("update default policy status=%d body=%s", updated.Code, updated.Body.String())
	}
	storedPolicy, err := s.DefaultSignInPolicy.Get(context.Background(), s.TenantID(t, defaultRealm))
	if err != nil || storedPolicy == nil || len(storedPolicy.Rules) != 1 {
		t.Fatalf("default policy not stored: policy=%+v err=%v", storedPolicy, err)
	}
}

// Application の作成は 403 で拒否され、Application は 1 つも増えない。
//
//spec:covers REQ-APPLICATION-004, EX-APPLICATION-004-02: applications:read だけのトークンによる
func TestApplicationsReadScopeAloneCannotChangeAnApplication(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens(), stack.WithApplicationApi())
	seedAssignedApplication(t, s)

	read, _ := s.IssueApiToken(t, defaultRealm, apitokendomain.ScopeApplicationsRead)
	refused := applicationApiTokenRequest(t, s, http.MethodPost,
		"/realms/default/api/admin/v1/applications", read,
		map[string]any{"name": "Refused", "type": "weblink", "launch_url": "https://refused.example"})

	if refused.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", refused.Code, refused.Body.String())
	}
	if got := applicationProblemCode(t, refused.Body.Bytes()); got != "urn:idmagic:error:insufficient_scope" {
		t.Fatalf("problem type=%q, want the insufficient_scope refusal", got)
	}
	applications, err := s.Applications.ListAll(context.Background(), s.TenantID(t, defaultRealm))
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	if len(applications) != 2 {
		t.Fatalf("application count=%d after refusal, want the 2 seeded ones", len(applications))
	}
}
