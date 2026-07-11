package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	idmmemory "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/memory"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestGetUserAttributeSchemaReturnsEmptyForUndefinedTenant(t *testing.T) {
	repo := idmmemory.NewTenantUserAttributeSchemaRepository()
	schema, err := GetUserAttributeSchema(context.Background(), repo, spec.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	if schema == nil || schema.TenantID != spec.DefaultTenantID || len(schema.Attributes) != 0 {
		t.Fatalf("expected empty schema, got %#v", schema)
	}
}

func TestUpdateUserAttributeSchemaPersistsCustomDefs(t *testing.T) {
	repo := idmmemory.NewTenantUserAttributeSchemaRepository()
	ctx := context.Background()
	defs := []idmdomain.UserAttributeDef{
		{Key: "region", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityClaimExposed, ClaimName: new("region")},
	}
	saved, err := UpdateUserAttributeSchema(ctx, repo, spec.DefaultTenantID, defs, time.Now().UTC())
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if len(saved.Attributes) != 1 || saved.Attributes[0].Key != "region" {
		t.Fatalf("unexpected saved schema: %#v", saved)
	}
	reloaded, err := GetUserAttributeSchema(ctx, repo, spec.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Attributes) != 1 || reloaded.Attributes[0].Key != "region" {
		t.Fatalf("schema not persisted: %#v", reloaded)
	}
}

func TestUpdateUserAttributeSchemaRejectsBuiltinCollision(t *testing.T) {
	repo := idmmemory.NewTenantUserAttributeSchemaRepository()
	defs := []idmdomain.UserAttributeDef{
		{Key: "nickname", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityClaimExposed},
	}
	if _, err := UpdateUserAttributeSchema(
		context.Background(), repo, spec.DefaultTenantID, defs, time.Now().UTC(),
	); !errors.Is(err, ErrInvalidUserAttributeSchema) {
		t.Fatalf("expected ErrInvalidUserAttributeSchema, got %v", err)
	}
}

func TestUpdateUserAttributeSchemaAllowsEmptyClear(t *testing.T) {
	repo := idmmemory.NewTenantUserAttributeSchemaRepository()
	ctx := context.Background()
	if _, err := UpdateUserAttributeSchema(ctx, repo, spec.DefaultTenantID,
		[]idmdomain.UserAttributeDef{{Key: "region", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityAdminReadable}},
		time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	cleared, err := UpdateUserAttributeSchema(ctx, repo, spec.DefaultTenantID, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if len(cleared.Attributes) != 0 {
		t.Fatalf("expected cleared schema, got %#v", cleared)
	}
}
