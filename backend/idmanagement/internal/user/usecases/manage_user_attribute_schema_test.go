package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/idmanagement/internal/user/db_memory"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/internal/group/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/tenancy/domain"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

func TestGetUserAttributeSchemaReturnsEmptyForUndefinedTenant(t *testing.T) {
	repo := memory.NewTenantUserAttributeSchemaRepository()
	schema, err := GetUserAttributeSchema(context.Background(), repo, domain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	if schema == nil || schema.TenantID != domain.DefaultTenantID || len(schema.Attributes) != 0 {
		t.Fatalf("expected empty schema, got %#v", schema)
	}
}

//spec:covers REQ-TENANCY-043: 参照中の属性は書き込み権威を変更できず、省略と upstream の相互変更は同じ分類として許す。
func TestReferencedAttributeWriteAuthorityCannotChange(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewTenantUserAttributeSchemaRepository()
	groups := groupmemory.NewGroupRepository()
	now := time.Now().UTC()
	defs := []userdomain.UserAttributeDef{{Key: "approval", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityAdminReadable}}
	if _, err := UpdateUserAttributeSchema(ctx, repo, groups, "acme", defs, now); err != nil {
		t.Fatal(err)
	}
	if err := groups.SaveDynamicRule(ctx, &groupdomain.DynamicGroupRule{TenantID: "acme", GroupID: "g", ReferencedAttributes: []string{"approval"}}); err != nil {
		t.Fatal(err)
	}
	defs[0].WriteAuthority = userdomain.AttributeAuthorityLocal
	if _, err := UpdateUserAttributeSchema(ctx, repo, groups, "acme", defs, now); !errors.Is(err, ErrAttributeReferencedByDynamicGroup) {
		t.Fatalf("authority change err=%v, want referenced refusal", err)
	}
	stored, err := repo.FindByTenant(ctx, "acme")
	if err != nil || stored.Attributes[0].WriteAuthority.Effective() != userdomain.AttributeAuthorityUpstream {
		t.Fatalf("refusal changed schema: %+v, %v", stored, err)
	}
	defs[0].WriteAuthority = userdomain.AttributeAuthorityUpstream
	if _, err := UpdateUserAttributeSchema(ctx, repo, groups, "acme", defs, now); err != nil {
		t.Fatalf("equivalent classification refused: %v", err)
	}
}

func TestUpdateUserAttributeSchemaPersistsCustomDefs(t *testing.T) {
	repo := memory.NewTenantUserAttributeSchemaRepository()
	ctx := context.Background()
	defs := []userdomain.UserAttributeDef{
		{Key: "region", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityClaimExposed, ClaimName: new("region")},
	}
	saved, err := UpdateUserAttributeSchema(ctx, repo, nil, domain.DefaultTenantID, defs, time.Now().UTC())
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if len(saved.Attributes) != 1 || saved.Attributes[0].Key != "region" {
		t.Fatalf("unexpected saved schema: %#v", saved)
	}
	reloaded, err := GetUserAttributeSchema(ctx, repo, domain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Attributes) != 1 || reloaded.Attributes[0].Key != "region" {
		t.Fatalf("schema not persisted: %#v", reloaded)
	}
}

func TestUpdateUserAttributeSchemaRejectsBuiltinCollision(t *testing.T) {
	repo := memory.NewTenantUserAttributeSchemaRepository()
	defs := []userdomain.UserAttributeDef{
		{Key: "nickname", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityClaimExposed},
	}
	if _, err := UpdateUserAttributeSchema(context.Background(), repo, nil, domain.DefaultTenantID, defs, time.Now().UTC()); !errors.Is(err, ErrInvalidUserAttributeSchema) {
		t.Fatalf("expected ErrInvalidUserAttributeSchema, got %v", err)
	}
}

func TestUpdateUserAttributeSchemaAllowsEmptyClear(t *testing.T) {
	repo := memory.NewTenantUserAttributeSchemaRepository()
	ctx := context.Background()
	if _, err := UpdateUserAttributeSchema(ctx, repo, nil, domain.DefaultTenantID,
		[]userdomain.UserAttributeDef{{Key: "region", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityAdminReadable}},
		time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	cleared, err := UpdateUserAttributeSchema(ctx, repo, nil, domain.DefaultTenantID, nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if len(cleared.Attributes) != 0 {
		t.Fatalf("expected cleared schema, got %#v", cleared)
	}
}
