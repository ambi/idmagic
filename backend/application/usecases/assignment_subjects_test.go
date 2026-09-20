package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/application/ports"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type subjectDirectoryFake struct {
	existing map[ports.SubjectRef]bool
	err      error
}

type existingSubjectDirectoryFake struct{}

func (existingSubjectDirectoryFake) SubjectExists(
	context.Context,
	string,
	domain.AssignmentSubjectType,
	string,
) (bool, error) {
	return true, nil
}

func (f subjectDirectoryFake) SubjectExists(
	_ context.Context,
	_ string,
	subjectType domain.AssignmentSubjectType,
	subjectID string,
) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.existing[ports.SubjectRef{Type: subjectType, ID: subjectID}], nil
}

//spec:covers REQ-APPLICATION-007: 実在する User と Group だけを保存し、主体の不在または照合失敗では割当も ApplicationAssigned も残さない。
func TestAssignApplicationAcceptsOnlyExistingTenantSubjects(t *testing.T) {
	lookupFailure := errors.New("subject lookup failed")
	tests := []struct {
		name        string
		subjectType domain.AssignmentSubjectType
		subjectID   string
		directory   *subjectDirectoryFake
		wantErr     error
		wantSaved   bool
	}{
		{
			name: "existing user", subjectType: domain.AssignmentSubjectUser, subjectID: "user-1",
			directory: &subjectDirectoryFake{existing: map[ports.SubjectRef]bool{{Type: domain.AssignmentSubjectUser, ID: "user-1"}: true}},
			wantSaved: true,
		},
		{
			name: "existing group", subjectType: domain.AssignmentSubjectGroup, subjectID: "group-1",
			directory: &subjectDirectoryFake{existing: map[ports.SubjectRef]bool{{Type: domain.AssignmentSubjectGroup, ID: "group-1"}: true}},
			wantSaved: true,
		},
		{name: "missing user", subjectType: domain.AssignmentSubjectUser, subjectID: "missing-user", directory: &subjectDirectoryFake{}, wantErr: appusecases.ErrSubjectNotFound},
		{name: "missing group", subjectType: domain.AssignmentSubjectGroup, subjectID: "missing-group", directory: &subjectDirectoryFake{}, wantErr: appusecases.ErrSubjectNotFound},
		{name: "directory not configured", subjectType: domain.AssignmentSubjectUser, subjectID: "user-1", wantErr: appusecases.ErrSubjectNotFound},
		{name: "lookup failure", subjectType: domain.AssignmentSubjectUser, subjectID: "user-1", directory: &subjectDirectoryFake{err: lookupFailure}, wantErr: lookupFailure},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := tenantContext()
			applications := appmemory.NewApplicationRepository()
			assignments := appmemory.NewApplicationAssignmentRepository()
			if err := applications.Save(ctx, &domain.Application{
				TenantID: "acme", ID: "app-1", Name: "App 1",
				Kind: domain.ApplicationWeblink, Status: domain.ApplicationActive,
			}); err != nil {
				t.Fatalf("seed application: %v", err)
			}
			var events []string
			deps := appusecases.AssignmentDeps{
				Repo: applications, AssignmentRepo: assignments,
				Emit: func(event spec.DomainEvent) { events = append(events, event.EventType()) },
			}
			if test.directory != nil {
				deps.SubjectDirectory = test.directory
			}

			_, err := appusecases.AssignApplication(ctx, deps, appusecases.AssignApplicationInput{
				ActorUserID: "admin", ApplicationID: "app-1",
				SubjectType: test.subjectType, SubjectID: test.subjectID,
				Now: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
			})

			if !errors.Is(err, test.wantErr) {
				t.Fatalf("AssignApplication() error=%v, want %v", err, test.wantErr)
			}
			saved, listErr := assignments.ListByApplication(ctx, "acme", "app-1")
			if listErr != nil {
				t.Fatalf("read assignments: %v", listErr)
			}
			if test.wantSaved {
				if len(saved) != 1 || saved[0].SubjectID != test.subjectID {
					t.Fatalf("assignments=%+v, want saved subject %q", saved, test.subjectID)
				}
				if len(events) != 1 || events[0] != "ApplicationAssigned" {
					t.Fatalf("events=%v, want ApplicationAssigned", events)
				}
				return
			}
			if len(saved) != 0 {
				t.Fatalf("refused assignment was saved: %+v", saved)
			}
			if len(events) != 0 {
				t.Fatalf("refused assignment emitted events: %v", events)
			}
		})
	}
}
