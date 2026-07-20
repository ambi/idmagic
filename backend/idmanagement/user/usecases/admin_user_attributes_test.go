package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	authnmemory "github.com/ambi/idmagic/backend/authentication/password/adapters/persistence/memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/adapters/persistence/memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"

	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func attrTestDeps(t *testing.T) (context.Context, userusecases.AdminUserDeps, *usermemory.TenantUserAttributeSchemaRepository) {
	t.Helper()
	schemaRepo := usermemory.NewTenantUserAttributeSchemaRepository()
	deps := userusecases.AdminUserDeps{
		UserRepo:            usermemory.NewUserRepository(),
		AttrSchemaRepo:      schemaRepo,
		PasswordHasher:      crypto.NewArgon2idPasswordHasher(),
		PasswordHistoryRepo: authnmemory.NewPasswordHistoryRepository(),
		Emit:                func(spec.DomainEvent) error { return nil },
	}
	return context.Background(), deps, schemaRepo
}

func createAttrUser(ctx context.Context, t *testing.T, deps userusecases.AdminUserDeps) *userdomain.User {
	t.Helper()
	user, err := userusecases.CreateUser(ctx, deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "carol", Password: "initial-password-9182",
		Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func TestUpdateUserAcceptsBuiltinAttribute(t *testing.T) {
	ctx, deps, _ := attrTestDeps(t)
	user := createAttrUser(ctx, t, deps)

	attrs := map[string]userdomain.AttributeValue{
		"nickname":     {Type: idmdomain.AttributeTypeString, String: new("cici")},
		"phone_number": {Type: idmdomain.AttributeTypeString, String: new("+819012345678")},
	}
	updated, err := userusecases.UpdateUser(ctx, deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID, GivenName: new("Carol"), Attributes: &attrs,
		Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.GivenName == nil || *updated.GivenName != "Carol" {
		t.Fatalf("given_name not updated: %+v", updated.GivenName)
	}
	if v := updated.Attributes["nickname"]; v.String == nil || *v.String != "cici" {
		t.Fatalf("nickname not stored: %+v", updated.Attributes)
	}
}

func TestUpdateUserRejectsUndefinedAttribute(t *testing.T) {
	ctx, deps, _ := attrTestDeps(t)
	user := createAttrUser(ctx, t, deps)

	attrs := map[string]userdomain.AttributeValue{
		"not_a_real_attribute": {Type: idmdomain.AttributeTypeString, String: new("x")},
	}
	_, err := userusecases.UpdateUser(ctx, deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID, Attributes: &attrs, Now: time.Now().UTC(),
	})
	if !errors.Is(err, userusecases.ErrInvalidAttribute) {
		t.Fatalf("expected ErrInvalidAttribute, got %v", err)
	}
}

func TestUpdateUserAcceptsTenantCustomAttribute(t *testing.T) {
	ctx, deps, schemaRepo := attrTestDeps(t)
	if err := schemaRepo.Save(ctx, &userdomain.TenantUserAttributeSchema{
		TenantID: tenancydomain.DefaultTenantID,
		Attributes: []userdomain.UserAttributeDef{
			{Key: "region", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityClaimExposed, ClaimName: new("region")},
		},
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	user := createAttrUser(ctx, t, deps)

	attrs := map[string]userdomain.AttributeValue{
		"region": {Type: idmdomain.AttributeTypeString, String: new("apac")},
	}
	updated, err := userusecases.UpdateUser(ctx, deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID, Attributes: &attrs, Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if v := updated.Attributes["region"]; v.String == nil || *v.String != "apac" {
		t.Fatalf("region not stored: %+v", updated.Attributes)
	}

	// schema 未定義の custom key は拒否される。
	bad := map[string]userdomain.AttributeValue{"zone": {Type: idmdomain.AttributeTypeString, String: new("z")}}
	if _, err := userusecases.UpdateUser(ctx, deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID, Attributes: &bad, Now: time.Now().UTC(),
	}); !errors.Is(err, userusecases.ErrInvalidAttribute) {
		t.Fatalf("expected ErrInvalidAttribute for undefined custom key, got %v", err)
	}
}
