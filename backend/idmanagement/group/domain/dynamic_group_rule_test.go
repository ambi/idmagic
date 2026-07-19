package domain

import (
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

func TestDynamicGroupRuleCEL(t *testing.T) {
	defs := append(userdomain.BuiltinUserAttributeDefs(), userdomain.UserAttributeDef{Key: "skills", Type: idmdomain.AttributeTypeStringArray, MultiValued: true})
	department := "Engineering"
	skills := []string{"platform-go", "ops"}
	user := userdomain.User{ID: "u1", PreferredUsername: "alice", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, Attributes: map[string]userdomain.AttributeValue{
		"department": {Type: idmdomain.AttributeTypeString, String: &department},
		"skills":     {Type: idmdomain.AttributeTypeStringArray, StringArray: skills},
	}}
	for _, expression := range []string{
		`user.department == "Engineering"`,
		`user.department.lowerAscii() == "engineering"`,
		`user.skills.exists(skill, skill.matches("^platform-.*$"))`,
	} {
		compiled, err := CompileDynamicGroupRule(expression, defs)
		if err != nil {
			t.Fatalf("compile %q: %v", expression, err)
		}
		matched, err := compiled.Evaluate(user)
		if err != nil || !matched {
			t.Fatalf("evaluate %q = %v, %v", expression, matched, err)
		}
	}
}

func TestDynamicGroupRuleRejectsUnsafeSurface(t *testing.T) {
	defs := userdomain.BuiltinUserAttributeDefs()
	for _, expression := range []string{
		`user.unknown == "x"`,
		`user.department.replace("x", "y") == "z"`,
		`user.department`,
		`user.department.matches(user.email)`,
	} {
		if _, err := CompileDynamicGroupRule(expression, defs); err == nil {
			t.Fatalf("expected %q to fail", expression)
		}
	}
}
