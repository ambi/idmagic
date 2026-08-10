package usecases

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

type importTestHasher struct{ err error }

func (h importTestHasher) Hash(string) (string, error)         { return "import-password-hash", h.err }
func (h importTestHasher) Verify(string, string) (bool, error) { return false, nil }

type importRowCommitter struct {
	mutations    []userports.UserImportRowMutation
	failUsername string
}

func (c *importRowCommitter) CommitUserImportRow(_ context.Context, mutation userports.UserImportRowMutation) error {
	if mutation.After.PreferredUsername == c.failUsername {
		return errors.New("commit failed")
	}
	c.mutations = append(c.mutations, mutation)
	return nil
}

func applyUserImportForTest(
	ctx context.Context,
	deps UserImportApplyDeps,
	input string,
) (userdomain.UserImportPlanSummary, []userdomain.UserImportRowPlan, error) {
	var rows []userdomain.UserImportRowPlan
	summary, err := ApplyUserImport(ctx, deps, strings.NewReader(input), userdomain.DefaultUserCSVTransferPolicy(), "admin", time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC), func(row userdomain.UserImportRowPlan) error {
		rows = append(rows, row)
		return nil
	})
	return summary, rows, err
}

func TestApplyUserImportCommitsEachRowAtomicallyAndContinuesAfterFailure(t *testing.T) {
	repo := usermemory.NewUserRepository()
	repo.Seed(importPlannerUser("user-alice", "alice"))
	repo.Seed(importPlannerUser("user-bob", "bob"))
	committer := &importRowCommitter{failUsername: "bob"}
	deps := UserImportApplyDeps{
		Plan:           importPlannerDeps(repo, perUserImportOwnershipGuard{}),
		Committer:      committer,
		PasswordHasher: importTestHasher{},
	}
	csv := "id,email,roles,required_actions,attr:department\n" +
		"user-alice,new-alice@example.com,admin|support,verify_email,Engineering\n" +
		"user-bob,new-bob@example.com,admin,update_password,Sales\n"

	summary, rows, err := applyUserImportForTest(importPlannerContext(), deps, csv)
	if err != nil {
		t.Fatal(err)
	}
	if summary.UpdatedRows != 1 || summary.RejectedRows != 1 || len(rows) != 2 {
		t.Fatalf("summary=%+v rows=%+v", summary, rows)
	}
	if rows[1].Error == nil || rows[1].Error.Code != "apply_failed" {
		t.Fatalf("failed row=%+v", rows[1])
	}
	if len(committer.mutations) != 1 {
		t.Fatalf("commits=%d, want 1", len(committer.mutations))
	}
	mutation := committer.mutations[0]
	if mutation.Before == nil || mutation.Before.ID != "user-alice" || mutation.After.ID != "user-alice" {
		t.Fatalf("mutation identity=%+v", mutation)
	}
	if mutation.After.Email == nil || *mutation.After.Email != "new-alice@example.com" ||
		len(mutation.After.Roles) != 2 || len(mutation.After.Lifecycle.RequiredActions) != 1 {
		t.Fatalf("after=%+v", mutation.After)
	}
	if got := mutation.After.Attributes["department"].String; got == nil || *got != "Engineering" {
		t.Fatalf("department=%+v", got)
	}
	if mutation.ActorUserID != "admin" || mutation.AuditEventType != "UserUpdated" || mutation.Now.IsZero() {
		t.Fatalf("audit metadata=%+v", mutation)
	}
}

func TestApplyUserImportCreateIncludesCredentialHistoryRequiredActionAndQuotaInOneCommit(t *testing.T) {
	repo := usermemory.NewUserRepository()
	committer := &importRowCommitter{}
	deps := UserImportApplyDeps{
		Plan:           importPlannerDeps(repo, perUserImportOwnershipGuard{}),
		Committer:      committer,
		PasswordHasher: importTestHasher{},
	}

	summary, rows, err := applyUserImportForTest(importPlannerContext(), deps, "preferred_username,email\nalice,alice@example.com\n")
	if err != nil {
		t.Fatal(err)
	}
	if summary.CreatedRows != 1 || len(rows) != 1 || len(committer.mutations) != 1 {
		t.Fatalf("summary=%+v rows=%+v commits=%d", summary, rows, len(committer.mutations))
	}
	mutation := committer.mutations[0]
	if mutation.Before != nil || mutation.After.ID == "" || mutation.After.PasswordHash != "import-password-hash" {
		t.Fatalf("create mutation=%+v", mutation)
	}
	if mutation.PasswordHistoryHash != mutation.After.PasswordHash || !mutation.ConsumesUserQuota {
		t.Fatalf("credential/quota metadata=%+v", mutation)
	}
	if !containsRequiredAction(mutation.After.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword) {
		t.Fatalf("required actions=%v", mutation.After.Lifecycle.RequiredActions)
	}
	if mutation.AuditEventType != "UserCreated" {
		t.Fatalf("audit event=%q", mutation.AuditEventType)
	}
}

func containsRequiredAction(actions []idmdomain.RequiredAction, want idmdomain.RequiredAction) bool {
	return slices.Contains(actions, want)
}
