package server_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

//spec:covers REQ-APPLICATION-007, EX-APPLICATION-007-02: 同じテナントに実在する User と Group だけを割り当て、存在しない主体と別テナントの主体は割当もイベントも残さず拒否する。
func TestApplicationAssignmentAcceptsOnlyExistingTenantSubjects(t *testing.T) {
	tests := []struct {
		name        string
		subjectType appdomain.AssignmentSubjectType
		subjectID   string
		seed        func(*testing.T, *stack.Stack)
		wantStatus  int
	}{
		{name: "existing user", subjectType: appdomain.AssignmentSubjectUser, subjectID: stack.UserID, wantStatus: http.StatusCreated},
		{
			name: "existing group", subjectType: appdomain.AssignmentSubjectGroup, subjectID: "group-1",
			seed: func(t *testing.T, s *stack.Stack) {
				t.Helper()
				if err := s.Groups.Save(t.Context(), &groupdomain.Group{ID: "group-1", TenantID: s.TenantID(t, defaultRealm), Name: "Group 1"}); err != nil {
					t.Fatalf("seed group: %v", err)
				}
			},
			wantStatus: http.StatusCreated,
		},
		{name: "missing user", subjectType: appdomain.AssignmentSubjectUser, subjectID: "missing-user", wantStatus: http.StatusBadRequest},
		{name: "missing group", subjectType: appdomain.AssignmentSubjectGroup, subjectID: "missing-group", wantStatus: http.StatusBadRequest},
		{
			name: "foreign user", subjectType: appdomain.AssignmentSubjectUser, subjectID: "foreign-user",
			seed: func(_ *testing.T, s *stack.Stack) {
				s.Users.Seed(&userdomain.User{
					ID: "foreign-user", TenantID: stack.OtherRealm, PreferredUsername: "foreign-user",
					Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
				})
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "foreign group", subjectType: appdomain.AssignmentSubjectGroup, subjectID: "foreign-group",
			seed: func(t *testing.T, s *stack.Stack) {
				t.Helper()
				if err := s.Groups.Save(t.Context(), &groupdomain.Group{ID: "foreign-group", TenantID: stack.OtherRealm, Name: "Foreign Group"}); err != nil {
					t.Fatalf("seed foreign group: %v", err)
				}
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := stack.New(t, stack.WithApiTokens(), stack.WithApplicationApi())
			tenantID := s.TenantID(t, defaultRealm)
			const applicationID = "assignment-app"
			now := time.Now().UTC()
			if err := s.Applications.Save(context.Background(), &appdomain.Application{
				TenantID: tenantID, ID: applicationID, Name: "Assignment App",
				Kind: appdomain.ApplicationWeblink, Status: appdomain.ApplicationActive,
				LaunchURL: "https://assignment.example", CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				t.Fatalf("seed application: %v", err)
			}
			if test.seed != nil {
				test.seed(t, s)
			}
			token, _ := s.IssueApiToken(t, defaultRealm, apitokendomain.ScopeApplicationsWrite)

			response := applicationApiTokenRequest(t, s, http.MethodPost,
				"/realms/default/api/admin/v1/applications/"+applicationID+"/assignments", token,
				map[string]any{"subject_type": test.subjectType, "subject_id": test.subjectID})

			if response.Code != test.wantStatus {
				t.Fatalf("status=%d body=%s, want %d", response.Code, response.Body.String(), test.wantStatus)
			}
			assignments, err := s.ApplicationAssignments.ListByApplication(context.Background(), tenantID, applicationID)
			if err != nil {
				t.Fatalf("read assignments: %v", err)
			}
			if test.wantStatus == http.StatusCreated {
				if len(assignments) != 1 || assignments[0].SubjectID != test.subjectID {
					t.Fatalf("assignments=%+v, want saved subject %q", assignments, test.subjectID)
				}
				s.Events.AssertEmitted(t, "ApplicationAssigned")
				return
			}
			var problem support.Problem
			if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
				t.Fatalf("decode problem: %v; body=%s", err, response.Body.String())
			}
			if problem.Type != "urn:idmagic:error:invalid_request" {
				t.Fatalf("problem type=%q, want invalid_request", problem.Type)
			}
			if len(assignments) != 0 {
				t.Fatalf("refused assignment was saved: %+v", assignments)
			}
			s.Events.AssertNotEmitted(t, "ApplicationAssigned")
		})
	}
}
