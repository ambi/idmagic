package domain_test

import (
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestUserValidateAcceptsAttributes(t *testing.T) {
	u := validUser()
	u.Name = new("Alice Q")
	u.Lifecycle = idmdomain.UserLifecycle{
		Status:          idmdomain.UserStatusActive,
		RequiredActions: []idmdomain.RequiredAction{idmdomain.RequiredActionUpdatePassword},
	}
	u.Attributes = map[string]idmdomain.AttributeValue{
		"nickname":     {Type: idmdomain.AttributeTypeString, String: new("ally")},
		"region":       {Type: idmdomain.AttributeTypeString, String: new("apac")},
		"phone_number": {Type: idmdomain.AttributeTypeString, String: new("+819012345678")},
	}
	if err := u.Validate(); err != nil {
		t.Fatalf("expected valid user, got %v", err)
	}
}

func TestUserZeroLifecycleIsActive(t *testing.T) {
	u := validUser() // Lifecycle 未設定
	if !u.IsActive() {
		t.Fatal("zero-value lifecycle must be treated as active")
	}
	if u.IsDeleted() {
		t.Fatal("zero-value lifecycle must not be deleted")
	}
	if err := u.Validate(); err != nil {
		t.Fatalf("zero-value lifecycle must validate, got %v", err)
	}
}

func TestUserStatusReflectsLifecycle(t *testing.T) {
	u := validUser()
	u.Lifecycle.Status = idmdomain.UserStatusDeleted
	if u.IsActive() || !u.IsDeleted() {
		t.Fatal("deleted status must be non-active and deleted")
	}
	u.Lifecycle.Status = idmdomain.UserStatusSuspended
	if u.IsActive() {
		t.Fatal("suspended must be non-active")
	}
}

func TestUserValidateRejectsBadAttributeValue(t *testing.T) {
	u := validUser()
	u.Attributes = map[string]idmdomain.AttributeValue{
		// type と設定フィールドが不一致。
		"region": {Type: idmdomain.AttributeTypeNumber, String: new("x")},
	}
	if err := u.Validate(); err == nil {
		t.Fatal("expected error for mismatched attribute value")
	}
}

func TestAttributeValueRequiresSingleMatchingField(t *testing.T) {
	cases := []struct {
		name  string
		value idmdomain.AttributeValue
		valid bool
	}{
		{"string ok", idmdomain.AttributeValue{Type: idmdomain.AttributeTypeString, String: new("x")}, true},
		{"number ok", idmdomain.AttributeValue{Type: idmdomain.AttributeTypeNumber, Number: new(3.5)}, true},
		{"array ok", idmdomain.AttributeValue{Type: idmdomain.AttributeTypeStringArray, StringArray: []string{"a"}}, true},
		{"type/field mismatch", idmdomain.AttributeValue{Type: idmdomain.AttributeTypeNumber, String: new("x")}, false},
		{"two fields", idmdomain.AttributeValue{Type: idmdomain.AttributeTypeString, String: new("x"), Boolean: new(true)}, false},
		{"no field", idmdomain.AttributeValue{Type: idmdomain.AttributeTypeString}, false},
		{"bad type", idmdomain.AttributeValue{Type: "bogus", String: new("x")}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.value.Validate()
			if c.valid && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
			if !c.valid && err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestBuiltinUserAttributeDefsCoverOIDCAndOrg(t *testing.T) {
	defs := idmdomain.BuiltinUserAttributeDefs()
	byKey := map[string]idmdomain.UserAttributeDef{}
	for _, d := range defs {
		byKey[d.Key] = d
	}
	if _, ok := byKey["nickname"]; !ok {
		t.Fatal("expected builtin nickname")
	}
	if d := byKey["phone_number"]; d.OIDCScope == nil || *d.OIDCScope != "phone" {
		t.Fatal("phone_number must map to phone scope")
	}
	if d := byKey["title"]; d.Visibility != idmdomain.AttrVisibilitySelfReadable || d.EditableByUser {
		t.Fatal("organization title must be self-readable and admin-managed")
	}
	// 返却値の変更がカタログ本体に波及しないこと。
	defs[0].Key = "mutated"
	if idmdomain.BuiltinUserAttributeDefs()[0].Key == "mutated" {
		t.Fatal("BuiltinUserAttributeDefs must return a copy")
	}
}

func sampleSchema() idmdomain.TenantUserAttributeSchema {
	return idmdomain.TenantUserAttributeSchema{
		TenantID: spec.DefaultTenantID,
		Attributes: []idmdomain.UserAttributeDef{
			{Key: "region", Type: idmdomain.AttributeTypeString, Required: true, Visibility: idmdomain.AttrVisibilityClaimExposed, ClaimName: new("region"), PII: false},
			{Key: "tags", Type: idmdomain.AttributeTypeStringArray, MultiValued: true, Visibility: idmdomain.AttrVisibilityAdminReadable, PII: true},
		},
		UpdatedAt: time.Now().UTC(),
	}
}

func TestTenantUserAttributeSchemaValidate(t *testing.T) {
	if err := sampleSchema().Validate(); err != nil {
		t.Fatalf("expected valid schema, got %v", err)
	}
}

func TestTenantUserAttributeSchemaRejectsBadKey(t *testing.T) {
	s := sampleSchema()
	s.Attributes[0].Key = "Region-1"
	if err := s.Validate(); err == nil {
		t.Fatal("expected error for non-snake_case key")
	}
}

func TestTenantUserAttributeSchemaRejectsBuiltinCollision(t *testing.T) {
	s := sampleSchema()
	s.Attributes[0].Key = "nickname" // builtin と衝突
	if err := s.Validate(); err == nil {
		t.Fatal("expected error for builtin collision")
	}
}

func TestValidateAttributesEnforcesEffectiveSchema(t *testing.T) {
	s := sampleSchema()
	defs := s.EffectiveDefs()

	ok := map[string]idmdomain.AttributeValue{
		"region":   {Type: idmdomain.AttributeTypeString, String: new("apac")},
		"nickname": {Type: idmdomain.AttributeTypeString, String: new("ally")}, // builtin
	}
	if err := idmdomain.ValidateAttributes(ok, defs); err != nil {
		t.Fatalf("expected valid values, got %v", err)
	}

	unknown := map[string]idmdomain.AttributeValue{
		"region":  {Type: idmdomain.AttributeTypeString, String: new("apac")},
		"unknown": {Type: idmdomain.AttributeTypeString, String: new("x")},
	}
	if err := idmdomain.ValidateAttributes(unknown, defs); err == nil {
		t.Fatal("expected error for undefined key")
	}

	if err := idmdomain.ValidateAttributes(map[string]idmdomain.AttributeValue{}, defs); err == nil {
		t.Fatal("expected error for missing required attribute")
	}

	wrongType := map[string]idmdomain.AttributeValue{
		"region": {Type: idmdomain.AttributeTypeNumber, Number: new(1.0)},
	}
	if err := idmdomain.ValidateAttributes(wrongType, defs); err == nil {
		t.Fatal("expected error for type mismatch")
	}
}

func TestClaimsForScopesExposesByScope(t *testing.T) {
	u := validUser()
	u.Attributes = map[string]idmdomain.AttributeValue{
		"nickname":     {Type: idmdomain.AttributeTypeString, String: new("ally")},
		"phone_number": {Type: idmdomain.AttributeTypeString, String: new("+819012345678")},
		"title":        {Type: idmdomain.AttributeTypeString, String: new("Engineer")}, // self_readable, never a claim
	}
	defs := idmdomain.BuiltinUserAttributeDefs()

	// profile scope は nickname を開示するが phone scope 無しでは phone_number を出さない。
	claims := idmdomain.ClaimsForScopes(u, defs, []string{"openid", "profile"})
	if claims["nickname"] != "ally" {
		t.Fatalf("nickname not exposed: %#v", claims)
	}
	if _, ok := claims["phone_number"]; ok {
		t.Fatalf("phone_number exposed without phone scope: %#v", claims)
	}
	if _, ok := claims["title"]; ok {
		t.Fatalf("self_readable title must never be a claim: %#v", claims)
	}

	// phone scope を足すと phone_number が出る。
	withPhone := idmdomain.ClaimsForScopes(u, defs, []string{"openid", "profile", "phone"})
	if withPhone["phone_number"] != "+819012345678" {
		t.Fatalf("phone_number not exposed with phone scope: %#v", withPhone)
	}
}

func TestClaimsForScopesReassemblesAddress(t *testing.T) {
	u := validUser()
	u.Attributes = map[string]idmdomain.AttributeValue{
		"address_locality": {Type: idmdomain.AttributeTypeString, String: new("Tokyo")},
		"address_country":  {Type: idmdomain.AttributeTypeString, String: new("JP")},
	}
	claims := idmdomain.ClaimsForScopes(u, idmdomain.BuiltinUserAttributeDefs(), []string{"openid", "address"})
	addr, ok := claims["address"].(map[string]any)
	if !ok {
		t.Fatalf("address not reassembled into nested object: %#v", claims)
	}
	if addr["locality"] != "Tokyo" || addr["country"] != "JP" {
		t.Fatalf("address fields wrong: %#v", addr)
	}
}

func TestClaimsForScopesCustomAttributeGatedByCustomScope(t *testing.T) {
	u := validUser()
	u.Attributes = map[string]idmdomain.AttributeValue{"region": {Type: idmdomain.AttributeTypeString, String: new("apac")}}
	defs := append(idmdomain.BuiltinUserAttributeDefs(), idmdomain.UserAttributeDef{
		Key: "region", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityClaimExposed, ClaimName: new("region"),
	})

	if c := idmdomain.ClaimsForScopes(u, defs, []string{"openid", "profile"}); c["region"] != nil {
		t.Fatalf("custom attribute exposed without custom_attribute scope: %#v", c)
	}
	c := idmdomain.ClaimsForScopes(u, defs, []string{"openid", "custom_attribute"})
	if c["region"] != "apac" {
		t.Fatalf("custom attribute not exposed with custom_attribute scope: %#v", c)
	}
}

func TestBuiltinUserAttributeDefsHaveLabels(t *testing.T) {
	for _, def := range idmdomain.BuiltinUserAttributeDefs() {
		if def.Label == "" {
			t.Fatalf("builtin attribute %q has no Japanese label", def.Key)
		}
	}
}
