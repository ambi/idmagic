package sourceuser

import (
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

func full() Attributes {
	return Attributes{
		PreferredUsername: "alice", Name: "Alice L", GivenName: "Alice", FamilyName: "Liddell", Email: "a@example.test",
		EmployeeNumber: "E1", Department: "Sales", ManagerSub: new("manager"), OnPremisesImmutableID: new("anchor"),
	}
}

//spec:covers REQ-SOURCING-003, REQ-SOURCING-016: 取り込み元の属性は管理する属性を置き換え、値のない属性を消し、管理しない属性を残す。
func TestApplyReplacesManagedAttributesOnly(t *testing.T) {
	user := &userdomain.User{Attributes: map[string]userdomain.AttributeValue{
		"cost_center": {Type: idmdomain.AttributeTypeString, String: new("C1")},
	}}
	full().Apply(user)
	if !full().Matches(*user) {
		t.Fatalf("applied attributes do not match: %+v", user)
	}
	if v, _ := String(user.Attributes, "cost_center"); v != "C1" {
		t.Fatalf("an unmanaged attribute changed: %+v", user.Attributes)
	}

	cleared := Attributes{PreferredUsername: "alice"}
	cleared.Apply(user)
	for _, key := range []string{AttrEmployeeNumber, AttrDepartment} {
		if _, ok := user.Attributes[key]; ok {
			t.Errorf("%s was kept after it was cleared", key)
		}
	}
	if v, _ := String(user.Attributes, AttrManagerSub); v != "manager" {
		t.Errorf("a source that does not manage manager_sub removed it")
	}
	if v, _ := String(user.Attributes, AttrOnPremisesImmutableID); v != "anchor" {
		t.Errorf("a source that does not manage on_premises_immutable_id removed it")
	}
	if *user.Name != "" || *user.Email != "" {
		t.Errorf("cleared strings = name %q email %q", *user.Name, *user.Email)
	}
	if !cleared.Matches(*user) {
		t.Errorf("cleared attributes do not match")
	}
}

//spec:covers REQ-SOURCING-016: 管理する属性のどれか一つが違えば、取り込み元の属性は User と一致しない。
func TestMatchesNoticesEveryManagedAttribute(t *testing.T) {
	user := &userdomain.User{}
	full().Apply(user)
	variants := []func(*Attributes){
		func(a *Attributes) { a.PreferredUsername = "x" }, func(a *Attributes) { a.Name = "x" },
		func(a *Attributes) { a.GivenName = "x" }, func(a *Attributes) { a.FamilyName = "x" },
		func(a *Attributes) { a.Email = "x" }, func(a *Attributes) { a.EmployeeNumber = "x" },
		func(a *Attributes) { a.Department = "x" }, func(a *Attributes) { a.ManagerSub = new("x") },
		func(a *Attributes) { a.OnPremisesImmutableID = new("x") },
	}
	for n, change := range variants {
		changed := full()
		change(&changed)
		if changed.Matches(*user) {
			t.Errorf("variant %d matched", n)
		}
	}
	unmanaged := full()
	unmanaged.ManagerSub, unmanaged.OnPremisesImmutableID = nil, nil
	if !unmanaged.Matches(*user) {
		t.Error("a source that does not manage manager_sub or on_premises_immutable_id compared it")
	}
	if !(Attributes{}).Matches(userdomain.User{}) {
		t.Error("absent values did not match empty attributes")
	}
}

//spec:covers REQ-SOURCING-007: 空白だけの値は属性を消し、文字列でない属性は読まない。
func TestSetStringRemovesBlankValues(t *testing.T) {
	user := &userdomain.User{}
	SetString(user, AttrDepartment, "Sales")
	if v, ok := String(user.Attributes, AttrDepartment); !ok || v != "Sales" {
		t.Fatalf("String = %q, %v", v, ok)
	}
	SetString(user, AttrDepartment, "  ")
	if _, ok := user.Attributes[AttrDepartment]; ok {
		t.Fatal("a blank value was stored")
	}
	user.Attributes["n"] = userdomain.AttributeValue{Type: idmdomain.AttributeTypeNumber}
	if _, ok := String(user.Attributes, "n"); ok {
		t.Fatal("a non-string attribute was read as a string")
	}
}
