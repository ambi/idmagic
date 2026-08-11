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

// REQ-SOURCING-007: EnterpriseUserSchema advertises exactly the employeeNumber /
// department / manager subset (costCenter / division / organization stay
// unadvertised, matching the WI's Out of Scope).
// interfaces.GetScimSchemas
func TestEnterpriseUserSchemaAttributes(t *testing.T) {
	schema := domain.EnterpriseUserSchema()
	if schema.ID != domain.EnterpriseUserSchemaURN {
		t.Errorf("ID = %q, want %q", schema.ID, domain.EnterpriseUserSchemaURN)
	}
	if len(schema.Attributes) == 0 {
		t.Fatal("expected non-empty attribute list")
	}

	byName := make(map[string]domain.SchemaAttribute, len(schema.Attributes))
	for _, attr := range schema.Attributes {
		byName[attr.Name] = attr
	}

	if byName["employeeNumber"].Type != "string" {
		t.Errorf("employeeNumber.Type = %q, want string", byName["employeeNumber"].Type)
	}
	if byName["department"].Type != "string" {
		t.Errorf("department.Type = %q, want string", byName["department"].Type)
	}
	manager, ok := byName["manager"]
	if !ok {
		t.Fatal("expected manager complex attribute")
	}
	if len(manager.SubAttributes) == 0 {
		t.Error("expected manager to declare subAttributes")
	}
	managerSubAttrs := make(map[string]domain.SchemaAttribute, len(manager.SubAttributes))
	for _, attr := range manager.SubAttributes {
		managerSubAttrs[attr.Name] = attr
	}
	if managerSubAttrs["value"].Type != "string" {
		t.Errorf("manager.value.Type = %q, want string", managerSubAttrs["value"].Type)
	}

	for _, unsupported := range []string{"costCenter", "division", "organization"} {
		if _, exists := byName[unsupported]; exists {
			t.Errorf("out-of-scope attribute %q must not be advertised", unsupported)
		}
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
