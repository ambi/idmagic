package domain_test

import (
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

func TestGroupValidateAcceptsEmailAndAttributes(t *testing.T) {
	now := time.Now().UTC()
	g := groupdomain.Group{
		ID: "group_x", TenantID: tenancydomain.DefaultTenantID, Name: "sales",
		Email: new("sales@example.test"),
		Attributes: map[string]userdomain.AttributeValue{
			"cost_center": {Type: idmdomain.AttributeTypeString, String: new("CC-100")},
		},
		Roles: []string{"catalog:read"}, CreatedAt: now, UpdatedAt: now,
	}
	if err := g.Validate(); err != nil {
		t.Fatalf("expected valid group, got %v", err)
	}
}

func TestGroupValidateRejectsBadEmail(t *testing.T) {
	now := time.Now().UTC()
	g := groupdomain.Group{
		ID: "group_x", TenantID: tenancydomain.DefaultTenantID, Name: "sales",
		Email: new("not-an-email"), Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	if err := g.Validate(); err == nil {
		t.Fatal("expected error for malformed email")
	}
}

func TestGroupValidateRejectsBadAttributeValue(t *testing.T) {
	now := time.Now().UTC()
	g := groupdomain.Group{
		ID: "group_x", TenantID: tenancydomain.DefaultTenantID, Name: "sales",
		Attributes: map[string]userdomain.AttributeValue{
			"cost_center": {Type: idmdomain.AttributeTypeNumber, String: new("x")},
		},
		Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	if err := g.Validate(); err == nil {
		t.Fatal("expected error for mismatched attribute value")
	}
}

func sampleGroupAttributeSchema() groupdomain.TenantGroupAttributeSchema {
	return groupdomain.TenantGroupAttributeSchema{
		TenantID: tenancydomain.DefaultTenantID,
		Attributes: []groupdomain.GroupAttributeDef{
			{Key: "cost_center", Type: idmdomain.AttributeTypeString, Required: true},
			{Key: "tags", Type: idmdomain.AttributeTypeStringArray, MultiValued: true},
		},
		UpdatedAt: time.Now().UTC(),
	}
}

func TestTenantGroupAttributeSchemaValidate(t *testing.T) {
	if err := sampleGroupAttributeSchema().Validate(); err != nil {
		t.Fatalf("expected valid schema, got %v", err)
	}
}

func TestTenantGroupAttributeSchemaRejectsBadKey(t *testing.T) {
	s := sampleGroupAttributeSchema()
	s.Attributes[0].Key = "Cost-Center"
	if err := s.Validate(); err == nil {
		t.Fatal("expected error for non-snake_case key")
	}
}

func TestTenantGroupAttributeSchemaRejectsDuplicateKey(t *testing.T) {
	s := sampleGroupAttributeSchema()
	s.Attributes[1].Key = "cost_center"
	if err := s.Validate(); err == nil {
		t.Fatal("expected error for duplicate key")
	}
}

func TestTenantGroupAttributeSchemaHasNoBuiltinCatalog(t *testing.T) {
	s := sampleGroupAttributeSchema()
	defs := s.EffectiveDefs()
	if len(defs) != len(s.Attributes) {
		t.Fatalf("expected effective defs to equal tenant-defined attributes only, got %d vs %d", len(defs), len(s.Attributes))
	}
}

func TestValidateGroupAttributesEnforcesEffectiveSchema(t *testing.T) {
	s := sampleGroupAttributeSchema()
	defs := s.EffectiveDefs()

	ok := map[string]userdomain.AttributeValue{
		"cost_center": {Type: idmdomain.AttributeTypeString, String: new("CC-100")},
	}
	if err := groupdomain.ValidateGroupAttributes(ok, defs); err != nil {
		t.Fatalf("expected valid values, got %v", err)
	}

	unknown := map[string]userdomain.AttributeValue{
		"cost_center": {Type: idmdomain.AttributeTypeString, String: new("CC-100")},
		"unknown":     {Type: idmdomain.AttributeTypeString, String: new("x")},
	}
	if err := groupdomain.ValidateGroupAttributes(unknown, defs); err == nil {
		t.Fatal("expected error for undefined key")
	}

	if err := groupdomain.ValidateGroupAttributes(map[string]userdomain.AttributeValue{}, defs); err == nil {
		t.Fatal("expected error for missing required attribute")
	}

	wrongType := map[string]userdomain.AttributeValue{
		"cost_center": {Type: idmdomain.AttributeTypeNumber, Number: new(float64)},
	}
	if err := groupdomain.ValidateGroupAttributes(wrongType, defs); err == nil {
		t.Fatal("expected error for type mismatch")
	}
}
