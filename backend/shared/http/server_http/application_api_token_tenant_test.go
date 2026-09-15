package server_http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

func applicationApiTokenRequest(t *testing.T, s *stack.Stack, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	request := httptest.NewRequest(method, stack.Issuer+path, &payload)
	request.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, request)
	return recorder
}

func assertApplicationTokenProblem(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus int, wantType string) {
	t.Helper()
	if recorder.Code != wantStatus {
		t.Fatalf("status=%d, want %d; body=%s", recorder.Code, wantStatus, recorder.Body.String())
	}
	var problem support.Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v; body=%s", err, recorder.Body.String())
	}
	if problem.Type != "urn:idmagic:error:"+wantType {
		t.Fatalf("problem type=%q, want %q", problem.Type, "urn:idmagic:error:"+wantType)
	}
	if challenge := recorder.Header().Get("WWW-Authenticate"); !bytes.Contains([]byte(challenge), []byte(`Bearer error="invalid_token"`)) {
		t.Fatalf("WWW-Authenticate=%q, want invalid_token challenge", challenge)
	}
}

func seedApplicationOrder(t *testing.T, s *stack.Stack, applicationIDs ...string) {
	t.Helper()
	ctx := s.RealmContext(t, stack.OtherRealm)
	now := time.Now().UTC()
	for _, applicationID := range applicationIDs {
		if err := s.Applications.Save(ctx, &appdomain.Application{
			TenantID: stack.OtherRealm, ID: applicationID, Name: applicationID,
			Kind: appdomain.ApplicationWeblink, Status: appdomain.ApplicationActive,
			LaunchURL: "https://" + applicationID + ".example", CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("seed application: %v", err)
		}
		if err := s.ApplicationAssignments.Save(ctx, &appdomain.ApplicationAssignment{
			TenantID: stack.OtherRealm, ApplicationID: applicationID,
			SubjectType: appdomain.AssignmentSubjectUser, SubjectID: stack.AdminUserID,
			Visibility: appdomain.AssignmentVisible, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("seed assignment: %v", err)
		}
	}
	if err := s.ApplicationOrderings.Save(ctx, &appdomain.ApplicationOrdering{
		UserID: stack.AdminUserID, ApplicationIDs: applicationIDs, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed ordering: %v", err)
	}
}

//spec:covers REQ-APPLICATION-003, EX-APPLICATION-003-02: 別テナント向けトークンを 401 invalid_token で拒否し、保存済みの並び順を変えない。
func TestForeignTenantApiTokenCannotReorderApplications(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens(), stack.WithApplicationApi())
	seedApplicationOrder(t, s, "kept-app", "foreign-app")
	token, _ := s.IssueApiToken(t, "default", apitokendomain.ScopeAccountWrite)

	recorder := applicationApiTokenRequest(t, s, http.MethodPut,
		"/realms/"+stack.OtherRealm+"/api/account/v1/applications/order", token,
		map[string]any{"application_ids": []string{"foreign-app"}})

	assertApplicationTokenProblem(t, recorder, http.StatusUnauthorized, "invalid_token")
	ordering, err := s.ApplicationOrderings.Get(context.Background(), stack.OtherRealm, stack.AdminUserID)
	if err != nil {
		t.Fatalf("read ordering: %v", err)
	}
	if ordering == nil || len(ordering.ApplicationIDs) != 2 || ordering.ApplicationIDs[0] != "kept-app" || ordering.ApplicationIDs[1] != "foreign-app" {
		t.Fatalf("ordering changed after refusal: %+v", ordering)
	}
	applications, err := s.Applications.ListAll(context.Background(), stack.OtherRealm)
	if err != nil || len(applications) != 2 {
		t.Fatalf("applications changed after refusal: count=%d err=%v", len(applications), err)
	}
	assignments, err := s.ApplicationAssignments.ListAll(context.Background(), stack.OtherRealm)
	if err != nil || len(assignments) != 2 {
		t.Fatalf("assignments changed after refusal: count=%d err=%v", len(assignments), err)
	}
}

//spec:covers REQ-APPLICATION-004, EX-APPLICATION-004-03: 別テナント向けトークンを 401 invalid_token で拒否し、Application を作成しない。
func TestForeignTenantApiTokenCannotCreateApplication(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens(), stack.WithApplicationApi())
	token, _ := s.IssueApiToken(t, "default", apitokendomain.ScopeApplicationsWrite)

	recorder := applicationApiTokenRequest(t, s, http.MethodPost,
		"/realms/"+stack.OtherRealm+"/api/admin/v1/applications", token,
		map[string]any{"name": "Foreign", "type": "weblink", "launch_url": "https://foreign.example"})

	assertApplicationTokenProblem(t, recorder, http.StatusUnauthorized, "invalid_token")
	applications, err := s.Applications.ListAll(context.Background(), stack.OtherRealm)
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	if len(applications) != 0 {
		t.Fatalf("applications created after refusal: %+v", applications)
	}
	assignments, err := s.ApplicationAssignments.ListAll(context.Background(), stack.OtherRealm)
	if err != nil {
		t.Fatalf("list assignments: %v", err)
	}
	if len(assignments) != 0 {
		t.Fatalf("assignments created after refusal: %+v", assignments)
	}
}
