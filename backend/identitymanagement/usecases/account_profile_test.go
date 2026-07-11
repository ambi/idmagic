package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	idmmemory "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/memory"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"

	authnmemory "github.com/ambi/idmagic/backend/authentication/adapters/persistence/memory"

	idmusecases "github.com/ambi/idmagic/backend/identitymanagement/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func accountTestDeps(t *testing.T) (context.Context, idmusecases.AccountProfileDeps, *idmdomain.User) {
	t.Helper()
	userRepo := idmmemory.NewUserRepository()
	schemaRepo := idmmemory.NewTenantUserAttributeSchemaRepository()
	adminDeps := idmusecases.AdminUserDeps{
		UserRepo: userRepo, AttrSchemaRepo: schemaRepo,
		PasswordHasher: crypto.NewArgon2idPasswordHasher(), PasswordHistoryRepo: authnmemory.NewPasswordHistoryRepository(),
		Emit: func(spec.DomainEvent) {},
	}
	ctx := context.Background()
	user, err := idmusecases.CreateUser(ctx, adminDeps, idmusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "dave", Password: "initial-password-9182", Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	// admin 管理属性 (organization, editable_by_user=false) を事前に入れておく。
	user.Attributes = map[string]idmdomain.AttributeValue{
		"department": {Type: idmdomain.AttributeTypeString, String: new("Platform")},
	}
	if err := userRepo.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	deps := idmusecases.AccountProfileDeps{UserRepo: userRepo, AttrSchemaRepo: schemaRepo, Emit: func(spec.DomainEvent) {}}
	return ctx, deps, user
}

func TestUpdateUserProfileEditsNameAndEditableAttribute(t *testing.T) {
	ctx, deps, user := accountTestDeps(t)
	attrs := map[string]idmdomain.AttributeValue{
		"nickname": {Type: idmdomain.AttributeTypeString, String: new("davey")},
	}
	updated, _, err := idmusecases.UpdateUserProfile(ctx, deps, idmusecases.UpdateUserProfileInput{
		Sub: user.ID, GivenName: new("Dave"), Attributes: &attrs, Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.GivenName == nil || *updated.GivenName != "Dave" {
		t.Fatalf("given_name not updated: %+v", updated.GivenName)
	}
	if v := updated.Attributes["nickname"]; v.String == nil || *v.String != "davey" {
		t.Fatalf("nickname not stored: %+v", updated.Attributes)
	}
	// admin 管理属性 (department) は merge で保持される。
	if v := updated.Attributes["department"]; v.String == nil || *v.String != "Platform" {
		t.Fatalf("admin-managed attribute lost on self merge: %+v", updated.Attributes)
	}
}

func TestUpdateUserProfileRejectsAdminManagedAttribute(t *testing.T) {
	ctx, deps, user := accountTestDeps(t)
	attrs := map[string]idmdomain.AttributeValue{
		"department": {Type: idmdomain.AttributeTypeString, String: new("Sales")}, // editable_by_user=false
	}
	_, _, err := idmusecases.UpdateUserProfile(ctx, deps, idmusecases.UpdateUserProfileInput{
		Sub: user.ID, Attributes: &attrs, Now: time.Now().UTC(),
	})
	if !errors.Is(err, idmusecases.ErrAttributeNotEditable) {
		t.Fatalf("expected ErrAttributeNotEditable, got %v", err)
	}
}

func TestUpdateUserProfileRejectsUndefinedAttribute(t *testing.T) {
	ctx, deps, user := accountTestDeps(t)
	attrs := map[string]idmdomain.AttributeValue{
		"not_a_real_attribute": {Type: idmdomain.AttributeTypeString, String: new("x")},
	}
	_, _, err := idmusecases.UpdateUserProfile(ctx, deps, idmusecases.UpdateUserProfileInput{
		Sub: user.ID, Attributes: &attrs, Now: time.Now().UTC(),
	})
	if !errors.Is(err, idmusecases.ErrInvalidAttribute) {
		t.Fatalf("expected ErrInvalidAttribute, got %v", err)
	}
}

func TestGetUserProfileShowsReadOnlyOrganizationAttributes(t *testing.T) {
	ctx, deps, user := accountTestDeps(t)
	_, defs, err := idmusecases.GetUserProfile(ctx, deps, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	// department は本人が参照できるが、self-service では編集できない。
	self := idmusecases.SelfReadableAttributes(user.Attributes, defs)
	if v, ok := self["department"]; !ok || v.String == nil || *v.String != "Platform" {
		t.Fatalf("self_readable organization attribute missing: %+v", self)
	}
}

func TestAccountProfileAttributeDefFilters(t *testing.T) {
	defs := []idmdomain.UserAttributeDef{
		{Key: "nickname", Visibility: idmdomain.AttrVisibilityClaimExposed, EditableByUser: true},
		{Key: "department", Visibility: idmdomain.AttrVisibilitySelfReadable, EditableByUser: false},
		{Key: "payroll_id", Visibility: idmdomain.AttrVisibilityAdminReadable, EditableByUser: false},
		{Key: "secret_note", Visibility: idmdomain.AttrVisibilityPrivate, EditableByUser: false},
	}

	readable := idmusecases.SelfReadableAttributeDefs(defs)
	if len(readable) != 2 || readable[0].Key != "nickname" || readable[1].Key != "department" {
		t.Fatalf("readable defs = %+v", readable)
	}
	editable := idmusecases.EditableAttributeDefs(defs)
	if len(editable) != 1 || editable[0].Key != "nickname" {
		t.Fatalf("editable defs = %+v", editable)
	}
}

func TestGetUserProfileRejectsMissingSelf(t *testing.T) {
	ctx, deps, _ := accountTestDeps(t)
	if _, _, err := idmusecases.GetUserProfile(ctx, deps, "ghost"); !errors.Is(err, idmusecases.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
