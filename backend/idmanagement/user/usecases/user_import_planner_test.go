package usecases

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type importSchemaReader struct {
	defs []userdomain.UserAttributeDef
	err  error
}

func (r importSchemaReader) EffectiveUserAttributeDefs(context.Context, string) ([]userdomain.UserAttributeDef, error) {
	return r.defs, r.err
}

type perUserImportOwnershipGuard struct {
	managed map[string]bool
	err     error
}

func (g perUserImportOwnershipGuard) SourceManagedUserIDs(_ context.Context, _ string, userIDs []string) (map[string]bool, error) {
	if g.err != nil {
		return nil, g.err
	}
	out := make(map[string]bool, len(userIDs))
	for _, userID := range userIDs {
		out[userID] = g.managed[userID]
	}
	return out, nil
}

func importPlannerContext() context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "acme"}, "", "")
}

func importPlannerUser(id, username string) *userdomain.User {
	now := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	email := username + "@example.com"
	department := "Old"
	return &userdomain.User{
		ID: id, TenantID: "acme", PreferredUsername: username, PasswordHash: "hash",
		Email: &email, Roles: []string{"support"},
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		Attributes: map[string]userdomain.AttributeValue{
			"department": {Type: idmdomain.AttributeTypeString, String: &department},
		},
		CreatedAt: now, UpdatedAt: now,
	}
}

func importPlannerDeps(repo *usermemory.UserRepository, guard perUserImportOwnershipGuard) UserImportPlanDeps {
	return UserImportPlanDeps{
		UserRepo: repo,
		SchemaReader: importSchemaReader{defs: []userdomain.UserAttributeDef{{
			Key: "department", Type: idmdomain.AttributeTypeString,
			Visibility: idmdomain.AttrVisibilityPrivate,
		}}},
		OwnershipGuard: guard,
	}
}

func planUserImportForTest(ctx context.Context, deps UserImportPlanDeps, input string) (userdomain.UserImportPlan, error) {
	plan := userdomain.UserImportPlan{}
	_, err := PlanUserImport(ctx, deps, strings.NewReader(input), idmdomain.DefaultCSVTransferPolicy(), func(row userdomain.UserImportRowPlan) error {
		plan.Rows = append(plan.Rows, row)
		return nil
	})
	return plan, err
}

// scenario: 管理者はエクスポートしたユーザー CSV を安全に再適用できる
func TestPlanUserImportCreateUpdateUnchangedAndFieldPresence(t *testing.T) {
	repo := usermemory.NewUserRepository()
	alice := importPlannerUser("user-alice", "alice")
	charlie := importPlannerUser("user-charlie", "charlie")
	repo.Seed(alice)
	repo.Seed(charlie)

	csv := "id,preferred_username,email,required_actions,attr:department\n" +
		"user-alice,alice,,update_password,Engineering\n" +
		",bob,bob@example.com,,Sales\n" +
		"user-charlie,charlie,charlie@example.com,,Old\n"
	plan, err := planUserImportForTest(importPlannerContext(), importPlannerDeps(repo, perUserImportOwnershipGuard{}), csv)
	if err != nil {
		t.Fatal(err)
	}
	if plan.CreatedRows() != 1 || plan.UpdatedRows() != 1 || plan.UnchangedRows() != 1 || plan.RejectedRows() != 0 {
		t.Fatalf("unexpected plan counts: %+v", plan)
	}
	updated := plan.Rows[0].User
	if updated.Email != nil {
		t.Fatalf("present empty email must clear, got %q", *updated.Email)
	}
	if len(updated.Roles) != 1 || updated.Roles[0] != "support" {
		t.Fatalf("absent roles must be preserved, got %v", updated.Roles)
	}
	if got := updated.Lifecycle.RequiredActions; len(got) != 1 || got[0] != idmdomain.RequiredActionUpdatePassword {
		t.Fatalf("required_actions=%v", got)
	}
	if got := updated.Attributes["department"].String; got == nil || *got != "Engineering" {
		t.Fatalf("department=%+v", got)
	}
}

func TestPlanUserImportRejectsIdentifierMismatchInvalidTypesAndMissingIdentifier(t *testing.T) {
	repo := usermemory.NewUserRepository()
	repo.Seed(importPlannerUser("user-alice", "alice"))
	repo.Seed(importPlannerUser("user-bob", "bob"))
	repo.Seed(importPlannerUser("user-carol", "carol"))
	csv := "id,preferred_username,email_verified,attr:department\n" +
		"user-alice,bob,true,Engineering\n" +
		"user-carol,carol,TRUE,Engineering\n" +
		",,,Engineering\n"

	plan, err := planUserImportForTest(importPlannerContext(), importPlannerDeps(repo, perUserImportOwnershipGuard{}), csv)
	if err != nil {
		t.Fatal(err)
	}
	want := []idmdomain.CSVErrorCode{"identifier_mismatch", "invalid_boolean", "missing_identifier"}
	if plan.RejectedRows() != len(want) {
		t.Fatalf("plan=%+v", plan)
	}
	for i, code := range want {
		if plan.Rows[i].Error == nil || plan.Rows[i].Error.Code != code {
			t.Fatalf("row[%d]=%+v, want %q", i, plan.Rows[i], code)
		}
	}
}

func TestPlanUserImportFailsClosedForSourceManagedUsers(t *testing.T) {
	for name, guard := range map[string]perUserImportOwnershipGuard{
		"managed":       {managed: map[string]bool{"user-alice": true}},
		"guard failure": {err: errors.New("source unavailable")},
		"missing guard": {},
	} {
		t.Run(name, func(t *testing.T) {
			repo := usermemory.NewUserRepository()
			repo.Seed(importPlannerUser("user-alice", "alice"))
			deps := importPlannerDeps(repo, guard)
			if name == "missing guard" {
				deps.OwnershipGuard = nil
			}
			plan, err := planUserImportForTest(importPlannerContext(), deps, "id,email\nuser-alice,new@example.com\n")
			if err != nil {
				t.Fatal(err)
			}
			if plan.RejectedRows() != 1 || plan.Rows[0].Error == nil || plan.Rows[0].Error.Code != "source_managed" {
				t.Fatalf("plan=%+v", plan)
			}
		})
	}
}

func TestPlanUserImportReplansAgainstCurrentRepositoryState(t *testing.T) {
	repo := usermemory.NewUserRepository()
	alice := importPlannerUser("user-alice", "alice")
	repo.Seed(alice)
	deps := importPlannerDeps(repo, perUserImportOwnershipGuard{})
	csv := "id,email\nuser-alice,new@example.com\n"

	preview, err := planUserImportForTest(importPlannerContext(), deps, csv)
	if err != nil || preview.UpdatedRows() != 1 {
		t.Fatalf("preview=%+v error=%v", preview, err)
	}
	newEmail := "new@example.com"
	alice.Email = &newEmail
	repo.Seed(alice)

	replanned, err := planUserImportForTest(importPlannerContext(), deps, csv)
	if err != nil || replanned.UnchangedRows() != 1 {
		t.Fatalf("replanned=%+v error=%v", replanned, err)
	}
}
