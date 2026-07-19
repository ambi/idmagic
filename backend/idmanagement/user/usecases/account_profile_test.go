package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	authnmemory "github.com/ambi/idmagic/backend/authentication/adapters/persistence/memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/adapters/persistence/memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"

	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func accountTestDeps(t *testing.T) (context.Context, userusecases.AccountProfileDeps, *userdomain.User) {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	schemaRepo := usermemory.NewTenantUserAttributeSchemaRepository()
	adminDeps := userusecases.AdminUserDeps{
		UserRepo: userRepo, AttrSchemaRepo: schemaRepo,
		PasswordHasher: crypto.NewArgon2idPasswordHasher(), PasswordHistoryRepo: authnmemory.NewPasswordHistoryRepository(),
		Emit: func(spec.DomainEvent) error { return nil },
	}
	ctx := context.Background()
	user, err := userusecases.CreateUser(ctx, adminDeps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "dave", Password: "initial-password-9182", Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	// admin 管理属性 (organization, editable_by_user=false) を事前に入れておく。
	user.Attributes = map[string]userdomain.AttributeValue{
		"department": {Type: idmdomain.AttributeTypeString, String: new("Platform")},
	}
	if err := userRepo.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	deps := userusecases.AccountProfileDeps{
		UserRepo: userRepo, AttrSchemaRepo: schemaRepo, Emit: func(spec.DomainEvent) error { return nil },
	}
	return ctx, deps, user
}

func TestUpdateUserProfileEditsNameAndEditableAttribute(t *testing.T) {
	ctx, deps, user := accountTestDeps(t)
	attrs := map[string]userdomain.AttributeValue{
		"nickname": {Type: idmdomain.AttributeTypeString, String: new("davey")},
	}
	updated, _, err := userusecases.UpdateUserProfile(ctx, deps, userusecases.UpdateUserProfileInput{
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
	attrs := map[string]userdomain.AttributeValue{
		"department": {Type: idmdomain.AttributeTypeString, String: new("Sales")}, // editable_by_user=false
	}
	_, _, err := userusecases.UpdateUserProfile(ctx, deps, userusecases.UpdateUserProfileInput{
		Sub: user.ID, Attributes: &attrs, Now: time.Now().UTC(),
	})
	if !errors.Is(err, userusecases.ErrAttributeNotEditable) {
		t.Fatalf("expected ErrAttributeNotEditable, got %v", err)
	}
}

func TestUpdateUserProfileRejectsUndefinedAttribute(t *testing.T) {
	ctx, deps, user := accountTestDeps(t)
	attrs := map[string]userdomain.AttributeValue{
		"not_a_real_attribute": {Type: idmdomain.AttributeTypeString, String: new("x")},
	}
	_, _, err := userusecases.UpdateUserProfile(ctx, deps, userusecases.UpdateUserProfileInput{
		Sub: user.ID, Attributes: &attrs, Now: time.Now().UTC(),
	})
	if !errors.Is(err, userusecases.ErrInvalidAttribute) {
		t.Fatalf("expected ErrInvalidAttribute, got %v", err)
	}
}

func TestGetUserProfileShowsReadOnlyOrganizationAttributes(t *testing.T) {
	ctx, deps, user := accountTestDeps(t)
	_, defs, err := userusecases.GetUserProfile(ctx, deps, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	// department は本人が参照できるが、self-service では編集できない。
	self := userusecases.SelfReadableAttributes(user.Attributes, defs)
	if v, ok := self["department"]; !ok || v.String == nil || *v.String != "Platform" {
		t.Fatalf("self_readable organization attribute missing: %+v", self)
	}
}

func TestAccountProfileAttributeDefFilters(t *testing.T) {
	defs := []userdomain.UserAttributeDef{
		{Key: "nickname", Visibility: idmdomain.AttrVisibilityClaimExposed, EditableByUser: true},
		{Key: "department", Visibility: idmdomain.AttrVisibilitySelfReadable, EditableByUser: false},
		{Key: "payroll_id", Visibility: idmdomain.AttrVisibilityAdminReadable, EditableByUser: false},
		{Key: "secret_note", Visibility: idmdomain.AttrVisibilityPrivate, EditableByUser: false},
	}

	readable := userusecases.SelfReadableAttributeDefs(defs)
	if len(readable) != 2 || readable[0].Key != "nickname" || readable[1].Key != "department" {
		t.Fatalf("readable defs = %+v", readable)
	}
	editable := userusecases.EditableAttributeDefs(defs)
	if len(editable) != 1 || editable[0].Key != "nickname" {
		t.Fatalf("editable defs = %+v", editable)
	}
}

func TestGetUserProfileRejectsMissingSelf(t *testing.T) {
	ctx, deps, _ := accountTestDeps(t)
	if _, _, err := userusecases.GetUserProfile(ctx, deps, "ghost"); !errors.Is(err, idmusecases.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
