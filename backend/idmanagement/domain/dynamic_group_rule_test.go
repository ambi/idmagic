package domain

import "testing"

func TestDynamicGroupRuleCEL(t *testing.T) {
	defs := append(BuiltinUserAttributeDefs(), UserAttributeDef{Key: "skills", Type: AttributeTypeStringArray, MultiValued: true})
	department := "Engineering"
	skills := []string{"platform-go", "ops"}
	user := User{ID: "u1", PreferredUsername: "alice", Lifecycle: UserLifecycle{Status: UserStatusActive}, Attributes: map[string]AttributeValue{
		"department": {Type: AttributeTypeString, String: &department},
		"skills":     {Type: AttributeTypeStringArray, StringArray: skills},
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
	defs := BuiltinUserAttributeDefs()
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
