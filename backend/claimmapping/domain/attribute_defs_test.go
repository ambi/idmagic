package domain_test

import (
	"testing"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

func TestMergeTenantAttributeDefsUsesBuiltinsWithoutASchema(t *testing.T) {
	got := claimdomain.MergeTenantAttributeDefs(nil)
	if want := userdomain.BuiltinUserAttributeDefs(); len(got) != len(want) {
		t.Fatalf("defs = %d, want the %d builtin defs", len(got), len(want))
	}
}

func TestMergeTenantAttributeDefsAppendsTheTenantSchema(t *testing.T) {
	custom := userdomain.UserAttributeDef{Key: "employee_number", Visibility: idmdomain.AttrVisibilityPrivate}
	got := claimdomain.MergeTenantAttributeDefs(&userdomain.TenantUserAttributeSchema{Attributes: []userdomain.UserAttributeDef{custom}})
	builtins := len(userdomain.BuiltinUserAttributeDefs())
	if len(got) != builtins+1 || got[builtins].Key != "employee_number" {
		t.Fatalf("defs = %+v, want the builtin defs followed by employee_number", got)
	}
}
