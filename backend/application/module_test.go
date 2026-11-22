package application

import (
	"context"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/idmanagement"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

//spec:covers REQ-APPLICATION-007: User は取得後の tenant_id を照合し、Group は tenant-scoped lookup を使って同一テナントの主体だけを公開する。
func TestIDManagementSubjectDirectoryIsTenantScoped(t *testing.T) {
	users := idmanagement.NewMemoryModule().UserRepo
	_ = users.Save(context.Background(), &userdomain.User{ID: "tenant-user", TenantID: "tenant-1", PreferredUsername: "tenant-user"})
	_ = users.Save(context.Background(), &userdomain.User{ID: "foreign-user", TenantID: "tenant-2", PreferredUsername: "foreign-user"})
	groups := idmanagement.NewMemoryModule().GroupRepo
	for _, group := range []*groupdomain.Group{
		{ID: "tenant-group", TenantID: "tenant-1", Name: "Tenant Group"},
		{ID: "foreign-group", TenantID: "tenant-2", Name: "Foreign Group"},
	} {
		if err := groups.Save(t.Context(), group); err != nil {
			t.Fatalf("seed group: %v", err)
		}
	}
	directory := idManagementSubjectDirectory{users: users, groups: groups}

	tests := []struct {
		name        string
		subjectType appdomain.AssignmentSubjectType
		subjectID   string
		want        bool
	}{
		{name: "tenant user", subjectType: appdomain.AssignmentSubjectUser, subjectID: "tenant-user", want: true},
		{name: "foreign user", subjectType: appdomain.AssignmentSubjectUser, subjectID: "foreign-user"},
		{name: "missing user", subjectType: appdomain.AssignmentSubjectUser, subjectID: "missing-user"},
		{name: "tenant group", subjectType: appdomain.AssignmentSubjectGroup, subjectID: "tenant-group", want: true},
		{name: "foreign group", subjectType: appdomain.AssignmentSubjectGroup, subjectID: "foreign-group"},
		{name: "missing group", subjectType: appdomain.AssignmentSubjectGroup, subjectID: "missing-group"},
		{name: "invalid type", subjectType: appdomain.AssignmentSubjectType("invalid"), subjectID: "tenant-user"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := directory.SubjectExists(t.Context(), "tenant-1", test.subjectType, test.subjectID)
			if err != nil {
				t.Fatalf("SubjectExists() error: %v", err)
			}
			if got != test.want {
				t.Fatalf("SubjectExists()=%t, want %t", got, test.want)
			}
		})
	}
}

type failingUserRepository struct {
	userports.UserRepository
	err error
}

func (r failingUserRepository) FindBySub(context.Context, string) (*userdomain.User, error) {
	return nil, r.err
}

type failingGroupRepository struct {
	groupports.GroupRepository
	err error
}

func (r failingGroupRepository) FindByID(context.Context, string, string) (*groupdomain.Group, error) {
	return nil, r.err
}

func TestIDManagementSubjectDirectoryPropagatesLookupFailures(t *testing.T) {
	lookupFailure := errors.New("lookup failed")
	tests := []struct {
		name        string
		directory   idManagementSubjectDirectory
		subjectType appdomain.AssignmentSubjectType
	}{
		{
			name: "user", subjectType: appdomain.AssignmentSubjectUser,
			directory: idManagementSubjectDirectory{users: failingUserRepository{err: lookupFailure}},
		},
		{
			name: "group", subjectType: appdomain.AssignmentSubjectGroup,
			directory: idManagementSubjectDirectory{groups: failingGroupRepository{err: lookupFailure}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.directory.SubjectExists(t.Context(), "tenant-1", test.subjectType, "subject-1")
			if !errors.Is(err, lookupFailure) {
				t.Fatalf("SubjectExists() error=%v, want %v", err, lookupFailure)
			}
		})
	}
}
