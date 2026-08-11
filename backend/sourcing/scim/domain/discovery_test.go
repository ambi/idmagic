package domain_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/sourcing/scim/domain"
)

// UserCoreSchema advertises exactly the RFC7643-CORE-RESOURCES
// adoption:partial attribute subset with correct mutability/required flags,
// so SCIM clients discover real capabilities instead of an empty array.
// interfaces.GetScimSchemas
func TestUserCoreSchemaAttributes(t *testing.T) {
	schema := domain.UserCoreSchema()
	if len(schema.Attributes) == 0 {
		t.Fatal("expected non-empty attribute list")
	}

	byName := make(map[string]domain.SchemaAttribute, len(schema.Attributes))
	for _, attr := range schema.Attributes {
		byName[attr.Name] = attr
	}

	userName, ok := byName["userName"]
	if !ok {
		t.Fatal("expected userName attribute")
	}
	if !userName.Required {
		t.Error("expected userName to be required")
	}
	if userName.Mutability != "readWrite" {
		t.Errorf("userName.Mutability = %q, want readWrite", userName.Mutability)
	}

	id, ok := byName["id"]
	if !ok {
		t.Fatal("expected id attribute")
	}
	if id.Mutability != "readOnly" {
		t.Errorf("id.Mutability = %q, want readOnly", id.Mutability)
	}

	active, ok := byName["active"]
	if !ok {
		t.Fatal("expected active attribute")
	}
	if active.Type != "boolean" {
		t.Errorf("active.Type = %q, want boolean", active.Type)
	}

	name, ok := byName["name"]
	if !ok {
		t.Fatal("expected name complex attribute")
	}
	if len(name.SubAttributes) == 0 {
		t.Error("expected name to declare subAttributes")
	}

	emails, ok := byName["emails"]
	if !ok {
		t.Fatal("expected emails attribute")
	}
	if !emails.MultiValued {
		t.Error("expected emails to be multiValued")
	}
	for _, unsupported := range []string{"phoneNumbers", "addresses"} {
		if _, exists := byName[unsupported]; exists {
			t.Errorf("unsupported attribute %q must not be advertised", unsupported)
		}
	}
	emailSubAttrs := make(map[string]domain.SchemaAttribute, len(emails.SubAttributes))
	for _, attr := range emails.SubAttributes {
		emailSubAttrs[attr.Name] = attr
	}
	if emailSubAttrs["type"].Type != "string" || emailSubAttrs["primary"].Type != "boolean" {
		t.Errorf("expected emails type/primary metadata, got %+v", emails.SubAttributes)
	}
}

func TestGroupCoreSchemaAttributes(t *testing.T) {
	schema := domain.GroupCoreSchema()
	byName := make(map[string]domain.SchemaAttribute, len(schema.Attributes))
	for _, attr := range schema.Attributes {
		byName[attr.Name] = attr
	}

	displayName, ok := byName["displayName"]
	if !ok {
		t.Fatal("expected displayName attribute")
	}
	if !displayName.Required {
		t.Error("expected displayName to be required")
	}

	members, ok := byName["members"]
	if !ok {
		t.Fatal("expected members attribute")
	}
	if !members.MultiValued {
		t.Error("expected members to be multiValued")
	}
	memberSubAttrs := make(map[string]domain.SchemaAttribute, len(members.SubAttributes))
	for _, attr := range members.SubAttributes {
		memberSubAttrs[attr.Name] = attr
	}
	ref, ok := memberSubAttrs["$ref"]
	if !ok {
		t.Fatal("expected members.$ref sub-attribute")
	}
	if len(ref.ReferenceTypes) != 1 || ref.ReferenceTypes[0] != "User" {
		t.Errorf("members.$ref.ReferenceTypes = %v, want [User]", ref.ReferenceTypes)
	}
	if memberSubAttrs["type"].Type != "string" {
		t.Errorf("expected members.type metadata, got %+v", members.SubAttributes)
	}
}
