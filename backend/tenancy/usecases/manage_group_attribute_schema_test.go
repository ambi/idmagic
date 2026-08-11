package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/tenancy/domain"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
)

func TestGetGroupAttributeSchemaReturnsEmptyForUndefinedTenant(t *testing.T) {
	repo := groupmemory.NewTenantGroupAttributeSchemaRepository()
	schema, err := GetGroupAttributeSchema(context.Background(), repo, domain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	if schema == nil || schema.TenantID != domain.DefaultTenantID || len(schema.Attributes) != 0 {
		t.Fatalf("expected empty schema, got %#v", schema)
	}
}

func TestUpdateGroupAttributeSchemaPersistsCustomDefs(t *testing.T) {
	repo := groupmemory.NewTenantGroupAttributeSchemaRepository()
	ctx := context.Background()
	defs := []groupdomain.GroupAttributeDef{
		{Key: "cost_center", Type: idmdomain.AttributeTypeString},
	}
	saved, err := UpdateGroupAttributeSchema(ctx, repo, domain.DefaultTenantID, defs, time.Now().UTC())
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if len(saved.Attributes) != 1 || saved.Attributes[0].Key != "cost_center" {
		t.Fatalf("unexpected saved schema: %#v", saved)
	}
	reloaded, err := GetGroupAttributeSchema(ctx, repo, domain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Attributes) != 1 || reloaded.Attributes[0].Key != "cost_center" {
		t.Fatalf("schema not persisted: %#v", reloaded)
	}
}

func TestUpdateGroupAttributeSchemaRejectsDuplicateKey(t *testing.T) {
	repo := groupmemory.NewTenantGroupAttributeSchemaRepository()
	defs := []groupdomain.GroupAttributeDef{
		{Key: "cost_center", Type: idmdomain.AttributeTypeString},
		{Key: "cost_center", Type: idmdomain.AttributeTypeNumber},
	}
	if _, err := UpdateGroupAttributeSchema(
		context.Background(), repo, domain.DefaultTenantID, defs, time.Now().UTC(),
	); !errors.Is(err, ErrInvalidGroupAttributeSchema) {
		t.Fatalf("expected ErrInvalidGroupAttributeSchema, got %v", err)
	}
}

func TestUpdateGroupAttributeSchemaAllowsEmptyClear(t *testing.T) {
	repo := groupmemory.NewTenantGroupAttributeSchemaRepository()
	ctx := context.Background()
	if _, err := UpdateGroupAttributeSchema(ctx, repo, domain.DefaultTenantID,
		[]groupdomain.GroupAttributeDef{{Key: "cost_center", Type: idmdomain.AttributeTypeString}},
		time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	cleared, err := UpdateGroupAttributeSchema(ctx, repo, domain.DefaultTenantID, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if len(cleared.Attributes) != 0 {
		t.Fatalf("expected cleared schema, got %#v", cleared)
	}
}
