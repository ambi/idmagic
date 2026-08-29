package usecases

import (
	"context"
	"errors"
	"io"
	"reflect"
	"slices"
	"time"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type UserImportApplyDeps struct {
	Plan           UserImportPlanDeps
	Committer      userports.UserImportRowCommitter
	PasswordHasher passwordports.PasswordHasher
}

// ApplyUserImport replans the immutable preview payload against current state,
// then submits one complete mutation per accepted row to the atomic commit port.
func ApplyUserImport(
	ctx context.Context,
	deps UserImportApplyDeps,
	input io.Reader,
	policy idmdomain.CSVTransferPolicy,
	actorUserID string,
	now time.Time,
	emit func(userdomain.UserImportRowPlan) error,
) (userdomain.UserImportPlanSummary, error) {
	var applied userdomain.UserImportPlanSummary
	if deps.Committer == nil || deps.PasswordHasher == nil {
		return applied, errors.New("user import apply dependencies are incomplete")
	}
	now = now.UTC()
	_, err := PlanUserImport(ctx, deps.Plan, input, policy, func(row userdomain.UserImportRowPlan) error {
		final := row
		if row.Action == userdomain.UserImportCreate || row.Action == userdomain.UserImportUpdate {
			mutation, err := prepareUserImportMutation(deps, row, actorUserID, now)
			if err == nil {
				err = deps.Committer.CommitUserImportRow(ctx, mutation)
			}
			if err != nil {
				final = rejectedUserImportRow(row.Row, "", "apply_failed")
			}
		}
		applied.Observe(final)
		if emit != nil {
			return emit(final)
		}
		return nil
	})
	return applied, err
}

func prepareUserImportMutation(deps UserImportApplyDeps, row userdomain.UserImportRowPlan, actorUserID string, now time.Time) (userports.UserImportRowMutation, error) {
	after := cloneImportUser(*row.User)
	mutation := userports.UserImportRowMutation{
		Before: row.Before, After: &after, ActorUserID: actorUserID, Now: now,
	}
	if row.Action == userdomain.UserImportCreate {
		password, err := randomImportPassword()
		if err != nil {
			return userports.UserImportRowMutation{}, err
		}
		hash, err := deps.PasswordHasher.Hash(password)
		if err != nil {
			return userports.UserImportRowMutation{}, err
		}
		id, err := spec.NewUUIDv4()
		if err != nil {
			return userports.UserImportRowMutation{}, err
		}
		after.ID = id
		after.PasswordHash = hash
		after.CreatedAt = now
		after.UpdatedAt = now
		after.Lifecycle.Status = idmdomain.UserStatusActive
		if !slices.Contains(after.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword) {
			after.Lifecycle.RequiredActions = append(after.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword)
			slices.Sort(after.Lifecycle.RequiredActions)
		}
		mutation.AuditEventType = "UserCreated"
		mutation.PasswordHistoryHash = hash
		mutation.ConsumesUserQuota = true
		mutation.Changed = []string{"user"}
	} else {
		after.UpdatedAt = now
		mutation.AuditEventType = "UserUpdated"
		mutation.Changed = importedUserChangedFields(*row.Before, after)
	}
	if err := after.Validate(); err != nil {
		return userports.UserImportRowMutation{}, err
	}
	return mutation, nil
}

func importedUserChangedFields(before, after userdomain.User) []string {
	changed := make([]string, 0, 9)
	appendIf := func(condition bool, field string) {
		if condition {
			changed = append(changed, field)
		}
	}
	appendIf(before.PreferredUsername != after.PreferredUsername, "preferred_username")
	appendIf(!reflect.DeepEqual(before.Name, after.Name), "name")
	appendIf(!reflect.DeepEqual(before.GivenName, after.GivenName), "given_name")
	appendIf(!reflect.DeepEqual(before.FamilyName, after.FamilyName), "family_name")
	appendIf(!reflect.DeepEqual(before.Email, after.Email), "email")
	appendIf(before.EmailVerified != after.EmailVerified, "email_verified")
	appendIf(!slices.Equal(before.Roles, after.Roles), "roles")
	appendIf(!slices.Equal(before.Lifecycle.RequiredActions, after.Lifecycle.RequiredActions), "required_actions")
	appendIf(!reflect.DeepEqual(before.Attributes, after.Attributes), "attributes")
	return changed
}
