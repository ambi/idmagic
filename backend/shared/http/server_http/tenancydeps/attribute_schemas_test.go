package tenancydeps_test

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/shared/http/server_http/tenancydeps"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

// FuzzAttributeSchemaProjection は管理用の値と保存型の変換が、検証前の値も含めて情報を失わないことを確かめる。
func FuzzAttributeSchemaProjection(f *testing.F) {
	f.Add("region", "地域", "string", "claim_exposed", "region", "profile", uint8(255))
	f.Add("cost_center", "", "number", "private", "", "", uint8(0))
	f.Add("unknown", "未知", "future_type", "future_visibility", "", "", uint8(128))
	f.Fuzz(func(t *testing.T, key, label, attributeType, visibility, claim, scope string, flags uint8) {
		ctx := context.Background()
		identity := idmanagement.NewMemoryModule()
		users := tenancydeps.UserAttributeSchemas(identity.AttrSchemaRepo)
		groups := tenancydeps.GroupAttributeSchemas(identity.GroupAttrSchemaRepo)
		now := time.Unix(1730000000, 123456789).UTC()
		definition := domain.UserAttributeDef{
			Key: key, Label: label, Type: attributeType, Visibility: visibility,
			MultiValued: flags&1 != 0, Required: flags&2 != 0,
			EditableByUser: flags&4 != 0, PII: flags&8 != 0,
		}
		if flags&16 != 0 {
			definition.ClaimName = new(claim)
		}
		if flags&32 != 0 {
			definition.OIDCScope = new(scope)
		}
		user := &domain.TenantUserAttributeSchema{
			TenantID: "tenant", CreatedAt: now, UpdatedAt: now.Add(time.Second),
		}
		group := &domain.TenantGroupAttributeSchema{
			TenantID: "tenant", CreatedAt: now, UpdatedAt: now.Add(time.Second),
		}
		if flags&64 != 0 {
			user.Attributes = []domain.UserAttributeDef{}
			group.Attributes = []domain.GroupAttributeDef{}
		}
		if flags&128 != 0 {
			user.Attributes = []domain.UserAttributeDef{definition}
			group.Attributes = []domain.GroupAttributeDef{{
				Key: key, Label: label, Type: attributeType,
				MultiValued: definition.MultiValued, Required: definition.Required,
			}}
		}
		if err := users.Save(ctx, user); err != nil {
			t.Fatal(err)
		}
		if err := groups.Save(ctx, group); err != nil {
			t.Fatal(err)
		}
		savedUser, err := users.FindByTenant(ctx, "tenant")
		if err != nil || !reflect.DeepEqual(savedUser, user) {
			t.Fatalf("user projection changed the value: got %#v, want %#v, error %v", savedUser, user, err)
		}
		savedGroup, err := groups.FindByTenant(ctx, "tenant")
		if err != nil || !reflect.DeepEqual(savedGroup, group) {
			t.Fatalf("group projection changed the value: got %#v, want %#v, error %v", savedGroup, group, err)
		}
		nativeUser, err := identity.AttrSchemaRepo.FindByTenant(ctx, "tenant")
		if err != nil {
			t.Fatal(err)
		}
		nativeGroup, err := identity.GroupAttrSchemaRepo.FindByTenant(ctx, "tenant")
		if err != nil {
			t.Fatal(err)
		}
		assertSameJSON(t, user, nativeUser)
		assertSameJSON(t, group, nativeGroup)
	})
}

func assertSameJSON(t *testing.T, managed, stored any) {
	t.Helper()
	expected, err := json.Marshal(managed)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatalf("projection changed the stored JSON: got %s, want %s", actual, expected)
	}
}
