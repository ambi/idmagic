package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	idmusecases "github.com/ambi/idmagic/backend/identitymanagement/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func attrTestDeps(t *testing.T) (context.Context, idmusecases.AdminUserDeps, *memory.TenantUserAttributeSchemaRepository) {
	t.Helper()
	schemaRepo := memory.NewTenantUserAttributeSchemaRepository()
	deps := idmusecases.AdminUserDeps{
		UserRepo:            memory.NewUserRepository(),
		AttrSchemaRepo:      schemaRepo,
		PasswordHasher:      crypto.NewArgon2idPasswordHasher(),
		PasswordHistoryRepo: memory.NewPasswordHistoryRepository(),
		Emit:                func(spec.DomainEvent) {},
	}
	return context.Background(), deps, schemaRepo
}

func createAttrUser(ctx context.Context, t *testing.T, deps idmusecases.AdminUserDeps) *spec.User {
	t.Helper()
	user, err := idmusecases.CreateUser(ctx, deps, idmusecases.CreateUserInput{
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

	attrs := map[string]spec.AttributeValue{
		"nickname":     {Type: spec.AttributeTypeString, String: new("cici")},
		"phone_number": {Type: spec.AttributeTypeString, String: new("+819012345678")},
	}
	updated, err := idmusecases.UpdateUser(ctx, deps, idmusecases.UpdateUserInput{
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

	attrs := map[string]spec.AttributeValue{
		"not_a_real_attribute": {Type: spec.AttributeTypeString, String: new("x")},
	}
	_, err := idmusecases.UpdateUser(ctx, deps, idmusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID, Attributes: &attrs, Now: time.Now().UTC(),
	})
	if !errors.Is(err, idmusecases.ErrInvalidAttribute) {
		t.Fatalf("expected ErrInvalidAttribute, got %v", err)
	}
}

func TestUpdateUserAcceptsTenantCustomAttribute(t *testing.T) {
	ctx, deps, schemaRepo := attrTestDeps(t)
	if err := schemaRepo.Save(ctx, &spec.TenantUserAttributeSchema{
		TenantID: spec.DefaultTenantID,
		Attributes: []spec.UserAttributeDef{
			{Key: "region", Type: spec.AttributeTypeString, Visibility: spec.AttrVisibilityClaimExposed, ClaimName: new("region")},
		},
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	user := createAttrUser(ctx, t, deps)

	attrs := map[string]spec.AttributeValue{
		"region": {Type: spec.AttributeTypeString, String: new("apac")},
	}
	updated, err := idmusecases.UpdateUser(ctx, deps, idmusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID, Attributes: &attrs, Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if v := updated.Attributes["region"]; v.String == nil || *v.String != "apac" {
		t.Fatalf("region not stored: %+v", updated.Attributes)
	}

	// schema 未定義の custom key は拒否される。
	bad := map[string]spec.AttributeValue{"zone": {Type: spec.AttributeTypeString, String: new("z")}}
	if _, err := idmusecases.UpdateUser(ctx, deps, idmusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID, Attributes: &bad, Now: time.Now().UTC(),
	}); !errors.Is(err, idmusecases.ErrInvalidAttribute) {
		t.Fatalf("expected ErrInvalidAttribute for undefined custom key, got %v", err)
	}
}
